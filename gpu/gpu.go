package gpu

import (
	"context"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/core/resource/plugins"
	"github.com/projecteru2/core/store/etcdv3/embedded"
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
	name  string
	store *nodestore.Store[*gputypes.NodeResourceInfo]
}

func NewPlugin(ctx context.Context, config coretypes.Config, embeddedETCD *embedded.Cluster) (*Plugin, error) {
	store, err := nodestore.New(ctx, config, nodeResourceInfoKey, func() *gputypes.NodeResourceInfo {
		return &gputypes.NodeResourceInfo{}
	}, embeddedETCD)
	if err != nil {
		log.WithFunc("resource.gpu.NewPlugin").Error(ctx, err)
		return nil, err
	}
	return &Plugin{name: name, store: store}, nil
}

func (p Plugin) Name() string {
	return p.name
}
