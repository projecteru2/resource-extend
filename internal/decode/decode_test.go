package decode_test

import (
	"encoding/json"
	"testing"

	resourcetypes "github.com/projecteru2/core/resource/types"

	"github.com/projecteru2/resource-extend/internal/decode"
)

var (
	nodePayload = resourcetypes.RawParams{
		"volumes": map[string]int64{
			"/data0": 1 << 40,
			"/data1": 1 << 40,
			"/data2": 1 << 40,
			"/data3": 1 << 40,
		},
		"disks": []*disk{
			{Device: "/dev/vda", Mounts: []string{"/", "/data"}, ReadIOPS: 1000, WriteIOPS: 1000, ReadBPS: 1 << 30, WriteBPS: 1 << 30},
			{Device: "/dev/vdb", Mounts: []string{"/data1"}, ReadIOPS: 500, WriteIOPS: 500, ReadBPS: 1 << 29, WriteBPS: 1 << 29},
		},
		"storage": 4 << 40,
	}

	workloadPayload = resourcetypes.RawParams{
		"prod_count_map": map[string]int{"nvidia-3070": 2, "nvidia-3090": 2},
	}

	wirePayload = overWire(nodePayload)
)

func BenchmarkWireJSONv2(b *testing.B) {
	for b.Loop() {
		out := &nodeResource{}
		if err := decode.Decode(wirePayload, out); err != nil {
			b.Fatalf("decode: %v", err)
		}
	}
}

func BenchmarkNodeJSONv2(b *testing.B) {
	for b.Loop() {
		out := &nodeResource{}
		if err := decode.Decode(nodePayload, out); err != nil {
			b.Fatalf("decode: %v", err)
		}
	}
}

func BenchmarkWorkloadJSONv2(b *testing.B) {
	for b.Loop() {
		out := &workloadResource{}
		if err := decode.Decode(workloadPayload, out); err != nil {
			b.Fatalf("decode: %v", err)
		}
	}
}

func overWire(in resourcetypes.RawParams) resourcetypes.RawParams {
	body, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	out := resourcetypes.RawParams{}
	if err := json.Unmarshal(body, &out); err != nil {
		panic(err)
	}
	return out
}

type nodeResource struct {
	Volumes map[string]int64 `json:"volumes" mapstructure:"volumes"`
	Disks   []*disk          `json:"disks" mapstructure:"disks"`
	Storage int64            `json:"storage" mapstructure:"storage"`
}

type disk struct {
	Device    string   `json:"device" mapstructure:"device"`
	Mounts    []string `json:"mounts" mapstructure:"mounts"`
	ReadIOPS  int64    `json:"read_IOPS" mapstructure:"read_IOPS"`
	WriteIOPS int64    `json:"write_IOPS" mapstructure:"write_IOPS"`
	ReadBPS   int64    `json:"read_bps" mapstructure:"read_bps"`
	WriteBPS  int64    `json:"write_bps" mapstructure:"write_bps"`
}

type workloadResource struct {
	ProdCountMap map[string]int `json:"prod_count_map" mapstructure:"prod_count_map"`
}
