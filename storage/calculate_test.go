package storage

import (
	"fmt"
	"testing"

	"github.com/docker/go-units"
	enginetypes "github.com/projecteru2/core/engine/types"
	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	resourcetypes "github.com/projecteru2/core/resource/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/sanity-io/litter"
	"github.com/stretchr/testify/assert"

	"github.com/projecteru2/resource-extend/storage/types"
)

func TestCalculateDeploy(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	vols := []string{"/data0:1T", "/data1:1T", "/data2:1T", "/data3:1T"}
	nodes := generateNodes(ctx, t, st, 1, vols, 0)
	node := nodes[0]

	req := plugintypes.WorkloadResourceRequest{"storage": "-1"}
	_, err := st.CalculateDeploy(ctx, node, 10, req)
	assert.ErrorIs(t, err, types.ErrInvalidStorage)

	req = plugintypes.WorkloadResourceRequest{
		"volumes": []string{"AUTO:/dir0:rwm:1G"},
	}
	_, err = st.CalculateDeploy(ctx, "no node", 10, req)
	assert.ErrorIs(t, err, coretypes.ErrNodeNotExists)

	req = plugintypes.WorkloadResourceRequest{
		"volumes": []string{"AUTO:/dir0:rwm:10T"},
	}
	_, err = st.CalculateDeploy(ctx, node, 10, req)
	assert.ErrorIs(t, err, coretypes.ErrInsufficientResource)

	req = plugintypes.WorkloadResourceRequest{
		"storage": fmt.Sprintf("%v", units.GiB),
	}
	d, err := st.CalculateDeploy(ctx, node, 10, req)
	assert.NoError(t, err)
	assert.Len(t, d.EnginesParams, 10)

	req = plugintypes.WorkloadResourceRequest{
		"volumes": []string{
			"AUTO:/dir0:rwm:1T",
		},
	}
	_, err = st.CalculateDeploy(ctx, node, 10, req)
	assert.Error(t, err)

	_, err = st.CalculateDeploy(ctx, node, 1, req)
	assert.NoError(t, err)
}

func TestCalculateRealloc(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	vols := []string{"/data0:1T", "/data1:1T", "/data2:1T", "/data3:1T"}
	nodes := generateNodes(ctx, t, st, 1, vols, 0)
	node := nodes[0]

	bindings, err := types.NewVolumeBindings([]string{
		"AUTO:/dir0:rw:100GiB",
		"AUTO:/dir1:mrw:100GiB",
		"AUTO:/dir2:rw:0",
	})
	assert.NoError(t, err)

	b1, err := types.NewVolumeBinding("AUTO:/dir0:rw:100GiB")
	assert.NoError(t, err)
	b2, err := types.NewVolumeBinding("AUTO:/dir1:mrw:100GiB")
	assert.NoError(t, err)
	b3, err := types.NewVolumeBinding("AUTO:/dir2:rw:0")
	assert.NoError(t, err)

	plan := types.VolumePlan{
		b1: types.Volumes{"/data0": 107374182400},
		b2: types.Volumes{"/data2": 1099511627776},
		b3: types.Volumes{"/data0": 0},
	}

	wrkResource := &types.WorkloadResource{
		VolumesRequest:    bindings,
		VolumesLimit:      bindings,
		VolumePlanRequest: plan,
		VolumePlanLimit:   plan,
		StorageRequest:    0,
		StorageLimit:      0,
	}
	resource := plugintypes.WorkloadResource(wrkResource.AsRawParams())

	_, err = st.SetNodeResourceUsage(ctx, node, nil, nil, []plugintypes.WorkloadResource{resource}, true, true)
	assert.NoError(t, err)

	req := plugintypes.WorkloadResourceRequest{}

	_, err = st.CalculateRealloc(ctx, "no node", resource, req)
	assert.ErrorIs(t, err, coretypes.ErrNodeNotExists)

	req = plugintypes.WorkloadResourceRequest{
		"volume-request":  []string{"AUTO:/dir0:rw:100GiB", "AUTO:/dir1:mrw:100GiB", "AUTO:/dir2:rw:0"},
		"storage-request": "-1",
		"storage-limit":   "-1",
	}
	_, err = st.CalculateRealloc(ctx, node, resource, req)
	assert.ErrorIs(t, err, types.ErrInvalidStorage)

	req = plugintypes.WorkloadResourceRequest{
		"volume-request":  []string{"AUTO:/dir1:mrw:100GiB"},
		"volume-limit":    []string{"AUTO:/dir1:mrw:100GiB"},
		"storage-request": fmt.Sprintf("%v", 4*units.TiB),
		"storage-limit":   fmt.Sprintf("%v", 4*units.TiB),
	}
	_, err = st.CalculateRealloc(ctx, node, resource, req)
	assert.ErrorIs(t, err, coretypes.ErrInsufficientResource)

	req = plugintypes.WorkloadResourceRequest{
		"volume-request": []string{"AUTO:/dir1:mrw:1TiB"},
		"volume-limit":   []string{"AUTO:/dir1:mrw:1TiB"},
	}
	_, err = st.CalculateRealloc(ctx, node, resource, req)
	assert.ErrorIs(t, err, coretypes.ErrInsufficientResource)

	req = plugintypes.WorkloadResourceRequest{
		"volume-request":  []string{"AUTO:/dir1:mrw:100GiB"},
		"volume-limit":    []string{"AUTO:/dir1:mrw:100GiB"},
		"storage-request": fmt.Sprintf("%v", units.GiB),
		"storage-limit":   fmt.Sprintf("%v", units.GiB),
	}
	d, err := st.CalculateRealloc(ctx, node, resource, req)
	assert.NoError(t, err)
	assert.False(t, parseEngineParams(t, d.EngineParams).VolumeChanged)

	wr := parseWorkloadResource(t, d.WorkloadResource)
	assert.Len(t, wr.VolumePlanRequest, 3)
	plan = types.VolumePlan{}
	assert.NoError(t, plan.UnmarshalJSON([]byte(`
	{
		"AUTO:/dir0:rw:100GiB": {
			"/data0": 107374182400
		  },
		  "AUTO:/dir1:mrw:200GiB": {
			"/data2": 1099511627776
		  },
		  "AUTO:/dir2:rw:0": {
			"/data0": 0
		  }
	}
	`)))
	assert.Equal(t, litter.Sdump(plan), litter.Sdump(wr.VolumePlanRequest))

	req = plugintypes.WorkloadResourceRequest{
		"storage-request": fmt.Sprintf("%v", units.GiB),
		"storage-limit":   fmt.Sprintf("%v", units.GiB),
	}
	d, err = st.CalculateRealloc(ctx, node, resource, req)
	assert.NoError(t, err)
	assert.False(t, parseEngineParams(t, d.EngineParams).VolumeChanged)

	wr = parseWorkloadResource(t, d.WorkloadResource)
	assert.Len(t, wr.VolumePlanRequest, 3)
	plan = types.VolumePlan{}
	assert.NoError(t, plan.UnmarshalJSON([]byte(`
	{
		"AUTO:/dir0:rw:100GiB": {
	        "/data0": 107374182400
	      },
	      "AUTO:/dir1:mrw:100GiB": {
	        "/data2": 1099511627776
	      },
	      "AUTO:/dir2:rw:0": {
	        "/data0": 0
	      }
	}
	`)))
	assert.Equal(t, litter.Sdump(plan), litter.Sdump(wr.VolumePlanRequest))
}

func TestCalculateRemap(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	vols := []string{"/data0:1T", "/data1:1T", "/data2:1T", "/data3:1T"}
	nodes := generateNodes(ctx, t, st, 1, vols, 0)
	node := nodes[0]
	d, err := st.CalculateRemap(ctx, node, nil)
	assert.NoError(t, err)
	assert.Nil(t, d.EngineParamsMap)
}

func TestCalculateReallocKeepsDisksWithoutReschedule(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	req := plugintypes.NodeResourceRequest{
		"volumes": []string{"/data0:1T"},
		"disks":   []string{"/dev/vda:/data0:1000:1000:1G:1G"},
	}
	_, err := st.AddNode(ctx, "disknode", req, &enginetypes.Info{StorageTotal: units.TB})
	assert.NoError(t, err)

	binding, err := types.NewVolumeBinding("AUTO:/dir0:rw:100GiB:100:100:1M:1M")
	assert.NoError(t, err)
	origin := &types.WorkloadResource{
		VolumesRequest:    types.VolumeBindings{binding},
		VolumesLimit:      types.VolumeBindings{binding},
		VolumePlanRequest: types.VolumePlan{binding: types.Volumes{"/data0": 107374182400}},
		VolumePlanLimit:   types.VolumePlan{binding: types.Volumes{"/data0": 107374182400}},
		DisksRequest:      types.Disks{{Device: "/dev/vda", ReadIOPS: 100, WriteIOPS: 100, ReadBPS: units.MiB, WriteBPS: units.MiB}},
	}

	d, err := st.CalculateRealloc(ctx, "disknode", origin.AsRawParams(), plugintypes.WorkloadResourceRequest{
		"storage-request": fmt.Sprintf("%v", units.GiB),
		"storage-limit":   fmt.Sprintf("%v", units.GiB),
	})
	assert.NoError(t, err)

	kept := parseWorkloadResource(t, d.WorkloadResource).DisksRequest.GetDiskByDevice("/dev/vda")
	assert.NotNil(t, kept)
	assert.Equal(t, int64(100), kept.ReadIOPS)

	delta := parseWorkloadResource(t, d.DeltaResource).DisksRequest.GetDiskByDevice("/dev/vda")
	if delta != nil {
		assert.Equal(t, int64(0), delta.ReadIOPS)
		assert.Equal(t, int64(0), delta.WriteIOPS)
	}
}

func parseEngineParams(t *testing.T, raw resourcetypes.RawParams) *types.EngineParams {
	t.Helper()
	ep := &types.EngineParams{}
	if err := ep.Parse(raw); err != nil {
		t.Fatalf("parse engine params: %v", err)
	}
	return ep
}

func parseWorkloadResource(t *testing.T, raw resourcetypes.RawParams) *types.WorkloadResource {
	t.Helper()
	wr := &types.WorkloadResource{}
	if err := wr.Parse(raw); err != nil {
		t.Fatalf("parse workload resource: %v", err)
	}
	return wr
}
