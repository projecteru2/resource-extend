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
