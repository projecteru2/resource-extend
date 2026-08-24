package types

import (
	"github.com/go-viper/mapstructure/v2"
	resourcetypes "github.com/projecteru2/core/resource/types"
)

type EngineParams struct {
	ProdCountMap ProdCountMap `json:"prod_count_map" mapstructure:"prod_count_map"`
}

func (ep *EngineParams) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		"prod_count_map": ep.ProdCountMap,
	}
}

func (ep *EngineParams) Parse(rawParams resourcetypes.RawParams) error {
	return mapstructure.Decode(rawParams, ep)
}

func (ep *EngineParams) Count() int {
	return ep.ProdCountMap.TotalCount()
}

func (ep *EngineParams) DeepCopy() *EngineParams {
	return &EngineParams{
		ProdCountMap: ep.ProdCountMap.DeepCopy(),
	}
}

func (ep *EngineParams) Sub(ep1 *EngineParams) {
	ep.ProdCountMap.Sub(ep1.ProdCountMap)
}

func (ep *EngineParams) Add(ep1 *EngineParams) {
	ep.ProdCountMap.Add(ep1.ProdCountMap)
}
