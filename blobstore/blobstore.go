package blobstore

import (
	"io"
)

type BlobStore interface {
	OpenWriter(blobpath string) (io.WriteCloser, error)
	OpenReader(blobpath string) (io.ReadCloser, error)
}

type BlobLister interface {
	ListBlobs() ([]string, error)
}

type BlobSizer interface {
	BlobSize(blobpath string) (int64, error)
}

type BlobRemover interface {
	RemoveBlob(blobpath string) error
}

type FlagsReader interface {
	Flags() int
}
