package testutils

import (
	"io"

	"github.com/nyaxt/otaru/blobstore"
)

type RWInterceptBlobStore struct {
	BE         blobstore.BlobStore
	WrapWriter func(orig io.WriteCloser) (io.WriteCloser, error)
	WrapReader func(orig io.ReadCloser) (io.ReadCloser, error)
}

var _ = blobstore.BlobStore(RWInterceptBlobStore{})

func (bs RWInterceptBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	orig, err := bs.BE.OpenWriter(blobpath)
	if err != nil {
		return nil, err
	}

	return bs.WrapWriter(orig)
}

func (bs RWInterceptBlobStore) OpenReader(blobpath string) (io.ReadCloser, error) {
	orig, err := bs.BE.OpenReader(blobpath)
	if err != nil {
		return nil, err
	}

	return bs.WrapReader(orig)
}

var _ = blobstore.BlobLister(RWInterceptBlobStore{})

func (bs RWInterceptBlobStore) ListBlobs() ([]string, error) {
	return bs.BE.(blobstore.BlobLister).ListBlobs()
}

var _ = blobstore.BlobSizer(RWInterceptBlobStore{})

func (bs RWInterceptBlobStore) BlobSize(blobpath string) (int64, error) {
	return bs.BE.(blobstore.BlobSizer).BlobSize(blobpath)
}
