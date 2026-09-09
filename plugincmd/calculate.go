package plugincmd

import (
	"context"

	"github.com/projecteru2/core/resource/plugins"
	"github.com/projecteru2/core/resource/plugins/binary"
	resourcetypes "github.com/projecteru2/core/resource/types"
	"github.com/urfave/cli/v3"
)

func (r *runner) calculateCommands() []*cli.Command {
	return []*cli.Command{
		r.command(binary.CalculateDeployCommand, "calculate deploy plan", calculateDeploy),
		r.command(binary.CalculateReallocCommand, "calculate realloc plan", calculateRealloc),
	}
}

func calculateDeploy(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	return p.CalculateDeploy(ctx, node, in.Int("deploy_count"), in.RawParams("workload_resource_request"))
}

func calculateRealloc(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	return p.CalculateRealloc(ctx, node, in.RawParams("workload_resource"), in.RawParams("workload_resource_request"))
}
