package otaru

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

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
	fi, err := h.fp.Stat()
	if err != nil {
		log.Fatalf("Stat failed: %v", err)
	}

	return fi.Size()
}

func (h fileBlobHandle) Truncate(size int64) {
	if err := h.fp.Truncate(size); err != nil {
		log.Fatalf("os.File.Truncate failed: %v", err)
	}
}

func (h fileBlobHandle) Close() error {
	return h.fp.Close()
}

type FileBlobStore struct {
	Base string
}

func NewFileBlobStore(base string) (*FileBlobStore, error) {
	base = path.Clean(base)

	fi, err := os.Stat(base)
	if err != nil {
		return nil, fmt.Errorf("Fstat base \"%s\" failed: %v", base, err)
	}
	if !fi.Mode().IsDir() {
		return nil, fmt.Errorf("Specified base \"%s\" is not a directory")
	}

	return &FileBlobStore{base}, nil
}

func (f *FileBlobStore) Open(blobpath string) (BlobHandle, error) {
	realpath := path.Join(f.Base, blobpath)
	fp, err := os.OpenFile(realpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return &fileBlobHandle{fp}, nil
}

func (f *FileBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	realpath := path.Join(f.Base, blobpath)
	return os.Create(realpath)
}

func (f *FileBlobStore) OpenReader(blobpath string) (io.ReadCloser, error) {
	realpath := path.Join(f.Base, blobpath)
	return os.Open(realpath)
}
