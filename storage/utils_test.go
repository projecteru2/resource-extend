package storage

import (
	"testing"

	"github.com/docker/go-units"
	"github.com/stretchr/testify/assert"

	"github.com/projecteru2/resource-extend/storage/types"
)

func TestGetDisksLimitSkipsUnknownMount(t *testing.T) {
	limit, err := types.NewVolumeBindings([]string{"/data:/data:rw:0:100:100:1M:1M"})
	assert.NoError(t, err)
	assert.Empty(t, getDisksLimit(limit, types.VolumePlan{}, types.Disks{}))
}

func TestGetDisksLimitSkipsUnknownPlanDevice(t *testing.T) {
	binding, err := types.NewVolumeBinding("AUTO:/data:rw:1G:100:100:1M:1M")
	assert.NoError(t, err)
	plan := types.VolumePlan{binding: types.Volumes{"/unknown": units.GiB}}
	assert.Empty(t, getDisksLimit(types.VolumeBindings{binding}, plan, types.Disks{}))
}

func TestGetDisksLimitUsesMatchingDisk(t *testing.T) {
	limit, err := types.NewVolumeBindings([]string{"/data:/data:rw:0:100:100:1M:1M"})
	assert.NoError(t, err)
	disks := types.Disks{{Device: "/dev/vda", Mounts: []string{"/data"}}}
	disksLimit := getDisksLimit(limit, types.VolumePlan{}, disks)
	assert.Len(t, disksLimit, 1)
	assert.Equal(t, "/dev/vda", disksLimit[0].Device)
	assert.Equal(t, int64(100), disksLimit[0].ReadIOPS)
}
