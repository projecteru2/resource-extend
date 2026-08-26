package gpu

import (
	"context"

	"github.com/projecteru2/core/log"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/projecteru2/core/utils"

	gputypes "github.com/projecteru2/resource-extend/gpu/types"
)

func (p Plugin) CalculateDeploy(ctx context.Context, nodename string, deployCount int, resourceRequest plugintypes.WorkloadResourceRequest) (*plugintypes.CalculateDeployResponse, error) {
	logger := log.WithFunc("resource.gpu.CalculateDeploy").WithField("node", nodename)
	req := &gputypes.WorkloadResourceRequest{}
	if err := req.Parse(resourceRequest); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		logger.Errorf(ctx, err, "invalid resource opts %+v", req)
		return nil, err
	}

	nodeResourceInfo, err := p.store.Get(ctx, nodename)
	if err != nil {
		logger.Error(ctx, err, "failed to get resource info of node")
		return nil, err
	}

	enginesParams, workloadsResource, err := p.doAlloc(nodeResourceInfo, deployCount, req)
	if err != nil {
		return nil, err
	}

	return &plugintypes.CalculateDeployResponse{
		EnginesParams:     utils.Map(enginesParams, (*gputypes.EngineParams).AsRawParams),
		WorkloadsResource: utils.Map(workloadsResource, (*gputypes.WorkloadResource).AsRawParams),
	}, nil
}

func (p Plugin) CalculateRealloc(ctx context.Context, nodename string, resource plugintypes.WorkloadResource, resourceRequest plugintypes.WorkloadResourceRequest) (*plugintypes.CalculateReallocResponse, error) {
	req := &gputypes.WorkloadResourceRequest{}
	if err := req.Parse(resourceRequest); err != nil {
		return nil, err
	}
	if err := req.ValidateProd(); err != nil {
		return nil, err
	}
	originResource := &gputypes.WorkloadResource{}
	if err := originResource.Parse(resource); err != nil {
		return nil, err
	}
	if err := originResource.Validate(); err != nil {
		return nil, err
	}
	nodeResourceInfo, err := p.store.Get(ctx, nodename)
	if err != nil {
		log.WithFunc("resource.gpu.CalculateRealloc").WithField("node", nodename).Error(ctx, err, "failed to get resource info of node")
		return nil, err
	}

	nodeResourceInfo.Usage.Sub(&gputypes.NodeResource{
		ProdCountMap: originResource.ProdCountMap,
	})

	newReq := req.DeepCopy()
	newReq.MergeFromResource(originResource)

	if err = newReq.Validate(); err != nil {
		return nil, err
	}

	enginesParams, workloadsResource, err := p.doAlloc(nodeResourceInfo, 1, newReq)
	if err != nil {
		return nil, err
	}

	engineParams := enginesParams[0]
	newResource := workloadsResource[0]

	deltaWorkloadResource := newResource.DeepCopy()
	deltaWorkloadResource.Sub(originResource)

	return &plugintypes.CalculateReallocResponse{
		EngineParams:     engineParams.AsRawParams(),
		DeltaResource:    deltaWorkloadResource.AsRawParams(),
		WorkloadResource: newResource.AsRawParams(),
	}, nil
}

func (p Plugin) CalculateRemap(context.Context, string, map[string]plugintypes.WorkloadResource) (*plugintypes.CalculateRemapResponse, error) {
	return &plugintypes.CalculateRemapResponse{}, nil
}

func (p Plugin) doAlloc(resourceInfo *gputypes.NodeResourceInfo, deployCount int, req *gputypes.WorkloadResourceRequest) ([]*gputypes.EngineParams, []*gputypes.WorkloadResource, error) {
	enginesParams := []*gputypes.EngineParams{}
	workloadsResource := []*gputypes.WorkloadResource{}

	availableResource := resourceInfo.GetAvailableResource()
	for range deployCount {
		prodCountMap := gputypes.ProdCountMap{}
		for reqProd, reqCount := range req.ProdCountMap {
			capCount, ok := availableResource.ProdCountMap[reqProd]
			if !ok || capCount < reqCount {
				return enginesParams, workloadsResource, coretypes.ErrInsufficientResource
			}
			availableResource.ProdCountMap[reqProd] -= reqCount
			prodCountMap[reqProd] = reqCount
		}
		workloadsResource = append(workloadsResource, &gputypes.WorkloadResource{ProdCountMap: prodCountMap.DeepCopy()})
		enginesParams = append(enginesParams, &gputypes.EngineParams{ProdCountMap: prodCountMap})
	}
	return enginesParams, workloadsResource, nil
}
