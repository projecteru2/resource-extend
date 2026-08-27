package types

import (
	resourcetypes "github.com/projecteru2/core/resource/types"
	"github.com/projecteru2/core/utils"
)

func parseSizeInBytes(rawParams resourcetypes.RawParams, key string) (int64, error) {
	switch value := rawParams[key].(type) {
	case nil:
		return 0, nil
	case string:
		return utils.ParseRAMInHuman(value)
	default:
		return rawParams.Int64(key), nil
	}
}
