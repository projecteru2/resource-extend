package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMetricsDescription(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	md, err := st.GetMetricsDescription(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, md)
	assert.Len(t, *md, 2)
}

func TestGetMetrics(t *testing.T) {
	ctx := t.Context()
	st := initStorage(ctx, t)
	_, err := st.GetMetrics(ctx, "", "")
	assert.Error(t, err)

	nodes := generateNodes(ctx, t, st, 1, defaultVols, 0)
	m, err := st.GetMetrics(ctx, "testpod", nodes[0])
	assert.NoError(t, err)
	assert.Len(t, *m, 2)

	keys := map[string]string{}
	for _, metric := range *m {
		keys[metric.Key] = metric.Name
	}
	assert.Len(t, keys, 2)
}
