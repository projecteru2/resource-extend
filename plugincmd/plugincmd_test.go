package plugincmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cockroachdb/errors"
	enginetypes "github.com/projecteru2/core/engine/types"
	"github.com/projecteru2/core/resource/plugins/binary"
	binarytypes "github.com/projecteru2/core/resource/plugins/binary/types"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	resourcetypes "github.com/projecteru2/core/resource/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/stretchr/testify/assert"
)

func TestHandlersDecodeCoreRequest(t *testing.T) {
	tests := []struct {
		name  string
		req   any
		run   handler
		check func(*testing.T, *stubPlugin)
	}{
		{
			name: "add-node",
			req: &binarytypes.AddNodeRequest{
				Nodename: "node0",
				Resource: plugintypes.NodeResource{"prod_count_map": map[string]int{"nvidia-3070": 2}},
				Info:     &enginetypes.Info{NCPU: 8, StorageTotal: 1024, Resources: map[string][]byte{"gpu": []byte(`{"a":1}`)}},
			},
			run: addNode,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, "node0", s.nodename)
				assert.Equal(t, 8, s.info.NCPU)
				assert.Equal(t, int64(1024), s.info.StorageTotal)
				assert.Equal(t, []byte(`{"a":1}`), s.info.Resources["gpu"])
				assert.NotNil(t, s.resourceRequest["prod_count_map"])
			},
		},
		{
			name: "remove-node",
			req:  &binarytypes.RemoveNodeRequest{Nodename: "node0"},
			run:  removeNode,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, "node0", s.nodename)
			},
		},
		{
			name: "get-nodes-deploy-capacity",
			req: &binarytypes.GetNodesDeployCapacityRequest{
				Nodenames:        []string{"node0", "node1"},
				WorkloadResource: plugintypes.WorkloadResource{"storage": "1G"},
			},
			run: getNodesDeployCapacity,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, []string{"node0", "node1"}, s.nodenames)
				assert.Equal(t, "1G", s.workloadResource["storage"])
			},
		},
		{
			name: "set-node-resource-capacity",
			req: &binarytypes.SetNodeResourceCapacityRequest{
				Nodename:        "node0",
				Resource:        plugintypes.NodeResource{"storage": 1},
				ResourceRequest: plugintypes.NodeResource{"storage": "2G"},
				Delta:           true,
				Incr:            true,
			},
			run: setNodeResourceCapacity,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, "node0", s.nodename)
				assert.Equal(t, float64(1), s.resource["storage"])
				assert.Equal(t, "2G", s.resourceRequest["storage"])
				assert.True(t, s.delta)
				assert.True(t, s.incr)
			},
		},
		{
			name: "set-node-resource-usage",
			req: &binarytypes.SetNodeResourceUsageRequest{
				Nodename:          "node0",
				Resource:          plugintypes.NodeResource{"storage": 1},
				ResourceRequest:   plugintypes.NodeResource{"storage": "2G"},
				WorkloadsResource: []plugintypes.WorkloadResource{{"storage_request": 3}},
			},
			run: setNodeResourceUsage,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, float64(1), s.resource["storage"])
				assert.Equal(t, "2G", s.resourceRequest["storage"])
				assert.Len(t, s.workloadsResource, 1)
				assert.False(t, s.delta)
				assert.False(t, s.incr)
			},
		},
		{
			name: "set-node-resource-info",
			req: &binarytypes.SetNodeResourceInfoRequest{
				Nodename: "node0",
				Capacity: plugintypes.NodeResource{"storage": 1},
				Usage:    plugintypes.NodeResource{"storage": 2},
			},
			run: setNodeResourceInfo,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, float64(1), s.capacity["storage"])
				assert.Equal(t, float64(2), s.usage["storage"])
			},
		},
		{
			name: "fix-node-resource",
			req: &binarytypes.GetNodeResourceInfoRequest{
				Nodename:          "node0",
				WorkloadsResource: []plugintypes.WorkloadResource{{"storage_request": 1}, {"storage_limit": 2}},
			},
			run: fixNodeResource,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Len(t, s.workloadsResource, 2)
			},
		},
		{
			name: "get-most-idle-node",
			req:  &binarytypes.GetMostIdleNodeRequest{Nodenames: []string{"node0"}},
			run:  getMostIdleNode,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, []string{"node0"}, s.nodenames)
			},
		},
		{
			name: "get-metrics",
			req:  &binarytypes.GetMetricsRequest{Nodes: []plugintypes.NodeRef{{Podname: "pod0", Nodename: "node0"}}},
			run:  getMetrics,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, "pod0", s.podname)
				assert.Equal(t, "node0", s.nodename)
			},
		},
		{
			name: "calculate-deploy",
			req: &binarytypes.CalculateDeployRequest{
				Nodename:                "node0",
				DeployCount:             3,
				WorkloadResourceRequest: plugintypes.WorkloadResourceRequest{"storage": "1G"},
			},
			run: calculateDeploy,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, 3, s.deployCount)
				assert.Equal(t, "1G", s.workloadResourceRequest["storage"])
			},
		},
		{
			name: "calculate-realloc",
			req: &binarytypes.CalculateReallocRequest{
				Nodename:                "node0",
				WorkloadResource:        plugintypes.WorkloadResource{"storage_request": 1},
				WorkloadResourceRequest: plugintypes.WorkloadResourceRequest{"storage": "1G"},
			},
			run: calculateRealloc,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Equal(t, float64(1), s.workloadResource["storage_request"])
				assert.Equal(t, "1G", s.workloadResourceRequest["storage"])
			},
		},
		{
			name: "calculate-remap",
			req: &binarytypes.CalculateRemapRequest{
				Nodename:          "node0",
				WorkloadsResource: map[string]plugintypes.WorkloadResource{"wrk0": {"storage_request": 1}},
			},
			run: calculateRemap,
			check: func(t *testing.T, s *stubPlugin) {
				assert.Len(t, s.workloadsResourceMap, 1)
				assert.Equal(t, float64(1), s.workloadsResourceMap["wrk0"]["storage_request"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stubPlugin{}
			_, err := tt.run(t.Context(), s, encodeRequest(t, tt.req))
			assert.NoError(t, err)
			tt.check(t, s)
		})
	}
}

func TestVerbsLeaveOutTheUnsupportedOnes(t *testing.T) {
	r := &runner{unsupported: []string{binary.CalculateRemapCommand}}
	names := []string{}
	for _, c := range r.commands() {
		names = append(names, c.Name)
	}
	assert.Contains(t, names, verbsCommand)
	assert.Contains(t, names, binary.CalculateDeployCommand)
	assert.NotContains(t, names, binary.CalculateRemapCommand)
}

func TestHandlersRejectEmptyNodename(t *testing.T) {
	handlers := map[string]handler{
		"add-node":                   addNode,
		"remove-node":                removeNode,
		"set-node-resource-capacity": setNodeResourceCapacity,
		"get-node-resource-info":     getNodeResourceInfo,
		"set-node-resource-info":     setNodeResourceInfo,
		"set-node-resource-usage":    setNodeResourceUsage,
		"fix-node-resource":          fixNodeResource,
		"calculate-deploy":           calculateDeploy,
		"calculate-realloc":          calculateRealloc,
		"calculate-remap":            calculateRemap,
		"get-nodes-deploy-capacity":  getNodesDeployCapacity,
		"get-most-idle-node":         getMostIdleNode,
	}
	for name, run := range handlers {
		t.Run(name, func(t *testing.T) {
			_, err := run(t.Context(), &stubPlugin{}, resourcetypes.RawParams{})
			assert.ErrorIs(t, err, coretypes.ErrEmptyNodeName)
		})
	}
}

func TestGetNodeResourceInfoTolerateUnknownNode(t *testing.T) {
	s := &stubPlugin{err: errors.Wrap(coretypes.ErrNodeNotExists, "key: node0")}
	out, err := getNodeResourceInfo(t.Context(), s, resourcetypes.RawParams{"nodename": "node0"})
	assert.NoError(t, err)

	data, err := json.Marshal(out)
	assert.NoError(t, err)
	assert.Equal(t, "null", string(data))
}

func TestGetNodeResourceInfoPropagatesOtherErrors(t *testing.T) {
	s := &stubPlugin{err: coretypes.ErrInvaildCount}
	_, err := getNodeResourceInfo(t.Context(), s, resourcetypes.RawParams{"nodename": "node0"})
	assert.ErrorIs(t, err, coretypes.ErrInvaildCount)
}

func encodeRequest(t *testing.T, req any) resourcetypes.RawParams {
	t.Helper()
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	in := resourcetypes.RawParams{}
	if err := json.Unmarshal(data, &in); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	return in
}

type stubPlugin struct {
	err error

	nodename             string
	nodenames            []string
	podname              string
	deployCount          int
	delta                bool
	incr                 bool
	info                 *enginetypes.Info
	resource             plugintypes.NodeResource
	resourceRequest      plugintypes.NodeResourceRequest
	capacity             plugintypes.NodeResource
	usage                plugintypes.NodeResource
	workloadResource     plugintypes.WorkloadResource
	workloadsResource    []plugintypes.WorkloadResource
	workloadsResourceMap map[string]plugintypes.WorkloadResource

	workloadResourceRequest plugintypes.WorkloadResourceRequest
}

func (s *stubPlugin) Name() string { return "stub" }

func (s *stubPlugin) AddNode(_ context.Context, nodename string, resource plugintypes.NodeResourceRequest, info *enginetypes.Info) (*plugintypes.AddNodeResponse, error) {
	s.nodename, s.resourceRequest, s.info = nodename, resource, info
	return &plugintypes.AddNodeResponse{}, s.err
}

func (s *stubPlugin) RemoveNode(_ context.Context, nodename string) (*plugintypes.RemoveNodeResponse, error) {
	s.nodename = nodename
	return &plugintypes.RemoveNodeResponse{}, s.err
}

func (s *stubPlugin) GetNodesDeployCapacity(_ context.Context, nodenames []string, resource plugintypes.WorkloadResourceRequest) (*plugintypes.GetNodesDeployCapacityResponse, error) {
	s.nodenames, s.workloadResource = nodenames, resource
	return &plugintypes.GetNodesDeployCapacityResponse{}, s.err
}

func (s *stubPlugin) SetNodeResourceCapacity(_ context.Context, nodename string, resource plugintypes.NodeResource, resourceRequest plugintypes.NodeResourceRequest, delta, incr bool) (*plugintypes.SetNodeResourceCapacityResponse, error) {
	s.nodename, s.resource, s.resourceRequest, s.delta, s.incr = nodename, resource, resourceRequest, delta, incr
	return &plugintypes.SetNodeResourceCapacityResponse{}, s.err
}

func (s *stubPlugin) GetNodeResourceInfo(_ context.Context, nodename string, workloadsResource []plugintypes.WorkloadResource) (*plugintypes.GetNodeResourceInfoResponse, error) {
	s.nodename, s.workloadsResource = nodename, workloadsResource
	if s.err != nil {
		return nil, s.err
	}
	return &plugintypes.GetNodeResourceInfoResponse{}, nil
}

func (s *stubPlugin) SetNodeResourceInfo(_ context.Context, nodename string, capacity, usage plugintypes.NodeResource) (*plugintypes.SetNodeResourceInfoResponse, error) {
	s.nodename, s.capacity, s.usage = nodename, capacity, usage
	return &plugintypes.SetNodeResourceInfoResponse{}, s.err
}

func (s *stubPlugin) SetNodeResourceUsage(_ context.Context, nodename string, resource plugintypes.NodeResource, resourceRequest plugintypes.NodeResourceRequest, workloadsResource []plugintypes.WorkloadResource, delta, incr bool) (*plugintypes.SetNodeResourceUsageResponse, error) {
	s.nodename, s.resource, s.resourceRequest, s.workloadsResource, s.delta, s.incr = nodename, resource, resourceRequest, workloadsResource, delta, incr
	return &plugintypes.SetNodeResourceUsageResponse{}, s.err
}

func (s *stubPlugin) GetMostIdleNode(_ context.Context, nodenames []string) (*plugintypes.GetMostIdleNodeResponse, error) {
	s.nodenames = nodenames
	return &plugintypes.GetMostIdleNodeResponse{}, s.err
}

func (s *stubPlugin) FixNodeResource(_ context.Context, nodename string, workloadsResource []plugintypes.WorkloadResource) (*plugintypes.GetNodeResourceInfoResponse, error) {
	s.nodename, s.workloadsResource = nodename, workloadsResource
	return &plugintypes.GetNodeResourceInfoResponse{}, s.err
}

func (s *stubPlugin) GetMetricsDescription(context.Context) (*plugintypes.GetMetricsDescriptionResponse, error) {
	return &plugintypes.GetMetricsDescriptionResponse{}, s.err
}

func (s *stubPlugin) GetMetrics(_ context.Context, nodes []plugintypes.NodeRef) (*plugintypes.GetMetricsResponse, error) {
	s.podname, s.nodename = nodes[0].Podname, nodes[0].Nodename
	return &plugintypes.GetMetricsResponse{}, s.err
}

func (s *stubPlugin) CalculateDeploy(_ context.Context, nodename string, deployCount int, resourceRequest plugintypes.WorkloadResourceRequest) (*plugintypes.CalculateDeployResponse, error) {
	s.nodename, s.deployCount, s.workloadResourceRequest = nodename, deployCount, resourceRequest
	return &plugintypes.CalculateDeployResponse{}, s.err
}

func (s *stubPlugin) CalculateRealloc(_ context.Context, nodename string, resource plugintypes.WorkloadResource, resourceRequest plugintypes.WorkloadResourceRequest) (*plugintypes.CalculateReallocResponse, error) {
	s.nodename, s.workloadResource, s.workloadResourceRequest = nodename, resource, resourceRequest
	return &plugintypes.CalculateReallocResponse{}, s.err
}

func (s *stubPlugin) CalculateRemap(_ context.Context, nodename string, workloadsResource map[string]plugintypes.WorkloadResource) (*plugintypes.CalculateRemapResponse, error) {
	s.nodename, s.workloadsResourceMap = nodename, workloadsResource
	return &plugintypes.CalculateRemapResponse{}, s.err
}
