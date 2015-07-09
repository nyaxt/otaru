package metadata

import (
	"strings"
)

const INodeDBSnapshotBlobpath = "META_INODEDB_SNAPSHOT"

func IsMetadataBlobpath(blobpath string) bool {
	return strings.HasPrefix(blobpath, "META_")
}
