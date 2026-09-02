package storage

import (
	"context"
	"fmt"
	"strings"

	plugintypes "github.com/projecteru2/core/resource/plugins/types"
)

func (p Plugin) GetMetricsDescription(context.Context) (*plugintypes.GetMetricsDescriptionResponse, error) {
	return &plugintypes.GetMetricsDescriptionResponse{
		{
			Name:   "storage_used",
			Help:   "node used storage.",
			Type:   "gauge",
			Labels: []string{"podname", "nodename"},
		},
		{
			Name:   "storage_capacity",
			Help:   "node available storage.",
			Type:   "gauge",
			Labels: []string{"podname", "nodename"},
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
		metrics = append(metrics,
			&plugintypes.Metrics{
				Name:   "storage_used",
				Labels: []string{node.Podname, node.Nodename},
				Value:  fmt.Sprintf("%+v", info.Usage.Storage),
				Key:    fmt.Sprintf("core.node.%s.storage.used", safeNodename),
			},
			&plugintypes.Metrics{
				Name:   "storage_capacity",
				Labels: []string{node.Podname, node.Nodename},
				Value:  fmt.Sprintf("%+v", info.Capacity.Storage),
				Key:    fmt.Sprintf("core.node.%s.storage.capacity", safeNodename),
			},
		)
	}
	return &metrics, nil
}
