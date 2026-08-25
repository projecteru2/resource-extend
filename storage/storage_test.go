package storage

import (
	"context"
	"fmt"
	"testing"

	enginetypes "github.com/projecteru2/core/engine/types"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	"github.com/projecteru2/core/store/etcdv3/embedded"
	coretypes "github.com/projecteru2/core/types"
	"github.com/stretchr/testify/assert"
)

const (
	mib = 1 << 20
	gib = 1 << 30
	tib = 1 << 40

	tb = 1000 * 1000 * 1000 * 1000
)

func TestName(t *testing.T) {
	st := initStorage(t.Context(), t)
	assert.Equal(t, name, st.Name())
}

func initStorage(ctx context.Context, t *testing.T) *Plugin {
	config := coretypes.Config{
		Etcd: coretypes.EtcdConfig{
			Prefix: "/storage",
		},
		Scheduler: coretypes.SchedulerConfig{
			MaxShare:       -1,
			ShareBase:      100,
			MaxDeployCount: 100,
		},
	}

	cluster, err := embedded.New(t.TempDir())
	assert.NoError(t, err)
	t.Cleanup(cluster.Close)
	st, err := NewPlugin(ctx, config, cluster)
	assert.NoError(t, err)
	return st
}

func generateNodes(
	ctx context.Context, t *testing.T, st *Plugin, nums int, vols []string, index int,
) []string {
	reqs := generateNodeResourceRequests(t, nums, vols, index)
	info := &enginetypes.Info{StorageTotal: tb}
	names := []string{}
	for name, req := range reqs {
		_, err := st.AddNode(ctx, name, req, info)
		assert.NoError(t, err)
		names = append(names, name)
	}
	t.Cleanup(func() {
		cleanupCtx := context.WithoutCancel(ctx)
		for name := range reqs {
			_, _ = st.RemoveNode(cleanupCtx, name)
		}
	})
	return names
}

func generateNodeResourceRequests(t *testing.T, nums int, vols []string, index int) map[string]plugintypes.NodeResourceRequest {
	infos := map[string]plugintypes.NodeResourceRequest{}
	for i := index; i < index+nums; i++ {
		info := plugintypes.NodeResourceRequest{
			"volumes": vols,
		}
		infos[fmt.Sprintf("test%v", i)] = info
	}
	return infos
}
