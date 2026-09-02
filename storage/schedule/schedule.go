package schedule

import (
	"cmp"
	"container/heap"
	"context"
	"math"
	"slices"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	coretypes "github.com/projecteru2/core/types"

	"github.com/projecteru2/resource-extend/storage/types"
)

type volume struct {
	device string
	size   int64
}

type volumes []*volume

func (v volumes) DeepCopy() volumes {
	res := make(volumes, len(v))
	for i, item := range v {
		res[i] = &volume{device: item.device, size: item.size}
	}
	return res
}

type volumeHeap volumes

func (v volumeHeap) Len() int {
	return len(v)
}

func (v volumeHeap) Less(i, j int) bool {
	return compareVolume(v[i], v[j]) < 0
}

func (v volumeHeap) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v *volumeHeap) Push(x any) {
	*v = append(*v, x.(*volume))
}

func (v *volumeHeap) Pop() any {
	old := *v
	n := len(old)
	x := old[n-1]
	*v = old[:n-1]
	return x
}

type requestClasses struct {
	normal    types.VolumeBindings
	mono      types.VolumeBindings
	unlimited types.VolumeBindings
	mount     types.VolumeBindings
}

type host struct {
	maxDeployCount int
	usedVolumes    volumes
	unusedVolumes  volumes
	disks          types.Disks
	diskByPath     map[string]*types.Disk
}

func newHost(resourceInfo *types.NodeResourceInfo, maxDeployCount int) *host {
	h := &host{
		maxDeployCount: maxDeployCount,
		usedVolumes:    []*volume{},
		unusedVolumes:  []*volume{},
		disks:          resourceInfo.Capacity.Disks.DeepCopy(),
		diskByPath:     map[string]*types.Disk{},
	}

	h.disks.Sub(resourceInfo.Usage.Disks)

	for device, size := range resourceInfo.Capacity.Volumes {
		used := resourceInfo.Usage.Volumes[device]
		if used == 0 {
			h.unusedVolumes = append(h.unusedVolumes, &volume{device: device, size: size})
		} else {
			h.usedVolumes = append(h.usedVolumes, &volume{device: device, size: size - used})
		}
	}

	slices.SortStableFunc(h.unusedVolumes, compareVolume)
	slices.SortStableFunc(h.usedVolumes, compareVolume)
	return h
}

func (h *host) clone() *host {
	return &host{
		maxDeployCount: h.maxDeployCount,
		usedVolumes:    h.usedVolumes.DeepCopy(),
		unusedVolumes:  h.unusedVolumes.DeepCopy(),
		disks:          h.disks.DeepCopy(),
		diskByPath:     map[string]*types.Disk{},
	}
}

func (h *host) emptyPlans() ([]types.VolumePlan, []types.Disks) {
	return repeatWith(h.maxDeployCount, func() types.VolumePlan {
			return types.VolumePlan{}
		}), repeatWith(h.maxDeployCount, func() types.Disks {
			return types.Disks{}
		})
}

// disk entries mutate in place and are never replaced, so resolved paths stay valid for the host's lifetime
func (h *host) getDiskByPath(path string) *types.Disk {
	disk, ok := h.diskByPath[path]
	if !ok {
		disk = h.disks.GetDiskByPath(path)
		h.diskByPath[path] = disk
	}
	if disk == nil {
		return &types.Disk{}
	}
	return disk
}

func (h *host) getMonoPlan(monoRequests types.VolumeBindings, volume *volume) (types.VolumePlan, *types.Disk, error) {
	var totalSize, totalReadIOPS, totalWriteIOPS, totalReadBPS, totalWriteBPS int64
	for _, req := range monoRequests {
		totalSize += req.SizeInBytes
		totalReadIOPS += req.ReadIOPS
		totalWriteIOPS += req.WriteIOPS
		totalReadBPS += req.ReadBPS
		totalWriteBPS += req.WriteBPS
	}

	if volume.size < totalSize {
		return nil, nil, coretypes.ErrInsufficientResource
	}

	total := &types.VolumeBinding{SizeInBytes: totalSize, ReadIOPS: totalReadIOPS, WriteIOPS: totalWriteIOPS, ReadBPS: totalReadBPS, WriteBPS: totalWriteBPS}
	disk := h.getDiskByPath(volume.device)
	if !isDiskIOPSQuotaQualified(disk, total) {
		return nil, nil, coretypes.ErrInsufficientResource
	}

	volumePlan := types.VolumePlan{}
	volumeSize := volume.size

	for _, req := range monoRequests {
		size := int64(float64(req.SizeInBytes) / float64(totalSize) * float64(volumeSize))
		volumePlan[req] = types.Volumes{volume.device: size}
		volume.size -= size
	}

	if volume.size != 0 {
		volumePlan[monoRequests[0]][volume.device] += volume.size
		volume.size = 0
	}

	var diskPlan *types.Disk
	if total.RequireIOPS() {
		diskPlan = total.DiskQuota(disk)
		h.disks.Sub(types.Disks{diskPlan})
	}

	return volumePlan, diskPlan, nil
}

func (h *host) getMonoPlans(monoRequests types.VolumeBindings) ([]types.VolumePlan, []types.Disks) {
	if len(monoRequests) == 0 {
		return make([]types.VolumePlan, h.maxDeployCount), make([]types.Disks, h.maxDeployCount)
	}
	if len(h.unusedVolumes) == 0 {
		return nil, nil
	}

	volumePlans := []types.VolumePlan{}
	diskPlans := []types.Disks{}

	// h.unusedVolumes is sorted by size, so we can allocate the volumes in order
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

func (h *host) getNormalPlan(normalRequests types.VolumeBindings) (types.VolumePlan, types.Disks, error) {
	vh := volumeHeap(h.usedVolumes)
	heap.Init(&vh)

	volumePlan := types.VolumePlan{}
	diskPlan := types.Disks{}

	if len(normalRequests) == 0 {
		return volumePlan, diskPlan, nil
	}

	// normalRequests is sorted by size, so we can allocate the volumes in order
	for _, req := range normalRequests {
		volumeToPush := []*volume{}
		allocated := false

		for vh.Len() > 0 {
			volume := heap.Pop(&vh).(*volume)
			disk := h.getDiskByPath(volume.device)
			if volume.size < req.SizeInBytes || !isDiskIOPSQuotaQualified(disk, req) {
				volumeToPush = append(volumeToPush, volume)
				continue
			}
			decreaseIOPSQuota(disk, req)
			volume.size -= req.SizeInBytes
			volumePlan[req] = types.Volumes{volume.device: req.SizeInBytes}
			if req.RequireIOPS() {
				diskPlan.Add(types.Disks{req.DiskQuota(disk)})
			}
			allocated = true
			volumeToPush = append(volumeToPush, volume)
			break
		}

		for _, volume := range volumeToPush {
			heap.Push(&vh, volume)
		}
		if !allocated {
			return nil, nil, coretypes.ErrInsufficientResource
		}
	}

	return volumePlan, diskPlan, nil
}

func (h *host) getNormalPlans(normalRequests, mountRequests types.VolumeBindings, bound int) ([]types.VolumePlan, []types.Disks) {
	if len(normalRequests) == 0 && !anyRequireIOPS(mountRequests) {
		return h.emptyPlans()
	}
	if len(normalRequests) == 0 {
		return h.getMountOnlyPlans(mountRequests, bound)
	}

	volumePlans := []types.VolumePlan{}
	diskPlans := []types.Disks{}

	for len(volumePlans) < bound {
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

func (h *host) getMountOnlyPlans(mountRequests types.VolumeBindings, bound int) ([]types.VolumePlan, []types.Disks) {
	capacity, prototype := h.applyMountPasses(mountRequests, bound)
	return repeatWith(capacity, func() types.VolumePlan {
			return types.VolumePlan{}
		}), repeatWith(capacity, func() types.Disks {
			return prototype.DeepCopy()
		})
}

// reproduces the enumeration loop in closed form: same count, same final disk quotas, same trailing failed pass
func (h *host) applyMountPasses(mountRequests types.VolumeBindings, bound int) (int, types.Disks) {
	prototype, err := h.getMountDiskPlan(mountRequests)
	if err != nil || bound < 1 {
		return 0, nil
	}

	quotas := map[*types.Disk]*types.VolumeBinding{}
	for _, req := range mountRequests {
		if !req.RequireIOPS() {
			continue
		}
		disk := h.getDiskByPath(req.Source)
		sum, ok := quotas[disk]
		if !ok {
			sum = &types.VolumeBinding{}
			quotas[disk] = sum
		}
		sum.ReadIOPS += req.ReadIOPS
		sum.WriteIOPS += req.WriteIOPS
		sum.ReadBPS += req.ReadBPS
		sum.WriteBPS += req.WriteBPS
	}

	capacity := bound
	for disk, sum := range quotas {
		capacity = min(capacity, 1+quotaHeadroom(disk, sum))
	}
	for disk, sum := range quotas {
		disk.ReadIOPS -= int64(capacity-1) * sum.ReadIOPS
		disk.WriteIOPS -= int64(capacity-1) * sum.WriteIOPS
		disk.ReadBPS -= int64(capacity-1) * sum.ReadBPS
		disk.WriteBPS -= int64(capacity-1) * sum.WriteBPS
	}
	if capacity < bound {
		_, _ = h.getMountDiskPlan(mountRequests)
	}
	return capacity, prototype
}

func (h *host) getUnlimitedPlans(normalPlans, monoPlans []types.VolumePlan, unlimitedRequests types.VolumeBindings, capacity int) ([]types.VolumePlan, error) {
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

func (h *host) getMountDiskPlan(reqs types.VolumeBindings) (types.Disks, error) {
	diskPlan := types.Disks{}
	for _, req := range reqs {
		if !req.RequireIOPS() {
			continue
		}
		disk := h.getDiskByPath(req.Source)
		if !isDiskIOPSQuotaQualified(disk, req) {
			return nil, coretypes.ErrInsufficientResource
		}
		decreaseIOPSQuota(disk, req)
		diskPlan.Add(types.Disks{req.DiskQuota(disk)})
	}
	return diskPlan, nil
}

func (h *host) getVolumePlans(ctx context.Context, requests types.VolumeBindings) ([]types.VolumePlan, []types.Disks) {
	if !requests.NeedSchedule() {
		return h.emptyPlans()
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

	// the classes couple only through disk quota that an IOPS-bearing monopoly group inspects
	normalBound := math.MaxInt
	if len(monoRequests) == 0 {
		normalBound = h.maxDeployCount
	} else if !anyRequireIOPS(monoRequests) || (!anyRequireIOPS(normalRequests) && !anyRequireIOPS(mountRequests)) {
		normalBound = max(h.maxDeployCount, len(h.unusedVolumes)+1)
	}

	getPlans := func() {
		scratch := h.clone()
		normalVolumePlans, normalDiskPlans := scratch.getNormalPlans(normalRequests, mountRequests, normalBound)
		monoVolumePlans, monoDiskPlans := scratch.getMonoPlans(monoRequests)
		normalCapacity = len(normalVolumePlans)
		monoCapacity = len(monoVolumePlans)
		bestCapacity = min(normalCapacity, monoCapacity, h.maxDeployCount)
		bestVolumePlans = [2][]types.VolumePlan{normalVolumePlans, monoVolumePlans}
		bestDiskPlans = [2][]types.Disks{normalDiskPlans, monoDiskPlans}
	}

	getPlans()

	for monoCapacity > normalCapacity && normalCapacity < h.maxDeployCount {
		p, _ := slices.BinarySearchFunc(h.unusedVolumes, minNormalRequestSize, compareVolumeSize)
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
	unlimitedVolumePlans, err := h.getUnlimitedPlans(normalVolumePlans, monoVolumePlans, unlimitedRequests, bestCapacity)
	if err != nil {
		log.WithFunc("resource.storage.getVolumePlans").Error(ctx, err, "failed to get unlimited volume plans")
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

func (h *host) getMonoAffinityPlan(ctx context.Context, monoRequests types.VolumeBindings, affinity map[*types.VolumeBinding]types.Volumes, originVolumePlan types.VolumePlan) (types.VolumePlan, *types.Disk, error) {
	logger := log.WithFunc("resource.storage.getMonoAffinityPlan")
	totalRequestSize := monoRequests.TotalSize()
	totalVolumeSize := int64(0)
	for req, volumeMap := range originVolumePlan {
		if req.RequireScheduleMonopoly() {
			totalVolumeSize += volumeMap.GetSize()
		}
	}

	if totalVolumeSize < totalRequestSize {
		logger.Errorf(ctx, coretypes.ErrInsufficientResource, "no space to expand, the size of %+v is %+v, requires %+v", affinity[monoRequests[0]].GetDevice(), totalVolumeSize, totalRequestSize)
		return nil, nil, coretypes.ErrInsufficientResource
	}

	var volume *volume
	for _, volumeMap := range affinity {
		volume = h.getVolumeByDevice(volumeMap.GetDevice())
		if volume == nil {
			logger.Errorf(ctx, types.ErrInvalidVolume, "volume %s not found", volumeMap.GetDevice())
			return nil, nil, types.ErrInvalidVolume
		}
		break
	}

	return h.getMonoPlan(monoRequests, volume)
}

func (h *host) getVolumeCapacity(requests types.VolumeBindings) int {
	if !requests.NeedSchedule() {
		return h.maxDeployCount
	}

	classes := classifyVolumeBindings(requests)
	normalRequests, monoRequests := classes.normal, classes.mono
	if len(normalRequests)+len(monoRequests)+len(classes.unlimited) > 0 && len(h.unusedVolumes)+len(h.usedVolumes) == 0 {
		return 0
	}

	minNormalRequestSize := int64(math.MaxInt)
	if len(normalRequests) > 0 {
		minNormalRequestSize = normalRequests[0].SizeInBytes
	}

	normalBound := math.MaxInt
	if len(monoRequests) == 0 {
		normalBound = h.maxDeployCount
	} else if !anyRequireIOPS(monoRequests) || (!anyRequireIOPS(normalRequests) && !anyRequireIOPS(classes.mount)) {
		normalBound = max(h.maxDeployCount, len(h.unusedVolumes)+1)
	}

	var normalCapacity, monoCapacity int
	getCapacities := func() {
		scratch := h.clone()
		normalCapacity = scratch.countNormalPlans(normalRequests, classes.mount, normalBound)
		monoCapacity = scratch.countMonoPlans(monoRequests)
	}

	getCapacities()

	for monoCapacity > normalCapacity && normalCapacity < h.maxDeployCount {
		p, _ := slices.BinarySearchFunc(h.unusedVolumes, minNormalRequestSize, compareVolumeSize)
		if p == len(h.unusedVolumes) {
			break
		}
		v := h.unusedVolumes[p]
		h.unusedVolumes = slices.Delete(h.unusedVolumes, p, p+1)
		h.usedVolumes = append(h.usedVolumes, v)

		getCapacities()
	}

	return min(normalCapacity, monoCapacity, h.maxDeployCount)
}

func (h *host) countNormalPlans(normalRequests, mountRequests types.VolumeBindings, bound int) int {
	if len(normalRequests) == 0 && !anyRequireIOPS(mountRequests) {
		return h.maxDeployCount
	}
	if len(normalRequests) == 0 {
		capacity, _ := h.applyMountPasses(mountRequests, bound)
		return capacity
	}

	capacity := 0
	for capacity < bound {
		if _, _, err := h.getNormalPlan(normalRequests); err != nil {
			break
		}
		if _, err := h.getMountDiskPlan(mountRequests); err != nil {
			break
		}
		capacity++
	}
	return capacity
}

func (h *host) countMonoPlans(monoRequests types.VolumeBindings) int {
	if len(monoRequests) == 0 {
		return h.maxDeployCount
	}
	capacity := 0
	for _, volume := range h.unusedVolumes {
		if _, _, err := h.getMonoPlan(monoRequests, volume); err == nil {
			capacity++
		}
	}
	return capacity
}

func (h *host) getVolumeByDevice(device string) *volume {
	hasDevice := func(v *volume) bool { return v.device == device }
	if i := slices.IndexFunc(h.usedVolumes, hasDevice); i >= 0 {
		return h.usedVolumes[i]
	}
	if i := slices.IndexFunc(h.unusedVolumes, hasDevice); i >= 0 {
		return h.unusedVolumes[i]
	}
	return nil
}

func (h *host) getAffinityPlan(ctx context.Context, requests types.VolumeBindings, originVolumePlan types.VolumePlan, originRequests types.VolumeBindings) (types.VolumePlan, types.Disks, error) {
	logger := log.WithFunc("resource.storage.getAffinityPlan")
	if !requests.NeedSchedule() {
		return types.VolumePlan{}, types.Disks{}, nil
	}

	oldMountRequests := classifyVolumeBindings(originRequests).mount
	for _, req := range oldMountRequests {
		if req.RequireIOPS() {
			disk := h.getDiskByPath(req.Source)
			if disk.Device == "" {
				err := errors.Wrapf(types.ErrInvalidVolume, "invalid path in the old mount requests: %s", req.Source)
				logger.Errorf(ctx, err, "invalid path in the old mount requests: %s", req.Source)
				return nil, nil, err
			}
			increaseIOPSQuota(disk, req)
		}
	}

	classes := classifyVolumeBindings(requests)
	normalRequests, monoRequests := classes.normal, classes.mono
	unlimitedRequests, mountRequests := classes.unlimited, classes.mount
	needRescheduleRequests := types.VolumeBindings{}
	volumePlan := types.VolumePlan{}
	diskPlan := types.Disks{}

	for req, volumeMap := range originVolumePlan {
		volume := h.getVolumeByDevice(volumeMap.GetDevice())
		if volume == nil {
			logger.Errorf(ctx, types.ErrInvalidVolume, "volume %s not found", volumeMap.GetDevice())
			return nil, nil, types.ErrInvalidVolume
		}
		volume.size += volumeMap.GetSize()
		if req.RequireIOPS() {
			disk := h.getDiskByPath(volume.device)
			if disk.Device == "" {
				logger.Errorf(ctx, types.ErrInvalidVolume, "invalid path: %s", volume.device)
				return nil, nil, types.ErrInvalidVolume
			}
			increaseIOPSQuota(disk, req)
		}
	}

	commonProcess := func(requests types.VolumeBindings) error {
		affinity, nonAffinity := classifyAffinityRequests(requests, originVolumePlan)
		for req, volumeMap := range affinity {
			device := volumeMap.GetDevice()

			volume := h.getVolumeByDevice(device)
			if req.SizeInBytes > volume.size {
				logger.Errorf(ctx, coretypes.ErrInsufficientResource, "no space to expand, %+v remains %+v, requires %+v", device, volume.size, req.SizeInBytes)
				return coretypes.ErrInsufficientResource
			}
			volume.size -= req.SizeInBytes
			volumePlan.Merge(types.VolumePlan{req: types.Volumes{volume.device: req.SizeInBytes}})

			if !req.RequireIOPS() {
				continue
			}
			disk := h.getDiskByPath(device)
			if !isDiskIOPSQuotaQualified(disk, req) {
				logger.Errorf(ctx, coretypes.ErrInsufficientResource, "no IOPS quota to expand, %+v remains %+v, requires %+v", device, disk, req)
				return coretypes.ErrInsufficientResource
			}
			decreaseIOPSQuota(disk, req)
			diskPlan.Add(types.Disks{req.DiskQuota(disk)})
		}
		needRescheduleRequests = append(needRescheduleRequests, nonAffinity...)
		return nil
	}

	mountDiskPlan, err := h.getMountDiskPlan(mountRequests)
	if err != nil {
		logger.Error(ctx, err, "alloc mount requests failed")
		return nil, nil, err
	}
	diskPlan.Add(mountDiskPlan)

	if err = commonProcess(normalRequests); err != nil {
		logger.Error(ctx, err, "alloc normal requests failed")
		return nil, nil, err
	}

	affinity, nonAffinity := classifyAffinityRequests(monoRequests, originVolumePlan)
	if len(affinity) == 0 {
		needRescheduleRequests = append(needRescheduleRequests, nonAffinity...)
	} else {
		monoVolumePlan, monoDiskPlan, err := h.getMonoAffinityPlan(ctx, monoRequests, affinity, originVolumePlan)
		if err != nil {
			logger.Error(ctx, err, "failed to get new mono plan")
			return nil, nil, err
		}
		volumePlan.Merge(monoVolumePlan)
		if monoDiskPlan != nil {
			diskPlan.Add(types.Disks{monoDiskPlan})
		}
	}

	if err := commonProcess(unlimitedRequests); err != nil {
		logger.Error(ctx, err, "alloc unlimited requests failed")
		return nil, nil, err
	}

	if len(needRescheduleRequests) == 0 {
		return volumePlan, diskPlan, nil
	}

	volumePlans, diskPlans := h.getVolumePlans(ctx, needRescheduleRequests)
	if len(volumePlans) == 0 {
		return nil, nil, coretypes.ErrInsufficientResource
	}
	volumePlan.Merge(volumePlans[0])
	diskPlan.Add(diskPlans[0])
	return volumePlan, diskPlan, nil
}

func GetAffinityPlan(ctx context.Context, resourceInfo *types.NodeResourceInfo, volumeRequest types.VolumeBindings, originVolumePlan types.VolumePlan, originVolumeRequest types.VolumeBindings) (types.VolumePlan, types.Disks, error) {
	h := newHost(resourceInfo, 1)
	return h.getAffinityPlan(ctx, volumeRequest, originVolumePlan, originVolumeRequest)
}

func GetVolumePlans(ctx context.Context, resourceInfo *types.NodeResourceInfo, volumeRequest types.VolumeBindings, maxDeployCount int) ([]types.VolumePlan, []types.Disks) {
	h := newHost(resourceInfo, maxDeployCount)
	return h.getVolumePlans(ctx, volumeRequest)
}

// GetVolumeCapacity counts the plans GetVolumePlans would return without materializing them.
func GetVolumeCapacity(resourceInfo *types.NodeResourceInfo, volumeRequest types.VolumeBindings, maxDeployCount int) int {
	h := newHost(resourceInfo, maxDeployCount)
	return h.getVolumeCapacity(volumeRequest)
}

func quotaHeadroom(disk *types.Disk, sum *types.VolumeBinding) int {
	head := math.MaxInt
	if sum.ReadIOPS > 0 {
		head = min(head, int(disk.ReadIOPS/sum.ReadIOPS))
	}
	if sum.WriteIOPS > 0 {
		head = min(head, int(disk.WriteIOPS/sum.WriteIOPS))
	}
	if sum.ReadBPS > 0 {
		head = min(head, int(disk.ReadBPS/sum.ReadBPS))
	}
	if sum.WriteBPS > 0 {
		head = min(head, int(disk.WriteBPS/sum.WriteBPS))
	}
	return head
}

func isDiskIOPSQuotaQualified(disk *types.Disk, req *types.VolumeBinding) bool {
	return disk.ReadBPS >= req.ReadBPS && disk.WriteBPS >= req.WriteBPS && disk.ReadIOPS >= req.ReadIOPS && disk.WriteIOPS >= req.WriteIOPS
}

func decreaseIOPSQuota(disk *types.Disk, req *types.VolumeBinding) {
	disk.ReadIOPS -= req.ReadIOPS
	disk.WriteIOPS -= req.WriteIOPS
	disk.ReadBPS -= req.ReadBPS
	disk.WriteBPS -= req.WriteBPS
}

func increaseIOPSQuota(disk *types.Disk, req *types.VolumeBinding) {
	disk.ReadIOPS += req.ReadIOPS
	disk.WriteIOPS += req.WriteIOPS
	disk.ReadBPS += req.ReadBPS
	disk.WriteBPS += req.WriteBPS
}

func classifyVolumeBindings(volumeBindings types.VolumeBindings) requestClasses {
	classes := requestClasses{}
	for _, binding := range volumeBindings {
		switch {
		case binding.RequireScheduleMonopoly():
			classes.mono = append(classes.mono, binding)
		case binding.RequireScheduleUnlimitedQuota():
			classes.unlimited = append(classes.unlimited, binding)
		case binding.RequireSchedule():
			classes.normal = append(classes.normal, binding)
		default:
			classes.mount = append(classes.mount, binding)
		}
	}

	slices.SortStableFunc(classes.mono, compareBindingSize)
	slices.SortStableFunc(classes.normal, compareBindingSize)

	return classes
}

func classifyAffinityRequests(requests types.VolumeBindings, existing types.VolumePlan) (affinity map[*types.VolumeBinding]types.Volumes, nonAffinity types.VolumeBindings) {
	affinity = map[*types.VolumeBinding]types.Volumes{}
	for _, req := range requests {
		found := false
		for binding, volumeMap := range existing {
			if req.Source == binding.Source && req.Destination == binding.Destination && req.Flags == binding.Flags {
				affinity[req] = volumeMap
				found = true
				break
			}
		}
		if !found {
			nonAffinity = append(nonAffinity, req)
		}
	}
	return affinity, nonAffinity
}

func anyRequireIOPS(vbs types.VolumeBindings) bool {
	return slices.ContainsFunc(vbs, (*types.VolumeBinding).RequireIOPS)
}

func compareVolumeSize(v *volume, size int64) int {
	return cmp.Compare(v.size, size)
}

func compareVolume(v, v1 *volume) int {
	return cmp.Or(cmp.Compare(v.size, v1.size), cmp.Compare(v.device, v1.device))
}

func compareBindingSize(b, b1 *types.VolumeBinding) int {
	return cmp.Compare(b.SizeInBytes, b1.SizeInBytes)
}
