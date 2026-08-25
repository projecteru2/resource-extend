package types

import (
	resourcetypes "github.com/projecteru2/core/resource/types"

	"github.com/projecteru2/resource-extend/internal/decode"
)

type EngineParams struct {
	ProdCountMap ProdCountMap `json:"prod_count_map"`
}

func (ep *EngineParams) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		prodCountMapKey: ep.ProdCountMap,
	}
}

func (ep *EngineParams) Parse(rawParams resourcetypes.RawParams) error {
	return decode.Decode(rawParams, ep)
}

func (ep *EngineParams) Count() int {
	return ep.ProdCountMap.TotalCount()
}
