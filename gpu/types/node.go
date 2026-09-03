package types

import (
	"fmt"

	resourcetypes "github.com/projecteru2/core/resource/types"
)

type NodeResource struct {
	ProdCountMap `json:"prod_count_map"`
}

func NewNodeResource(prodCountMap ProdCountMap) *NodeResource {
	if prodCountMap == nil {
		prodCountMap = ProdCountMap{}
	}
	return &NodeResource{ProdCountMap: prodCountMap}
}

func (r *NodeResource) Parse(rawParams resourcetypes.RawParams) error {
	return resourcetypes.Decode(rawParams, r)
}

func (r *NodeResource) DeepCopy() *NodeResource {
	return &NodeResource{ProdCountMap: r.ProdCountMap.DeepCopy()}
}

func (r *NodeResource) Add(r1 *NodeResource) {
	r.ProdCountMap.Add(r1.ProdCountMap)
}

func (r *NodeResource) Sub(r1 *NodeResource) {
	r.ProdCountMap.Sub(r1.ProdCountMap)
}

type NodeResourceInfo struct {
	Capacity *NodeResource `json:"capacity"`
	Usage    *NodeResource `json:"usage"`
}

func NewNodeResourceInfo() *NodeResourceInfo {
	return &NodeResourceInfo{Capacity: NewNodeResource(nil), Usage: NewNodeResource(nil)}
}

func (n *NodeResourceInfo) CapCount() int {
	return n.Capacity.Count()
}

func (n *NodeResourceInfo) UsageCount() int {
	return n.Usage.Count()
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
	ProdCountMap `json:"prod_count_map"`
}

func (n *NodeResourceRequest) Parse(rawParams resourcetypes.RawParams) error {
	if err := resourcetypes.Decode(rawParams, n); err != nil {
		return err
	}
	if n.ProdCountMap == nil {
		n.ProdCountMap = ProdCountMap{}
	}
	return nil
}

func (n *NodeResourceRequest) LoadFromOrigin(nodeResource *NodeResource, resourceRequest resourcetypes.RawParams) {
	if !resourceRequest.IsSet(prodCountMapKey) {
		n.ProdCountMap = nodeResource.ProdCountMap
	}
}
