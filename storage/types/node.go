package types

import (
	"maps"
	"slices"
	"strings"

	"github.com/cockroachdb/errors"
	resourcetypes "github.com/projecteru2/core/resource/types"
	"github.com/projecteru2/core/utils"
)

type NodeResource struct {
	Volumes Volumes `json:"volumes"`
	Disks   Disks   `json:"disks"`
	Storage int64   `json:"storage"`
}

func (n *NodeResource) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		"volumes": n.Volumes,
		"disks":   n.Disks,
		"storage": n.Storage,
	}
}

func (n *NodeResource) Parse(rawParams resourcetypes.RawParams) error {
	return resourcetypes.Decode(rawParams, n)
}

func (n *NodeResource) DeepCopy() *NodeResource {
	return &NodeResource{Volumes: n.Volumes.DeepCopy(), Storage: n.Storage, Disks: n.Disks.DeepCopy()}
}

func (n *NodeResource) RemoveEmpty() {
	maps.DeleteFunc(n.Volumes, func(_ string, size int64) bool { return size == 0 })
}

func (n *NodeResource) Add(n1 *NodeResource) {
	n.Volumes.Add(n1.Volumes)
	n.Storage += n1.Storage
	n.Disks.Add(n1.Disks)
}

func (n *NodeResource) Sub(n1 *NodeResource) {
	n.Volumes.Sub(n1.Volumes)
	n.Storage -= n1.Storage
	n.Disks.Sub(n1.Disks)
}

type NodeResourceInfo struct {
	Capacity *NodeResource `json:"capacity"`
	Usage    *NodeResource `json:"usage"`
}

// NewNodeResourceInfo is the empty record of a node this plugin has nothing stored for.
func NewNodeResourceInfo() *NodeResourceInfo {
	return &NodeResourceInfo{
		Capacity: &NodeResource{Volumes: Volumes{}, Disks: Disks{}},
		Usage:    &NodeResource{Volumes: Volumes{}, Disks: Disks{}},
	}
}

func (n *NodeResourceInfo) Validate() error {
	if n.Capacity == nil {
		return ErrInvalidCapacity
	}
	if n.Usage == nil {
		n.Usage = &NodeResource{Volumes: Volumes{}, Disks: Disks{}}
		for device := range n.Capacity.Volumes {
			n.Usage.Volumes[device] = 0
		}
		for _, disk := range n.Capacity.Disks {
			n.Usage.Disks = append(n.Usage.Disks, &Disk{Device: disk.Device, Mounts: disk.Mounts})
		}
	}

	slices.SortFunc(n.Usage.Disks, compareDiskDevice)
	slices.SortFunc(n.Capacity.Disks, compareDiskDevice)

	return errors.Join(n.validateVolume(), n.validateStorage(), n.validateDisks())
}

func (n *NodeResourceInfo) validateDisks() error {
	for _, disk := range n.Capacity.Disks {
		if disk.ReadIOPS < 0 || disk.WriteIOPS < 0 || disk.ReadBPS < 0 || disk.WriteBPS < 0 {
			return errors.Wrap(ErrInvalidDisk, "disk IOPS / BPS can't be negative")
		}

		usage := n.Usage.Disks.GetDiskByDevice(disk.Device)
		if usage == nil {
			continue
		}
		usage.Mounts = disk.Mounts
		if usage.ReadIOPS < 0 || usage.WriteIOPS < 0 || usage.ReadBPS < 0 || usage.WriteBPS < 0 {
			return errors.Wrap(ErrInvalidDisk, "disk IOPS / BPS can't be negative")
		}
		if usage.ReadIOPS > disk.ReadIOPS || usage.WriteIOPS > disk.WriteIOPS || usage.ReadBPS > disk.ReadBPS || usage.WriteBPS > disk.WriteBPS {
			return errors.Wrap(ErrInvalidDisk, "disk IOPS / BPS usage can't be greater than capacity")
		}
	}

	toRemoveMap := map[string]struct{}{}
	for _, disk := range n.Usage.Disks {
		capacity := n.Capacity.Disks.GetDiskByDevice(disk.Device)
		if capacity == nil {
			if disk.ReadIOPS == 0 && disk.WriteIOPS == 0 && disk.ReadBPS == 0 && disk.WriteBPS == 0 {
				toRemoveMap[disk.Device] = struct{}{}
			} else {
				return errors.Wrapf(ErrInvalidDisk, "disk %+v not found in capacity", disk.Device)
			}
		}
	}
	n.Usage.Disks = slices.DeleteFunc(n.Usage.Disks, func(disk *Disk) bool {
		_, ok := toRemoveMap[disk.Device]
		return ok
	})
	return nil
}

func (n *NodeResourceInfo) validateVolume() error {
	for key, value := range n.Capacity.Volumes {
		if value < 0 {
			return errors.Wrap(ErrInvalidVolume, "volume size should not be less than 0")
		}
		if usage, ok := n.Usage.Volumes[key]; ok && (usage > value || usage < 0) {
			return errors.Wrap(ErrInvalidVolume, "invalid size in usage")
		}
	}
	return nil
}

func (n *NodeResourceInfo) validateStorage() error {
	if n.Capacity.Storage < 0 {
		return errors.Wrap(ErrInvalidStorage, "storage capacity can't be negative")
	}
	if n.Usage.Storage < 0 {
		return errors.Wrap(ErrInvalidStorage, "storage usage can't be negative")
	}
	return nil
}

type NodeResourceRequest struct {
	Volumes Volumes
	Storage int64
	Disks   Disks
	RMDisks []string

	RawParams resourcetypes.RawParams
}

func (n *NodeResourceRequest) Parse(rawParams resourcetypes.RawParams) (err error) {
	n.RawParams = rawParams

	volumes := Volumes{}
	for _, volume := range n.RawParams.StringSlice("volumes") {
		parts := strings.Split(volume, ":")
		if len(parts) != 2 {
			return errors.Wrap(ErrInvalidVolume, "volume should have 2 parts")
		}

		size, parseErr := utils.ParseRAMInHuman(parts[1])
		if parseErr != nil {
			return parseErr
		}
		volumes[parts[0]] = size
	}
	n.Volumes = volumes

	if n.Storage, err = n.RawParams.SizeInBytes("storage"); err != nil {
		return err
	}

	n.Storage += n.Volumes.Total()

	disks := Disks{}
	for _, rawDiskStr := range n.RawParams.StringSlice("disks") {
		disk := &Disk{}
		if err = disk.Parse(rawDiskStr); err != nil {
			return errors.Wrapf(ErrInvalidDisk, "wrong disk format: %+v, %+v", rawDiskStr, err)
		}
		disks = append(disks, disk)
	}
	n.Disks = disks

	if n.RawParams.IsSet("rm-disks") {
		n.RMDisks = strings.Split(n.RawParams.String("rm-disks"), ",")
	}
	return nil
}

// SkipEmpty used for setting node resource capacity in absolute mode
func (n *NodeResourceRequest) SkipEmpty(nodeResource *NodeResource) {
	if !n.RawParams.IsSet("volumes") {
		n.Volumes = nodeResource.Volumes
	}
	if !n.RawParams.IsSet("storage") {
		if n.RawParams.IsSet("volumes") {
			n.Storage = nodeResource.Storage - nodeResource.Volumes.Total() + n.Volumes.Total()
		} else {
			n.Storage = nodeResource.Storage
		}
	}
	if !n.RawParams.IsSet("disks") {
		n.Disks = nodeResource.Disks
	}
}
