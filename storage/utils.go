package storage

import (
	"fmt"

	storagetypes "github.com/projecteru2/resource-extend/storage/types"
)

func toIOPSOptions(disks storagetypes.Disks) map[string]string {
	iopsOptions := make(map[string]string, len(disks))
	for _, disk := range disks {
		iopsOptions[disk.Device] = fmt.Sprintf("%d:%d:%d:%d", disk.ReadIOPS, disk.WriteIOPS, disk.ReadBPS, disk.WriteBPS)
	}
	return iopsOptions
}

func getVolumePlanLimit(volumeRequest, volumeLimit storagetypes.VolumeBindings, volumePlan storagetypes.VolumePlan) storagetypes.VolumePlan {
	volumePlanLimit := storagetypes.VolumePlan{}

	volumeBindingToVolumes := map[[3]string]storagetypes.Volumes{}
	for binding, volumeMap := range volumePlan {
		volumeBindingToVolumes[binding.GetMapKey()] = volumeMap
	}

	for index, binding := range volumeLimit {
		if !binding.RequireSchedule() {
			continue
		}
		if volumeMap, ok := volumeBindingToVolumes[binding.GetMapKey()]; ok {
			volumePlanLimit[binding] = storagetypes.Volumes{volumeMap.GetDevice(): volumeMap.GetSize() + binding.SizeInBytes - volumeRequest[index].SizeInBytes}
		}
	}
	return volumePlanLimit
}

func getDisksLimit(volumeLimit storagetypes.VolumeBindings, volumePlanLimit storagetypes.VolumePlan, disks storagetypes.Disks) storagetypes.Disks {
	disksLimit := storagetypes.Disks{}
	for _, binding := range volumeLimit {
		if !binding.RequireIOPS() || binding.RequireSchedule() {
			continue
		}
		if disk := disks.GetDiskByPath(binding.Source); disk != nil {
			disksLimit.Add(getDiskLimit(disk, binding))
		}
	}
	for binding, volumeMap := range volumePlanLimit {
		if !binding.RequireIOPS() {
			continue
		}
		if disk := disks.GetDiskByPath(volumeMap.GetDevice()); disk != nil {
			disksLimit.Add(getDiskLimit(disk, binding))
		}
	}
	return disksLimit
}

func getDiskLimit(disk *storagetypes.Disk, binding *storagetypes.VolumeBinding) storagetypes.Disks {
	return storagetypes.Disks{{
		Device:    disk.Device,
		Mounts:    disk.Mounts,
		ReadIOPS:  binding.ReadIOPS,
		WriteIOPS: binding.WriteIOPS,
		ReadBPS:   binding.ReadBPS,
		WriteBPS:  binding.WriteBPS,
	}}
}

func getDeltaWorkloadResourceArgs(originResource, targetWorkloadResource *storagetypes.WorkloadResource) *storagetypes.WorkloadResource {
	deltaVolumes := storagetypes.Volumes{}
	for _, volumeMap := range targetWorkloadResource.VolumePlanRequest {
		deltaVolumes.Add(volumeMap)
	}
	for _, volumeMap := range originResource.VolumePlanRequest {
		deltaVolumes.Sub(volumeMap)
	}

	deltaDisks := targetWorkloadResource.DisksRequest.DeepCopy()
	deltaDisks.Sub(originResource.DisksRequest)

	return &storagetypes.WorkloadResource{
		VolumePlanRequest: storagetypes.VolumePlan{&storagetypes.VolumeBinding{
			Source:      "fake-source",
			Destination: "fake-destination",
			Flags:       "fake-flags",
			SizeInBytes: 0,
		}: deltaVolumes},
		StorageRequest: targetWorkloadResource.StorageRequest - originResource.StorageRequest,
		DisksRequest:   deltaDisks,
	}
}
