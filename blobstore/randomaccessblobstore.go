package blobstore

import (
	"io"

	"github.com/nyaxt/otaru/flags"
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
	flags.FlagsReader
}
