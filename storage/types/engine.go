package types

import (
	resourcetypes "github.com/projecteru2/core/resource/types"

	"github.com/projecteru2/resource-extend/internal/decode"
)

type EngineParams struct {
	Volumes       []string          `json:"volumes"`
	VolumeChanged bool              `json:"volume_changed"` // indicates whether the realloc request includes new volumes
	Storage       int64             `json:"storage"`
	IOPSOptions   map[string]string `json:"iops_options"`
}

func (ep *EngineParams) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		"volumes":        ep.Volumes,
		"volume_changed": ep.VolumeChanged,
		"storage":        ep.Storage,
		"iops_options":   ep.IOPSOptions,
	}
}

func (ep *EngineParams) Parse(rawParams resourcetypes.RawParams) error {
	return decode.Decode(rawParams, ep)
}
