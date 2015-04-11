package otaru

import (
	"io"
	"os"
)

type FileRandomAccessBlobStore struct{}

type FileBlobStore struct{}

func (f *FileBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	return os.Create(blobpath)
}

func (f *FileBlobStore) OpenReader(blobpath string) (io.ReadCloser, error) {
	return os.Open(blobpath)
}

type fileBlobHandle struct {
	fp *os.File
}

func (h fileBlobHandle) PRead(offset int64, p []byte) error {
	if _, err := h.fp.Seek(offset, os.SEEK_SET); err != nil {
		return err
	}
	if _, err := io.ReadFull(h.fp, p); err != nil {
		return err
	}
	return nil
}

func (h fileBlobHandle) PWrite(offset int64, p []byte) error {
	if _, err := h.fp.WriteAt(p, offset); err != nil {
		return err
	}
	return nil
}

func (h fileBlobHandle) Size() int64 {
	panic("Not Implemented")
	return 0
}

func (h fileBlobHandle) Close() error {
	return h.fp.Close()
}

func (f *FileBlobStore) Open(blobpath string) (BlobHandle, error) {
	fp, err := os.OpenFile(blobpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return &fileBlobHandle{fp}, nil
}
