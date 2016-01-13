package blobstore

import (
	"io"

	"github.com/nyaxt/otaru/flags"
)

type MemBlobHandle struct {
	Content []byte
}

func NewMemBlobHandle() *MemBlobHandle {
	return &MemBlobHandle{
		Content: []byte{},
	}
}

func (bh *MemBlobHandle) PRead(p []byte, offset int64) error {
	if offset < 0 || offset > bh.Size() {
		return io.EOF
	}
	if offset+int64(len(p)) > int64(len(bh.Content)) {
		return io.EOF
	}
	copy(p, bh.Content[offset:offset+int64(len(p))])
	return nil
}

func (bh *MemBlobHandle) PWrite(p []byte, offset int64) error {
	if len(p) == 0 {
		return nil
	}

	right := offset + int64(len(p))
	if right > int64(len(bh.Content)) {
		bh.Truncate(right)
	}
	copy(bh.Content[offset:int(offset)+len(p)], p)
	return nil
}

func (bh *MemBlobHandle) Size() int64 {
	return int64(len(bh.Content))
}

func (bh *MemBlobHandle) Truncate(newSize int64) error {
	if newSize > bh.Size() {
		newContent := make([]byte, newSize)
		copy(newContent[:len(bh.Content)], bh.Content)
		bh.Content = newContent
	} else if newSize < bh.Size() {
		bh.Content = bh.Content[:int(newSize)]
	}
	return nil
}

func (bh *MemBlobHandle) Close() error {
	return nil
}

type MemBlobStore struct {
	Paths map[string]*MemBlobHandle
}

func NewMemBlobStore() *MemBlobStore {
	return &MemBlobStore{make(map[string]*MemBlobHandle)}
}

func (bs *MemBlobStore) Open(blobpath string, flags int) (BlobHandle, error) {
	bh := bs.Paths[blobpath]
	if bh == nil {
		bh = NewMemBlobHandle()
		bs.Paths[blobpath] = bh
	}
	return bh, nil
}

func (bs *MemBlobStore) Flags() int {
	return flags.O_RDWR
}
