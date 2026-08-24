package schedule

import (
	"testing"

	"github.com/docker/go-units"
	"github.com/projecteru2/core/utils"
	"github.com/stretchr/testify/assert"

	"github.com/projecteru2/resource-extend/storage/types"
)

const maxDeployCount = 1000

func TestGetVolumePlans(t *testing.T) {
	resourceInfo := generateEmptyResourceInfo()

	volumeRequest := generateVolumeBindings(t, []string{
		"AUTO:/dir1:rw:500GiB",
	})

	plans, _ := GetVolumePlans(resourceInfo, volumeRequest, maxDeployCount)
	assert.Equal(t, len(plans), 0)

	requests := []types.VolumeBindings{
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:500GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:500GiB",
			"AUTO:/dir2:rw:500GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:1GiB",
			"AUTO:/dir2:rwm:1GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:500GiB",
			"AUTO:/dir2:rw:500GiB",
			"AUTO:/dir3:rwm:100GiB",
			"AUTO:/dir4:rwm:100GiB",
			"AUTO:/dir5:rwm:100GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:500GiB",
			"AUTO:/dir2:rw:0",
			"AUTO:/dir3:rwm:100GiB",
			"AUTO:/dir4:rwm:100GiB",
			"AUTO:/dir5:rwm:100GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:0",
		}),
	}

	for _, volumeRequest := range requests {
		resourceInfo = generateResourceInfo()
		plans, _ = GetVolumePlans(resourceInfo, volumeRequest, maxDeployCount)
		validateVolumePlans(t, resourceInfo, volumeRequest, plans)
	}

	requests = []types.VolumeBindings{
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:2TiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:800GiB",
			"AUTO:/dir2:rw:800GiB",
			"AUTO:/dir3:rw:800GiB",
			"AUTO:/dir4:rw:800GiB",
			"AUTO:/dir5:rw:800GiB",
			"AUTO:/dir6:rw:800GiB",
			"AUTO:/dir7:rw:800GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rwm:500GiB",
			"AUTO:/dir2:rwm:500GiB",
			"AUTO:/dir3:rwm:500GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:800GiB",
			"AUTO:/dir2:rw:800GiB",
			"AUTO:/dir3:rw:800GiB",
			"AUTO:/dir4:rw:800GiB",
			"AUTO:/dir5:rwm:500GiB",
			"AUTO:/dir6:rwm:500GiB",
		}),
	}

	for _, volumeRequest := range requests {
		resourceInfo = generateResourceInfo()
		plans, _ = GetVolumePlans(resourceInfo, volumeRequest, maxDeployCount)
		assert.Equal(t, len(plans), 0)
	}
}

func TestGetVolumePlansWithIOPS(t *testing.T) {
	requests := []types.VolumeBindings{
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:500GiB:500:100:100M:100M",
			":/dir2:rw:500GiB:100:100:100M:100M",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rw:500GiB:500:100:100M:100M",
			":/dir2:rw:500GiB:100:100:100M:10000M",
		}),
		generateVolumeBindings(t, []string{
			":/dir2:rw",
		}),
	}

	for _, volumeRequest := range requests {
		resourceInfo := generateResourceInfo()
		plans, _ := GetVolumePlans(resourceInfo, volumeRequest, maxDeployCount)
		validateVolumePlans(t, resourceInfo, volumeRequest, plans)
	}
}

func TestGetAffinityPlan(t *testing.T) {
	requests := []types.VolumeBindings{
		generateVolumeBindings(t, []string{
			"AUTO:/dir0:rw:1GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir3:rw:1GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rwm:1GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rwm:1GiB",
			"AUTO:/dir3:rwm:1GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rwm:-1TiB",
			"AUTO:/dir3:rwm:100GiB",
			"AUTO:/dir4:rwm:100GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir0:rw:-100GiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir0:rw:-100GiB",
			"AUTO:/dir2:rw:100GiB",
			"AUTO:/dir3:rwm:100GiB",
		}),
	}

	for _, request := range requests {
		resourceInfo := generateResourceInfo()
		originRequest, existing := generateExistingVolumePlan(t)
		for _, volumeMap := range existing {
			resourceInfo.Usage.Volumes[volumeMap.GetDevice()] += volumeMap.GetSize()
		}
		mergedRequest := types.MergeVolumeBindings(request, originRequest)

		plan, _, _ := GetAffinityPlan(resourceInfo, mergedRequest, existing, originRequest)
		validateVolumePlan(t, resourceInfo, mergedRequest, plan)
	}

	resourceInfo := generateResourceInfo()
	originRequest, existing := generateExistingVolumePlan(t)
	for _, volumeMap := range existing {
		resourceInfo.Usage.Volumes[volumeMap.GetDevice()] += volumeMap.GetSize()
	}
	emptyRequest := types.VolumeBindings{}
	mergedRequest := types.MergeVolumeBindings(emptyRequest, originRequest)

	plan, _, _ := GetAffinityPlan(resourceInfo, mergedRequest, existing, originRequest)
	assert.Equal(t, existing.String(), plan.String())

	invalidRequests := []types.VolumeBindings{
		generateVolumeBindings(t, []string{
			"AUTO:/dir0:rw:1TiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir1:rwm:1TiB",
		}),
		generateVolumeBindings(t, []string{
			"AUTO:/dir3:rw:1TiB",
			"AUTO:/dir4:rw:1TiB",
		}),
	}

	for _, request := range invalidRequests {
		resourceInfo := generateResourceInfo()
		originRequest, existing := generateExistingVolumePlan(t)
		for _, volumeMap := range existing {
			resourceInfo.Usage.Volumes[volumeMap.GetDevice()] += volumeMap.GetSize()
		}
		mergedRequest := types.MergeVolumeBindings(request, originRequest)

		plan, _, _ := GetAffinityPlan(resourceInfo, mergedRequest, existing, originRequest)
		assert.Equal(t, len(plan), 0)
	}
}

func TestAffinityPlan2(t *testing.T) {
	req := generateVolumeBindings(t, []string{
		"AUTO:/dir0:rw:800GiB",
		"AUTO:/dir10:rw:800GiB",
	})
	resourceInfo := generateResourceInfo()
	resourceInfo.Usage.Volumes["/data2"] = units.TiB
	resourceInfo.Usage.Volumes["/data3"] = units.TiB
	originRequest, existing := generateExistingVolumePlan(t)
	mergedRequest := types.MergeVolumeBindings(req, originRequest)
	plan, _, _ := GetAffinityPlan(resourceInfo, mergedRequest, existing, originRequest)
	assert.Equal(t, len(plan), 0)
}

func generateResourceInfo() *types.NodeResourceInfo {
	return &types.NodeResourceInfo{
		Capacity: &types.NodeResource{
			Volumes: types.Volumes{
				"/data0": units.TiB,
				"/data1": units.TiB,
				"/data2": units.TiB,
				"/data3": units.TiB,
			},
			Disks: []*types.Disk{
				{
					Device:    "/dev/vda",
					Mounts:    []string{"/", "/data"},
					ReadIOPS:  1000,
					WriteIOPS: 1000,
					ReadBPS:   units.GiB,
					WriteBPS:  units.GiB,
				},
				{
					Device:    "/dev/vdb",
					Mounts:    []string{"/data1"},
					ReadIOPS:  0,
					WriteIOPS: 0,
					ReadBPS:   units.GiB,
					WriteBPS:  units.GiB,
				},
			},
		},
		Usage: &types.NodeResource{
			Volumes: types.Volumes{
				"/data0": 200 * units.GiB,
				"/data1": 300 * units.GiB,
			},
			Disks: []*types.Disk{
				{
					Device:    "/dev/vda",
					Mounts:    []string{"/", "/data"},
					ReadIOPS:  0,
					WriteIOPS: 0,
					ReadBPS:   0,
					WriteBPS:  0,
				},
				{
					Device:    "/dev/vdb",
					Mounts:    []string{"/data1"},
					ReadIOPS:  0,
					WriteIOPS: 0,
					ReadBPS:   0,
					WriteBPS:  0,
				},
			},
		},
	}
}

func generateEmptyResourceInfo() *types.NodeResourceInfo {
	return &types.NodeResourceInfo{
		Capacity: &types.NodeResource{
			Volumes: types.Volumes{},
		},
		Usage: &types.NodeResource{
			Volumes: types.Volumes{},
		},
	}
}

func applyPlans(resourceInfo *types.NodeResourceInfo, plans []types.VolumePlan) {
	for _, plan := range plans {
		for _, volumeMap := range plan {
			for device, size := range volumeMap {
				resourceInfo.Usage.Volumes[device] += size
			}
		}
	}
}

func noMorePlans(t *testing.T, resourceInfo *types.NodeResourceInfo, volumePlans []types.VolumePlan, volumeRequest types.VolumeBindings) {
	applyPlans(resourceInfo, volumePlans)
	assert.Nil(t, resourceInfo.Validate())
	plan, _ := GetVolumePlans(resourceInfo, volumeRequest, maxDeployCount)
	assert.Equal(t, len(plan), 0)
}

func validateVolumePlan(t *testing.T, resourceInfo *types.NodeResourceInfo, volumeRequest types.VolumeBindings, volumePlan types.VolumePlan) {
	t.Logf("volume plan: %v", volumePlan)
	t.Logf("volume request: %v", volumeRequest)

	allocIOPSQuotaForMountRequests(resourceInfo, volumeRequest)

	monoDevice := ""
	monoTotalSize := int64(0)

	for _, binding := range volumeRequest {
		if !binding.RequireSchedule() {
			continue
		}
		volumeMap, ok := volumePlan[binding]
		assert.True(t, ok)
		disk := resourceInfo.Usage.Disks.GetDiskByPath(volumeMap.GetDevice())
		assert.NotNil(t, disk)
		disk.ReadIOPS += binding.ReadIOPS
		disk.WriteIOPS += binding.WriteIOPS
		disk.ReadBPS += binding.ReadBPS
		disk.WriteBPS += binding.WriteBPS

		switch {
		case binding.RequireScheduleMonopoly():
			if monoDevice == "" {
				monoDevice = volumeMap.GetDevice()
			}
			assert.Equal(t, monoDevice, volumeMap.GetDevice())
			monoTotalSize += volumeMap.GetSize()
		case binding.RequireSchedule():
			assert.Equal(t, volumeMap.GetSize(), binding.SizeInBytes)
		}
	}

	assert.Equal(t, monoTotalSize, resourceInfo.Capacity.Volumes[monoDevice])
	assert.Nil(t, resourceInfo.Validate())
}

func allocIOPSQuotaForMountRequests(resourceInfo *types.NodeResourceInfo, volumeRequest types.VolumeBindings) {
	for _, binding := range volumeRequest {
		if binding.RequireSchedule() {
			continue
		}
		disk := resourceInfo.Usage.Disks.GetDiskByPath(binding.Source)
		disk.ReadIOPS += binding.ReadIOPS
		disk.WriteIOPS += binding.WriteIOPS
		disk.ReadBPS += binding.ReadBPS
		disk.WriteBPS += binding.WriteBPS
	}
}

func validateVolumePlans(t *testing.T, resourceInfo *types.NodeResourceInfo, volumeRequest types.VolumeBindings, volumePlans []types.VolumePlan) {
	t.Logf("%v plans in total", len(volumePlans))
	t.Logf("plans: %v", volumePlans)
	for _, plan := range volumePlans {
		validateVolumePlan(t, resourceInfo, volumeRequest, plan)
	}
	if utils.Any(volumeRequest, func(plan *types.VolumeBinding) bool {
		return plan.RequireSchedule() && !plan.RequireScheduleUnlimitedQuota()
	}) {
		noMorePlans(t, resourceInfo, volumePlans, volumeRequest)
	}
}

func generateVolumeBindings(t *testing.T, str []string) types.VolumeBindings {
	bindings, err := types.NewVolumeBindings(str)
	assert.Nil(t, err)
	return bindings
}

func generateExistingVolumePlan(t *testing.T) (types.VolumeBindings, types.VolumePlan) {
	plan := types.VolumePlan{}
	err := plan.UnmarshalJSON([]byte(`
{
	"AUTO:/dir0:rw:100GiB:100:100:100M:100M": {
        "/data0": 107374182400
      },
      "AUTO:/dir1:mrw:100GiB": {
        "/data2": 1099511627776
      },
      "AUTO:/dir2:rw:0": {
        "/data0": 0
      }
}
`))
	assert.Nil(t, err)
	bindings := generateVolumeBindings(t, []string{
		"AUTO:/dir0:rw:100GiB:100:100:100M:100M",
		"AUTO:/dir1:mrw:100GiB",
		"AUTO:/dir2:rw:0",
	})
	return bindings, plan
}
