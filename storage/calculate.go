package storage

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/projecteru2/core/utils"
	"github.com/sanity-io/litter"

	"github.com/projecteru2/resource-extend/storage/schedule"
	storagetypes "github.com/projecteru2/resource-extend/storage/types"
)

func (p Plugin) CalculateDeploy(ctx context.Context, nodename string, deployCount int, resourceRequest plugintypes.WorkloadResourceRequest) (*plugintypes.CalculateDeployResponse, error) {
	logger := log.WithFunc("resource.storage.CalculateDeploy").WithField("node", nodename)
	req := &storagetypes.WorkloadResourceRequest{}
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

	enginesParams, workloadsResource, err := p.doAlloc(ctx, nodeResourceInfo, deployCount, req)
	if err != nil {
		return nil, err
	}

	return &plugintypes.CalculateDeployResponse{
		EnginesParams:     utils.Map(enginesParams, (*storagetypes.EngineParams).AsRawParams),
		WorkloadsResource: utils.Map(workloadsResource, (*storagetypes.WorkloadResource).AsRawParams),
	}, nil
}

func (p Plugin) CalculateRealloc(ctx context.Context, nodename string, resource plugintypes.WorkloadResource, resourceRequest plugintypes.WorkloadResourceRequest) (*plugintypes.CalculateReallocResponse, error) {
	logger := log.WithFunc("resource.storage.CalculateRealloc").WithField("node", nodename)
	req := &storagetypes.WorkloadResourceRequest{}
	if err := req.Parse(resourceRequest); err != nil {
		return nil, err
	}
	originResource := &storagetypes.WorkloadResource{}
	if err := originResource.Parse(resource); err != nil {
		return nil, err
	}
	resourceInfo, err := p.store.Get(ctx, nodename)
	if err != nil {
		logger.Error(ctx, err, "failed to get resource info of node")
		return nil, err
	}

	needVolumeReschedule := req.VolumesRequest.NeedSchedule()

	req = &storagetypes.WorkloadResourceRequest{
		VolumesRequest: storagetypes.MergeVolumeBindings(req.VolumesRequest, originResource.VolumesRequest),
		VolumesLimit:   storagetypes.MergeVolumeBindings(req.VolumesLimit, originResource.VolumesLimit),
		StorageRequest: req.StorageRequest + originResource.StorageRequest + req.VolumesRequest.TotalSize(),
		StorageLimit:   req.StorageLimit + originResource.StorageLimit + req.VolumesLimit.TotalSize(),
	}
	req.SkipAddStorage()

	if err = req.Validate(); err != nil {
		logger.Errorf(ctx, err, "invalid resource opts %+v", litter.Sdump(req))
		return nil, err
	}

	targetWorkloadResource := &storagetypes.WorkloadResource{
		VolumesRequest:    req.VolumesRequest,
		VolumesLimit:      req.VolumesLimit,
		VolumePlanRequest: nil,
		VolumePlanLimit:   nil,
		StorageRequest:    req.StorageRequest,
		StorageLimit:      req.StorageLimit,
	}

	if targetWorkloadResource.StorageRequest-originResource.StorageRequest > resourceInfo.Capacity.Storage-resourceInfo.Usage.Storage {
		return nil, coretypes.ErrInsufficientResource
	}

	var volumePlan storagetypes.VolumePlan
	var diskPlan storagetypes.Disks

	if needVolumeReschedule {
		if volumePlan, diskPlan, err = schedule.GetAffinityPlan(ctx, resourceInfo, req.VolumesRequest, originResource.VolumePlanRequest, originResource.VolumesRequest); err != nil {
			return nil, coretypes.ErrInsufficientResource
		}
	} else {
		volumePlan = originResource.VolumePlanRequest
		diskPlan = originResource.DisksRequest
	}

	targetWorkloadResource.VolumePlanRequest = volumePlan
	targetWorkloadResource.DisksRequest = diskPlan
	targetWorkloadResource.VolumePlanLimit = getVolumePlanLimit(targetWorkloadResource.VolumesRequest, targetWorkloadResource.VolumesLimit, volumePlan)
	targetWorkloadResource.DisksLimit = getDisksLimit(req.VolumesLimit, targetWorkloadResource.VolumePlanLimit, resourceInfo.Capacity.Disks)

	originBindingSet := map[[3]string]struct{}{}
	for _, binding := range originResource.VolumesLimit.ApplyPlan(originResource.VolumePlanLimit) {
		originBindingSet[binding.GetMapKey()] = struct{}{}
	}

	engineParams := &storagetypes.EngineParams{Storage: targetWorkloadResource.StorageLimit, IOPSOptions: toIOPSOptions(targetWorkloadResource.DisksLimit)}
	newBindings := req.VolumesLimit.ApplyPlan(volumePlan)
	if len(newBindings) != len(originBindingSet) {
		engineParams.VolumeChanged = true
	}
	for _, binding := range newBindings {
		engineParams.Volumes = append(engineParams.Volumes, binding.ToString(true))
		if _, ok := originBindingSet[binding.GetMapKey()]; !ok {
			engineParams.VolumeChanged = true
		}
	}

	deltaWorkloadResource := getDeltaWorkloadResourceArgs(originResource, targetWorkloadResource)

	return &plugintypes.CalculateReallocResponse{
		EngineParams:     engineParams.AsRawParams(),
		DeltaResource:    deltaWorkloadResource.AsRawParams(),
		WorkloadResource: targetWorkloadResource.AsRawParams(),
	}, nil
}

func (p Plugin) CalculateRemap(context.Context, string, map[string]plugintypes.WorkloadResource) (*plugintypes.CalculateRemapResponse, error) {
	return &plugintypes.CalculateRemapResponse{}, nil
}

func (p Plugin) doAlloc(ctx context.Context, resourceInfo *storagetypes.NodeResourceInfo, deployCount int, req *storagetypes.WorkloadResourceRequest) ([]*storagetypes.EngineParams, []*storagetypes.WorkloadResource, error) {
	if req.StorageRequest > 0 {
		storageCapacity := int((resourceInfo.Capacity.Storage - resourceInfo.Usage.Storage) / req.StorageRequest)
		if storageCapacity < deployCount {
			return nil, nil, errors.Wrapf(coretypes.ErrInsufficientResource, "not enough storage, request: %+v, available: %+v", req.StorageRequest, storageCapacity)
		}
	}

	var enginesParams []*storagetypes.EngineParams
	var workloadsResource []*storagetypes.WorkloadResource

	var volumePlans []storagetypes.VolumePlan
	var diskPlans []storagetypes.Disks

	if !req.VolumesRequest.NeedSchedule() {
		for range deployCount {
			volumePlans = append(volumePlans, storagetypes.VolumePlan{})
			diskPlans = append(diskPlans, storagetypes.Disks{})
		}
	} else {
		volumePlans, diskPlans = schedule.GetVolumePlans(ctx, resourceInfo, req.VolumesRequest, deployCount)
		if len(volumePlans) < deployCount {
			return nil, nil, errors.Wrapf(coretypes.ErrInsufficientResource, "not enough volume plan, need %+v, available %+v", deployCount, len(volumePlans))
		}
	}

	for index, volumePlan := range volumePlans {
		engineParam := &storagetypes.EngineParams{Storage: req.StorageLimit}
		for _, binding := range req.VolumesLimit.ApplyPlan(volumePlan) {
			engineParam.Volumes = append(engineParam.Volumes, binding.ToString(true))
		}

		volumePlanLimit := getVolumePlanLimit(req.VolumesLimit, req.VolumesLimit, volumePlan)
		disksLimit := getDisksLimit(req.VolumesLimit, volumePlanLimit, resourceInfo.Capacity.Disks)

		engineParam.IOPSOptions = toIOPSOptions(disksLimit)

		workloadResource := &storagetypes.WorkloadResource{
			VolumesRequest:    req.VolumesRequest,
			VolumesLimit:      req.VolumesLimit,
			VolumePlanRequest: volumePlan,
			VolumePlanLimit:   volumePlanLimit,
			StorageRequest:    req.StorageRequest,
			StorageLimit:      req.StorageLimit,
			DisksRequest:      diskPlans[index],
			DisksLimit:        disksLimit,
		}

		enginesParams = append(enginesParams, engineParam)
		workloadsResource = append(workloadsResource, workloadResource)
	}

	return enginesParams, workloadsResource, nil
}
