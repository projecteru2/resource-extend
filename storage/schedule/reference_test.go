package schedule

import (
	"cmp"
	"context"
	"math"
	"slices"

	coretypes "github.com/projecteru2/core/types"

	"github.com/projecteru2/resource-extend/storage/types"
)

func refGetVolumePlans(ctx context.Context, resourceInfo *types.NodeResourceInfo, volumeRequest types.VolumeBindings, maxDeployCount int) ([]types.VolumePlan, []types.Disks) {
	h := newHost(resourceInfo, maxDeployCount)
	return h.refVolumePlans(ctx, volumeRequest)
}

func (h *host) refVolumePlans(ctx context.Context, requests types.VolumeBindings) ([]types.VolumePlan, []types.Disks) {
	if !requests.NeedSchedule() {
		return h.refEmptyPlans()
	}

	classes := classifyVolumeBindings(requests)
	normalRequests, monoRequests := classes.normal, classes.mono
	unlimitedRequests, mountRequests := classes.unlimited, classes.mount
	if len(normalRequests)+len(monoRequests)+len(unlimitedRequests) > 0 && len(h.unusedVolumes)+len(h.usedVolumes) == 0 {
		return nil, nil
	}

	minNormalRequestSize := int64(math.MaxInt)
	if len(normalRequests) > 0 {
		minNormalRequestSize = normalRequests[0].SizeInBytes
	}

	var (
		normalCapacity, monoCapacity, bestCapacity int
		bestVolumePlans                            [2][]types.VolumePlan
		bestDiskPlans                              [2][]types.Disks
	)

	getPlans := func() {
		scratch := h.clone()
		normalVolumePlans, normalDiskPlans := scratch.refNormalPlans(normalRequests, mountRequests)
		monoVolumePlans, monoDiskPlans := scratch.refMonoPlans(monoRequests)
		normalCapacity = len(normalVolumePlans)
		monoCapacity = len(monoVolumePlans)
		bestCapacity = min(normalCapacity, monoCapacity, h.maxDeployCount)
		bestVolumePlans = [2][]types.VolumePlan{normalVolumePlans, monoVolumePlans}
		bestDiskPlans = [2][]types.Disks{normalDiskPlans, monoDiskPlans}
	}

	getPlans()

	for monoCapacity > normalCapacity {
		p, _ := slices.BinarySearchFunc(h.unusedVolumes, minNormalRequestSize, func(v *volume, size int64) int { return cmp.Compare(v.size, size) })
		if p == len(h.unusedVolumes) {
			break
		}
		v := h.unusedVolumes[p]
		h.unusedVolumes = slices.Delete(h.unusedVolumes, p, p+1)
		h.usedVolumes = append(h.usedVolumes, v)

		getPlans()
	}

	normalVolumePlans, monoVolumePlans := bestVolumePlans[0], bestVolumePlans[1]
	normalDiskPlans, monoDiskPlans := bestDiskPlans[0], bestDiskPlans[1]
	unlimitedVolumePlans, err := h.refUnlimitedPlans(normalVolumePlans, monoVolumePlans, unlimitedRequests, bestCapacity)
	if err != nil {
		return nil, nil
	}

	resVolumePlans := make([]types.VolumePlan, bestCapacity)
	resDiskPlans := make([]types.Disks, bestCapacity)

	for i := range bestCapacity {
		resVolumePlans[i] = normalVolumePlans[i]
		resVolumePlans[i].Merge(monoVolumePlans[i])
		resVolumePlans[i].Merge(unlimitedVolumePlans[i])
		resDiskPlans[i] = normalDiskPlans[i]
		resDiskPlans[i].Add(monoDiskPlans[i])
	}

	return resVolumePlans, resDiskPlans
}

func (h *host) refNormalPlans(normalRequests, mountRequests types.VolumeBindings) ([]types.VolumePlan, []types.Disks) {
	needScheduleMountRequest := slices.ContainsFunc(mountRequests, func(req *types.VolumeBinding) bool { return req.RequireIOPS() })
	if len(normalRequests) == 0 && !needScheduleMountRequest {
		return h.refEmptyPlans()
	}

	volumePlans := []types.VolumePlan{}
	diskPlans := []types.Disks{}

	for {
		volumePlan, diskPlan, err := h.getNormalPlan(normalRequests)
		if err != nil {
			break
		}
		mountDiskPlan, err := h.getMountDiskPlan(mountRequests)
		if err != nil {
			break
		}
		diskPlan.Add(mountDiskPlan)
		volumePlans = append(volumePlans, volumePlan)
		diskPlans = append(diskPlans, diskPlan)
	}

	return volumePlans, diskPlans
}

func (h *host) refMonoPlans(monoRequests types.VolumeBindings) ([]types.VolumePlan, []types.Disks) {
	if len(monoRequests) == 0 {
		return make([]types.VolumePlan, h.maxDeployCount), make([]types.Disks, h.maxDeployCount)
	}
	if len(h.unusedVolumes) == 0 {
		return nil, nil
	}

	volumePlans := []types.VolumePlan{}
	diskPlans := []types.Disks{}

	for _, volume := range h.unusedVolumes {
		volumePlan, diskPlan, err := h.getMonoPlan(monoRequests, volume)
		if err != nil {
			continue
		}
		monoDisks := types.Disks{}
		if diskPlan != nil {
			monoDisks = types.Disks{diskPlan}
		}
		volumePlans = append(volumePlans, volumePlan)
		diskPlans = append(diskPlans, monoDisks)
	}

	return volumePlans, diskPlans
}

func (h *host) refUnlimitedPlans(normalPlans, monoPlans []types.VolumePlan, unlimitedRequests types.VolumeBindings, capacity int) ([]types.VolumePlan, error) {
	if len(unlimitedRequests) == 0 {
		return make([]types.VolumePlan, capacity), nil
	}
	allVolumes := slices.Concat(h.usedVolumes.DeepCopy(), h.unusedVolumes.DeepCopy())
	if len(allVolumes) == 0 {
		return nil, coretypes.ErrInsufficientResource
	}
	volumeMap := map[string]*volume{}
	for _, vol := range allVolumes {
		volumeMap[vol.device] = vol
	}

	for _, plans := range [2][]types.VolumePlan{normalPlans, monoPlans} {
		for _, plan := range plans {
			for _, vm := range plan {
				volumeMap[vm.GetDevice()].size -= vm.GetSize()
			}
		}
	}

	volumeWithLargestSize := slices.MaxFunc(allVolumes, func(a, b *volume) int { return cmp.Compare(a.size, b.size) })

	return repeatWith(capacity, func() types.VolumePlan {
		volumePlan := types.VolumePlan{}
		for _, req := range unlimitedRequests {
			volumePlan[req] = types.Volumes{volumeWithLargestSize.device: req.SizeInBytes}
		}
		return volumePlan
	}), nil
}

func (h *host) refEmptyPlans() ([]types.VolumePlan, []types.Disks) {
	return repeatWith(h.maxDeployCount, func() types.VolumePlan {
			return types.VolumePlan{}
		}), repeatWith(h.maxDeployCount, func() types.Disks {
			return types.Disks{}
		})
}
