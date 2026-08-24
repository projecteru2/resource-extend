// Package nodestore keeps the per-node resource info of a resource plugin in etcd.
package nodestore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/store/etcdv3/embedded"
	"github.com/projecteru2/core/store/etcdv3/meta"
	coretypes "github.com/projecteru2/core/types"
	"github.com/projecteru2/core/utils"
)

// Info is the per-node resource info a Store persists.
type Info interface {
	Validate() error
}

// Factory allocates an empty Info to decode a stored value into.
type Factory[T Info] func() T

// Store reads and writes node resource info under one etcd key namespace.
type Store[T Info] struct {
	kv      meta.KV
	keyFmt  string
	newInfo Factory[T]
}

// New opens the store behind keyFmt, a key template with one node name verb.
func New[T Info](ctx context.Context, config coretypes.Config, keyFmt string, newInfo Factory[T], embeddedETCD *embedded.Cluster) (*Store[T], error) {
	if embeddedETCD == nil && len(config.Etcd.Machines) < 1 {
		return nil, coretypes.ErrConfigInvaild
	}
	kv, err := meta.NewETCD(config.Etcd, embeddedETCD)
	if err != nil {
		return nil, err
	}
	return &Store[T]{kv: kv, keyFmt: keyFmt, newInfo: newInfo}, nil
}

// Get returns the resource info of one node, ErrNodeNotExists when it has none.
func (s *Store[T]) Get(ctx context.Context, nodename string) (T, error) {
	info := s.newInfo()
	resp, err := s.kv.Get(ctx, s.key(nodename))
	if err != nil {
		return info, err
	}
	switch resp.Count {
	case 0:
		return info, errors.Wrapf(coretypes.ErrNodeNotExists, "key: %s", nodename)
	case 1:
		return info, json.Unmarshal(resp.Kvs[0].Value, info)
	default:
		return info, errors.Wrapf(coretypes.ErrInvaildCount, "key: %s", nodename)
	}
}

// GetMulti returns the resource info of every node, ErrInvaildCount when one has none.
func (s *Store[T]) GetMulti(ctx context.Context, nodenames []string) (map[string]T, error) {
	keys := make([]string, 0, len(nodenames))
	for _, nodename := range nodenames {
		keys = append(keys, s.key(nodename))
	}
	kvs, err := s.kv.GetMulti(ctx, keys)
	if err != nil {
		return nil, err
	}
	infos := make(map[string]T, len(kvs))
	for _, kv := range kvs {
		info := s.newInfo()
		if err := json.Unmarshal(kv.Value, info); err != nil {
			return nil, err
		}
		infos[utils.Tail(string(kv.Key))] = info
	}
	return infos, nil
}

// Put validates info and stores it as the resource info of one node.
func (s *Store[T]) Put(ctx context.Context, nodename string, info T) error {
	if err := info.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	_, err = s.kv.Put(ctx, s.key(nodename), string(data))
	return err
}

// Delete drops the resource info of one node.
func (s *Store[T]) Delete(ctx context.Context, nodename string) error {
	_, err := s.kv.Delete(ctx, s.key(nodename))
	return err
}

func (s *Store[T]) key(nodename string) string {
	return fmt.Sprintf(s.keyFmt, nodename)
}
