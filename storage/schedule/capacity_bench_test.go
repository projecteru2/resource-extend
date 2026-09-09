package schedule

import (
	"strconv"
	"testing"

	"github.com/projecteru2/resource-extend/storage/types"
)

const (
	benchMaxDeployCount = 10000

	benchVolumeCount = 12
	benchVolumeIOPS  = 1000000
	benchVolumeBPS   = 1000 * gib
)

var capacitySink int

func BenchmarkGetVolumeCapacity(b *testing.B) {
	for _, tt := range capacityCases(b) {
		b.Run(tt.name, func(b *testing.B) {
			resourceInfo := generateCapacityResourceInfo()
			b.ReportAllocs()
			for b.Loop() {
				capacitySink = GetVolumeCapacity(resourceInfo, tt.requests, benchMaxDeployCount)
			}
		})
	}
}

func BenchmarkVolumeCapacityByPlans(b *testing.B) {
	for _, tt := range capacityCases(b) {
		b.Run(tt.name, func(b *testing.B) {
			resourceInfo := generateCapacityResourceInfo()
			ctx := b.Context()
			b.ReportAllocs()
			for b.Loop() {
				plans, _ := GetVolumePlans(ctx, resourceInfo, tt.requests, benchMaxDeployCount)
				capacitySink = min(len(plans), benchMaxDeployCount)
			}
		})
	}
}

type capacityCase struct {
	name     string
	requests types.VolumeBindings
}

func capacityCases(b *testing.B) []capacityCase {
	specs := []struct {
		name    string
		volumes []string
	}{
		{"normal", []string{
			"AUTO:/dir0:rw:10GiB",
			"AUTO:/dir1:rw:20GiB:100:100:1M:1M",
			"/data0:/mnt:rw:0:100:100:1M:1M",
		}},
		{"mono", []string{
			"AUTO:/dir0:rwm:100GiB",
			"AUTO:/dir1:rwm:200GiB",
		}},
		{"mixed", []string{
			"AUTO:/dir0:rw:10GiB",
			"AUTO:/dir1:rw:20GiB:100:100:1M:1M",
			"AUTO:/dir2:rwm:100GiB",
			"AUTO:/dir3:rwm:200GiB",
			"/data0:/mnt:rw:0:100:100:1M:1M",
		}},
	}

	cases := make([]capacityCase, len(specs))
	for i, spec := range specs {
		requests, err := types.NewVolumeBindings(spec.volumes)
		if err != nil {
			b.Fatalf("setup: %v", err)
		}
		plans, _ := GetVolumePlans(b.Context(), generateCapacityResourceInfo(), requests, benchMaxDeployCount)
		capacity := GetVolumeCapacity(generateCapacityResourceInfo(), requests, benchMaxDeployCount)
		if len(plans) == 0 || len(plans) != capacity {
			b.Fatalf("setup: %s plans %d capacity %d", spec.name, len(plans), capacity)
		}
		cases[i] = capacityCase{name: spec.name, requests: requests}
	}
	return cases
}

func generateCapacityResourceInfo() *types.NodeResourceInfo {
	capacity := &types.NodeResource{Volumes: types.Volumes{}, Disks: types.Disks{}}
	usage := &types.NodeResource{Volumes: types.Volumes{}, Disks: types.Disks{}}
	for i := range benchVolumeCount {
		device := "/data" + strconv.Itoa(i)
		capacity.Volumes[device] = tib
		if i%2 == 1 {
			usage.Volumes[device] = int64(i) * gib
		}
		capacity.Disks = append(capacity.Disks, &types.Disk{
			Device:    "/dev/vd" + strconv.Itoa(i),
			Mounts:    []string{device},
			ReadIOPS:  benchVolumeIOPS,
			WriteIOPS: benchVolumeIOPS,
			ReadBPS:   benchVolumeBPS,
			WriteBPS:  benchVolumeBPS,
		})
		usage.Disks = append(usage.Disks, &types.Disk{Device: "/dev/vd" + strconv.Itoa(i), Mounts: []string{device}})
	}
	return &types.NodeResourceInfo{Capacity: capacity, Usage: usage}
}
