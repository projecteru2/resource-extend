package storage

import (
	"context"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/core/resource/plugins"
	coretypes "github.com/projecteru2/core/types"

	"github.com/projecteru2/resource-extend/nodestore"
	storagetypes "github.com/projecteru2/resource-extend/storage/types"
)

const (
	Name                = "storage"
	rate                = 8
	nodeResourceInfoKey = "/resource/storage/%s"
	priority            = 1
)

var _ plugins.Plugin = (*Plugin)(nil)

type Plugin struct {
	config coretypes.Config
	store  *nodestore.Store[*storagetypes.NodeResourceInfo]
}

func (p Plugin) Name() string {
	return Name
}

func NewPlugin(ctx context.Context, config coretypes.Config) (*Plugin, error) {
	kv, err := nodestore.Open(ctx, config)
	if err != nil {
		log.WithFunc("resource.storage.NewPlugin").Error(ctx, err)
		return nil, err
	}
	return &Plugin{config: config, store: newStore(kv)}, nil
}

func newStore(kv nodestore.KV) *nodestore.Store[*storagetypes.NodeResourceInfo] {
	return nodestore.New(kv, nodeResourceInfoKey, storagetypes.NewNodeResourceInfo)
}
