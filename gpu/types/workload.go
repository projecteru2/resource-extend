package types

import (
	"maps"

	"github.com/go-viper/mapstructure/v2"
	resourcetypes "github.com/projecteru2/core/resource/types"
)

type WorkloadResource struct {
	ProdCountMap ProdCountMap `json:"prod_count_map" mapstructure:"prod_count_map"`
}

func (w *WorkloadResource) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		prodCountMapKey: w.ProdCountMap,
	}
}

func (w *WorkloadResource) Validate() error {
	return w.ProdCountMap.Validate()
}

func (w *WorkloadResource) Parse(rawParams resourcetypes.RawParams) error {
	return mapstructure.Decode(rawParams, w)
}

func (w *WorkloadResource) DeepCopy() *WorkloadResource {
	res := &WorkloadResource{
		ProdCountMap: w.ProdCountMap.DeepCopy(),
	}
	return res
}

func (w *WorkloadResource) Add(w1 *WorkloadResource) {
	w.ProdCountMap.Add(w1.ProdCountMap)
}

func (w *WorkloadResource) Sub(w1 *WorkloadResource) {
	w.ProdCountMap.Sub(w1.ProdCountMap)
}

func (w *WorkloadResource) Count() int {
	return w.ProdCountMap.TotalCount()
}

type WorkloadResourceRequest struct {
	ProdCountMap ProdCountMap `json:"prod_count_map" mapstructure:"prod_count_map"`
}

func (w *WorkloadResourceRequest) ValidateProd() error {
	// in order to support realloc, the count can be negative, so only validate prod here
	return w.ProdCountMap.ValidateProd()
}

func (w *WorkloadResourceRequest) Validate() error {
	return w.ProdCountMap.Validate()
}

func (w *WorkloadResourceRequest) Parse(rawParams resourcetypes.RawParams) (err error) {
	return mapstructure.Decode(rawParams, w)
}

func (w *WorkloadResourceRequest) MergeFromResource(r *WorkloadResource) {
	w.ProdCountMap.Add(r.ProdCountMap)
	maps.DeleteFunc(w.ProdCountMap, func(_ string, count int) bool { return count <= 0 })
}

func (w *WorkloadResourceRequest) DeepCopy() *WorkloadResourceRequest {
	return &WorkloadResourceRequest{
		ProdCountMap: w.ProdCountMap.DeepCopy(),
	}
}

func (w *WorkloadResourceRequest) Count() int {
	return w.ProdCountMap.TotalCount()
}
