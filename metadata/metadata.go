package metadata

import (
	"strings"
)

const INodeDBSnapshotBlobpath = "META_INODEDB_SNAPSHOT"
const VersionCacheBlobpath = "META_VERSION_CACHE"

func IsMetadataBlobpath(blobpath string) bool {
	return strings.HasPrefix(blobpath, "META_")
}
