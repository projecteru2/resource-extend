package main

import (
	"context"

	"github.com/projecteru2/core/resource/plugins"
	"github.com/projecteru2/core/resource/plugins/binary"
	coretypes "github.com/projecteru2/core/types"

	"github.com/projecteru2/resource-extend/gpu"
	"github.com/projecteru2/resource-extend/plugincmd"
)

func main() {
	plugincmd.Main("resource-gpu", "Run eru resource GPU plugin", "gpu.yaml",
		func(ctx context.Context, config coretypes.Config) (plugins.Plugin, error) {
			return gpu.NewPlugin(ctx, config)
		}, binary.CalculateRemapCommand)
}
