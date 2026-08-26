package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/projecteru2/resource-extend/storage/types"
)

func TestPromotionSurvivesNonMonotoneCapacity(t *testing.T) {
	requests := generateVolumeBindings(t, []string{
		"AUTO:/n:rw:10GiB:85:0:0:0",
		"AUTO:/o:rwm:10GiB",
		"/mnt/shared/x:/mm:rw:0:10:0:0:0",
	})

	plans, _ := GetVolumePlans(t.Context(), buildSharedDiskFixture(), requests, maxDeployCount)
	require.Len(t, plans, 4)

	refPlans, _ := refGetVolumePlans(t.Context(), buildSharedDiskFixture(), requests, maxDeployCount)
	assert.Equal(t, len(refPlans), len(plans))

	assert.Equal(t, 4, GetVolumeCapacity(buildSharedDiskFixture(), requests, maxDeployCount))
}

func buildSharedDiskFixture() *types.NodeResourceInfo {
	huge := int64(1) << 50
	return &types.NodeResourceInfo{
		Capacity: &types.NodeResource{
			Volumes: types.Volumes{
				"/dataA": 40 * gib,
				"/v1":    11 * gib,
				"/v2":    12 * gib,
				"/v3":    100 * gib,
				"/v4":    100 * gib,
				"/v5":    100 * gib,
			},
			Disks: types.Disks{
				{Device: "/dev/a", Mounts: []string{"/dataA"}, ReadIOPS: huge, WriteIOPS: huge, ReadBPS: huge, WriteBPS: huge},
				{Device: "/dev/b", Mounts: []string{"/mnt/shared", "/v2"}, ReadIOPS: 100, WriteIOPS: huge, ReadBPS: huge, WriteBPS: huge},
				{Device: "/dev/c", Mounts: []string{"/v1", "/v3", "/v4", "/v5"}, ReadIOPS: huge, WriteIOPS: huge, ReadBPS: huge, WriteBPS: huge},
			},
		},
		Usage: &types.NodeResource{
			Volumes: types.Volumes{"/dataA": 10 * gib, "/v1": 0, "/v2": 0, "/v3": 0, "/v4": 0, "/v5": 0},
			Disks:   types.Disks{},
		},
	}
}
