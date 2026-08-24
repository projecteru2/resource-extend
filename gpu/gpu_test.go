package gpu

import (
	"context"
	"fmt"
	"maps"
	"testing"

	enginetypes "github.com/projecteru2/core/engine/types"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	"github.com/projecteru2/core/store/etcdv3/embedded"
	coretypes "github.com/projecteru2/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/projecteru2/resource-extend/gpu/types"
)

func TestName(t *testing.T) {
	cm := initGPU(t.Context(), t)
	assert.Equal(t, name, cm.Name())
}

func initGPU(ctx context.Context, t *testing.T) *Plugin {
	config := coretypes.Config{
		Etcd: coretypes.EtcdConfig{
			Prefix: "/gpu",
		},
		Scheduler: coretypes.SchedulerConfig{
			MaxShare:  -1,
			ShareBase: 100,
		},
	}

	cluster, err := embedded.New(t.TempDir())
	assert.NoError(t, err)
	t.Cleanup(cluster.Close)
	cm, err := NewPlugin(ctx, config, cluster)
	assert.NoError(t, err)
	return cm
}

func generateNodes(
	ctx context.Context, t *testing.T, cm *Plugin,
	nums, index int,
) []string {
	reqs := generateNodeResourceRequests(t, nums, index, "test", 8)
	info := &enginetypes.Info{NCPU: 8, MemTotal: 2048}
	names := []string{}
	for name, req := range reqs {
		_, err := cm.AddNode(ctx, name, req, info)
		assert.NoError(t, err)
		names = append(names, name)
	}
	t.Cleanup(func() {
		cleanupCtx := context.WithoutCancel(ctx)
		for name := range reqs {
			_, err := cm.RemoveNode(cleanupCtx, name)
			assert.NoError(t, err)
		}
	})
	return names
}

func generateEmptyNodes(
	ctx context.Context, t *testing.T, cm *Plugin,
	nums, index int,
) []string {
	reqs := generateNodeResourceRequests(t, nums, index, "test-empty", 0)
	info := &enginetypes.Info{NCPU: 8, MemTotal: 2048}
	names := []string{}
	for name, req := range reqs {
		_, err := cm.AddNode(ctx, name, req, info)
		assert.NoError(t, err)
		names = append(names, name)
	}
	t.Cleanup(func() {
		cleanupCtx := context.WithoutCancel(ctx)
		for name := range reqs {
			_, err := cm.RemoveNode(cleanupCtx, name)
			assert.NoError(t, err)
		}
	})
	return names
}

func generateNodeResourceRequests(t *testing.T, nums, index int, namePrefix string, numGPUs int) map[string]plugintypes.NodeResourceRequest {
	gpuMap := types.ProdCountMap{
		"nvidia-3070": numGPUs / 2,
		"nvidia-3090": numGPUs / 2,
	}

	maps.DeleteFunc(gpuMap, func(_ string, count int) bool { return count <= 0 })

	infos := map[string]plugintypes.NodeResourceRequest{}
	for i := index; i < index+nums; i++ {
		info := plugintypes.NodeResourceRequest{
			"prod_count_map": gpuMap,
		}
		infos[fmt.Sprintf("%s%d", namePrefix, i)] = info
	}
	return infos
}
