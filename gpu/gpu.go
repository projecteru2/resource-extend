package gpu

import (
	"context"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/core/resource/plugins"
	coretypes "github.com/projecteru2/core/types"

	gputypes "github.com/projecteru2/resource-extend/gpu/types"
	"github.com/projecteru2/resource-extend/nodestore"
)

const (
	name                = "gpu"
	nodeResourceInfoKey = "/resource/gpu/%s"
	priority            = 100
)

var _ plugins.Plugin = (*Plugin)(nil)

type Plugin struct {
	store *nodestore.Store[*gputypes.NodeResourceInfo]
}

func (p Plugin) Name() string {
	return name
}

func NewPlugin(ctx context.Context, config coretypes.Config) (*Plugin, error) {
	kv, err := nodestore.Open(ctx, config)
	if err != nil {
		log.WithFunc("resource.gpu.NewPlugin").Error(ctx, err)
		return nil, err
	}
	return &Plugin{store: newStore(kv)}, nil
}

func newStore(kv nodestore.KV) *nodestore.Store[*gputypes.NodeResourceInfo] {
	return nodestore.New(kv, nodeResourceInfoKey, gputypes.NewNodeResourceInfo)
}
