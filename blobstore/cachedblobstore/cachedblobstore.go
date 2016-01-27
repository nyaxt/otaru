package cachedblobstore

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/blobstore/version"
	"github.com/nyaxt/otaru/btncrypt"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("cachedbs")

const (
	EPERM  = syscall.Errno(syscall.EPERM)
	ENFILE = syscall.Errno(syscall.ENFILE)
)

type CachedBlobStoreStats struct {
	NumWritebackOnClose int `json:num_writeback_on_close`
}

type CachedBlobStore struct {
	backendbs blobstore.BlobStore
	cachebs   blobstore.RandomAccessBlobStore
	s         *scheduler.Scheduler

	flags int

	queryVersion version.QueryFunc
	bever        *CachedBackendVersion

	entriesmgr CachedBlobEntriesManager
	usagestats *CacheUsageStats
	syncer     *CacheSyncer
	stats      CachedBlobStoreStats
}

func New(backendbs blobstore.BlobStore, cachebs blobstore.RandomAccessBlobStore, s *scheduler.Scheduler, flags int, queryVersion version.QueryFunc) (*CachedBlobStore, error) {
	if fl.IsWriteAllowed(flags) {
		if fr, ok := backendbs.(fl.FlagsReader); ok {
			if !fl.IsWriteAllowed(fr.Flags()) {
				return nil, fmt.Errorf("Writable CachedBlobStore requested, but backendbs doesn't allow writes")
			}
		}
	}
	if !fl.IsWriteAllowed(cachebs.Flags()) {
		return nil, fmt.Errorf("CachedBlobStore requested, but cachebs doesn't allow writes")
	}

	cbs := &CachedBlobStore{
		backendbs:    backendbs,
		cachebs:      cachebs,
		s:            s,
		flags:        flags,
		queryVersion: queryVersion,
		bever:        NewCachedBackendVersion(backendbs, queryVersion),
		entriesmgr:   NewCachedBlobEntriesManager(),
		usagestats:   NewCacheUsageStats(),
		stats:        CachedBlobStoreStats{},
	}
	cbs.syncer = NewCacheSyncer(&cbs.entriesmgr, defaultNumWorkers)

	if _, ok := cachebs.(blobstore.BlobRemover); !ok {
		return nil, fmt.Errorf("cachebs backend must support blob removals for failed-to-invalidate caches")
	}
	if lister, ok := cachebs.(blobstore.BlobLister); ok {
		bps, err := lister.ListBlobs()
		if err != nil {
			return nil, fmt.Errorf("Failed to list blobs to init CacheUsageStats: %v", err)
		}
		cbs.usagestats.ImportBlobList(bps)
	}

	go cbs.entriesmgr.Run()
	return cbs, nil
}

func (cbs *CachedBlobStore) RestoreState(c *btncrypt.Cipher) error {
	errs := []error{}

	if err := cbs.bever.RestoreStateFromBlobstore(c, cbs.backendbs); err != nil {
		errs = append(errs, err)
	}

	return util.ToErrors(errs)
}

func (cbs *CachedBlobStore) SaveState(c *btncrypt.Cipher) error {
	errs := []error{}

	if err := cbs.bever.SaveStateToBlobstore(c, cbs.backendbs); err != nil {
		errs = append(errs, err)
	}

	return util.ToErrors(errs)
}

func (cbs *CachedBlobStore) Sync() error {
	ss := cbs.entriesmgr.FindAllSyncable()
	return cbs.syncer.SyncAll(ss)
}

func (cbs *CachedBlobStore) Quit() error {
	err := cbs.Sync()
	cbs.syncer.Quit()
	cbs.entriesmgr.Quit()
	return err
}

var _ = blobstore.BlobStore(&CachedBlobStore{})

func (cbs *CachedBlobStore) OpenReader(blobpath string) (io.ReadCloser, error) {
	bh, err := cbs.Open(blobpath, fl.O_RDONLY)
	if err != nil {
		return nil, err
	}
	return &struct {
		blobstore.OffsetReader // for io.Reader
		blobstore.BlobHandle   // for io.Closer
	}{
		blobstore.OffsetReader{bh, 0},
		bh,
	}, nil
}

func (cbs *CachedBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	bh, err := cbs.Open(blobpath, fl.O_WRONLY|fl.O_CREATE)
	if err != nil {
		return nil, err
	}
	if err := bh.Truncate(0); err != nil {
		return nil, err
	}
	return &struct {
		blobstore.OffsetWriter // for io.Writer
		blobstore.BlobHandle   // for io.Closer
	}{
		blobstore.OffsetWriter{bh, 0},
		bh,
	}, nil
}

var _ = blobstore.RandomAccessBlobStore(&CachedBlobStore{})

func (cbs *CachedBlobStore) Flags() int {
	return cbs.flags
}

func (cbs *CachedBlobStore) Open(blobpath string, flags int) (blobstore.BlobHandle, error) {
	if !fl.IsWriteAllowed(cbs.flags) && fl.IsWriteAllowed(flags) {
		return nil, EPERM
	}

	cbs.usagestats.ObserveOpen(blobpath, flags)
	be, err := cbs.entriesmgr.OpenEntry(blobpath, cbs)
	if err != nil {
		return nil, err
	}
	return be.OpenHandle(flags)
}

func (cbs *CachedBlobStore) DumpEntriesInfo() []*CachedBlobEntryInfo {
	return cbs.entriesmgr.DumpEntriesInfo()
}

func (*CachedBlobStore) ImplName() string { return "CachedBlobStore" }

var _ = blobstore.BlobLister(&CachedBlobStore{})

func (cbs *CachedBlobStore) ListBlobs() ([]string, error) {
	belister, ok := cbs.backendbs.(blobstore.BlobLister)
	if !ok {
		return nil, fmt.Errorf("Backendbs \"%v\" doesn't support listing blobs.", util.TryGetImplName(cbs.backendbs))
	}

	belist, err := belister.ListBlobs()
	if err != nil {
		return nil, fmt.Errorf("Backendbs failed to ListBlobs: %v", err)
	}
	cachelist := cbs.entriesmgr.ListBlobs()
	cachelistset := make(map[string]struct{})
	for _, blobpath := range cachelist {
		cachelistset[blobpath] = struct{}{}
	}

	// list = append(cachelist, ...belist but entries already in cachelist)
	list := cachelist
	for _, blobpath := range belist {
		if _, ok := cachelistset[blobpath]; ok {
			// already in cachelist
			continue
		}
		list = append(list, blobpath)
	}

	return list, nil
}

var _ = blobstore.BlobRemover(&CachedBlobStore{})

func (cbs *CachedBlobStore) RemoveBlob(blobpath string) error {
	backendrm, ok := cbs.backendbs.(blobstore.BlobRemover)
	if !ok {
		return fmt.Errorf("Backendbs \"%v\" doesn't support removing blobs.", util.TryGetImplName(cbs.backendbs))
	}
	cacherm, ok := cbs.cachebs.(blobstore.BlobRemover)
	if !ok {
		return fmt.Errorf("Cachebs \"%v\" doesn't support removing blobs.", util.TryGetImplName(cbs.cachebs))
	}

	if err := cbs.entriesmgr.RemoveBlob(blobpath); err != nil {
		return err
	}
	cbs.bever.Delete(blobpath)
	if err := backendrm.RemoveBlob(blobpath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Backendbs RemoveBlob failed: %v", err)
	}
	cbs.usagestats.ObserveRemoveBlob(blobpath)
	if err := cacherm.RemoveBlob(blobpath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Cachebs RemoveBlob failed: %v", err)
	}

	return nil
}

func (cbs *CachedBlobStore) ReduceCache(ctx context.Context, desiredSize int64, dryrun bool) error {
	start := time.Now()

	tsizer, ok := cbs.cachebs.(blobstore.TotalSizer)
	if !ok {
		return fmt.Errorf("Cache backend \"%s\" doesn't support TotalSize() method, required to ReduceCache(). aborting.", util.TryGetImplName(cbs.cachebs))
	}

	blobsizer, ok := cbs.cachebs.(blobstore.BlobSizer)
	if !ok {
		return fmt.Errorf("Cache backend \"%s\" doesn't support BlobSize() method, required to ReduceCache(). aborting.", util.TryGetImplName(cbs.cachebs))
	}

	blobremover, ok := cbs.cachebs.(blobstore.BlobRemover)
	if !ok {
		return fmt.Errorf("Cache backend \"%s\" doesn't support RemoveBlob() method, required to ReduceCache(). aborting.", util.TryGetImplName(cbs.cachebs))
	}

	totalSizeBefore, err := tsizer.TotalSize()
	if err != nil {
		return fmt.Errorf("Failed to query current total cache size: %v", err)
	}

	needsReduce := totalSizeBefore - desiredSize
	if needsReduce < 0 {
		logger.Infof(mylog, "ReduceCache: No need to reduce cache as its already under desired size! No-op.")
		return nil
	}
	logger.Infof(mylog, "ReduceCache: Current cache bs total size: %s. Desired size: %s. Needs to reduce %s.",
		humanize.IBytes(uint64(totalSizeBefore)), humanize.IBytes(uint64(desiredSize)), humanize.IBytes(uint64(needsReduce)))

	bps := cbs.usagestats.FindLeastUsed()
	for _, bp := range bps {
		size, err := blobsizer.BlobSize(bp)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Infof(mylog, "Attempted to drop blob cache \"%s\", but not found. Maybe it's already removed.", bp)
				continue
			}
			return fmt.Errorf("Failed to query size for cache blob \"%s\": %v", bp, err)
		}

		logger.Infof(mylog, "ReduceCache: Drop entry \"%s\" to release %s", bp, humanize.IBytes(uint64(size)))

		if !dryrun {
			if err := cbs.entriesmgr.DropCacheEntry(bp, cbs, blobremover); err != nil {
				return fmt.Errorf("Failed to remove cache blob \"%s\": %v", bp, err)
			}
		}

		needsReduce -= size
		if needsReduce < 0 {
			break
		}
	}

	totalSizeAfter, err := tsizer.TotalSize()
	if err != nil {
		return fmt.Errorf("Failed to query current total cache size: %v", err)
	}

	logger.Infof(mylog, "ReduceCache done. Cache bs total size: %s -> %s. Dryrun: %t. Took: %s",
		humanize.IBytes(uint64(totalSizeBefore)), humanize.IBytes(uint64(totalSizeAfter)),
		dryrun, time.Since(start))
	return nil
}

type Stats struct {
	CacheUsageStatsView  `json:"usage_stats"`
	CbvStats             `json:"cbv_stats"`
	CachedBlobStoreStats `json:"stats"`
}

func (cbs *CachedBlobStore) GetStats() Stats {
	return Stats{
		CacheUsageStatsView:  cbs.usagestats.View(),
		CbvStats:             cbs.bever.GetStats(),
		CachedBlobStoreStats: cbs.stats,
	}
}

func (cbs *CachedBlobStore) CloseEntryForTesting(blobpath string) {
	cbs.entriesmgr.CloseEntryForTesting(blobpath)
}
