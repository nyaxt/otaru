package otaru

import (
	"io"
)

type BlobHandle interface {
	RandomAccessIO
	Size() int64
	Truncate(int64) error
	io.Closer
}

type RandomAccessBlobStore interface {
	Open(blobpath string, flags int) (BlobHandle, error)
	Flags() int
}
