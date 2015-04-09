package otaru

import (
	"io"
)

type BlobStore interface {
	OpenWriter(blobpath string) (io.WriteCloser, error)
	OpenReader(blobpath string) (io.ReadCloser, error)
}

type BlobHandle interface {
	RandomAccessIO
	io.Closer
}

type RandomAccessBlobStore interface {
	Open(blobpath string) (BlobHandle, error)
}
