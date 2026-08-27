package types

import (
	"testing"

	resourcetypes "github.com/projecteru2/core/resource/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeResourceInfoValidate(t *testing.T) {
	assert.ErrorIs(t, (&NodeResourceInfo{}).Validate(), ErrInvalidCapacity)

	info := &NodeResourceInfo{
		Capacity: &NodeResource{
			Volumes: Volumes{"/data0": testGiB},
			Disks:   Disks{{Device: "/dev/vda", Mounts: []string{"/"}, ReadIOPS: 100}},
			Storage: testGiB,
		},
	}
	require.NoError(t, info.Validate())
	assert.Equal(t, Volumes{"/data0": 0}, info.Usage.Volumes)
	require.Len(t, info.Usage.Disks, 1)
	assert.Equal(t, &Disk{Device: "/dev/vda", Mounts: []string{"/"}}, info.Usage.Disks[0])

	info.Usage.Volumes["/data0"] = 2 * testGiB
	assert.ErrorIs(t, info.Validate(), ErrInvalidVolume)
	info.Usage.Volumes["/data0"] = 0

	info.Usage.Storage = -1
	assert.ErrorIs(t, info.Validate(), ErrInvalidStorage)
	info.Usage.Storage = 0

	info.Usage.Disks[0].ReadIOPS = 200
	assert.ErrorIs(t, info.Validate(), ErrInvalidDisk)
	info.Usage.Disks[0].ReadIOPS = 0
}

func TestValidateScrubsOrphanUsageDisks(t *testing.T) {
	info := &NodeResourceInfo{
		Capacity: &NodeResource{Volumes: Volumes{}, Disks: Disks{}},
		Usage: &NodeResource{
			Volumes: Volumes{},
			Disks:   Disks{{Device: "/dev/gone"}},
		},
	}
	require.NoError(t, info.Validate())
	assert.Empty(t, info.Usage.Disks)

	info.Usage.Disks = Disks{{Device: "/dev/gone", ReadIOPS: 1}}
	assert.ErrorIs(t, info.Validate(), ErrInvalidDisk)
}

func TestNodeResourceRequestParse(t *testing.T) {
	req := &NodeResourceRequest{}
	require.NoError(t, req.Parse(resourcetypes.RawParams{
		"volumes":  []string{"/data0:1T", "/data1:1G"},
		"storage":  "1G",
		"disks":    []string{"/dev/vda:/,/data0:100:200:1G:2G"},
		"rm-disks": "/dev/vdb,/dev/vdc",
	}))
	assert.Equal(t, Volumes{"/data0": int64(1) << 40, "/data1": testGiB}, req.Volumes)
	assert.Equal(t, (int64(1)<<40)+2*testGiB, req.Storage)
	require.Len(t, req.Disks, 1)
	assert.Equal(t, "/dev/vda", req.Disks[0].Device)
	assert.Equal(t, []string{"/dev/vdb", "/dev/vdc"}, req.RMDisks)

	assert.ErrorIs(t, (&NodeResourceRequest{}).Parse(resourcetypes.RawParams{"volumes": []string{"/data0"}}), ErrInvalidVolume)
	assert.Error(t, (&NodeResourceRequest{}).Parse(resourcetypes.RawParams{"volumes": []string{"/data0:xx"}}))
}

func TestNodeResourceRequestParseStorageForms(t *testing.T) {
	tests := []struct {
		name    string
		storage any
	}{
		{"human string", "100G"},
		{"json number", float64(100 * testGiB)},
		{"int64", 100 * testGiB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &NodeResourceRequest{}
			require.NoError(t, req.Parse(resourcetypes.RawParams{
				"volumes": []string{"/data0:1G"},
				"storage": tt.storage,
			}))
			assert.Equal(t, Volumes{"/data0": testGiB}, req.Volumes)
			assert.Equal(t, 101*testGiB, req.Storage)
		})
	}
}

func TestSkipEmpty(t *testing.T) {
	origin := &NodeResource{
		Volumes: Volumes{"/data0": testGiB},
		Disks:   Disks{{Device: "/dev/vda"}},
		Storage: 3 * testGiB,
	}

	req := &NodeResourceRequest{}
	require.NoError(t, req.Parse(resourcetypes.RawParams{}))
	req.SkipEmpty(origin)
	assert.Equal(t, origin.Volumes, req.Volumes)
	assert.Equal(t, origin.Disks, req.Disks)
	assert.Equal(t, int64(3*testGiB), req.Storage)

	req = &NodeResourceRequest{}
	require.NoError(t, req.Parse(resourcetypes.RawParams{"volumes": []string{"/data0:4G"}}))
	req.SkipEmpty(origin)
	assert.Equal(t, Volumes{"/data0": 4 * testGiB}, req.Volumes)
	assert.Equal(t, int64(6*testGiB), req.Storage)
}

func TestNodeResourceArithmetic(t *testing.T) {
	res := &NodeResource{Volumes: Volumes{"/data0": 10}, Disks: Disks{}, Storage: 100}
	res.Add(&NodeResource{Volumes: Volumes{"/data1": 5}, Storage: 50})
	assert.Equal(t, int64(150), res.Storage)
	assert.Equal(t, Volumes{"/data0": 10, "/data1": 5}, res.Volumes)

	res.Sub(&NodeResource{Volumes: Volumes{"/data1": 5}, Storage: 50})
	res.RemoveEmpty()
	assert.Equal(t, Volumes{"/data0": 10}, res.Volumes)

	clone := res.DeepCopy()
	clone.Volumes["/data0"] = 0
	assert.Equal(t, int64(10), res.Volumes["/data0"])
}
