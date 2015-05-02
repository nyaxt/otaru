package otaru

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"syscall"
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

func (h fileBlobHandle) Truncate(size int64) error {
	if err := h.fp.Truncate(size); err != nil {
		return err
	}
	return nil
}

func (h fileBlobHandle) Close() error {
	return h.fp.Close()
}

type FileBlobStore struct {
	base  string
	flags int
	fmask int
}

func NewFileBlobStore(base string, flags int) (*FileBlobStore, error) {
	base = path.Clean(base)

	fi, err := os.Stat(base)
	if err != nil {
		return nil, fmt.Errorf("Fstat base \"%s\" failed: %v", base, err)
	}
	if !fi.Mode().IsDir() {
		return nil, fmt.Errorf("Specified base \"%s\" is not a directory")
	}

	fmask := O_RDONLY
	if IsWriteAllowed(flags) {
		fmask = O_RDONLY | O_WRONLY | O_RDWR | O_CREATE | O_EXCL
	}

	return &FileBlobStore{base, flags, fmask}, nil
}

func (f *FileBlobStore) Open(blobpath string, flags int) (BlobHandle, error) {
	realpath := path.Join(f.base, blobpath)

	fp, err := os.OpenFile(realpath, flags&f.fmask, 0644)
	if err != nil {
		if pe, ok := err.(*os.PathError); ok {
			if pe.Err == syscall.ENOENT {
				return nil, ENOENT
			}
		}
		return nil, err
	}
	return &fileBlobHandle{fp}, nil
}

func (f *FileBlobStore) Flags() int {
	return f.flags
}

func (f *FileBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	if !IsWriteAllowed(f.flags) {
		return nil, EPERM
	}

	realpath := path.Join(f.base, blobpath)
	return os.Create(realpath)
}

func (f *FileBlobStore) OpenReader(blobpath string) (io.ReadCloser, error) {
	if !IsReadAllowed(f.flags) {
		return nil, EPERM
	}

	realpath := path.Join(f.base, blobpath)
	return os.Open(realpath)
}
