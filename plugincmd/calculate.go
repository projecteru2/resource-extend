package plugincmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/projecteru2/core/resource/plugins"
	"github.com/projecteru2/core/resource/plugins/binary"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	resourcetypes "github.com/projecteru2/core/resource/types"
)

func (r *runner) calculateCommands() []*cli.Command {
	return []*cli.Command{
		r.command(binary.CalculateDeployCommand, "calculate deploy plan", calculateDeploy),
		r.command(binary.CalculateReallocCommand, "calculate realloc plan", calculateRealloc),
		r.command(binary.CalculateRemapCommand, "remap resource", calculateRemap),
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

func calculateRemap(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	workloadsResource := map[string]plugintypes.WorkloadResource{}
	for id, raw := range in.RawParams("workloads_resource") {
		resource, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("workloads_resource[%s] is not an object", id)
		}
		workloadsResource[id] = resource
	}
	return p.CalculateRemap(ctx, node, workloadsResource)
}
