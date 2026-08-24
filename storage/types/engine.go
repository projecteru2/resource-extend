package types

import (
	"github.com/go-viper/mapstructure/v2"
	resourcetypes "github.com/projecteru2/core/resource/types"
)

// EngineParams .
type EngineParams struct {
	Volumes       []string          `json:"volumes" mapstructure:"volumes"`
	VolumeChanged bool              `json:"volume_changed" mapstructure:"volume_changed"` // indicates whether the realloc request includes new volumes
	Storage       int64             `json:"storage" mapstructure:"storage"`
	IOPSOptions   map[string]string `json:"iops_options" mapstructure:"iops_options"`
}

// AsRawParams .
func (ep *EngineParams) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		"volumes":        ep.Volumes,
		"volume_changed": ep.VolumeChanged,
		"storage":        ep.Storage,
		"iops_options":   ep.IOPSOptions,
	}
}

// Parse .
func (ep *EngineParams) Parse(rawParams resourcetypes.RawParams) error {
	return mapstructure.Decode(rawParams, ep)
}
