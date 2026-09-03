package plugincmd

import (
	"context"

	"github.com/cockroachdb/errors"
	enginetypes "github.com/projecteru2/core/engine/types"
	"github.com/projecteru2/core/resource/plugins"
	"github.com/projecteru2/core/resource/plugins/binary"
	resourcetypes "github.com/projecteru2/core/resource/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/urfave/cli/v3"
)

func (r *runner) nodeCommands() []*cli.Command {
	return []*cli.Command{
		r.command(binary.AddNodeCommand, "add node", addNode),
		r.command(binary.RemoveNodeCommand, "remove node", removeNode),
		r.command(binary.GetNodesDeployCapacityCommand, "get deploy capacity", getNodesDeployCapacity),
		r.command(binary.SetNodeResourceCapacityCommand, "set node capacity", setNodeResourceCapacity),
		r.command(binary.GetNodeResourceInfoCommand, "get node resource info", getNodeResourceInfo),
		r.command(binary.SetNodeResourceInfoCommand, "set node resource info", setNodeResourceInfo),
		r.command(binary.SetNodeResourceUsageCommand, "set node usage", setNodeResourceUsage),
		r.command(binary.GetMostIdleNodeCommand, "get most idle node", getMostIdleNode),
		r.command(binary.FixNodeResourceCommand, "fix node resource", fixNodeResource),
	}
}

func addNode(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	info := &enginetypes.Info{}
	if err := resourcetypes.Decode(in.RawParams("info"), info); err != nil {
		return nil, err
	}
	return p.AddNode(ctx, node, in.RawParams("resource"), info)
}

func removeNode(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	return p.RemoveNode(ctx, node)
}

func getNodesDeployCapacity(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	nodenames := in.StringSlice("nodenames")
	if len(nodenames) == 0 {
		return nil, coretypes.ErrEmptyNodeName
	}
	return p.GetNodesDeployCapacity(ctx, nodenames, in.RawParams("workload_resource"))
}

func setNodeResourceCapacity(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	return p.SetNodeResourceCapacity(ctx, node,
		in.RawParams("resource"), in.RawParams("resource_request"),
		in.Bool("delta"), in.Bool("incr"))
}

func getNodeResourceInfo(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	resp, err := p.GetNodeResourceInfo(ctx, node, in.SliceRawParams("workloads_resource"))
	// a node the plugin never saw has no resource of this kind, which is not a failure
	if errors.Is(err, coretypes.ErrNodeNotExists) {
		return resp, nil
	}
	return resp, err
}

func setNodeResourceInfo(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	return p.SetNodeResourceInfo(ctx, node, in.RawParams("capacity"), in.RawParams("usage"))
}

func setNodeResourceUsage(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	return p.SetNodeResourceUsage(ctx, node,
		in.RawParams("resource"), in.RawParams("resource_request"), in.SliceRawParams("workloads_resource"),
		in.Bool("delta"), in.Bool("incr"))
}

func getMostIdleNode(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	nodenames := in.StringSlice("nodenames")
	if len(nodenames) == 0 {
		return nil, coretypes.ErrEmptyNodeName
	}
	return p.GetMostIdleNode(ctx, nodenames)
}

func fixNodeResource(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	return p.FixNodeResource(ctx, node, in.SliceRawParams("workloads_resource"))
}
