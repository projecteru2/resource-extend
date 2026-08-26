package types

import (
	"testing"

	resourcetypes "github.com/projecteru2/core/resource/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRejectsSizelessMonopoly(t *testing.T) {
	tests := []struct {
		name    string
		volumes []string
	}{
		{"zero size with IOPS", []string{"AUTO:/dir:mrw:0:100:100:1M:1M"}},
		{"zero size without IOPS", []string{"AUTO:/dir:mrw"}},
		{"negative size", []string{"AUTO:/dir:mrw:-1G"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &WorkloadResourceRequest{}
			require.NoError(t, req.Parse(resourcetypes.RawParams{"volumes": tt.volumes}))
			assert.ErrorIs(t, req.Validate(), ErrInvalidVolume)
		})
	}

	req := &WorkloadResourceRequest{}
	require.NoError(t, req.Parse(resourcetypes.RawParams{"volumes": []string{"AUTO:/dir:mrw:1G"}}))
	assert.NoError(t, req.Validate())
}

func TestWorkloadResourceRequestParse(t *testing.T) {
	req := &WorkloadResourceRequest{}
	require.NoError(t, req.Parse(resourcetypes.RawParams{
		"volume-request": []string{"AUTO:/dir:rw:1G"},
		"volume-limit":   []string{"AUTO:/dir:rw:2G"},
		"storage":        "1G",
	}))
	assert.Equal(t, int64(testGiB), req.VolumesRequest[0].SizeInBytes)
	assert.Equal(t, int64(2*testGiB), req.VolumesLimit[0].SizeInBytes)
	assert.Equal(t, int64(testGiB), req.StorageRequest)
	assert.Equal(t, int64(testGiB), req.StorageLimit)

	req = &WorkloadResourceRequest{}
	require.NoError(t, req.Parse(resourcetypes.RawParams{"volumes": []string{"AUTO:/dir:rw:1G"}}))
	assert.Equal(t, req.VolumesLimit.String(), req.VolumesRequest.String())
}

func TestWorkloadValidateRaisesSoftLimits(t *testing.T) {
	req := &WorkloadResourceRequest{}
	require.NoError(t, req.Parse(resourcetypes.RawParams{
		"volume-request":  []string{"AUTO:/dir:rw:2G:100:100:0:0"},
		"volume-limit":    []string{"AUTO:/dir:rw:1G:50:50:0:0"},
		"storage-request": "2G",
		"storage-limit":   "1G",
	}))
	require.NoError(t, req.Validate())
	assert.Equal(t, int64(2*testGiB), req.VolumesLimit[0].SizeInBytes)
	assert.Equal(t, int64(100), req.VolumesLimit[0].ReadIOPS)
	assert.Equal(t, req.StorageRequest, req.StorageLimit)
}

func TestWorkloadValidateMismatches(t *testing.T) {
	req := &WorkloadResourceRequest{
		VolumesRequest: mustBindings(t, []string{"AUTO:/dir1:rw:1G", "AUTO:/dir2:rw:1G"}),
		VolumesLimit:   mustBindings(t, []string{"AUTO:/dir1:rw:1G"}),
	}
	assert.ErrorIs(t, req.Validate(), ErrInvalidVolume)

	req = &WorkloadResourceRequest{
		VolumesRequest: mustBindings(t, []string{"AUTO:/dir1:rw:1G"}),
		VolumesLimit:   mustBindings(t, []string{"AUTO:/other:rw:1G"}),
	}
	assert.ErrorIs(t, req.Validate(), ErrInvalidVolume)
}

func TestWorkloadValidateFoldsStorageOnce(t *testing.T) {
	params := resourcetypes.RawParams{
		"volumes":         []string{"AUTO:/dir:rw:1G"},
		"storage-request": "1G",
	}

	req := &WorkloadResourceRequest{}
	require.NoError(t, req.Parse(params))
	require.NoError(t, req.Validate())
	assert.Equal(t, int64(2*testGiB), req.StorageRequest)
	require.NoError(t, req.Validate())
	assert.Equal(t, int64(2*testGiB), req.StorageRequest)

	req = &WorkloadResourceRequest{}
	require.NoError(t, req.Parse(params))
	req.SkipAddStorage()
	require.NoError(t, req.Validate())
	assert.Equal(t, int64(testGiB), req.StorageRequest)
}
