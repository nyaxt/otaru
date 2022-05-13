package cachedblobstore_test

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"context"

	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/scheduler"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
)

func init() { tu.EnsureLogger() }

func TestCachedBlobStore(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	if err := tu.AssertBlobVersion(backendbs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}
	// assert cache not yet filled
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 0); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersionRA(bs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	// assert cache fill
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	if err := tu.WriteVersionedBlobRA(bs, "backendonly", 10); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}

	if err := tu.AssertBlobVersionRA(bs, "backendonly", 10); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 10); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(backendbs, "backendonly", 10); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestCachedBlobStore_WritebackDirty(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	tu.ObservedCriticalLog = false

	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	if err := tu.WriteVersionedBlob(bs, "hoge", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("%v", err)
		return
	}

	if tu.ObservedCriticalLog {
		t.Errorf("ObservedCriticalLog should be false")
		return
	}
}

type PausableReader struct {
	BE      io.ReadCloser
	OnReadC chan struct{}
	WaitC   chan struct{}
}

func (r PausableReader) Read(p []byte) (int, error) {
	r.OnReadC <- struct{}{}
	<-r.WaitC
	fmt.Printf("Read!!!\n")
	return r.BE.Read(p)
}

func (r PausableReader) Close() error {
	return r.BE.Close()
}

func TestCachedBlobStore_InvalidateCancel(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	onReadC := make(chan struct{})
	waitC := make(chan struct{})

	backendbs := tu.RWInterceptBlobStore{
		BE:         tu.TestFileBlobStoreOfName("backend"),
		WrapWriter: func(orig io.WriteCloser) (io.WriteCloser, error) { return orig, nil },
		WrapReader: func(orig io.ReadCloser) (io.ReadCloser, error) {
			return PausableReader{orig, onReadC, waitC}, nil
		},
	}
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	if err := tu.WriteVersionedBlob(cachebs, "backendnewer", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlob(backendbs, "backendnewer", 3); err != nil {
		t.Errorf("%v", err)
		return
	}

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	join := make(chan struct{})
	go func() {
		if err := tu.AssertBlobVersionRA(bs, "backendnewer", 3); err == nil {
			t.Errorf("Unexpected read succeed. Expected invalidate failed error.")
		}
		close(join)
	}()

	// Allow version query.
	<-onReadC
	waitC <- struct{}{}

	// But block on invalidate.
	<-onReadC
	s.AbortAllAndStop()

	close(waitC)
	<-join
}

type PausableWriter struct {
	BE       io.WriteCloser
	OnWriteC chan struct{}
	WaitC    chan struct{}
}

func (r PausableWriter) Write(p []byte) (int, error) {
	r.OnWriteC <- struct{}{}
	<-r.WaitC
	fmt.Printf("Write!!!\n")
	return r.BE.Write(p)
}

func (r PausableWriter) Close() error {
	return r.BE.Close()
}

func TestCachedBlobStore_OpenWhileClosing(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	onWriteC := make(chan struct{})
	waitC := make(chan struct{})

	backendbs := tu.RWInterceptBlobStore{
		BE: tu.TestFileBlobStoreOfName("backend"),
		WrapWriter: func(orig io.WriteCloser) (io.WriteCloser, error) {
			return PausableWriter{orig, onWriteC, waitC}, nil
		},
		WrapReader: func(orig io.ReadCloser) (io.ReadCloser, error) { return orig, nil },
	}
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	// defer bs.Quit() // Won't play well with RWInterceptBlobStore

	if err := tu.WriteVersionedBlob(bs, "hoge", 1); err != nil {
		t.Errorf("%v", err)
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		// this would block
		bs.CloseEntryForTesting("hoge")
		fmt.Printf("CloseEntryForTesting done\n")
	}()

	// wait until write attempt
	<-onWriteC

	wg.Add(1)
	go func() {
		defer wg.Done()

		// try writing to the "closing" entry
		if err := tu.WriteVersionedBlob(bs, "hoge", 2); err != nil {
			t.Errorf("%v", err)
			return
		}
	}()

	// unblock write to continue closing
	waitC <- struct{}{}

	wg.Wait()

	if err := tu.AssertBlobVersion(cachebs, "hoge", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(bs, "hoge", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestCachedBlobStore_NewEntry(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	if err := tu.WriteVersionedBlobRA(bs, "newentry", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersionRA(bs, "newentry", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}
	if err := tu.AssertBlobVersion(cachebs, "newentry", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(backendbs, "newentry", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestCachedBlobStore_AutoExpandLen(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	bh, err := bs.Open("hoge", flags.O_RDWRCREATE)
	if err != nil {
		t.Errorf("Failed to open blobhandle")
	}
	defer bh.Close()

	if size := bh.Size(); size != 0 {
		t.Errorf("New bh size non-zero: %d", size)
	}

	if err := bh.PWrite([]byte("Hello"), 0); err != nil {
		t.Errorf("PWrite failed: %v", err)
	}

	if size := bh.Size(); size != 5 {
		t.Errorf("bh size not auto expanded! size  %d", size)
	}
}

func TestCachedBlobStore_ListBlobs(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlob(cachebs, "cacheonly", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlobRA(bs, "synced", 3); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}
	if err := tu.WriteVersionedBlobRA(bs, "unsynced", 4); err != nil {
		t.Errorf("%v", err)
		return
	}

	bpaths, err := bs.ListBlobs()
	if err != nil {
		t.Errorf("ListBlobs failed: %v", err)
		return
	}
	sort.Strings(bpaths)
	if !reflect.DeepEqual([]string{"backendonly", "synced", "unsynced"}, bpaths) {
		t.Errorf("ListBlobs returned unexpected result: %v", bpaths)
	}
}

func TestCachedBlobStore_RemoveBlob(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlob(cachebs, "cacheonly", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlobRA(bs, "synced", 3); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}
	if err := tu.WriteVersionedBlobRA(bs, "unsynced", 4); err != nil {
		t.Errorf("%v", err)
		return
	}

	if err := bs.RemoveBlob("backendonly"); err != nil {
		t.Errorf("RemoveBlob failed: %v", err)
		return
	}
	if err := bs.RemoveBlob("synced"); err != nil {
		t.Errorf("RemoveBlob failed: %v", err)
		return
	}
	if err := bs.RemoveBlob("unsynced"); err != nil {
		t.Errorf("RemoveBlob failed: %v", err)
		return
	}

	bpaths, err := bs.ListBlobs()
	if err != nil {
		t.Errorf("ListBlobs failed: %v", err)
		return
	}
	if len(bpaths) > 0 {
		t.Errorf("Left over blobs: %v", bpaths)
	}

	for _, bp := range []string{"backendonly", "synced", "unsynced"} {
		if err := tu.AssertBlobVersionRA(bs, bp, 0); err != nil {
			t.Errorf("left over blob in bs: %s", bp)
		}
		if err := tu.AssertBlobVersion(cachebs, bp, 0); err != nil {
			t.Errorf("left over blob in cachebs: %s", bp)
		}
		if err := tu.AssertBlobVersion(backendbs, bp, 0); err != nil {
			t.Errorf("left over blob in backendbs: %s", bp)
		}
	}
}

func TestCachedBlobStore_CancelInvalidatingBlobsOnExit(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	onReadC := make(chan struct{})
	waitC := make(chan struct{})

	backendbs := tu.RWInterceptBlobStore{
		BE:         tu.TestFileBlobStoreOfName("backend"),
		WrapWriter: func(orig io.WriteCloser) (io.WriteCloser, error) { return orig, nil },
		WrapReader: func(orig io.ReadCloser) (io.ReadCloser, error) {
			return PausableReader{orig, onReadC, waitC}, nil
		},
	}
	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 2); err != nil {
		t.Errorf("%v", err)
		return
	}

	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := bs.OpenReader("backendonly")
		if err != nil {
			t.Errorf("Unexpected OpenReader failure: %v", err)
			return
		}

		if _, err := tu.TestQueryVersion(r); err == nil {
			t.Errorf("Unexpected read succeed. Expected invalidate failed error.")
		}
	}()

	// Allow version query.
	<-onReadC
	waitC <- struct{}{}

	// But cancel invalidate.
	<-onReadC
	s.AbortAllAndStop()

	wg.Wait()

	// FIXME: Wait for Close(abandonAndClose) goroutine to run.
	time.Sleep(100 * time.Millisecond)

	if _, err := cachebs.OpenReader("backendonly"); !util.IsNotExist(err) {
		t.Errorf("invalidate failed blob is still cached! Err: %v", err)
	}
}

func TestCachedBlobStore_CancelInvalidatingBlobsByClose(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	onReadC := make(chan struct{})
	waitC := make(chan struct{})

	backendbs := tu.RWInterceptBlobStore{
		BE:         tu.TestFileBlobStoreOfName("backend"),
		WrapWriter: func(orig io.WriteCloser) (io.WriteCloser, error) { return orig, nil },
		WrapReader: func(orig io.ReadCloser) (io.ReadCloser, error) {
			return PausableReader{orig, onReadC, waitC}, nil
		},
	}
	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 2); err != nil {
		t.Errorf("%v", err)
		return
	}

	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := bs.OpenReader("backendonly")
		if err != nil {
			t.Errorf("Unexpected OpenReader failure: %v", err)
			return
		}
		if err := r.Close(); err != nil {
			t.Errorf("Unexpected Close failure: %v", err)
			return
		}
	}()

	// Allow version query.
	<-onReadC
	waitC <- struct{}{}

	// But block invalidate.
	<-onReadC

	bs.CloseEntryForTesting("backendonly")
	wg.Wait()

	// FIXME: Wait for Close(abandonAndClose) goroutine to run.
	time.Sleep(100 * time.Millisecond)

	if _, err := cachebs.OpenReader("backendonly"); !util.IsNotExist(err) {
		t.Errorf("invalidate cancelled blob is still cached!")
	}
}

type sequenceRecorder struct {
	events []string
}

func NewSequenceRecorder() *sequenceRecorder {
	return &sequenceRecorder{[]string{}}
}

func (sr *sequenceRecorder) RecordEvent(e string) {
	sr.events = append(sr.events, e)
}

func (sr *sequenceRecorder) AssertSequence(t *testing.T, expected []string) {
	if !reflect.DeepEqual(sr.events, expected) {
		t.Errorf("Expected seq: %+v, actual: %+v", expected, sr.events)
	}
}

func TestCachedBlobStore_WaitForPreviousSync(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	onWriteC := make(chan struct{})
	waitC := make(chan struct{})

	backendbs := tu.RWInterceptBlobStore{
		BE: tu.TestFileBlobStoreOfName("backend"),
		WrapWriter: func(orig io.WriteCloser) (io.WriteCloser, error) {
			return PausableWriter{orig, onWriteC, waitC}, nil
		},
		WrapReader: func(orig io.ReadCloser) (io.ReadCloser, error) { return orig, nil },
	}
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()
	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	if err := tu.WriteVersionedBlob(bs, "hoge", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	bh, err := bs.Open("hoge", flags.O_WRONLY)
	if err != nil {
		t.Errorf("Open err: %v", err)
		return
	}
	bhs := bh.(util.Syncer)
	defer bh.Close()

	joinC := make(chan struct{})

	sr := NewSequenceRecorder()
	go func() {
		sr.RecordEvent("sync1b")
		if err := bhs.Sync(); err != nil {
			t.Errorf("Sync err: %v", err)
		}
		sr.RecordEvent("sync1e")
		joinC <- struct{}{}
	}()
	time.Sleep(10 * time.Millisecond)
	go func() {
		for {
			<-onWriteC
			time.Sleep(20 * time.Millisecond)
			sr.RecordEvent("write")
			waitC <- struct{}{}
		}
	}()
	sr.RecordEvent("sync2b")
	if err := bhs.Sync(); err != nil {
		t.Errorf("Sync err: %v", err)
	}
	sr.RecordEvent("sync2e")

	<-joinC
	sr.AssertSequence(t, []string{"sync1b", "sync2b", "write", "sync1e", "sync2e"})
}

func TestCachedBlobStore_ReduceCache(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")

	if err := tu.WriteVersionedBlob(cachebs, "cacheonly", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlob(cachebs, "cacheonlyunopened", 2); err != nil {
		t.Errorf("%v", err)
		return
	}

	s := scheduler.NewScheduler()

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}

	rc, err := bs.OpenReader("cacheonly")
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	rc.Close()

	if err := tu.WriteVersionedBlob(bs, "both", 2); err != nil {
		t.Errorf("%v", err)
		return
	}

	if err := bs.ReduceCache(context.TODO(), 0, false); err != nil {
		t.Errorf("ReduceCache err: %v", err)
	}

	bs.Quit()

	if err := tu.AssertBlobVersion(cachebs, "cacheonly", 0); err != nil {
		t.Errorf("%v", err)
	}
	if err := tu.AssertBlobVersion(cachebs, "cacheonlyunopened", 0); err != nil {
		t.Errorf("%v", err)
	}
	if err := tu.AssertBlobVersion(cachebs, "both", 0); err != nil {
		t.Errorf("%v", err)
	}

	if err := tu.AssertBlobVersion(backendbs, "cacheonly", 2); err != nil {
		t.Errorf("backend sync failed: %v", err)
	}
	if err := tu.AssertBlobVersion(backendbs, "cacheonlyunopened", 2); err != nil {
		t.Errorf("backend sync failed: %v", err)
	}
	if err := tu.AssertBlobVersion(backendbs, "both", 2); err != nil {
		t.Errorf("backend sync failed: %v", err)
	}
}

func TestCachedBlobStore_ReadOnly(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")
	s := scheduler.NewScheduler()

	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	bs, err := cachedblobstore.New(backendbs, cachebs, s, flags.O_RDONLY, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	defer bs.Quit()

	if err := tu.AssertBlobVersion(backendbs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}
	// assert cache not yet filled
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 0); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersionRA(bs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	// assert cache fill
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	// assert Open(O_RDWRCREATE) fails
	_, err = bs.Open("backendonly", flags.O_RDWRCREATE)
	if err == nil {
		t.Errorf("Unexpected write succeed.")
		return
	}
	if err != util.EACCES {
		t.Errorf("Expected EACCES error. Got: %v", err)
		return
	}

	// assert OpenWriter() fails
	_, err = bs.OpenWriter("backendonly")
	if err == nil {
		t.Errorf("Unexpected write succeed.")
		return
	}
	if err != util.EACCES {
		t.Errorf("Expected EACCES error. Got: %v", err)
		return
	}

	// assert RemoveBlob() fails
	err = bs.RemoveBlob("backendonly")
	if err == nil {
		t.Errorf("Unexpected remove succeed.")
		return
	}
	if err != util.EACCES {
		t.Errorf("Expected EACCES error. Got: %v", err)
		return
	}

	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}

	if err := tu.AssertBlobVersionRA(bs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(backendbs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}
}
