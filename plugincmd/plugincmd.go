// Package plugincmd exposes a core resource plugin as an eru binary plugin.
package plugincmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/projecteru2/core/resource/plugins"
	resourcetypes "github.com/projecteru2/core/resource/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/projecteru2/core/utils"
	"github.com/urfave/cli/v3"

	"github.com/projecteru2/resource-extend/version"
)

const (
	nameCommand = "name"
	exitCode    = 128
)

// Factory builds the plugin a command tree drives.
type Factory func(ctx context.Context, config coretypes.Config) (plugins.Plugin, error)

type handler func(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error)

type runner struct {
	configPath string
	newPlugin  Factory
}

func (r *runner) commands() []*cli.Command {
	return slices.Concat(
		[]*cli.Command{r.command(nameCommand, "show plugin name", name)},
		r.metricsCommands(),
		r.nodeCommands(),
		r.calculateCommands(),
	)
}

func (r *runner) command(name, usage string, h handler) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: usage,
		Action: func(ctx context.Context, _ *cli.Command) error {
			return r.serve(ctx, h)
		},
	}
}

// serve reads the request from stdin and writes the JSON result to stdout.
// core merges the child stderr into stdout, so nothing but the result is printed.
func (r *runner) serve(ctx context.Context, h handler) error {
	config, err := utils.LoadConfig(r.configPath)
	if err != nil {
		return cli.Exit(err, exitCode)
	}

	p, err := r.newPlugin(ctx, config)
	if err != nil {
		return cli.Exit(err, exitCode)
	}

	in := resourcetypes.RawParams{}
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		return cli.Exit(fmt.Errorf("decode plugin input: %w", err), exitCode)
	}

	out, err := h(ctx, p, in)
	if err != nil {
		return cli.Exit(err, exitCode)
	}

	data, err := json.Marshal(out)
	if err != nil {
		return cli.Exit(err, exitCode)
	}
	fmt.Print(string(data))
	return nil
}

// New builds the command tree of a resource plugin binary.
func New(name, usage, configPath string, newPlugin Factory) *cli.Command {
	r := &runner{newPlugin: newPlugin}
	return &cli.Command{
		Name:    name,
		Usage:   usage,
		Version: version.VERSION,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Value:       configPath,
				Usage:       "config file path for plugin, in yaml",
				Destination: &r.configPath,
				Sources:     cli.EnvVars("ERU_RESOURCE_CONFIG_PATH"),
			},
		},
		Commands: r.commands(),
	}
}

func name(_ context.Context, p plugins.Plugin, _ resourcetypes.RawParams) (any, error) {
	return p.Name(), nil
}

func nodename(in resourcetypes.RawParams) (string, error) {
	if n := in.String("nodename"); n != "" {
		return n, nil
	}
	return "", coretypes.ErrEmptyNodeName
}
