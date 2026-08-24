package storage

import (
	"context"
	"fmt"
	"strings"

	plugintypes "github.com/projecteru2/core/resource/plugins/types"
)

// GetMetricsDescription .
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

// GetMetrics .
func (p Plugin) GetMetrics(ctx context.Context, podname, nodename string) (*plugintypes.GetMetricsResponse, error) {
	nodeResourceInfo, err := p.store.Get(ctx, nodename)
	if err != nil {
		return nil, err
	}
	safeNodename := strings.ReplaceAll(nodename, ".", "_")
	return &plugintypes.GetMetricsResponse{
		{
			Name:   "storage_used",
			Labels: []string{podname, nodename},
			Value:  fmt.Sprintf("%+v", nodeResourceInfo.Usage.Storage),
			Key:    fmt.Sprintf("core.node.%s.storage.used", safeNodename),
		},
		{
			Name:   "storage_capacity",
			Labels: []string{podname, nodename},
			Value:  fmt.Sprintf("%+v", nodeResourceInfo.Capacity.Storage),
			Key:    fmt.Sprintf("core.node.%s.storage.used", safeNodename),
		},
	}, nil
}
