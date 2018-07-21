package path

import (
	"path/filepath"
)

func ResolveLocalPath(localRootPath, relPath string) string {
	return filepath.Join(localRootPath, filepath.Clean("/"+relPath))
}
