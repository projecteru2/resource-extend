package gpu

import (
	"testing"

	plugintypes "github.com/projecteru2/core/resource/plugins/types"
	coretypes "github.com/projecteru2/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/projecteru2/resource-extend/gpu/types"
)

func TestCalculateDeploy(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]

	req := plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": -1,
			"nvidia-3090": 1,
		},
	}
	_, err := cm.CalculateDeploy(ctx, node, 100, req)
	assert.ErrorIs(t, err, types.ErrInvalidGPUMap)

	req = plugintypes.WorkloadResourceRequest{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
			"nvidia-3090": 1,
		},
	}
	_, err = cm.CalculateDeploy(ctx, "xxx", 100, req)
	assert.ErrorIs(t, err, coretypes.ErrNodeNotExists)

	parse := func(d *plugintypes.CalculateDeployResponse) (eps []*types.EngineParams, wrs []*types.WorkloadResource) {
		assert.NotNil(t, d.EnginesParams)
		assert.NotNil(t, d.WorkloadsResource)
		for _, epRaw := range d.EnginesParams {
			ep := &types.EngineParams{}
			err := ep.Parse(epRaw)
			assert.Nil(t, err)
			eps = append(eps, ep)
		}
		for _, wrRaw := range d.WorkloadsResource {
			wr := &types.WorkloadResource{}
			err := wr.Parse(wrRaw)
			assert.Nil(t, err)
			wrs = append(wrs, wr)
		}
		return eps, wrs
	}
	d, err := cm.CalculateDeploy(ctx, node, 4, nil)
	assert.Nil(t, err)
	assert.NotNil(t, d.EnginesParams)
	eParams, wResources := parse(d)
	assert.Len(t, eParams, 4)
	assert.Len(t, wResources, 4)
	for i := range 4 {
		assert.Equal(t, eParams[i].Count(), 0)
		assert.Equal(t, wResources[i].Count(), 0)
	}
	d, err = cm.CalculateDeploy(ctx, node, 4, req)
	assert.Nil(t, err)
	eParams, _ = parse(d)
	assert.Len(t, eParams, 4)

	_, err = cm.CalculateDeploy(ctx, node, 5, req)
	assert.Error(t, err)
}

func TestCalculateRealloc(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]

	resource := plugintypes.NodeResource{
		"prod_count_map": types.ProdCountMap{
			"nvidia-3070": 1,
			"nvidia-3090": 1,
		},
	}

	_, err := cm.SetNodeResourceUsage(ctx, node, resource, nil, nil, false, true)
	assert.Nil(t, err)

	origin := plugintypes.WorkloadResource{}
	req := plugintypes.WorkloadResourceRequest{}

	_, err = cm.CalculateRealloc(ctx, "xxx", origin, req)
	assert.ErrorIs(t, err, coretypes.ErrNodeNotExists)

	parse := func(d *plugintypes.CalculateReallocResponse) (*types.EngineParams, *types.WorkloadResource, *types.WorkloadResource) {
		assert.NotNil(t, d.EngineParams)
		assert.NotNil(t, d.WorkloadResource)
		ep := &types.EngineParams{}
		err := ep.Parse(d.EngineParams)
		assert.Nil(t, err)

		wr := &types.WorkloadResource{}
		err = wr.Parse(d.WorkloadResource)
		assert.Nil(t, err)

		dwr := &types.WorkloadResource{}
		err = dwr.Parse(d.DeltaResource)
		assert.Nil(t, err)
		return ep, wr, dwr
	}
	tests := []struct {
		name               string
		origin             plugintypes.WorkloadResource
		req                plugintypes.WorkloadResourceRequest
		wantEParamsCount   int
		wantEParamsMap     types.ProdCountMap
		wantWResourceCount int
		wantWResourceMap   types.ProdCountMap
		wantDResourceCount int
		wantDResourceMap   types.ProdCountMap
	}{
		{
			name: "no origin and no request",
		},
		{
			name: "origin only",
			origin: plugintypes.WorkloadResource{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": 1,
				},
			},
			wantEParamsCount:   1,
			wantEParamsMap:     types.ProdCountMap{"nvidia-3090": 1},
			wantWResourceCount: 1,
			wantWResourceMap:   types.ProdCountMap{"nvidia-3090": 1},
		},
		{
			name: "request scales up the existing model",
			origin: plugintypes.WorkloadResource{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": 1,
				},
			},
			req: plugintypes.WorkloadResourceRequest{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": 2,
				},
			},
			wantEParamsCount:   3,
			wantWResourceCount: 3,
			wantWResourceMap:   types.ProdCountMap{"nvidia-3090": 3},
			wantDResourceCount: 2,
			wantDResourceMap:   types.ProdCountMap{"nvidia-3090": 2},
		},
		{
			name: "request adds a second model",
			origin: plugintypes.WorkloadResource{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": 1,
				},
			},
			req: plugintypes.WorkloadResourceRequest{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": 1,
					"nvidia-3070": 1,
				},
			},
			wantEParamsCount:   3,
			wantEParamsMap:     types.ProdCountMap{"nvidia-3070": 1, "nvidia-3090": 2},
			wantWResourceCount: 3,
			wantWResourceMap:   types.ProdCountMap{"nvidia-3070": 1, "nvidia-3090": 2},
			wantDResourceCount: 2,
			wantDResourceMap:   types.ProdCountMap{"nvidia-3070": 1, "nvidia-3090": 1},
		},
		{
			name: "request scales down the existing model while adding another",
			origin: plugintypes.WorkloadResource{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": 1,
				},
			},
			req: plugintypes.WorkloadResourceRequest{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": -1,
					"nvidia-3070": 1,
				},
			},
			wantEParamsCount:   1,
			wantWResourceCount: 1,
			wantWResourceMap:   types.ProdCountMap{"nvidia-3070": 1},
			wantDResourceMap:   types.ProdCountMap{"nvidia-3070": 1, "nvidia-3090": -1},
		},
		{
			name: "request scale-down below zero is clamped",
			origin: plugintypes.WorkloadResource{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": 1,
				},
			},
			req: plugintypes.WorkloadResourceRequest{
				"prod_count_map": types.ProdCountMap{
					"nvidia-3090": -5,
					"nvidia-3070": 1,
				},
			},
			wantEParamsCount:   1,
			wantWResourceCount: 1,
			wantWResourceMap:   types.ProdCountMap{"nvidia-3070": 1},
			wantDResourceMap:   types.ProdCountMap{"nvidia-3070": 1, "nvidia-3090": -1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := cm.CalculateRealloc(ctx, node, tt.origin, tt.req)
			assert.Nil(t, err)
			eParams, wResource, dResource := parse(d)

			assert.Equal(t, tt.wantEParamsCount, eParams.Count())
			for k, v := range tt.wantEParamsMap {
				count, ok := eParams.ProdCountMap[k]
				assert.True(t, ok)
				assert.Equal(t, v, count)
			}

			assert.Equal(t, tt.wantWResourceCount, wResource.Count())
			for k, v := range tt.wantWResourceMap {
				count, ok := wResource.ProdCountMap[k]
				assert.True(t, ok)
				assert.Equal(t, v, count)
			}

			assert.Equal(t, tt.wantDResourceCount, dResource.Count())
			for k, v := range tt.wantDResourceMap {
				count, ok := dResource.ProdCountMap[k]
				assert.True(t, ok)
				assert.Equal(t, v, count)
			}
		})
	}
}

func TestCalculateRemap(t *testing.T) {
	ctx := t.Context()
	cm := initGPU(ctx, t)
	nodes := generateNodes(ctx, t, cm, 1, 0)
	node := nodes[0]
	d, err := cm.CalculateRemap(ctx, node, nil)

	assert.NoError(t, err)
	assert.Nil(t, d.EngineParamsMap)
}
