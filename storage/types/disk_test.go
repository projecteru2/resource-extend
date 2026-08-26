package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskParse(t *testing.T) {
	disk := &Disk{}
	require.NoError(t, disk.Parse("/dev/vda:/,/data:100:200:1G:2G"))
	assert.Equal(t, &Disk{
		Device:    "/dev/vda",
		Mounts:    []string{"/", "/data"},
		ReadIOPS:  100,
		WriteIOPS: 200,
		ReadBPS:   testGiB,
		WriteBPS:  2 * testGiB,
	}, disk)

	for _, invalid := range []string{"/dev/vda:/:100:200:1G", "/dev/vda:/:x:200:1G:2G", "/dev/vda:/:100:200:xx:2G"} {
		assert.Error(t, (&Disk{}).Parse(invalid), invalid)
	}
}

func TestDiskStringRoundTrip(t *testing.T) {
	disk := &Disk{Device: "/dev/vda", Mounts: []string{"/", "/data"}, ReadIOPS: 100, WriteIOPS: 200, ReadBPS: 300, WriteBPS: 400}
	decoded := &Disk{}
	require.NoError(t, decoded.Parse(disk.String()))
	assert.Equal(t, disk, decoded)
}

func TestDisksAdd(t *testing.T) {
	disks := Disks{{Device: "/dev/vda", Mounts: []string{"/"}, ReadIOPS: 100}}
	extra := &Disk{Device: "/dev/vdb", ReadIOPS: 10}

	disks.Add(Disks{{Device: "/dev/vda", Mounts: []string{"/data"}, ReadIOPS: 1, WriteIOPS: 2, ReadBPS: 3, WriteBPS: 4}, extra})
	require.Len(t, disks, 2)
	assert.Equal(t, &Disk{Device: "/dev/vda", Mounts: []string{"/data"}, ReadIOPS: 101, WriteIOPS: 2, ReadBPS: 3, WriteBPS: 4}, disks[0])

	extra.ReadIOPS = 99
	assert.Equal(t, int64(10), disks[1].ReadIOPS)

	disks.Add(Disks{{Device: "/dev/vda", ReadIOPS: 1}})
	assert.Equal(t, []string{"/data"}, disks[0].Mounts)
}

func TestDisksSub(t *testing.T) {
	disks := Disks{{Device: "/dev/vda", ReadIOPS: 100, WriteIOPS: 100, ReadBPS: 100, WriteBPS: 100}}

	disks.Sub(Disks{
		{Device: "/dev/vda", ReadIOPS: 40, WriteIOPS: 30, ReadBPS: 20, WriteBPS: 10},
		{Device: "/dev/vdb", Mounts: []string{"/data"}, ReadIOPS: 5, WriteIOPS: 6, ReadBPS: 7, WriteBPS: 8},
	})
	require.Len(t, disks, 2)
	assert.Equal(t, &Disk{Device: "/dev/vda", ReadIOPS: 60, WriteIOPS: 70, ReadBPS: 80, WriteBPS: 90}, disks[0])
	assert.Equal(t, &Disk{Device: "/dev/vdb", Mounts: []string{"/data"}, ReadIOPS: -5, WriteIOPS: -6, ReadBPS: -7, WriteBPS: -8}, disks[1])

	disks.Add(Disks{{Device: "/dev/vdb", ReadIOPS: 5, WriteIOPS: 6, ReadBPS: 7, WriteBPS: 8}})
	assert.Equal(t, &Disk{Device: "/dev/vdb", Mounts: []string{"/data"}}, disks[1])
}

func TestDisksDeepCopy(t *testing.T) {
	disks := Disks{{Device: "/dev/vda", Mounts: []string{"/"}, ReadIOPS: 100}}
	clone := disks.DeepCopy()
	clone[0].ReadIOPS = 1
	assert.Equal(t, int64(100), disks[0].ReadIOPS)
}

func TestGetDiskByPath(t *testing.T) {
	disks := Disks{
		{Device: "/dev/vda", Mounts: []string{"/"}},
		{Device: "/dev/vdb", Mounts: []string{"/data"}},
		{Device: "/dev/vdc", Mounts: []string{"/data/sub/"}},
	}

	tests := []struct {
		path string
		want string
	}{
		{"/", "/dev/vda"},
		{"/etc", "/dev/vda"},
		{"/data", "/dev/vdb"},
		{"/data/x", "/dev/vdb"},
		{"/database", "/dev/vda"},
		{"/data/sub", "/dev/vdc"},
		{"/data/sub/x", "/dev/vdc"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			disk := disks.GetDiskByPath(tt.path)
			require.NotNil(t, disk)
			assert.Equal(t, tt.want, disk.Device)
		})
	}

	assert.Nil(t, Disks{{Device: "/dev/vdb", Mounts: []string{"/data"}}}.GetDiskByPath("/etc"))
}

func TestRemoveMounts(t *testing.T) {
	disks := Disks{{Device: "/dev/vda", Mounts: []string{"/"}, ReadIOPS: 100}}
	stripped := disks.RemoveMounts()
	assert.Nil(t, stripped[0].Mounts)
	assert.Equal(t, int64(100), stripped[0].ReadIOPS)
	assert.Equal(t, []string{"/"}, disks[0].Mounts)
}
