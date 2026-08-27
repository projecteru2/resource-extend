package types

import (
	resourcetypes "github.com/projecteru2/core/resource/types"
	"github.com/projecteru2/core/utils"
)

func parseSizeInBytes(rawParams resourcetypes.RawParams, key string) (int64, error) {
	if size, ok := rawParams[key].(string); ok {
		return utils.ParseRAMInHuman(size)
	}
	return rawParams.Int64(key), nil
}
