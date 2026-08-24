package main

import (
	"context"
	"fmt"
	"os"

	"github.com/projecteru2/core/resource/plugins"
	coretypes "github.com/projecteru2/core/types"
	"github.com/urfave/cli/v3"

	"github.com/projecteru2/resource-extend/plugincmd"
	"github.com/projecteru2/resource-extend/storage"
	"github.com/projecteru2/resource-extend/version"
)

func main() {
	cli.VersionPrinter = func(_ *cli.Command) {
		fmt.Print(version.String())
	}

	app := plugincmd.New("resource-storage", "Run eru resource storage plugin", "storage.yaml",
		func(ctx context.Context, config coretypes.Config) (plugins.Plugin, error) {
			return storage.NewPlugin(ctx, config, nil)
		})

	if err := app.Run(context.Background(), os.Args); err != nil {
		cli.HandleExitCoder(err)
		os.Exit(1)
	}
}
