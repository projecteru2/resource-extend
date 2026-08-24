package main

import (
	"context"
	"fmt"
	"os"

	"github.com/projecteru2/core/resource/plugins"
	coretypes "github.com/projecteru2/core/types"
	"github.com/urfave/cli/v3"

	"github.com/projecteru2/resource-extend/gpu"
	"github.com/projecteru2/resource-extend/plugincmd"
	"github.com/projecteru2/resource-extend/version"
)

func main() {
	cli.VersionPrinter = func(_ *cli.Command) {
		fmt.Print(version.String())
	}

	app := plugincmd.New("resource-gpu", "Run eru resource GPU plugin", "gpu.yaml",
		func(ctx context.Context, config coretypes.Config) (plugins.Plugin, error) {
			return gpu.NewPlugin(ctx, config, nil)
		})

	if err := app.Run(context.Background(), os.Args); err != nil {
		cli.HandleExitCoder(err)
		os.Exit(1)
	}
}
