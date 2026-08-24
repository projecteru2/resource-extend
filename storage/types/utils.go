package types

import "strings"

func hasPrefix(path, mount string) bool {
	mount = addSlash(mount)
	path = addSlash(path)
	return strings.HasPrefix(path, mount)
}

func addSlash(dir string) string {
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}
