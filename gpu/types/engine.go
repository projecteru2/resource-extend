package types

import (
	resourcetypes "github.com/projecteru2/core/resource/types"
)

type EngineParams struct {
	ProdCountMap `json:"prod_count_map"`
}

func (ep *EngineParams) Parse(rawParams resourcetypes.RawParams) error {
	return resourcetypes.Decode(rawParams, ep)
}
