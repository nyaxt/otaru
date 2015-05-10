package otaru

import (
	"fmt"
	"io"
	"log"
)

// FIXME: handle overflows
type BlobVersion int64

type QueryVersionFunc func(r io.Reader) (BlobVersion, error)

type CachedBlobStore struct {
	backendbs BlobStore
	cachebs   RandomAccessBlobStore

	flags int

	queryVersion QueryVersionFunc
	beversion    map[string]BlobVersion
}

type CachedBlobHandle struct {
	cbs      *CachedBlobStore
	blobpath string
	cachebh  BlobHandle

	flags int

	isDirty bool
}

func (bh *CachedBlobHandle) PRead(offset int64, p []byte) error {
	return bh.cachebh.PRead(offset, p)
}

func (bh *CachedBlobHandle) PWrite(offset int64, p []byte) error {
	if !IsWriteAllowed(bh.flags) {
		return EPERM
	}
	if len(p) == 0 {
		return nil
	}
	bh.isDirty = true
	return bh.cachebh.PWrite(offset, p)
}

func (bh *CachedBlobHandle) Size() int64 {
	return bh.cachebh.Size()
}

func (bh *CachedBlobHandle) Truncate(newsize int64) error {
	if !IsWriteAllowed(bh.flags) {
		return EPERM
	}
	if bh.cachebh.Size() == newsize {
		return nil
	}
	bh.isDirty = true
	return bh.cachebh.Truncate(newsize)
}

func (bh *CachedBlobHandle) writeBack() error {
	if !bh.isDirty {
		return nil
	}
	if !IsWriteAllowed(bh.flags) {
		log.Printf("Write disallowed, but dirty flag is on somehow")
		return EPERM
	}

	w, err := bh.cbs.backendbs.OpenWriter(bh.blobpath)
	if err != nil {
		return fmt.Errorf("Failed to open backend blob writer: %v", err)
	}
	r := io.LimitReader(&OffsetReader{bh.cachebh, 0}, bh.Size())
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("Failed to copy dirty data to backend blob writer: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("Failed to close backend blob writer: %v", err)
	}

	bh.isDirty = false
	return nil
}

func (bh *CachedBlobHandle) Close() error {
	if bh.isDirty {
		if err := bh.writeBack(); err != nil {
			return err
		}
	}

	if err := bh.cachebh.Close(); err != nil {
		return err
	}
	return nil
}

func (cbs *CachedBlobStore) invalidateCache(blobpath string) error {
	backendr, err := cbs.backendbs.OpenReader(blobpath)
	if err != nil {
		return fmt.Errorf("Failed to open backend blob: %v", err)
	}

	bs, ok := cbs.cachebs.(BlobStore)
	if !ok {
		return fmt.Errorf("FIXME: only cachebs supporting OpenWriter is currently supported")
	}

	cachew, err := bs.OpenWriter(blobpath)
	if _, err := io.Copy(cachew, backendr); err != nil {
		return fmt.Errorf("Failed to copy blob from backend: %v", err)
	}
	if err := backendr.Close(); err != nil {
		return fmt.Errorf("Failed to close backend blob reader: %v", err)
	}
	if err := cachew.Close(); err != nil {
		return fmt.Errorf("Failed to close cache blob writer: %v", err)
	}

	// FIXME: check integrity here?

	return nil
}

func (cbs *CachedBlobStore) queryBackendVersion(blobpath string) (BlobVersion, error) {
	backendr, err := cbs.backendbs.OpenReader(blobpath)
	if err != nil {
		return -1, fmt.Errorf("Failed to open backend blob: %v", err)
	}
	// FIXME: use cache
	backendver, err := cbs.queryVersion(backendr)
	if err != nil {
		return -1, fmt.Errorf("Failed to query backend blob ver: %v", err)
	}
	if err := backendr.Close(); err != nil {
		return -1, fmt.Errorf("Failed to close backend blob handle for querying version: %v", err)
	}

	return backendver, nil
}

func NewCachedBlobStore(backendbs BlobStore, cachebs RandomAccessBlobStore, flags int, queryVersion QueryVersionFunc) (*CachedBlobStore, error) {
	if IsWriteAllowed(flags) {
		if fr, ok := backendbs.(FlagsReader); ok {
			if !IsWriteAllowed(fr.Flags()) {
				return nil, fmt.Errorf("Writable CachedBlobStore requested, but backendbs doesn't allow writes")
			}
		}
	}
	if !IsWriteAllowed(cachebs.Flags()) {
		return nil, fmt.Errorf("CachedBlobStore requested, but cachebs doesn't allow writes")
	}

	return &CachedBlobStore{
		backendbs: backendbs, cachebs: cachebs,
		flags:        flags,
		queryVersion: queryVersion,
	}, nil
}

func (cbs *CachedBlobStore) Open(blobpath string, flags int) (BlobHandle, error) {
	if !IsWriteAllowed(cbs.flags) && IsWriteAllowed(flags) {
		return nil, EPERM
	}

	cachebh, err := cbs.cachebs.Open(blobpath, O_RDWRCREATE)
	if err != nil {
		return nil, fmt.Errorf("Failed to open cache blob: %v", err)
	}

	cachever, err := cbs.queryVersion(&OffsetReader{cachebh, 0})
	if err != nil {
		return nil, fmt.Errorf("Failed to query cached blob ver: %v", err)
	}
	backendver, err := cbs.queryBackendVersion(blobpath)
	if err != nil {
		return nil, err
	}

	cbh := &CachedBlobHandle{
		cbs: cbs, blobpath: blobpath, cachebh: cachebh,
		flags:   flags,
		isDirty: false,
	}
	if cachever > backendver {
		log.Printf("FIXME: cache is newer than backend when open")
		cbh.isDirty = true
	} else if cachever == backendver {
		// ok
	} else {
		if err := cbs.invalidateCache(blobpath); err != nil {
			return nil, err
		}

		// reopen cachebh
		if err := cbh.cachebh.Close(); err != nil {
			return nil, fmt.Errorf("Failed to close cache blob for re-opening: %v", err)
		}
		var err error
		cbh.cachebh, err = cbs.cachebs.Open(blobpath, flags)
		if err != nil {
			return nil, fmt.Errorf("Failed to reopen cache blob: %v", err)
		}
	}

	return cbh, nil
}

func (cbs *CachedBlobStore) Flags() int {
	return cbs.flags
}
