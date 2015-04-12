package otaru

import (
	"fmt"
	"io"
	"math"
)

type BlobStore interface {
	OpenWriter(blobpath string) (io.WriteCloser, error)
	OpenReader(blobpath string) (io.ReadCloser, error)
}

type BlobHandle interface {
	RandomAccessIO
	Size() int64
	io.Closer
}

type TestBlobHandle struct {
	Buf []byte
}

func (bh *TestBlobHandle) PRead(offset int64, p []byte) error {
	if offset < 0 || int64(len(bh.Buf)) < offset+int64(len(p)) {
		return fmt.Errorf("offset out of bound. buf len: %d while given offset: %d and len: %p", len(bh.Buf), offset, len(p))
	}

	copy(p, bh.Buf[offset:])
	return nil
}

func (bh *TestBlobHandle) PWrite(offset int64, p []byte) error {
	if offset < 0 || math.MaxInt32 < offset+int64(len(p)) {
		return fmt.Errorf("offset out of bound. buf len: %d while given offset: %d and len: %p", len(bh.Buf), offset, len(p))
	}
	if int64(len(bh.Buf)) < offset+int64(len(p)) {
		newsize := offset + int64(len(p))
		buf := make([]byte, newsize)
		copy(buf[:len(bh.Buf)], bh.Buf)
		bh.Buf = buf
	}

	copy(bh.Buf[offset:], p)
	return nil
}

func (bh *TestBlobHandle) Size() int64 {
	return int64(len(bh.Buf))
}

func (TestBlobHandle) Close() error {
	return nil
}

type RandomAccessBlobStore interface {
	Open(blobpath string) (BlobHandle, error)
}
