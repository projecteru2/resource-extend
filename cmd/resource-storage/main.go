package main

import (
	"context"

	"github.com/projecteru2/core/resource/plugins"
	"github.com/projecteru2/core/resource/plugins/binary"
	coretypes "github.com/projecteru2/core/types"

	"github.com/projecteru2/resource-extend/plugincmd"
	"github.com/projecteru2/resource-extend/storage"
)

func main() {
	plugincmd.Main("resource-storage", "Run eru resource storage plugin", "storage.yaml",
		func(ctx context.Context, config coretypes.Config) (plugins.Plugin, error) {
			return storage.NewPlugin(ctx, config, nil)
		}, binary.CalculateRemapCommand)
}
