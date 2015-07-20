package metadata

import (
	"fmt"
	"strings"
	"time"
)

const INodeDBSnapshotBlobpathPrefix = "META_INODEDB_SNAPSHOT"
const VersionCacheBlobpath = "META_VERSION_CACHE"

func IsMetadataBlobpath(blobpath string) bool {
	return strings.HasPrefix(blobpath, "META_")
}

func GenINodeDBSnapshotBlobpath() string {
	return fmt.Sprintf("%s.%s",
		INodeDBSnapshotBlobpathPrefix,
		time.Now().Format("2006-01-02.150405.000"))
}
