package blobstore // FIXME: make blobstore.cached pkg

import (
	"fmt"
	"log"
	"sync"
)

type CachedBackendVersion struct {
	backendbs    BlobStore
	queryVersion QueryVersionFunc

	mu    sync.Mutex
	cache map[string]BlobVersion
}

func NewCachedBackendVersion(backendbs BlobStore, queryVersion QueryVersionFunc) *CachedBackendVersion {
	return &CachedBackendVersion{
		backendbs:    backendbs,
		queryVersion: queryVersion,

		cache: make(map[string]BlobVersion),
	}
}

func (cbv *CachedBackendVersion) Set(blobpath string, ver BlobVersion) {
	cbv.mu.Lock()
	defer cbv.mu.Unlock()

	cbv.cache[blobpath] = ver
}

func (cbv *CachedBackendVersion) Query(blobpath string) (BlobVersion, error) {
	cbv.mu.Lock()
	defer cbv.mu.Unlock() // FIXME: unlock earlier?

	if ver, ok := cbv.cache[blobpath]; ok {
		log.Printf("return cached ver for \"%s\" -> %d", blobpath, ver)
		return ver, nil
	}

	r, err := cbv.backendbs.OpenReader(blobpath)
	if err != nil {
		if err == ENOENT {
			cbv.cache[blobpath] = 0
			return 0, nil
		}
		return -1, fmt.Errorf("Failed to open backend blob for ver query: %v", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("Failed to close backend blob handle for querying version: %v", err)
		}
	}()
	ver, err := cbv.queryVersion(r)
	if err != nil {
		return -1, fmt.Errorf("Failed to query backend blob ver: %v", err)
	}

	cbv.cache[blobpath] = ver
	return ver, nil
}

func (cbv *CachedBackendVersion) Delete(blobpath string) {
	cbv.mu.Lock()
	defer cbv.mu.Unlock()
	delete(cbv.cache, blobpath)
}

/*
// FIXME: dedupe below w/ blobstoredbstatesnapshotio to separate pkg

func (cbv *CachedBackendVersion) SaveStateToBlobstore(c btncrypt.Cipher, bs BlobStore) {
	raw, err := bs.OpenWriter(metadata.VersionCacheBlobpath)
	if err != nil {
		return err
	}

	cw := otaru.NewChunkWriter(raw, sio.c, ChunkHeader{
		OrigFilename: metadata.VersionCacheBlobpath,
		OrigOffset:   0,
	})
	bufio := bufio.NewWriter(cw)
	zw := zlib.NewWriter(bufio)
	enc := gob.NewEncoder(zw)

	es := []error{}
	if err := s.EncodeToGob(enc); err != nil {
		es = append(es, fmt.Errorf("Failed to encode DBState: %v", err))
	}
	if err := zw.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close zlib Writer: %v", err))
	}
	if err := bufio.Flush(); err != nil {
		es = append(es, fmt.Errorf("Failed to close bufio: %v", err))
	}
	if err := cio.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close ChunkIO: %v", err))
	}
	if err := raw.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close blobhandle: %v", err))
	}

	if err := util.ToErrors(es); err != nil {
		return err
	}
	sio.snapshotVer = s.Version()
	return nil
}
*/
