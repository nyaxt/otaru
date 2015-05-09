package otaru

import (
	"io"
)

type SizeQueryable interface {
	Size() int64
}

type BlobHandle interface {
	RandomAccessIO
	SizeQueryable
	Truncate(int64) error
	io.Closer
}

type RandomAccessBlobStore interface {
	Open(blobpath string, flags int) (BlobHandle, error)
	FlagsReader
}
