package gpu

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	plugintypes "github.com/projecteru2/core/resource/plugins/types"
)

// GetMetricsDescription .
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

// GetMetrics .
func (p Plugin) GetMetrics(ctx context.Context, podname, nodename string) (*plugintypes.GetMetricsResponse, error) {
	nodeResourceInfo, err := p.store.Get(ctx, nodename)
	if err != nil {
		return nil, err
	}
	safeNodename := strings.ReplaceAll(nodename, ".", "_")
	metrics := plugintypes.GetMetricsResponse{}
	for prod, count := range nodeResourceInfo.Capacity.ProdCountMap {
		metrics = append(metrics,
			&plugintypes.Metrics{
				Name:   "gpu_capacity",
				Labels: []string{podname, nodename, prod},
				Value:  strconv.Itoa(count),
				Key:    fmt.Sprintf("core.node.%s.gpu.capacity", safeNodename),
			},
			&plugintypes.Metrics{
				Name:   "gpu_used",
				Labels: []string{podname, nodename, prod},
				Value:  strconv.Itoa(nodeResourceInfo.Usage.ProdCountMap[prod]),
				Key:    fmt.Sprintf("core.node.%s.gpu.used", safeNodename),
			},
		)
	}
	return &metrics, nil
}
