package gpu

import (
	"encoding/json"
	"testing"

	enginetypes "github.com/projecteru2/core/engine/types"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	resourcetypes "github.com/projecteru2/core/resource/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/projecteru2/resource-extend/gpu/types"
)

const gb = 1000 * 1000 * 1000

func TestAddNode(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]
	nodeForAdd := "test2"

	req := plugintypes.NodeResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 2,
		},
	}

	info := &enginetypes.Info{NCPU: 2, MemTotal: 4 * gb}

	_, err := cm.AddNode(ctx, node, req, info)
	assert.Equal(t, err, coretypes.ErrNodeExists)

	r, err := cm.AddNode(ctx, "xxx", nil, nil)
	assert.Nil(t, err)
	cv := parseNodeResource(t, r.Capacity)
	nr, err := cm.GetNodeResourceInfo(ctx, "xxx", nil)
	assert.Nil(t, err)
	cv = parseNodeResource(t, nr.Capacity)
	assert.Equal(t, cv.Count(), 0)
	assert.NotNil(t, cv.ProdCountMap)
	cm.RemoveNode(ctx, "xxx")

	r, err = cm.AddNode(ctx, nodeForAdd, req, info)
	assert.Nil(t, err)
	cv = parseNodeResource(t, r.Capacity)
	assert.Equal(t, cv.Count(), 2)

	nRes := types.NodeResource{
		ProdCountMap: types.ProdCountMap{
			"nvidia-3070": 2,
		},
	}
	data, err := json.Marshal(&nRes)
	assert.Nil(t, err)
	eInfo := &enginetypes.Info{
		Resources: map[string][]byte{
			"gpu": data,
		},
	}
	r, err = cm.AddNode(ctx, "xxx1", nil, eInfo)
	assert.Nil(t, err)

	nr, err = cm.GetNodeResourceInfo(ctx, "xxx1", nil)
	assert.Nil(t, err)
	cv = parseNodeResource(t, nr.Capacity)
	assert.Equal(t, cv.Count(), 2)
	assert.NotNil(t, cv.ProdCountMap)
	cm.RemoveNode(ctx, "xxx1")
}

func TestRemoveNode(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]
	nodeForDel := "test2"

	_, err := cm.RemoveNode(ctx, "xxx")
	assert.Nil(t, err)

	_, err = cm.RemoveNode(ctx, node)
	assert.Nil(t, err)
	_, err = cm.RemoveNode(ctx, nodeForDel)
	assert.Nil(t, err)
}

func TestGetNodesDeployCapacity(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateEmptyNodes(ctx, t, cm, 2, 0)
	r, err := cm.GetNodesDeployCapacity(ctx, nodes, nil)
	assert.Nil(t, err)
	assert.Equal(t, 2*maxCapacity, r.Total)
	for _, node := range nodes {
		nodeCap := r.NodeDeployCapacityMap[node]
		assert.Equal(t, maxCapacity, nodeCap.Capacity)
		assert.Equal(t, float64(0), nodeCap.Usage)
		assert.Equal(t, float64(0), nodeCap.Rate)
	}

	nodes = generateNodes(ctx, t, cm, 2, 0)

	req := plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 2,
		},
	}

	_, err = cm.GetNodesDeployCapacity(ctx, []string{"xxx"}, req)
	assert.ErrorIs(t, err, coretypes.ErrInvaildCount)

	r, err = cm.GetNodesDeployCapacity(ctx, nodes, nil)
	assert.Nil(t, err)
	assert.Equal(t, 2*maxCapacity, r.Total)
	for _, node := range nodes {
		nodeCap := r.NodeDeployCapacityMap[node]
		assert.Equal(t, maxCapacity, nodeCap.Capacity)
	}

	r, err = cm.GetNodesDeployCapacity(ctx, nodes, req)
	assert.Nil(t, err)
	assert.Equal(t, 4, r.Total)

	req = plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 3,
		},
	}
	r, err = cm.GetNodesDeployCapacity(ctx, nodes, req)
	assert.Nil(t, err)
	assert.Equal(t, 2, r.Total)
	for _, node := range nodes {
		nodeCap := r.NodeDeployCapacityMap[node]
		assert.Equal(t, 1, nodeCap.Capacity)
	}

	req = plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 5,
		},
	}
	r, err = cm.GetNodesDeployCapacity(ctx, nodes, req)
	assert.Nil(t, err)
	assert.Equal(t, 0, r.Total)
	assert.Len(t, r.NodeDeployCapacityMap, 0)

	req = plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
			"nvidia-3090": 1,
		},
	}
	r, err = cm.GetNodesDeployCapacity(ctx, nodes, req)
	assert.Nil(t, err)
	assert.Equal(t, 8, r.Total)
	for _, node := range nodes {
		nodeCap := r.NodeDeployCapacityMap[node]
		assert.Equal(t, 4, nodeCap.Capacity)
	}

	req = plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
			"nvidia-3090": 2,
		},
	}
	r, err = cm.GetNodesDeployCapacity(ctx, nodes, req)
	assert.Nil(t, err)
	assert.Equal(t, 4, r.Total)
	for _, node := range nodes {
		nodeCap := r.NodeDeployCapacityMap[node]
		assert.Equal(t, 2, nodeCap.Capacity)
	}

	req = plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 2,
			"nvidia-3090": 2,
		},
	}
	r, err = cm.GetNodesDeployCapacity(ctx, nodes, req)
	assert.Nil(t, err)
	assert.Equal(t, 4, r.Total)
	for _, node := range nodes {
		nodeCap := r.NodeDeployCapacityMap[node]
		assert.Equal(t, 2, nodeCap.Capacity)
	}

	req = plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 4,
			"nvidia-3090": 4,
		},
	}
	r, err = cm.GetNodesDeployCapacity(ctx, nodes, req)
	assert.Nil(t, err)
	assert.Equal(t, 2, r.Total)
	for _, node := range nodes {
		nodeCap := r.NodeDeployCapacityMap[node]
		assert.Equal(t, 1, nodeCap.Capacity)
	}

	req = plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 5,
			"nvidia-3090": 4,
		},
	}
	r, err = cm.GetNodesDeployCapacity(ctx, nodes, req)
	assert.Nil(t, err)
	assert.Equal(t, 0, r.Total)
	assert.Len(t, r.NodeDeployCapacityMap, 0)
}

func TestSetNodeResourceCapacity(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]

	gr, err := cm.GetNodeResourceInfo(ctx, node, nil)
	assert.Nil(t, err)
	capacity := parseNodeResource(t, gr.Capacity)
	assert.Equal(t, capacity.Count(), 8)

	nodeResource := plugintypes.NodeResource{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
		},
	}

	nodeResourceRequest := plugintypes.NodeResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
		},
	}

	r, err := cm.SetNodeResourceCapacity(ctx, node, nil, nil, true, true)
	assert.Nil(t, err)
	v := parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 8)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nil, nil, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 8)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nil, nodeResourceRequest, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 9)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nil, nodeResourceRequest, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 8)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nodeResource, nil, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 9)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nil, nodeResource, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 8)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nil, nodeResourceRequest, false, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nodeResource, nil, false, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nodeResource, nodeResourceRequest, false, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceCapacity(ctx, node, nil, nil, false, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)

	nodeResourceRequest1 := plugintypes.NodeResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
		},
	}
	r, err = cm.SetNodeResourceCapacity(ctx, node, nil, nodeResourceRequest1, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	nodeResourceRequest1 = plugintypes.NodeResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": -1,
		},
	}
	r, err = cm.SetNodeResourceCapacity(ctx, node, nil, nodeResourceRequest1, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)
}

func TestGetAndFixNodeResourceInfo(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]

	_, err := cm.GetNodeResourceInfo(ctx, "xxx", nil)
	assert.ErrorIs(t, err, coretypes.ErrNodeNotExists)

	r, err := cm.GetNodeResourceInfo(ctx, node, nil)
	assert.Nil(t, err)
	assert.Len(t, r.Diffs, 0)

	workloadsResource := []plugintypes.WorkloadResource{
		{
			"prod_count_map": types.ProdCountMap{
				"nvidia-3070": 1,
				"nvidia-3090": 1,
			},
		},
	}
	r, err = cm.GetNodeResourceInfo(ctx, node, workloadsResource)
	assert.Nil(t, err)
	assert.Len(t, r.Diffs, 3)

	r, err = cm.FixNodeResource(ctx, node, workloadsResource)
	assert.Nil(t, err)
	assert.Len(t, r.Diffs, 3)
	usage := parseNodeResource(t, r.Usage)
	assert.Equal(t, usage.Count(), 2)
}

func TestSetNodeResourceInfo(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]

	r, err := cm.GetNodeResourceInfo(ctx, node, nil)
	assert.Nil(t, err)
	capacity := parseNodeResource(t, r.Capacity)
	usage := parseNodeResource(t, r.Usage)
	assert.Equal(t, 8, capacity.Count())
	assert.Equal(t, 0, usage.Count())

	rcv := resourcetypes.RawParams{
		"prod_count_map": capacity.ProdCountMap,
	}
	ucv := resourcetypes.RawParams{
		"prod_count_map": usage.ProdCountMap,
	}
	_, err = cm.SetNodeResourceInfo(ctx, "node-2", rcv, ucv)
	assert.Nil(t, err)
}

func TestSetNodeResourceUsage(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]

	gr, err := cm.GetNodeResourceInfo(ctx, node, nil)
	assert.Nil(t, err)
	usage := parseNodeResource(t, gr.Usage)
	assert.Equal(t, usage.Count(), 0)

	nodeResource := plugintypes.NodeResource{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
		},
	}

	nodeResourceRequest := plugintypes.NodeResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
		},
	}

	workloadsResource := []plugintypes.WorkloadResource{
		{
			"prod_count_map": types.ProdCountMap{
				"nvidia-3070": 1,
			},
		},
	}

	r, err := cm.SetNodeResourceUsage(ctx, node, nil, nil, nil, true, true)
	assert.Nil(t, err)
	v := parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nil, nil, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nodeResourceRequest, nil, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nodeResourceRequest, nil, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)

	r, err = cm.SetNodeResourceUsage(ctx, node, nodeResource, nil, nil, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nodeResource, nil, nil, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nil, workloadsResource, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nil, workloadsResource, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nil, nil, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nodeResourceRequest, nil, false, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nodeResource, nil, nil, false, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nil, workloadsResource, false, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nodeResource, nodeResourceRequest, nil, false, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nodeResourceRequest, workloadsResource, false, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nodeResource, nil, workloadsResource, false, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nodeResource, nodeResourceRequest, workloadsResource, false, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	r, err = cm.SetNodeResourceUsage(ctx, node, nodeResource, nodeResourceRequest, workloadsResource, true, false)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)

	nodeResourceRequest1 := plugintypes.NodeResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
		},
	}
	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nodeResourceRequest1, nil, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 1)

	nodeResourceRequest1 = plugintypes.NodeResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": -1,
		},
	}
	r, err = cm.SetNodeResourceUsage(ctx, node, nil, nodeResourceRequest1, nil, true, true)
	assert.Nil(t, err)
	v = parseNodeResource(t, r.After)
	assert.Equal(t, v.Count(), 0)
}

func TestGetMostIdleNode(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 2, 0)

	usage := plugintypes.NodeResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
		},
	}

	_, err := cm.SetNodeResourceUsage(ctx, nodes[1], usage, nil, nil, false, false)
	assert.Nil(t, err)

	r, err := cm.GetMostIdleNode(ctx, nodes)
	assert.Nil(t, err)
	assert.Equal(t, r.Nodename, nodes[0])

	nodes = append(nodes, "node-x")
	_, err = cm.GetMostIdleNode(ctx, nodes)
	assert.Error(t, err)
}

func parseNodeResource(t *testing.T, raw resourcetypes.RawParams) *types.NodeResource {
	res := &types.NodeResource{}
	assert.NoError(t, res.Parse(raw))
	return res
}
