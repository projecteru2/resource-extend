package schedule

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/projecteru2/resource-extend/storage/types"
)

const propertySeeds = 300

func TestGetVolumePlansMatchesReference(t *testing.T) {
	for seed := range int64(propertySeeds) {
		rng := rand.New(rand.NewSource(seed))
		resourceInfo, requests := generateRandomFixture(t, rng)

		plans, diskPlans := GetVolumePlans(t.Context(), resourceInfo, requests, maxDeployCount)
		refPlans, refDiskPlans := refGetVolumePlans(t.Context(), resourceInfo, requests, maxDeployCount)

		require.Equal(t, len(refPlans), len(plans), "seed %d requests %s", seed, requests.String())
		require.Equal(t, len(refDiskPlans), len(diskPlans), "seed %d", seed)
		for _, i := range sampleIndexes(len(plans)) {
			require.Equal(t, refPlans[i].String(), plans[i].String(), "seed %d plan %d requests %s", seed, i, requests.String())
			require.Equal(t, disksKey(refDiskPlans[i]), disksKey(diskPlans[i]), "seed %d disks %d requests %s", seed, i, requests.String())
		}
	}
}

func sampleIndexes(n int) []int {
	if n <= 64 {
		indexes := make([]int, n)
		for i := range n {
			indexes[i] = i
		}
		return indexes
	}
	indexes := []int{0, 1, 2, n / 2, n - 2, n - 1}
	for i := 3; i < n-3; i += 97 {
		indexes = append(indexes, i)
	}
	return indexes
}

func disksKey(disks types.Disks) string {
	parts := make([]string, len(disks))
	for i, disk := range disks {
		parts[i] = disk.String()
	}
	slices.Sort(parts)
	return strings.Join(parts, "|")
}

func generateRandomFixture(t *testing.T, rng *rand.Rand) (*types.NodeResourceInfo, types.VolumeBindings) {
	volumeCount := rng.Intn(9)
	capacity := &types.NodeResource{Volumes: types.Volumes{}, Disks: types.Disks{}}
	usage := &types.NodeResource{Volumes: types.Volumes{}, Disks: types.Disks{}}
	for i := range volumeCount {
		device := fmt.Sprintf("/data%d", i)
		size := int64(1+rng.Intn(64)) * gib
		capacity.Volumes[device] = size
		if rng.Intn(2) == 0 {
			usage.Volumes[device] = int64(rng.Intn(int(size/gib))) * gib
		} else {
			usage.Volumes[device] = 0
		}
	}

	for i := range rng.Intn(4) {
		mounts := []string{}
		if volumeCount > 0 {
			mounts = append(mounts, fmt.Sprintf("/data%d", rng.Intn(volumeCount)))
		}
		if rng.Intn(3) == 0 {
			mounts = append(mounts, "/")
		}
		if len(mounts) == 0 {
			mounts = []string{"/mnt"}
		}
		disk := &types.Disk{
			Device:    fmt.Sprintf("/dev/vd%c", 'a'+i),
			Mounts:    mounts,
			ReadIOPS:  int64(rng.Intn(1000)),
			WriteIOPS: int64(rng.Intn(1000)),
			ReadBPS:   int64(rng.Intn(1000)) << 20,
			WriteBPS:  int64(rng.Intn(1000)) << 20,
		}
		capacity.Disks = append(capacity.Disks, disk)
		usage.Disks = append(usage.Disks, &types.Disk{Device: disk.Device, Mounts: disk.Mounts})
	}

	specs := []string{}
	for i := range 1 + rng.Intn(5) {
		dst := fmt.Sprintf("/dir%d", i)
		switch rng.Intn(5) {
		case 0:
			specs = append(specs, fmt.Sprintf("AUTO:%s:rw:%dGiB", dst, 1+rng.Intn(16)))
		case 1:
			specs = append(specs, fmt.Sprintf("AUTO:%s:rw:%dGiB:%d:%d:%dM:%dM", dst, 1+rng.Intn(16), rng.Intn(200), rng.Intn(200), rng.Intn(100), rng.Intn(100)))
		case 2:
			specs = append(specs, fmt.Sprintf("AUTO:%s:rwm:%dGiB", dst, 1+rng.Intn(16)))
		case 3:
			specs = append(specs, fmt.Sprintf("AUTO:%s:rw", dst))
		default:
			source := "/mnt/other"
			if volumeCount > 0 {
				source = fmt.Sprintf("/data%d/sub", rng.Intn(volumeCount))
			}
			specs = append(specs, fmt.Sprintf("%s:%s:rw:0:%d:%d:%dM:%dM", source, dst, rng.Intn(200), rng.Intn(200), rng.Intn(100), rng.Intn(100)))
		}
	}

	return &types.NodeResourceInfo{Capacity: capacity, Usage: usage}, generateVolumeBindings(t, specs)
}
