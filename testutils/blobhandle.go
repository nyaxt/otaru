package otaru_test

import (
	"fmt"
	"math"
)

type TestBlobHandle struct {
	Buf []byte
}

func (bh *TestBlobHandle) PRead(p []byte, offset int64) error {
	if offset < 0 || int64(len(bh.Buf)) < offset+int64(len(p)) {
		return fmt.Errorf("PRead offset out of bound. buf len: %d while given offset: %d and len: %d", len(bh.Buf), offset, len(p))
	}

	copy(p, bh.Buf[offset:])
	return nil
}

func (bh *TestBlobHandle) PWrite(p []byte, offset int64) error {
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
