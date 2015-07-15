package blobstore

import (
	"errors"
	"fmt"
	"io"

	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/util"
)

var (
	ErrEmptyMux = errors.New("blobstore.Mux is empty.")
)

type BlobMatcher func(blobpath string) bool

type MuxEntry struct {
	BlobMatcher
	BlobStore
}

type Mux []MuxEntry

var _ = BlobStore(Mux{})

func (m Mux) findBlobStoreFor(blobpath string) BlobStore {
	for i, e := range m {
		lastEntry := i == len(m)-1
		if lastEntry || e.BlobMatcher(blobpath) {
			return e.BlobStore
		}
	}
	return nil
}

func (m Mux) OpenWriter(blobpath string) (io.WriteCloser, error) {
	bs := m.findBlobStoreFor(blobpath)
	if bs == nil {
		return nil, ErrEmptyMux
	}
	return bs.OpenWriter(blobpath)
}

func (m Mux) OpenReader(blobpath string) (io.ReadCloser, error) {
	bs := m.findBlobStoreFor(blobpath)
	if bs == nil {
		return nil, ErrEmptyMux
	}
	return bs.OpenReader(blobpath)
}

var _ = RandomAccessBlobStore(Mux{})

func (m Mux) Open(blobpath string, flags int) (BlobHandle, error) {
	bs := m.findBlobStoreFor(blobpath)
	if bs == nil {
		return nil, ErrEmptyMux
	}
	rabs, ok := bs.(RandomAccessBlobStore)
	if !ok {
		return nil, fmt.Errorf("Backend blobstore \"%s\" don't support Open()", util.TryGetImplName(bs))
	}
	return rabs.Open(blobpath, flags)
}

var _ = fl.FlagsReader(Mux{})

func (m Mux) Flags() int {
	flags := fl.O_RDWRCREATE

	for _, e := range m {
		if flagsreader, ok := e.BlobStore.(fl.FlagsReader); ok {
			flags = fl.Mask(flags, flagsreader.Flags())
		}
	}

	return flags
}

var _ = BlobLister(Mux{})

func (m Mux) ListBlobs() ([]string, error) {
	ret := make([]string, 0)
	for _, e := range m {
		blobLister, ok := e.BlobStore.(BlobLister)
		if !ok {
			return nil, fmt.Errorf("Backend blobstore \"%s\" don't support ListBlobs()", util.TryGetImplName(e.BlobStore))
		}
		entries, err := blobLister.ListBlobs()
		if err != nil {
			return nil, fmt.Errorf("Backend blobstore \"%s\" failed to ListBlobs: %v", util.TryGetImplName(e.BlobStore), err)
		}
		ret = append(ret, entries...)
	}
	return ret, nil
}

var _ = BlobSizer(Mux{})

func (m Mux) BlobSize(blobpath string) (int64, error) {
	bs := m.findBlobStoreFor(blobpath)
	if bs == nil {
		return -1, ErrEmptyMux
	}
	sizer, ok := bs.(BlobSizer)
	if !ok {
		return -1, fmt.Errorf("Backend blobstore \"%s\" don't support BlobSize()", util.TryGetImplName(bs))
	}
	return sizer.BlobSize(blobpath)
}

var _ = BlobRemover(Mux{})

func (m Mux) RemoveBlob(blobpath string) error {
	bs := m.findBlobStoreFor(blobpath)
	if bs == nil {
		return ErrEmptyMux
	}
	remover, ok := bs.(BlobRemover)
	if !ok {
		return fmt.Errorf("Backend blobstore \"%s\" don't support RemoveBlob()", util.TryGetImplName(bs))
	}
	return remover.RemoveBlob(blobpath)
}

var _ = util.ImplNamed(Mux{})

func (Mux) ImplName() string { return "blobstore.Mux" }
