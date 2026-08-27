package storage

import (
	"context"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/core/resource/plugins"
	"github.com/projecteru2/core/store/etcdv3/embedded"
	coretypes "github.com/projecteru2/core/types"

	"github.com/projecteru2/resource-extend/nodestore"
	storagetypes "github.com/projecteru2/resource-extend/storage/types"
)

const (
	name                = "storage"
	rate                = 8
	nodeResourceInfoKey = "/resource/storage/%s"
	priority            = 1
)

var _ plugins.Plugin = (*Plugin)(nil)

type Plugin struct {
	config coretypes.Config
	store  *nodestore.Store[*storagetypes.NodeResourceInfo]
}

func NewPlugin(ctx context.Context, config coretypes.Config, embeddedETCD *embedded.Cluster) (*Plugin, error) {
	store, err := nodestore.New(ctx, config, nodeResourceInfoKey, func() *storagetypes.NodeResourceInfo {
		return &storagetypes.NodeResourceInfo{}
	}, embeddedETCD)
	if err != nil {
		log.WithFunc("resource.storage.NewPlugin").Error(ctx, err)
		return nil, err
	}
	return &Plugin{config: config, store: store}, nil
}

func (p Plugin) Name() string {
	return name
}
