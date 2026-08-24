package plugincmd

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/projecteru2/core/resource/plugins"
	"github.com/projecteru2/core/resource/plugins/binary"
	resourcetypes "github.com/projecteru2/core/resource/types"
)

func (r *runner) metricsCommands() []*cli.Command {
	return []*cli.Command{
		r.command(binary.GetMetricsDescriptionCommand, "show metrics descriptions", getMetricsDescription),
		r.command(binary.GetMetricsCommand, "show metrics", getMetrics),
	}
}

func getMetricsDescription(ctx context.Context, p plugins.Plugin, _ resourcetypes.RawParams) (any, error) {
	return p.GetMetricsDescription(ctx)
}

func getMetrics(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	return p.GetMetrics(ctx, in.String("podname"), in.String("nodename"))
}
