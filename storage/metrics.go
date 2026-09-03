package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	"github.com/projecteru2/core/utils"
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
	infos, err := p.store.GetMulti(ctx, utils.Map(nodes, func(node plugintypes.NodeRef) string { return node.Nodename }))
	if err != nil {
		return nil, err
	}
	metrics := make(plugintypes.GetMetricsResponse, 0, 2*len(nodes))
	for _, node := range nodes {
		info := infos[node.Nodename]
		safeNodename := strings.ReplaceAll(node.Nodename, ".", "_")
		metrics = append(metrics,
			&plugintypes.Metrics{
				Name:   "storage_used",
				Labels: []string{node.Podname, node.Nodename},
				Value:  strconv.FormatInt(info.Usage.Storage, 10),
				Key:    fmt.Sprintf("core.node.%s.storage.used", safeNodename),
			},
			&plugintypes.Metrics{
				Name:   "storage_capacity",
				Labels: []string{node.Podname, node.Nodename},
				Value:  strconv.FormatInt(info.Capacity.Storage, 10),
				Key:    fmt.Sprintf("core.node.%s.storage.capacity", safeNodename),
			},
		)
	}
	return &metrics, nil
}
