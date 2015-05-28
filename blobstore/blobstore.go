package blobstore

import (
	"io"
)

type BlobStore interface {
	OpenWriter(blobpath string) (io.WriteCloser, error)
	OpenReader(blobpath string) (io.ReadCloser, error)
}

type FlagsReader interface {
	Flags() int
}
