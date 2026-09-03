package storage

import (
	"context"
	"fmt"
	"testing"

	enginetypes "github.com/projecteru2/core/engine/types"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	resourcetypes "github.com/projecteru2/core/resource/types"
	"github.com/projecteru2/core/store/etcdv3/embedded"
	"github.com/projecteru2/core/store/etcdv3/meta"
	coretypes "github.com/projecteru2/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/projecteru2/resource-extend/nodestore"
	"github.com/projecteru2/resource-extend/storage/types"
)

func TestAddNode(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	vols := []string{"/data0:1T"}
	nodes := generateNodes(ctx, t, st, 1, vols, 0)
	node := nodes[0]
	nodeForAdd := "test2"

	req := plugintypes.NodeResourceRequest{
		"volumes": vols,
	}
	info := &enginetypes.Info{StorageTotal: tb}

	_, err := st.AddNode(ctx, node, req, info)
	assert.Equal(t, err, coretypes.ErrNodeExists)

	r, err := st.AddNode(ctx, nodeForAdd, req, info)
	assert.Nil(t, err)
	assert.Equal(t, int64(tib), parseNodeResource(t, r.Capacity).Storage)
}

func TestRemoveNode(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	nodes := generateNodes(ctx, t, st, 1, defaultVols, 0)
	node := nodes[0]
	nodeForDel := "test2"

	_, err := st.RemoveNode(ctx, node)
	assert.Nil(t, err)
	_, err = st.RemoveNode(ctx, nodeForDel)
	assert.Nil(t, err)
}

func TestGetNodesDeployCapacity(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	nodes := generateNodes(ctx, t, st, 10, defaultVols, 0)

	_, err := st.GetNodesDeployCapacity(ctx, nodes, plugintypes.WorkloadResourceRequest{"storage": "-1"})
	assert.ErrorIs(t, err, types.ErrInvalidStorage)

	req := plugintypes.WorkloadResourceRequest{"storage": "1"}
	r0, err := st.GetNodesDeployCapacity(ctx, []string{"??"}, req)
	assert.NoError(t, err)
	assert.Empty(t, r0.NodeDeployCapacityMap)

	r0, err = st.GetNodesDeployCapacity(ctx, []string{"??"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, st.config.Scheduler.MaxDeployCount, r0.NodeDeployCapacityMap["??"].Capacity)

	req = plugintypes.WorkloadResourceRequest{"storage": fmt.Sprintf("%v", tib)}
	r, err := st.GetNodesDeployCapacity(ctx, nodes, req)
	assert.NoError(t, err)
	assert.Equal(t, 40, r.Total)

	req = plugintypes.WorkloadResourceRequest{"storage": "1G"}
	r, err = st.GetNodesDeployCapacity(ctx, nodes, req)
	assert.NoError(t, err)
	assert.Equal(t, 1000, r.Total)

	req = plugintypes.WorkloadResourceRequest{
		"volumes": []string{"AUTO:/dir0:rwm:1G"},
	}
	r, err = st.GetNodesDeployCapacity(ctx, nodes, req)
	assert.NoError(t, err)
	assert.Equal(t, 40, r.Total)

	req = plugintypes.WorkloadResourceRequest{
		"volumes": []string{"AUTO:/dir0:rwm:1G"},
		"storage": fmt.Sprintf("%v", tib),
	}
	r, err = st.GetNodesDeployCapacity(ctx, nodes, req)
	assert.NoError(t, err)
	assert.Equal(t, 30, r.Total)
}

func TestSetNodeResourceCapacityCreatesAnUnknownNode(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)

	r, err := st.SetNodeResourceCapacity(ctx, "never-added", nil, plugintypes.NodeResourceRequest{"storage": "1T"}, true, true)
	require.NoError(t, err)
	assert.Equal(t, int64(tib), parseNodeResource(t, r.After).Storage)

	gr, err := st.GetNodeResourceInfo(ctx, "never-added", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(tib), parseNodeResource(t, gr.Capacity).Storage)
}

func TestSetNodeResourceCapacity(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	nodes := generateNodes(ctx, t, st, 1, defaultVols, 0)
	node := nodes[0]

	r, err := st.GetNodeResourceInfo(ctx, node, nil)
	assert.Nil(t, err)
	assert.Equal(t, int64(4*tib), parseNodeResource(t, r.Capacity).Storage)

	resourceRequest := plugintypes.NodeResourceRequest{
		"volumes": []string{"/data4:1T"},
		"storage": "1T",
	}

	nodeResource := plugintypes.NodeResource{
		"volumes": types.Volumes{"/data4": tib},
		"storage": tib,
	}

	d, err := st.SetNodeResourceCapacity(ctx, node, nodeResource, nil, true, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(5*tib), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceCapacity(ctx, node, nodeResource, nil, true, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(4*tib), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceCapacity(ctx, node, nil, resourceRequest, true, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(6*tib), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceCapacity(ctx, node, nil, resourceRequest, true, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(4*tib), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceCapacity(ctx, node, nil, resourceRequest, false, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(2*tib), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceCapacity(ctx, node, nil, plugintypes.NodeResourceRequest{}, false, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(2*tib), parseNodeResource(t, d.After).Storage)
}

func TestSetNodeResourceCapacityRollback(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	node := generateNodes(ctx, t, st, 1, defaultVols, 0)[0]
	_, err := st.SetNodeResourceCapacity(ctx, node, nil, plugintypes.NodeResourceRequest{
		"disks": []string{"/dev/vda:/,/data0:1000:1000:1G:1G"},
	}, false, true)
	require.NoError(t, err)

	original, err := st.GetNodeResourceInfo(ctx, node, nil)
	require.NoError(t, err)

	updated, err := st.SetNodeResourceCapacity(ctx, node, nil, plugintypes.NodeResourceRequest{
		"volumes": []string{"/data4:1T"},
		"storage": "1T",
	}, false, true)
	require.NoError(t, err)

	rollbackRequest := plugintypes.NodeResourceRequest{}
	require.NoError(t, resourcetypes.Decode(updated.Before, &rollbackRequest))
	for range 2 {
		_, err = st.SetNodeResourceCapacity(ctx, node, nil, rollbackRequest, false, false)
		require.NoError(t, err)
	}

	restored, err := st.GetNodeResourceInfo(ctx, node, nil)
	require.NoError(t, err)
	assert.Equal(t, parseNodeResource(t, original.Capacity), parseNodeResource(t, restored.Capacity))
}

func TestSetNodeResourceCapacityRMDisks(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	node := generateNodes(ctx, t, st, 1, defaultVols, 0)[0]
	_, err := st.SetNodeResourceCapacity(ctx, node, nil, plugintypes.NodeResourceRequest{
		"disks": []string{"/dev/vda:/,/data0:1000:1000:1G:1G", "/dev/vdb:/data1:1000:1000:1G:1G"},
	}, false, true)
	require.NoError(t, err)

	_, err = st.SetNodeResourceCapacity(ctx, node, nil, plugintypes.NodeResourceRequest{"rm-disks": "/dev/vda"}, true, true)
	assert.ErrorIs(t, err, coretypes.ErrInvalidEngineArgs)

	_, err = st.SetNodeResourceUsage(ctx, node, plugintypes.NodeResource{
		"disks": types.Disks{{Device: "/dev/vdb"}},
	}, nil, nil, true, true)
	require.NoError(t, err)

	d, err := st.SetNodeResourceCapacity(ctx, node, nil, plugintypes.NodeResourceRequest{"rm-disks": "/dev/vdb"}, false, false)
	require.NoError(t, err)
	disks := parseNodeResource(t, d.After).Disks
	require.Len(t, disks, 1)
	assert.Equal(t, "/dev/vda", disks[0].Device)

	r, err := st.GetNodeResourceInfo(ctx, node, nil)
	require.NoError(t, err)
	for _, disk := range parseNodeResource(t, r.Usage).Disks {
		assert.NotEqual(t, "/dev/vdb", disk.Device)
	}

	_, err = st.SetNodeResourceUsage(ctx, node, plugintypes.NodeResource{
		"disks": types.Disks{{Device: "/dev/vda", ReadIOPS: 10}},
	}, nil, nil, true, true)
	require.NoError(t, err)
	_, err = st.SetNodeResourceCapacity(ctx, node, nil, plugintypes.NodeResourceRequest{"rm-disks": "/dev/vda"}, false, false)
	assert.ErrorIs(t, err, types.ErrInvalidDisk)
}

func TestGetNodeResourceInfo(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	nodes := generateNodes(ctx, t, st, 1, defaultVols, 0)
	node := nodes[0]

	_, err := st.GetNodeResourceInfo(ctx, "abc", nil)
	assert.ErrorIs(t, err, coretypes.ErrNodeNotExists)

	d, err := st.GetNodeResourceInfo(ctx, node, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(4*tib), parseNodeResource(t, d.Capacity).Storage)

	workloadsResource := []plugintypes.WorkloadResource{
		{"storage_request": 1},
		{"storage_limit": 1},
	}

	d, err = st.GetNodeResourceInfo(ctx, node, workloadsResource)
	assert.NoError(t, err)
	assert.NotEmpty(t, d.Diffs)
}

func TestSetNodeResourceInfo(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	nodes := generateNodes(ctx, t, st, 1, defaultVols, 0)
	node := nodes[0]

	capacity := plugintypes.NodeResource{
		"volumes": types.Volumes{"/data4": tib},
		"storage": tib,
	}

	usage := plugintypes.NodeResource{
		"volumes": types.Volumes{"/data3": tib},
		"storage": 4 * tib,
	}

	r, err := st.GetNodeResourceInfo(ctx, node, nil)
	assert.Nil(t, err)
	assert.Equal(t, int64(4*tib), parseNodeResource(t, r.Capacity).Storage)

	_, err = st.SetNodeResourceInfo(ctx, node, capacity, usage)
	assert.NoError(t, err)

	r, err = st.GetNodeResourceInfo(ctx, node, nil)
	assert.Nil(t, err)
	assert.Equal(t, int64(4*tib), parseNodeResource(t, r.Usage).Storage)
	assert.Equal(t, int64(2*tib), parseNodeResource(t, r.Capacity).Storage)
}

func TestSetNodeResourceUsage(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	nodes := generateNodes(ctx, t, st, 1, defaultVols, 0)
	node := nodes[0]

	r, err := st.GetNodeResourceInfo(ctx, node, nil)
	assert.Nil(t, err)
	assert.Equal(t, int64(4*tib), parseNodeResource(t, r.Capacity).Storage)

	resourceRequest := plugintypes.NodeResourceRequest{
		"volumes": []string{"/data4:1T"},
		"storage": "1T",
	}

	nodeResource := plugintypes.NodeResource{
		"volumes": types.Volumes{"/data4": tib},
		"storage": tib,
	}

	workloadsResource := []plugintypes.WorkloadResource{
		{"storage_request": 1},
		{"storage_limit": 1},
	}

	d, err := st.SetNodeResourceUsage(ctx, node, nodeResource, nil, nil, true, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(tib), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceUsage(ctx, node, nodeResource, nil, nil, true, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceUsage(ctx, node, nil, resourceRequest, nil, true, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(2*tib), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceUsage(ctx, node, nil, resourceRequest, nil, true, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceUsage(ctx, node, nil, nil, nil, true, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceUsage(ctx, node, nil, nil, workloadsResource, true, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceUsage(ctx, node, nil, nil, workloadsResource, true, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), parseNodeResource(t, d.After).Storage)

	d, err = st.SetNodeResourceUsage(ctx, node, nodeResource, nil, nil, false, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(tib), parseNodeResource(t, d.After).Storage)
}

func TestGetMostIdleNode(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	nodes := generateNodes(ctx, t, st, 1, defaultVols, 0)

	d, err := st.GetMostIdleNode(ctx, nodes)
	assert.NoError(t, err)
	assert.Equal(t, nodes[0], d.Nodename)

	_, err = st.GetMostIdleNode(ctx, nil)
	assert.ErrorIs(t, err, coretypes.ErrEmptyNodeName)
}

func TestFixNodeResource(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	nodes := generateNodes(ctx, t, st, 1, defaultVols, 0)
	node := nodes[0]

	_, err := st.FixNodeResource(ctx, "abc", nil)
	assert.ErrorIs(t, err, coretypes.ErrNodeNotExists)

	workloadsResource := []plugintypes.WorkloadResource{
		{"storage_request": 1},
		{"storage_limit": 1},
	}

	d, err := st.FixNodeResource(ctx, node, workloadsResource)
	assert.NoError(t, err)
	assert.NotEmpty(t, d.Diffs)

	d, err = st.GetNodeResourceInfo(ctx, node, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), parseNodeResource(t, d.Usage).Storage)
}

func TestFixNodeResourceReturnsPutError(t *testing.T) {
	ctx := t.Context()
	st, kv := initStorageWithKV(ctx, t)
	node := generateNodes(ctx, t, st, 1, defaultVols, 0)[0]
	st.store = newStore(putErrorKV{KV: kv})

	_, err := st.FixNodeResource(ctx, node, []plugintypes.WorkloadResource{{"storage_request": 1}})
	assert.ErrorIs(t, err, assert.AnError)
}

func BenchmarkDoGetNodeDeployCapacity(b *testing.B) {
	plugin := Plugin{config: coretypes.Config{Scheduler: coretypes.SchedulerConfig{MaxDeployCount: 10000}}}
	nodeResourceInfo := &types.NodeResourceInfo{
		Capacity: &types.NodeResource{Volumes: types.Volumes{"/data0": 1 << 40}, Disks: types.Disks{}, Storage: 1 << 40},
		Usage:    &types.NodeResource{Volumes: types.Volumes{"/data0": 0}, Disks: types.Disks{}},
	}
	req := &types.WorkloadResourceRequest{}
	if err := req.Parse(resourcetypes.RawParams{"storage": "1G"}); err != nil {
		b.Fatalf("setup: %v", err)
	}

	b.ReportAllocs()
	for b.Loop() {
		plugin.doGetNodeDeployCapacity(nodeResourceInfo, req)
	}
}

func BenchmarkGetNodesDeployCapacityScaling(b *testing.B) {
	for _, nodes := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("nodes=%d", nodes), func(b *testing.B) {
			ctx := b.Context()
			config := coretypes.Config{Etcd: coretypes.EtcdConfig{Prefix: "/storage"}, Scheduler: coretypes.SchedulerConfig{MaxShare: -1, ShareBase: 100, MaxDeployCount: 10000}}
			cluster, err := embedded.New(b.TempDir())
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(cluster.Close)
			kv, err := meta.NewETCD(ctx, config.Etcd, cluster)
			if err != nil {
				b.Fatal(err)
			}
			st := &Plugin{config: config, store: newStore(kv)}
			names := make([]string, 0, nodes)
			for name, req := range generateNodeResourceRequests(nodes, defaultVols, 0) {
				if _, err := st.AddNode(ctx, name, req, &enginetypes.Info{StorageTotal: tb}); err != nil {
					b.Fatal(err)
				}
				names = append(names, name)
			}
			req := plugintypes.WorkloadResourceRequest{"volumes": []string{"AUTO:/dir0:rwm:1G"}, "storage": "1G"}
			b.ResetTimer()
			for b.Loop() {
				if _, err := st.GetNodesDeployCapacity(ctx, names, req); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

type putErrorKV struct {
	nodestore.KV
}

func (putErrorKV) Put(context.Context, string, string) (*clientv3.PutResponse, error) {
	return nil, assert.AnError
}

func parseNodeResource(t *testing.T, raw resourcetypes.RawParams) *types.NodeResource {
	t.Helper()
	r := &types.NodeResource{}
	if err := r.Parse(raw); err != nil {
		t.Fatalf("parse node resource: %v", err)
	}
	return r
}
