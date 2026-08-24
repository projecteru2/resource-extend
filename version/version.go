package version

import (
	"fmt"
	"runtime"
)

var (
	// VERSION is the release the binary was built from.
	VERSION = "unknown"
	// REVISION is the git commit the binary was built from.
	REVISION = "HEAD"
	// BUILTAT is the build timestamp.
	BUILTAT = "now"
)

// String reports the build identity of the running binary.
func String() string {
	return fmt.Sprintf(
		"Version:        %s\nGit hash:       %s\nBuilt:          %s\nGolang version: %s\nOS/Arch:        %s/%s\n",
		VERSION, REVISION, BUILTAT, runtime.Version(), runtime.GOOS, runtime.GOARCH,
	)
}
