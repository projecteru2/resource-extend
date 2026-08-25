package gpu

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/cockroachdb/errors"
	enginetypes "github.com/projecteru2/core/engine/types"
	"github.com/projecteru2/core/log"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/sanity-io/litter"

	gputypes "github.com/projecteru2/resource-extend/gpu/types"
)

const maxCapacity = 1000000

func (p Plugin) AddNode(ctx context.Context, nodename string, resource plugintypes.NodeResourceRequest, info *enginetypes.Info) (*plugintypes.AddNodeResponse, error) {
	switch _, err := p.store.Get(ctx, nodename); {
	case err == nil:
		return nil, coretypes.ErrNodeExists
	case !errors.IsAny(err, coretypes.ErrInvaildCount, coretypes.ErrNodeNotExists):
		log.WithFunc("resource.gpu.AddNode").WithField("node", nodename).Error(ctx, err, "failed to get resource info of node")
		return nil, err
	}

	req := &gputypes.NodeResourceRequest{}
	if err := req.Parse(resource); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	capacity := gputypes.NewNodeResource(req.ProdCountMap)
	if info != nil && capacity.Count() == 0 {
		if b, ok := info.Resources[name]; ok {
			if err := json.Unmarshal(b, capacity); err != nil {
				return nil, err
			}
		}
	}
	nodeResourceInfo := &gputypes.NodeResourceInfo{
		Capacity: capacity,
		Usage:    gputypes.NewNodeResource(nil),
	}

	if err := p.store.Put(ctx, nodename, nodeResourceInfo); err != nil {
		return nil, err
	}
	return &plugintypes.AddNodeResponse{
		Capacity: nodeResourceInfo.Capacity.AsRawParams(),
		Usage:    nodeResourceInfo.Usage.AsRawParams(),
	}, nil
}

func (p Plugin) RemoveNode(ctx context.Context, nodename string) (*plugintypes.RemoveNodeResponse, error) {
	err := p.store.Delete(ctx, nodename)
	if err != nil {
		log.WithFunc("resource.gpu.RemoveNode").WithField("node", nodename).Error(ctx, err, "failed to delete node")
	}
	return &plugintypes.RemoveNodeResponse{}, err
}

func (p Plugin) GetNodesDeployCapacity(ctx context.Context, nodenames []string, resource plugintypes.WorkloadResourceRequest) (*plugintypes.GetNodesDeployCapacityResponse, error) {
	logger := log.WithFunc("resource.gpu.GetNodesDeployCapacity")
	req := &gputypes.WorkloadResourceRequest{}
	if err := req.Parse(resource); err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Errorf(ctx, err, "invalid resource opts %+v", req)
		return nil, err
	}

	nodesDeployCapacityMap := map[string]*plugintypes.NodeDeployCapacity{}
	total := 0

	nodesResourceInfos, err := p.store.GetMulti(ctx, nodenames)
	if err != nil {
		return nil, err
	}

	for nodename, nodeResourceInfo := range nodesResourceInfos {
		nodeDeployCapacity := p.doGetNodeDeployCapacity(nodeResourceInfo, req)
		if nodeDeployCapacity.Capacity > 0 {
			nodesDeployCapacityMap[nodename] = nodeDeployCapacity
			total += nodeDeployCapacity.Capacity
		}
	}
	return &plugintypes.GetNodesDeployCapacityResponse{
		NodeDeployCapacityMap: nodesDeployCapacityMap,
		Total:                 total,
	}, nil
}

func (p Plugin) SetNodeResourceCapacity(ctx context.Context, nodename string, resource plugintypes.NodeResource, resourceRequest plugintypes.NodeResourceRequest, delta, incr bool) (*plugintypes.SetNodeResourceCapacityResponse, error) {
	logger := log.WithFunc("resource.gpu.SetNodeResourceCapacity").WithField("node", nodename)
	req, nodeResource, _, err := p.parseNodeResourceInfos(resourceRequest, resource, nil)
	if err != nil {
		return nil, err
	}
	nodeResourceInfo, err := p.store.Get(ctx, nodename)
	if err != nil {
		return nil, err
	}

	origin := nodeResourceInfo.Capacity
	before := origin.DeepCopy()

	if !delta && req != nil {
		req.LoadFromOrigin(origin, resourceRequest)
	}
	nodeResourceInfo.Capacity = p.calculateNodeResource(req, nodeResource, origin, nil, delta, incr)

	if err := p.store.Put(ctx, nodename, nodeResourceInfo); err != nil {
		logger.Errorf(ctx, err, "node resource info %+v", litter.Sdump(nodeResourceInfo))
		return nil, err
	}

	return &plugintypes.SetNodeResourceCapacityResponse{
		Before: before.AsRawParams(),
		After:  nodeResourceInfo.Capacity.AsRawParams(),
	}, nil
}

func (p Plugin) GetNodeResourceInfo(ctx context.Context, nodename string, workloadsResource []plugintypes.WorkloadResource) (*plugintypes.GetNodeResourceInfoResponse, error) {
	nodeResourceInfo, _, diffs, err := p.getNodeResourceInfo(ctx, nodename, workloadsResource)
	if err != nil {
		return nil, err
	}

	return &plugintypes.GetNodeResourceInfoResponse{
		Capacity: nodeResourceInfo.Capacity.AsRawParams(),
		Usage:    nodeResourceInfo.Usage.AsRawParams(),
		Diffs:    diffs,
	}, nil
}

func (p Plugin) SetNodeResourceInfo(ctx context.Context, nodename string, capacity, usage plugintypes.NodeResource) (*plugintypes.SetNodeResourceInfoResponse, error) {
	capacityResource := &gputypes.NodeResource{}
	usageResource := &gputypes.NodeResource{}
	if err := capacityResource.Parse(capacity); err != nil {
		return nil, err
	}
	if err := usageResource.Parse(usage); err != nil {
		return nil, err
	}
	resourceInfo := &gputypes.NodeResourceInfo{
		Capacity: capacityResource,
		Usage:    usageResource,
	}

	return &plugintypes.SetNodeResourceInfoResponse{}, p.store.Put(ctx, nodename, resourceInfo)
}

func (p Plugin) SetNodeResourceUsage(ctx context.Context, nodename string, resource plugintypes.NodeResource, resourceRequest plugintypes.NodeResourceRequest, workloadsResource []plugintypes.WorkloadResource, delta, incr bool) (*plugintypes.SetNodeResourceUsageResponse, error) {
	logger := log.WithFunc("resource.gpu.SetNodeResourceUsage").WithField("node", nodename)
	req, nodeResource, wrksResource, err := p.parseNodeResourceInfos(resourceRequest, resource, workloadsResource)
	if err != nil {
		return nil, err
	}
	nodeResourceInfo, err := p.store.Get(ctx, nodename)
	if err != nil {
		return nil, err
	}

	origin := nodeResourceInfo.Usage
	before := origin.DeepCopy()

	nodeResourceInfo.Usage = p.calculateNodeResource(req, nodeResource, origin, wrksResource, delta, incr)

	if err := p.store.Put(ctx, nodename, nodeResourceInfo); err != nil {
		logger.Errorf(ctx, err, "node resource info %+v", litter.Sdump(nodeResourceInfo))
		return nil, err
	}

	return &plugintypes.SetNodeResourceUsageResponse{
		Before: before.AsRawParams(),
		After:  nodeResourceInfo.Usage.AsRawParams(),
	}, nil
}

func (p Plugin) GetMostIdleNode(ctx context.Context, nodenames []string) (*plugintypes.GetMostIdleNodeResponse, error) {
	var mostIdleNode string
	minIdle := math.MaxFloat64

	nodesResourceInfo, err := p.store.GetMulti(ctx, nodenames)
	if err != nil {
		return nil, err
	}

	for nodename, nodeResourceInfo := range nodesResourceInfo {
		var idle float64
		if nodeResourceInfo.CapCount() > 0 {
			idle = float64(nodeResourceInfo.UsageCount()) / float64(nodeResourceInfo.CapCount())
		}

		if idle < minIdle {
			mostIdleNode = nodename
			minIdle = idle
		}
	}
	return &plugintypes.GetMostIdleNodeResponse{
		Nodename: mostIdleNode,
		Priority: priority,
	}, nil
}

func (p Plugin) FixNodeResource(ctx context.Context, nodename string, workloadsResource []plugintypes.WorkloadResource) (*plugintypes.GetNodeResourceInfoResponse, error) {
	nodeResourceInfo, actuallyWorkloadsUsage, diffs, err := p.getNodeResourceInfo(ctx, nodename, workloadsResource)
	if err != nil {
		return nil, err
	}

	if len(diffs) != 0 {
		nodeResourceInfo.Usage = &gputypes.NodeResource{
			ProdCountMap: actuallyWorkloadsUsage.ProdCountMap,
		}
		if err = p.store.Put(ctx, nodename, nodeResourceInfo); err != nil {
			log.WithFunc("resource.gpu.FixNodeResource").Error(ctx, err)
			diffs = append(diffs, err.Error())
		}
	}
	return &plugintypes.GetNodeResourceInfoResponse{
		Capacity: nodeResourceInfo.Capacity.AsRawParams(),
		Usage:    nodeResourceInfo.Usage.AsRawParams(),
		Diffs:    diffs,
	}, nil
}

func (p Plugin) getNodeResourceInfo(ctx context.Context, nodename string, workloadsResource []plugintypes.WorkloadResource) (*gputypes.NodeResourceInfo, *gputypes.WorkloadResource, []string, error) {
	logger := log.WithFunc("resource.gpu.getNodeResourceInfo").WithField("node", nodename)
	nodeResourceInfo, err := p.store.Get(ctx, nodename)
	if err != nil {
		logger.Error(ctx, err)
		return nodeResourceInfo, nil, nil, err
	}

	actuallyWorkloadsUsage := &gputypes.WorkloadResource{ProdCountMap: gputypes.ProdCountMap{}}
	for _, workloadResource := range workloadsResource {
		workloadUsage := &gputypes.WorkloadResource{}
		if err := workloadUsage.Parse(workloadResource); err != nil {
			logger.Error(ctx, err)
			return nil, nil, nil, err
		}
		actuallyWorkloadsUsage.Add(workloadUsage)
	}

	diffs := []string{}

	if actuallyWorkloadsUsage.Count() != nodeResourceInfo.UsageCount() {
		diffs = append(diffs, fmt.Sprintf("node.usage != sum(workload.request): %d != %d", nodeResourceInfo.UsageCount(), actuallyWorkloadsUsage.Count()))
	}
	for prod, count1 := range actuallyWorkloadsUsage.ProdCountMap {
		count2, ok := nodeResourceInfo.Usage.ProdCountMap[prod]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("%s not in usage", prod))
			continue
		}
		if count1 != count2 {
			diffs = append(diffs, fmt.Sprintf("%s: actual(%d) != usage(%d)", prod, count1, count2))
		}
	}

	return nodeResourceInfo, actuallyWorkloadsUsage, diffs, nil
}

func (p Plugin) doGetNodeDeployCapacity(nodeResourceInfo *gputypes.NodeResourceInfo, req *gputypes.WorkloadResourceRequest) *plugintypes.NodeDeployCapacity {
	availableResource := nodeResourceInfo.GetAvailableResource()

	capacityInfo := &plugintypes.NodeDeployCapacity{
		Weight:   1,
		Capacity: maxCapacity,
	}
	if req.Count() > 0 {
		for reqProd, reqCount := range req.ProdCountMap {
			count := availableResource.ProdCountMap[reqProd]
			capacityInfo.Capacity = min(capacityInfo.Capacity, count/reqCount)
			if capacityInfo.Capacity <= 0 {
				capacityInfo.Capacity = 0
				break
			}
		}
	}
	if nodeResourceInfo.CapCount() > 0 {
		capacityInfo.Usage = float64(nodeResourceInfo.UsageCount()) / float64(nodeResourceInfo.CapCount())
		capacityInfo.Rate = float64(req.Count()) / float64(nodeResourceInfo.CapCount())
	}
	return capacityInfo
}

// calculateNodeResource priority: node resource request > node resource > workload resource args list
func (p Plugin) calculateNodeResource(req *gputypes.NodeResourceRequest, nodeResource, origin *gputypes.NodeResource, workloadsResource []*gputypes.WorkloadResource, delta, incr bool) *gputypes.NodeResource {
	var resp *gputypes.NodeResource
	if origin == nil || !delta {
		resp = gputypes.NewNodeResource(nil)
		incr = true
	} else {
		resp = origin.DeepCopy()
	}

	if req != nil {
		nodeResource = &gputypes.NodeResource{ProdCountMap: req.ProdCountMap}
	}

	if nodeResource != nil {
		if incr {
			resp.Add(nodeResource)
		} else {
			resp.Sub(nodeResource)
		}
		return resp
	}

	for _, workloadResource := range workloadsResource {
		nodeResource = &gputypes.NodeResource{ProdCountMap: workloadResource.ProdCountMap}
		if incr {
			resp.Add(nodeResource)
		} else {
			resp.Sub(nodeResource)
		}
	}
	return resp
}

func (p Plugin) parseNodeResourceInfos(resourceRequest plugintypes.NodeResourceRequest, resource plugintypes.NodeResource, workloadsResource []plugintypes.WorkloadResource) (*gputypes.NodeResourceRequest, *gputypes.NodeResource, []*gputypes.WorkloadResource, error) {
	var req *gputypes.NodeResourceRequest
	var nodeResource *gputypes.NodeResource
	wrksResource := []*gputypes.WorkloadResource{}

	if resourceRequest != nil {
		req = &gputypes.NodeResourceRequest{}
		if err := req.Parse(resourceRequest); err != nil {
			return nil, nil, nil, err
		}
	}

	if resource != nil {
		nodeResource = &gputypes.NodeResource{}
		if err := nodeResource.Parse(resource); err != nil {
			return nil, nil, nil, err
		}
	}

	for _, workloadResource := range workloadsResource {
		wrkResource := &gputypes.WorkloadResource{}
		if err := wrkResource.Parse(workloadResource); err != nil {
			return nil, nil, nil, err
		}
		wrksResource = append(wrksResource, wrkResource)
	}

	return req, nodeResource, wrksResource, nil
}
