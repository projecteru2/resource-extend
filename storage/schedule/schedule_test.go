package schedule

import (
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/projecteru2/resource-extend/storage/types"
)

const (
	maxDeployCount = 1000

	gib = 1 << 30
	tib = 1 << 40
)

func TestGetVolumePlans(t *testing.T) {
	resourceInfo := generateEmptyResourceInfo()

	volumeRequest := generateVolumeBindings(t, []string{
		"AUTO:/dir1:rw:500GiB",
	})

	plans, _ := GetVolumePlans(t.Context(), resourceInfo, volumeRequest, maxDeployCount)
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
		plans, _ = GetVolumePlans(t.Context(), resourceInfo, volumeRequest, maxDeployCount)
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
		plans, _ = GetVolumePlans(t.Context(), resourceInfo, volumeRequest, maxDeployCount)
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
		plans, _ := GetVolumePlans(t.Context(), resourceInfo, volumeRequest, maxDeployCount)
		validateVolumePlans(t, resourceInfo, volumeRequest, plans)
	}
}

func TestGetVolumePlansSkipQuotaFreeDisks(t *testing.T) {
	requests := generateVolumeBindings(t, []string{
		"AUTO:/dir1:rw:100GiB",
		"AUTO:/dir2:rwm:1GiB",
		"/mnt/data:/dir3:rw",
	})
	resourceInfo := generateResourceInfo()
	plans, diskPlans := GetVolumePlans(t.Context(), resourceInfo, requests, maxDeployCount)
	assert.NotEmpty(t, plans)
	for _, diskPlan := range diskPlans {
		assert.Empty(t, diskPlan)
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
		resourceInfo, mergedRequest, _, plan := runAffinityPlan(t, request)
		validateVolumePlan(t, resourceInfo, mergedRequest, plan)
	}

	_, _, existing, plan := runAffinityPlan(t, types.VolumeBindings{})
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
		_, _, _, plan := runAffinityPlan(t, request)
		assert.Equal(t, len(plan), 0)
	}
}

func TestAffinityPlan2(t *testing.T) {
	req := generateVolumeBindings(t, []string{
		"AUTO:/dir0:rw:800GiB",
		"AUTO:/dir10:rw:800GiB",
	})
	resourceInfo := generateResourceInfo()
	resourceInfo.Usage.Volumes["/data2"] = tib
	resourceInfo.Usage.Volumes["/data3"] = tib
	originRequest, existing := generateExistingVolumePlan(t)
	mergedRequest := types.MergeVolumeBindings(req, originRequest)
	plan, _, _ := GetAffinityPlan(t.Context(), resourceInfo, mergedRequest, existing, originRequest)
	assert.Equal(t, len(plan), 0)
}

func BenchmarkGetVolumePlans(b *testing.B) {
	cases := map[string][]string{
		"no-schedule": nil,
		"normal": {
			"AUTO:/dir0:rw:10GiB",
			"AUTO:/dir1:rw:20GiB:100:100:1M:1M",
			"/data0:/mnt:rw:0:100:100:1M:1M",
		},
		"mono": {
			"AUTO:/dir0:rw:10GiB",
			"AUTO:/dir1:rw:20GiB:100:100:1M:1M",
			"AUTO:/dir2:mrw:100GiB",
			"/data0:/mnt:rw:0:100:100:1M:1M",
		},
	}

	for name, volumes := range cases {
		b.Run(name, func(b *testing.B) {
			resourceInfo := generateBenchResourceInfo()
			requests, err := types.NewVolumeBindings(volumes)
			if err != nil {
				b.Fatalf("setup: %v", err)
			}
			if plans, _ := GetVolumePlans(b.Context(), resourceInfo, requests, maxDeployCount); len(plans) == 0 {
				b.Fatal("setup: benchmark must schedule at least one plan")
			}

			b.ReportAllocs()
			for b.Loop() {
				GetVolumePlans(b.Context(), resourceInfo, requests, maxDeployCount)
			}
		})
	}
}

func BenchmarkGetAffinityPlan(b *testing.B) {
	resourceInfo := generateBenchResourceInfo()
	originRequest, err := types.NewVolumeBindings([]string{
		"AUTO:/dir0:rw:100GiB:100:100:1M:1M",
		"AUTO:/dir1:mrw:100GiB",
	})
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	existing := types.VolumePlan{}
	if err := existing.UnmarshalJSON([]byte(`{"AUTO:/dir0:rw:100GiB:100:100:1M:1M": {"/data0": 107374182400}, "AUTO:/dir1:mrw:100GiB": {"/data1": 107374182400}}`)); err != nil {
		b.Fatalf("setup: %v", err)
	}
	if _, _, err := GetAffinityPlan(b.Context(), resourceInfo, originRequest, existing, originRequest); err != nil {
		b.Fatalf("setup: %v", err)
	}

	b.ReportAllocs()
	for b.Loop() {
		GetAffinityPlan(b.Context(), resourceInfo, originRequest, existing, originRequest)
	}
}

func generateBenchResourceInfo() *types.NodeResourceInfo {
	capacity := &types.NodeResource{Volumes: types.Volumes{}, Disks: types.Disks{}}
	usage := &types.NodeResource{Volumes: types.Volumes{}, Disks: types.Disks{}}
	for i := range 8 {
		device := "/data" + strconv.Itoa(i)
		capacity.Volumes[device] = tib
		usage.Volumes[device] = int64(i) * gib
		capacity.Disks = append(capacity.Disks, &types.Disk{
			Device:    "/dev/vd" + strconv.Itoa(i),
			Mounts:    []string{device},
			ReadIOPS:  1000000,
			WriteIOPS: 1000000,
			ReadBPS:   1000 * gib,
			WriteBPS:  1000 * gib,
		})
		usage.Disks = append(usage.Disks, &types.Disk{Device: "/dev/vd" + strconv.Itoa(i), Mounts: []string{device}})
	}
	return &types.NodeResourceInfo{Capacity: capacity, Usage: usage}
}

func generateResourceInfo() *types.NodeResourceInfo {
	return &types.NodeResourceInfo{
		Capacity: &types.NodeResource{
			Volumes: types.Volumes{
				"/data0": tib,
				"/data1": tib,
				"/data2": tib,
				"/data3": tib,
			},
			Disks: []*types.Disk{
				{
					Device:    "/dev/vda",
					Mounts:    []string{"/", "/data"},
					ReadIOPS:  1000,
					WriteIOPS: 1000,
					ReadBPS:   gib,
					WriteBPS:  gib,
				},
				{
					Device:   "/dev/vdb",
					Mounts:   []string{"/data1"},
					ReadBPS:  gib,
					WriteBPS: gib,
				},
			},
		},
		Usage: &types.NodeResource{
			Volumes: types.Volumes{
				"/data0": 200 * gib,
				"/data1": 300 * gib,
			},
			Disks: []*types.Disk{
				{Device: "/dev/vda", Mounts: []string{"/", "/data"}},
				{Device: "/dev/vdb", Mounts: []string{"/data1"}},
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
	plan, _ := GetVolumePlans(t.Context(), resourceInfo, volumeRequest, maxDeployCount)
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
		increaseIOPSQuota(disk, binding)

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
		increaseIOPSQuota(disk, binding)
	}
}

func validateVolumePlans(t *testing.T, resourceInfo *types.NodeResourceInfo, volumeRequest types.VolumeBindings, volumePlans []types.VolumePlan) {
	t.Logf("%v plans in total", len(volumePlans))
	t.Logf("plans: %v", volumePlans)
	for _, plan := range volumePlans {
		validateVolumePlan(t, resourceInfo, volumeRequest, plan)
	}
	if slices.ContainsFunc(volumeRequest, func(plan *types.VolumeBinding) bool {
		return plan.RequireSchedule() && !plan.RequireScheduleUnlimitedQuota()
	}) {
		noMorePlans(t, resourceInfo, volumePlans, volumeRequest)
	}
}

func runAffinityPlan(t *testing.T, request types.VolumeBindings) (*types.NodeResourceInfo, types.VolumeBindings, types.VolumePlan, types.VolumePlan) {
	resourceInfo := generateResourceInfo()
	originRequest, existing := generateExistingVolumePlan(t)
	for _, volumeMap := range existing {
		resourceInfo.Usage.Volumes[volumeMap.GetDevice()] += volumeMap.GetSize()
	}
	mergedRequest := types.MergeVolumeBindings(request, originRequest)
	plan, _, _ := GetAffinityPlan(t.Context(), resourceInfo, mergedRequest, existing, originRequest)
	return resourceInfo, mergedRequest, existing, plan
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
