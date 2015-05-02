package otaru

import (
	"fmt"
	"io"
	"math"
	"os"
)

type BlobStore interface {
	OpenWriter(blobpath string) (io.WriteCloser, error)
	OpenReader(blobpath string) (io.ReadCloser, error)
}

type BlobHandle interface {
	RandomAccessIO
	Size() int64
	Truncate(int64) error
	io.Closer
}

type TestBlobHandle struct {
	Buf []byte
}

func (bh *TestBlobHandle) PRead(offset int64, p []byte) error {
	if offset < 0 || int64(len(bh.Buf)) < offset+int64(len(p)) {
		return fmt.Errorf("PRead offset out of bound. buf len: %d while given offset: %d and len: %d", len(bh.Buf), offset, len(p))
	}

	copy(p, bh.Buf[offset:])
	return nil
}

func (bh *TestBlobHandle) PWrite(offset int64, p []byte) error {
	if offset < 0 || math.MaxInt32 < offset+int64(len(p)) {
		return fmt.Errorf("PWrite offset out of bound. buf len: %d while given offset: %d and len: %d", len(bh.Buf), offset, len(p))
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

func (bh *TestBlobHandle) Truncate(size int64) error {
	if size < int64(len(bh.Buf)) {
		bh.Buf = bh.Buf[:int(size)]
	}

	return nil
}

func (bh *TestBlobHandle) Size() int64 {
	return int64(len(bh.Buf))
}

func (TestBlobHandle) Close() error {
	return nil
}

const (
	O_RDONLY    int = os.O_RDONLY
	O_WRONLY    int = os.O_WRONLY
	O_RDWR      int = os.O_RDWR
	O_CREATE    int = os.O_CREATE
	O_EXCL      int = os.O_EXCL
	O_VALIDMASK int = O_RDONLY | O_WRONLY | O_RDWR | O_CREATE | O_EXCL
)

func IsReadAllowed(flags int) bool {
	return flags&(O_RDWR|O_RDONLY) != 0
}

func IsWriteAllowed(flags int) bool {
	return flags&(O_RDWR|O_WRONLY) != 0
}

func IsReadWriteAllowed(flags int) bool {
	return flags&O_RDWR != 0
}

type RandomAccessBlobStore interface {
	Open(blobpath string, flags int) (BlobHandle, error)

	Flags() int
}
