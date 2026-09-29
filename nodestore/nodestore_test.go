package nodestore

import (
	"context"
	"errors"
	"testing"

	coretypes "github.com/projecteru2/core/types"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestGetAndCheckAbsent(t *testing.T) {
	down := errors.New("etcd unavailable")
	tests := []struct {
		name       string
		kv         *fakeKV
		wantGet    error
		wantAbsent error
	}{
		{"absent", &fakeKV{}, coretypes.ErrNodeNotExists, nil},
		{"present", &fakeKV{value: `{"name":"n1"}`}, nil, coretypes.ErrNodeExists},
		{"store error", &fakeKV{err: down}, down, down},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.kv, "/resource/%s", newTestInfo)
			info, err := s.Get(t.Context(), "n1")
			assert.ErrorIs(t, err, tt.wantGet)
			if tt.wantGet == nil {
				assert.Equal(t, "n1", info.Name)
			}
			assert.ErrorIs(t, s.CheckAbsent(t.Context(), "n1"), tt.wantAbsent)
		})
	}
}

type testInfo struct {
	Name string `json:"name"`
}

func newTestInfo() *testInfo { return &testInfo{} }

func (*testInfo) Validate() error { return nil }

type fakeKV struct {
	value string
	err   error
}

func (f *fakeKV) Get(_ context.Context, key string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.value == "" {
		return &clientv3.GetResponse{}, nil
	}
	return &clientv3.GetResponse{Count: 1, Kvs: []*mvccpb.KeyValue{{Key: []byte(key), Value: []byte(f.value)}}}, nil
}

func (*fakeKV) Put(context.Context, string, string) (*clientv3.PutResponse, error) {
	return &clientv3.PutResponse{}, nil
}

func (*fakeKV) Delete(context.Context, string) (*clientv3.DeleteResponse, error) {
	return &clientv3.DeleteResponse{}, nil
}
