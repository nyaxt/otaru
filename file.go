package otaru

import (
	"fmt"
	"log"
	"syscall"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/util"
)

const (
	EEXIST    = syscall.Errno(syscall.EEXIST)
	EISDIR    = syscall.Errno(syscall.EISDIR)
	ENOENT    = syscall.Errno(syscall.ENOENT)
	ENOTDIR   = syscall.Errno(syscall.ENOTDIR)
	ENOTEMPTY = syscall.Errno(syscall.ENOTEMPTY)
	EPERM     = syscall.Errno(syscall.EPERM)
)

const (
	FileWriteCacheMaxPatches         = 32
	FileWriteCacheMaxPatchContentLen = 256 * 1024
)

type FileSystem struct {
	idb inodedb.DBHandler

	bs blobstore.RandomAccessBlobStore
	c  btncrypt.Cipher

	newChunkedFileIO func(bs blobstore.RandomAccessBlobStore, c btncrypt.Cipher, caio ChunksArrayIO) blobstore.BlobHandle

	wcmap map[inodedb.ID]*FileWriteCache
}

func newFileSystemCommon(idb inodedb.DBHandler, bs blobstore.RandomAccessBlobStore, c btncrypt.Cipher) *FileSystem {
	fs := &FileSystem{
		idb: idb,
		bs:  bs,
		c:   c,

		newChunkedFileIO: func(bs blobstore.RandomAccessBlobStore, c btncrypt.Cipher, caio ChunksArrayIO) blobstore.BlobHandle {
			return NewChunkedFileIO(bs, c, caio)
		},

		wcmap: make(map[inodedb.ID]*FileWriteCache),
	}

	return fs
}

func NewFileSystemEmpty(bs blobstore.RandomAccessBlobStore, c btncrypt.Cipher) (*FileSystem, error) {
	// FIXME: refactor here and FromSnapshot

	snapshotio := NewBlobStoreDBStateSnapshotIO(bs, c)
	txio := inodedb.NewSimpleDBTransactionLogIO() // FIXME!
	idb, err := inodedb.NewEmptyDB(snapshotio, txio)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize inodedb: %v", err)
	}

	return newFileSystemCommon(idb, bs, c), nil
}

func NewFileSystemFromSnapshot(bs blobstore.RandomAccessBlobStore, c btncrypt.Cipher) (*FileSystem, error) {
	snapshotio := NewBlobStoreDBStateSnapshotIO(bs, c)
	txio := inodedb.NewSimpleDBTransactionLogIO() // FIXME!
	idb, err := inodedb.NewDB(snapshotio, txio)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize inodedb: %v", err)
	}

	return newFileSystemCommon(idb, bs, c), nil
}

func (fs *FileSystem) Sync() error {
	if s, ok := fs.idb.(util.Syncer); ok {
		if err := s.Sync(); err != nil {
			return fmt.Errorf("Failed to sync INodeDB: %v", err)
		}
	}

	return nil
}

func (fs *FileSystem) getOrCreateFileWriteCache(id inodedb.ID) *FileWriteCache {
	wc := fs.wcmap[id]
	if wc == nil {
		wc = NewFileWriteCache()
		fs.wcmap[id] = wc
	}
	return wc
}

func (fs *FileSystem) OverrideNewChunkedFileIOForTesting(newChunkedFileIO func(blobstore.RandomAccessBlobStore, btncrypt.Cipher, ChunksArrayIO) blobstore.BlobHandle) {
	fs.newChunkedFileIO = newChunkedFileIO
}

func (fs *FileSystem) DirEntries(id inodedb.ID) (map[string]inodedb.ID, error) {
	v, _, err := fs.idb.QueryNode(id, false)
	if err != nil {
		return nil, err
	}
	if v.GetType() != inodedb.DirNodeT {
		return nil, ENOTDIR
	}

	dv := v.(inodedb.DirNodeView)
	return dv.GetEntries(), err
}

func (fs *FileSystem) Rename(srcDirID inodedb.ID, srcName string, dstDirID inodedb.ID, dstName string) error {
	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.RenameOp{
			SrcDirID: srcDirID, SrcName: srcName,
			DstDirID: dstDirID, DstName: dstName,
		},
	}}
	if _, err := fs.idb.ApplyTransaction(tx); err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) Remove(dirID inodedb.ID, name string) error {
	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.RemoveOp{
			NodeLock: inodedb.NodeLock{dirID, inodedb.NoTicket}, Name: name,
		},
	}}
	if _, err := fs.idb.ApplyTransaction(tx); err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) createNode(dirID inodedb.ID, name string, typ inodedb.Type) (inodedb.ID, error) {
	nlock, err := fs.idb.LockNode(inodedb.AllocateNewNodeID)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := fs.idb.UnlockNode(nlock); err != nil {
			log.Printf("Failed to unlock node when creating file: %v", err)
		}
	}()

	origpath := name // FIXME

	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.CreateNodeOp{NodeLock: nlock, OrigPath: origpath, Type: typ},
		&inodedb.HardLinkOp{NodeLock: inodedb.NodeLock{dirID, inodedb.NoTicket}, Name: name, TargetID: nlock.ID},
	}}
	if _, err := fs.idb.ApplyTransaction(tx); err != nil {
		return 0, err
	}

	return nlock.ID, nil
}

func (fs *FileSystem) CreateFile(dirID inodedb.ID, name string) (inodedb.ID, error) {
	return fs.createNode(dirID, name, inodedb.FileNodeT)
}

func (fs *FileSystem) CreateDir(dirID inodedb.ID, name string) (inodedb.ID, error) {
	return fs.createNode(dirID, name, inodedb.DirNodeT)
}

type Attr struct {
	ID   inodedb.ID
	Type inodedb.Type
	Size int64
}

func (fs *FileSystem) Attr(id inodedb.ID) (Attr, error) {
	v, _, err := fs.idb.QueryNode(id, false)
	if err != nil {
		return Attr{}, err
	}

	size := int64(0)
	if fn, ok := v.(inodedb.FileNodeView); ok {
		size = fn.GetSize()
	}

	a := Attr{
		ID:   v.GetID(),
		Type: v.GetType(),
		Size: size,
	}
	return a, nil
}

func (fs *FileSystem) IsDir(id inodedb.ID) (bool, error) {
	v, _, err := fs.idb.QueryNode(id, false)
	if err != nil {
		return false, err
	}

	return v.GetType() == inodedb.DirNodeT, nil
}

// FIXME: Multiple FileHandle may exist for same file at once. Support it!
type FileHandle struct {
	fs    *FileSystem
	nlock inodedb.NodeLock
	wc    *FileWriteCache
	cfio  blobstore.BlobHandle
}

func (fs *FileSystem) OpenFile(id inodedb.ID, flags int) (*FileHandle, error) {
	tryLock := fl.IsWriteAllowed(flags)
	v, nlock, err := fs.idb.QueryNode(id, tryLock)
	if err != nil {
		return nil, err
	}
	if v.GetType() == inodedb.DirNodeT {
		return nil, EISDIR
	}
	fv, ok := v.(inodedb.FileNodeView)
	if !ok {
		return nil, fmt.Errorf("Specified node not file but has type %v", v.GetType())
	}

	wc := fs.getOrCreateFileWriteCache(id)
	var caio ChunksArrayIO
	if tryLock {
		caio = NewINodeDBChunksArrayIO(fs.idb, nlock, fv)
	} else {
		caio = NewReadOnlyINodeDBChunksArrayIO(fs.idb, nlock)
	}
	cfio := fs.newChunkedFileIO(fs.bs, fs.c, caio)
	return &FileHandle{fs: fs, nlock: nlock, wc: wc, cfio: cfio}, nil
}

func (h *FileHandle) ID() inodedb.ID {
	return h.nlock.ID
}

func (h *FileHandle) updateSize(newsize int64) error {
	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.UpdateSizeOp{NodeLock: h.nlock, Size: newsize},
	}}
	if _, err := h.fs.idb.ApplyTransaction(tx); err != nil {
		return fmt.Errorf("Failed to update FileNode size: %v", err)
	}
	return nil
}

func (h *FileHandle) SizeMayFail() (int64, error) {
	v, _, err := h.fs.idb.QueryNode(h.nlock.ID, false)
	if err != nil {
		return 0, fmt.Errorf("Failed to QueryNode inodedb: %v", err)
	}
	fv, ok := v.(inodedb.FileNodeView)
	if !ok {
		return 0, fmt.Errorf("Non-FileNodeView returned from QueryNode. Type: %v", v.GetType())
	}
	return fv.GetSize(), nil
}

func (h *FileHandle) PWrite(offset int64, p []byte) error {
	currentSize, err := h.SizeMayFail()
	if err != nil {
		return err
	}

	if err := h.wc.PWrite(offset, p); err != nil {
		return err
	}

	if h.wc.NeedsFlush() {
		if err := h.wc.Flush(h.cfio); err != nil {
			return err
		}
	}

	right := offset + int64(len(p))
	if right > currentSize {
		return h.updateSize(right)
	}

	return nil
}

func (h *FileHandle) PRead(offset int64, p []byte) error {
	return h.wc.PReadThrough(offset, p, h.cfio)
}

func (h *FileHandle) Flush() error {
	return h.wc.Flush(h.cfio)
}

func (h *FileHandle) Size() int64 {
	v, _, err := h.fs.idb.QueryNode(h.nlock.ID, false)
	if err != nil {
		log.Printf("Failed to QueryNode inodedb: %v", err)
		return 0
	}
	fv, ok := v.(inodedb.FileNodeView)
	if !ok {
		log.Printf("Non-FileNodeView returned from QueryNode. Type: %v", v.GetType())
		return 0
	}

	return fv.GetSize()
}

func (h *FileHandle) Truncate(newsize int64) error {
	oldsize, err := h.SizeMayFail()
	if err != nil {
		return err
	}

	if newsize > oldsize {
		return h.updateSize(newsize)
	} else if newsize < oldsize {
		h.wc.Truncate(newsize)
		h.cfio.Truncate(newsize)
		return h.updateSize(newsize)
	} else {
		return nil
	}
}
