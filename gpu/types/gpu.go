package types

import (
	"maps"
	"strings"

	"github.com/cockroachdb/errors"
	resourcetypes "github.com/projecteru2/core/resource/types"
)

const prodCountMapKey = "prod_count_map"

type ProdCountMap map[string]int

func (pcm ProdCountMap) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		prodCountMapKey: pcm,
	}
}

func (pcm ProdCountMap) Validate() error {
	for prod, count := range pcm {
		if count <= 0 {
			return errors.Wrapf(ErrInvalidGPUMap, "count is less than or equal to zero: <product: %s, count: %d>", prod, count)
		}
		if strings.TrimSpace(prod) == "" {
			return errors.Wrap(ErrInvalidGPUProduct, "product is empty")
		}
	}
	return nil
}

// in order to support realloc, the count can be negative, so only validate prod here
func (pcm ProdCountMap) ValidateProd() error {
	for prod := range pcm {
		if strings.TrimSpace(prod) == "" {
			return errors.Wrap(ErrInvalidGPUProduct, "product is empty")
		}
	}
	return nil
}

func (pcm ProdCountMap) Add(pcm1 ProdCountMap) {
	for prod, count := range pcm1 {
		pcm[prod] += count
		if pcm[prod] == 0 {
			delete(pcm, prod)
		}
	}
}

func (pcm ProdCountMap) Sub(pcm1 ProdCountMap) {
	for prod, count := range pcm1 {
		pcm[prod] -= count
		if pcm[prod] == 0 {
			delete(pcm, prod)
		}
	}
}

func (pcm ProdCountMap) DeepCopy() ProdCountMap {
	cp := make(ProdCountMap, len(pcm))
	maps.Copy(cp, pcm)
	return cp
}

func (pcm ProdCountMap) Count() int {
	totalCount := 0
	for _, count := range pcm {
		totalCount += count
	}
	return totalCount
}
