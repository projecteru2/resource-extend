package types

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	resourcetypes "github.com/projecteru2/core/resource/types"
)

type NodeResource struct {
	ProdCountMap ProdCountMap `json:"prod_count_map" mapstructure:"prod_count_map"`
}

func NewNodeResource(gm ProdCountMap) *NodeResource {
	r := &NodeResource{
		ProdCountMap: gm,
	}
	if r.ProdCountMap == nil {
		r.ProdCountMap = ProdCountMap{}
	}
	return r
}

func (r *NodeResource) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		"prod_count_map": r.ProdCountMap,
	}
}

func (r *NodeResource) Parse(rawParams resourcetypes.RawParams) error {
	return mapstructure.Decode(rawParams, r)
}

func (r *NodeResource) Validate() error {
	return r.ProdCountMap.Validate()
}

func (r *NodeResource) DeepCopy() *NodeResource {
	res := &NodeResource{
		ProdCountMap: r.ProdCountMap.DeepCopy(),
	}
	return res
}

func (r *NodeResource) Add(r1 *NodeResource) {
	r.ProdCountMap.Add(r1.ProdCountMap)
}

func (r *NodeResource) Sub(r1 *NodeResource) {
	r.ProdCountMap.Sub(r1.ProdCountMap)
}

func (r *NodeResource) Count() int {
	return r.ProdCountMap.TotalCount()
}

type NodeResourceInfo struct {
	Capacity *NodeResource `json:"capacity"`
	Usage    *NodeResource `json:"usage"`
}

func (n *NodeResourceInfo) CapCount() int {
	return n.Capacity.Count()
}

func (n *NodeResourceInfo) UsageCount() int {
	return n.Usage.Count()
}

func (n *NodeResourceInfo) DeepCopy() *NodeResourceInfo {
	return &NodeResourceInfo{
		Capacity: n.Capacity.DeepCopy(),
		Usage:    n.Usage.DeepCopy(),
	}
}

func (n *NodeResourceInfo) Validate() error {
	if err := n.Capacity.Validate(); err != nil {
		return fmt.Errorf("invalid capacity: %w", err)
	}
	if err := n.Usage.Validate(); err != nil {
		return fmt.Errorf("invalid usage: %w", err)
	}
	return nil
}

func (n *NodeResourceInfo) GetAvailableResource() *NodeResource {
	availableResource := n.Capacity.DeepCopy()
	availableResource.Sub(n.Usage)

	return availableResource
}

type NodeResourceRequest struct {
	ProdCountMap ProdCountMap `json:"prod_count_map" mapstructure:"prod_count_map"`
}

func (n *NodeResourceRequest) Parse(rawParams resourcetypes.RawParams) error {
	if err := mapstructure.Decode(rawParams, n); err != nil {
		return err
	}
	if n.ProdCountMap == nil {
		n.ProdCountMap = ProdCountMap{}
	}
	return nil
}

func (n *NodeResourceRequest) Validate() error {
	return n.ProdCountMap.Validate()
}

func (n *NodeResourceRequest) Count() int {
	return n.ProdCountMap.TotalCount()
}

func (n *NodeResourceRequest) LoadFromOrigin(nodeResource *NodeResource, resourceRequest resourcetypes.RawParams) {
	if n == nil {
		return
	}
	if !resourceRequest.IsSet("prod_count_map") {
		n.ProdCountMap = nodeResource.ProdCountMap
	}
}
