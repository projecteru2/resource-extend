package types

import (
	"maps"

	resourcetypes "github.com/projecteru2/core/resource/types"
)

type WorkloadResource struct {
	ProdCountMap `json:"prod_count_map"`
}

func (w *WorkloadResource) Parse(rawParams resourcetypes.RawParams) error {
	return resourcetypes.Decode(rawParams, w)
}

func (w *WorkloadResource) DeepCopy() *WorkloadResource {
	return &WorkloadResource{ProdCountMap: w.ProdCountMap.DeepCopy()}
}

func (w *WorkloadResource) Add(w1 *WorkloadResource) {
	w.ProdCountMap.Add(w1.ProdCountMap)
}

func (w *WorkloadResource) Sub(w1 *WorkloadResource) {
	w.ProdCountMap.Sub(w1.ProdCountMap)
}

type WorkloadResourceRequest struct {
	ProdCountMap `json:"prod_count_map"`
}

func (w *WorkloadResourceRequest) Parse(rawParams resourcetypes.RawParams) error {
	return resourcetypes.Decode(rawParams, w)
}

func (w *WorkloadResourceRequest) MergeFromResource(r *WorkloadResource) {
	w.Add(r.ProdCountMap)
	maps.DeleteFunc(w.ProdCountMap, func(_ string, count int) bool { return count <= 0 })
}

func (w *WorkloadResourceRequest) DeepCopy() *WorkloadResourceRequest {
	return &WorkloadResourceRequest{
		ProdCountMap: w.ProdCountMap.DeepCopy(),
	}
}
