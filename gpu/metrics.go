package gpu

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	plugintypes "github.com/projecteru2/core/resource/plugins/types"
)

func (p Plugin) GetMetricsDescription(context.Context) (*plugintypes.GetMetricsDescriptionResponse, error) {
	return &plugintypes.GetMetricsDescriptionResponse{
		{
			Name:   "gpu_capacity",
			Help:   "node available gpu.",
			Type:   "gauge",
			Labels: []string{"podname", "nodename", "product"},
		},
		{
			Name:   "gpu_used",
			Help:   "node used gpu.",
			Type:   "gauge",
			Labels: []string{"podname", "nodename", "product"},
		},
	}, nil
}

func (p Plugin) GetMetrics(ctx context.Context, nodes []plugintypes.NodeRef) (*plugintypes.GetMetricsResponse, error) {
	nodenames := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodenames = append(nodenames, node.Nodename)
	}
	infos, err := p.store.GetMulti(ctx, nodenames)
	if err != nil {
		return nil, err
	}
	metrics := plugintypes.GetMetricsResponse{}
	for _, node := range nodes {
		info := infos[node.Nodename]
		safeNodename := strings.ReplaceAll(node.Nodename, ".", "_")
		for prod, count := range info.Capacity.ProdCountMap {
			metrics = append(metrics,
				&plugintypes.Metrics{
					Name:   "gpu_capacity",
					Labels: []string{node.Podname, node.Nodename, prod},
					Value:  strconv.Itoa(count),
					Key:    fmt.Sprintf("core.node.%s.gpu.capacity", safeNodename),
				},
				&plugintypes.Metrics{
					Name:   "gpu_used",
					Labels: []string{node.Podname, node.Nodename, prod},
					Value:  strconv.Itoa(info.Usage.ProdCountMap[prod]),
					Key:    fmt.Sprintf("core.node.%s.gpu.used", safeNodename),
				},
			)
		}
	}
	return &metrics, nil
}
