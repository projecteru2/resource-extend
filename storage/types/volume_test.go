package types

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testGiB = int64(1) << 30

func TestNewVolumeBinding(t *testing.T) {
	tests := []struct {
		volume string
		want   VolumeBinding
	}{
		{"AUTO:/dir:rw:1G", VolumeBinding{Source: "AUTO", Destination: "/dir", Flags: "rw", SizeInBytes: testGiB}},
		{"/src:/dst", VolumeBinding{Source: "/src", Destination: "/dst", Flags: "rw"}},
		{"AUTO:/dir:wrm:1G", VolumeBinding{Source: "AUTO", Destination: "/dir", Flags: "mrw", SizeInBytes: testGiB}},
		{"AUTO:/dir::-1G", VolumeBinding{Source: "AUTO", Destination: "/dir", Flags: "rw", SizeInBytes: -testGiB}},
		{"/mnt:/dir:rw:0:100:200:1M:2M", VolumeBinding{Source: "/mnt", Destination: "/dir", Flags: "rw", ReadIOPS: 100, WriteIOPS: 200, ReadBPS: 1 << 20, WriteBPS: 2 << 20}},
	}
	for _, tt := range tests {
		t.Run(tt.volume, func(t *testing.T) {
			vb, err := NewVolumeBinding(tt.volume)
			require.NoError(t, err)
			assert.Equal(t, tt.want, *vb)
		})
	}

	for _, invalid := range []string{"one", "a:b:c:d:e:f:g:h:i", "AUTO:/dir:rw:xx", "AUTO::rw:1G"} {
		_, err := NewVolumeBinding(invalid)
		assert.Error(t, err, invalid)
	}
}

func TestVolumeBindingToString(t *testing.T) {
	vb, err := NewVolumeBinding("AUTO:/dir:mrw:1G")
	require.NoError(t, err)
	assert.Equal(t, "AUTO:/dir:mrw:1073741824:0:0:0:0", vb.ToString(false))
	assert.Equal(t, "AUTO:/dir:rw:1073741824", vb.ToString(true))

	vb, err = NewVolumeBinding("/src:/dst")
	require.NoError(t, err)
	assert.Equal(t, "/src:/dst:rw:0", vb.ToString(true))

	vb, err = NewVolumeBinding("AUTO:/dir:orw:1G:100:100:1M:1M")
	require.NoError(t, err)
	assert.Equal(t, "AUTO:/dir:rowo:1073741824:100:100:1048576:1048576", vb.ToString(true))
}

func TestVolumeBindingClassification(t *testing.T) {
	auto := mustBinding(t, "AUTO:/dir:rw:1G")
	assert.True(t, auto.RequireSchedule())
	assert.False(t, auto.RequireScheduleMonopoly())
	assert.False(t, auto.RequireScheduleUnlimitedQuota())

	mono := mustBinding(t, "AUTO:/dir:mrw:1G")
	assert.True(t, mono.RequireScheduleMonopoly())

	unlimited := mustBinding(t, "AUTO:/dir:rw")
	assert.True(t, unlimited.RequireScheduleUnlimitedQuota())

	mount := mustBinding(t, "/mnt:/dir:rw:0:100:100:1M:1M")
	assert.False(t, mount.RequireSchedule())
	assert.True(t, mount.RequireIOPS())
}

func TestVolumeBindingsJSONRoundTrip(t *testing.T) {
	vbs, err := NewVolumeBindings([]string{"AUTO:/dir1:rw:1G", "/mnt:/dir2:rw"})
	require.NoError(t, err)

	data, err := json.Marshal(vbs)
	require.NoError(t, err)

	decoded := VolumeBindings{}
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, vbs.String(), decoded.String())
}

func TestMergeVolumeBindings(t *testing.T) {
	base := mustBindings(t, []string{"AUTO:/dir1:rw:2G:100:100:0:0"})

	merged := MergeVolumeBindings(mustBindings(t, []string{"AUTO:/dir1:rw:1G:50:50:0:0"}), base)
	require.Len(t, merged, 1)
	assert.Equal(t, int64(3*testGiB), merged[0].SizeInBytes)
	assert.Equal(t, int64(150), merged[0].ReadIOPS)

	merged = MergeVolumeBindings(mustBindings(t, []string{"AUTO:/dir1:rw:-2G"}), base)
	require.Len(t, merged, 1)
	assert.Equal(t, int64(0), merged[0].SizeInBytes)

	merged = MergeVolumeBindings(mustBindings(t, []string{"AUTO:/dir1:rw:-3G"}), base)
	assert.Empty(t, merged)

	merged = MergeVolumeBindings(mustBindings(t, []string{"AUTO:/dir2:rw:1G"}), base)
	slices.SortFunc(merged, compareBindingString)
	assert.Len(t, merged, 2)
	assert.Equal(t, "/dir1", merged[0].Destination)
	assert.Equal(t, "/dir2", merged[1].Destination)
}

func TestVolumePlanMerge(t *testing.T) {
	b1 := mustBinding(t, "AUTO:/dir1:rw:1G")
	b2 := mustBinding(t, "AUTO:/dir2:rw:1G")
	plan := VolumePlan{b1: Volumes{"/data0": testGiB}}

	plan.Merge(VolumePlan{b2: Volumes{"/data1": testGiB}})
	assert.Len(t, plan, 2)

	expand := mustBinding(t, "AUTO:/dir1:rw:2G")
	plan.Merge(VolumePlan{expand: Volumes{"/data0": 2 * testGiB}})
	assert.Len(t, plan, 2)
	vmap, binding := plan.GetVolumes(b1)
	assert.Equal(t, int64(3*testGiB), vmap.GetSize())
	assert.Equal(t, int64(3*testGiB), binding.SizeInBytes)
}

func TestVolumePlanJSONRoundTrip(t *testing.T) {
	plan := VolumePlan{mustBinding(t, "AUTO:/dir1:rw:1G"): Volumes{"/data0": testGiB}}

	data, err := json.Marshal(plan)
	require.NoError(t, err)

	decoded := VolumePlan{}
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, plan.String(), decoded.String())
}

func TestApplyPlan(t *testing.T) {
	vbs := mustBindings(t, []string{"AUTO:/dir1:mrw:1G", "/mnt:/dir2:rw"})
	plan := VolumePlan{vbs[0]: Volumes{"/data0": 4 * testGiB}}

	applied := vbs.ApplyPlan(plan)
	require.Len(t, applied, 2)
	assert.Equal(t, "/data0", applied[0].Source)
	assert.Equal(t, int64(4*testGiB), applied[0].SizeInBytes)
	assert.Equal(t, "/mnt", applied[1].Source)
}

func TestVolumesArithmetic(t *testing.T) {
	v := Volumes{"/data0": 10, "/data1": 20}
	v.Add(Volumes{"/data0": 5, "/data2": 1})
	assert.Equal(t, Volumes{"/data0": 15, "/data1": 20, "/data2": 1}, v)

	v.Sub(Volumes{"/data1": 20})
	assert.Equal(t, int64(0), v["/data1"])
	assert.Equal(t, int64(16), v.Total())

	clone := v.DeepCopy()
	clone["/data0"] = 0
	assert.Equal(t, int64(15), v["/data0"])
}

func mustBinding(t *testing.T, volume string) *VolumeBinding {
	vb, err := NewVolumeBinding(volume)
	require.NoError(t, err)
	return vb
}

func mustBindings(t *testing.T, volumes []string) VolumeBindings {
	vbs, err := NewVolumeBindings(volumes)
	require.NoError(t, err)
	return vbs
}
