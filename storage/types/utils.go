package types

import (
	resourcetypes "github.com/projecteru2/core/resource/types"
)

func firstStringSlice(rawParams resourcetypes.RawParams, keys ...string) []string {
	for _, key := range keys {
		if res := rawParams.StringSlice(key); len(res) > 0 {
			return res
		}
	}
	return nil
}
