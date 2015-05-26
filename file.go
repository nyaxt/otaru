package otaru

import (
	"fmt"
	"syscall"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/util"
)

const (
	EPERM     = syscall.Errno(syscall.EPERM)
	ENOENT    = syscall.Errno(syscall.ENOENT)
	ENOTDIR   = syscall.Errno(syscall.ENOTDIR)
	ENOTEMPTY = syscall.Errno(syscall.ENOTEMPTY)
	EEXIST    = syscall.Errno(syscall.EEXIST)
)

const (
	FileWriteCacheMaxPatches         = 32
	FileWriteCacheMaxPatchContentLen = 256 * 1024
)

type FileSystem struct {
	idb inodedb.DBHandler

	bs blobstore.RandomAccessBlobStore
	c  Cipher

	newChunkedFileIO func(bs blobstore.RandomAccessBlobStore, c Cipher, caio ChunksArrayIO) blobstore.BlobHandle

	wcmap map[inodedb.ID]*FileWriteCache
}

func newFileSystemCommon(idb inodedb.DBHandler, bs blobstore.RandomAccessBlobStore, c Cipher) *FileSystem {
	fs := &FileSystem{
		idb: idb,
		bs:  bs,
		c:   c,

		newChunkedFileIO: func(bs blobstore.RandomAccessBlobStore, c Cipher, caio ChunksArrayIO) blobstore.BlobHandle {
			return NewChunkedFileIO(bs, c, caio)
		},

		wcmap: make(map[inodedb.ID]*FileWriteCache),
	}

	return fs
}

func NewFileSystemEmpty(bs blobstore.RandomAccessBlobStore, c Cipher) (*FileSystem, error) {
	// FIXME: refactor here and FromSnapshot

	snapshotio := NewBlobStoreDBStateSnapshotIO(bs, c)
	txio := inodedb.NewSimpleDBTransactionLogIO() // FIXME!
	idb, err := inodedb.NewEmptyDB(snapshotio, txio)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize inodedb: %v", err)
	}

	return newFileSystemCommon(idb, bs, c), nil
}

func NewFileSystemFromSnapshot(bs blobstore.RandomAccessBlobStore, c Cipher) (*FileSystem, error) {
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

func (fs *FileSystem) OverrideNewChunkedFileIOForTesting(newChunkedFileIO func(blobstore.RandomAccessBlobStore, Cipher, ChunksArrayIO) blobstore.BlobHandle) {
	fs.newChunkedFileIO = newChunkedFileIO
}

/*

type DirHandle struct {
	fs *FileSystem
	id inodedb.ID
}

func (fs *FileSystem) OpenDir(id inodedb.ID) (*DirHandle, error) {
	node := fs.INodeDB.Get(id)
	if node == nil {
		return nil, ENOENT
	}
	if node.Type() != DirNodeT {
		return nil, ENOTDIR
	}
	dirnode := node.(*DirNode)

	h := &DirHandle{fs: fs, n: dirnode}
	return h, nil
}

func (dh *DirHandle) FileSystem() *FileSystem {
	return dh.fs
}

func (dh *DirHandle) ID() inodedb.ID {
	return dh.id
}

func (dh *DirHandle) Entries() map[string]inodedb.ID {
	return dh.n.Entries
}

// FIXME
func (dh *DirHandle) Path() string {
	return ""
}

func (dh *DirHandle) Rename(oldname string, tgtdh *DirHandle, newname string) error {
	es := dh.n.Entries
	id, ok := es[oldname]
	if !ok {
		return ENOENT
	}

	es2 := tgtdh.n.Entries
	_, ok = es2[newname]
	if ok {
		return EEXIST
	}

	es2[newname] = id
	delete(es, oldname)
	return nil
}

func (dh *DirHandle) Remove(name string) error {
	es := dh.n.Entries

	id, ok := es[name]
	if !ok {
		return ENOENT
	}
	n := dh.fs.INodeDB.Get(id)
	if n.Type() == DirNodeT {
		sdn := n.(*DirNode)
		if len(sdn.Entries) != 0 {
			return ENOTEMPTY
		}
	}

	delete(es, name)
	return nil
}

func (dh *DirHandle) createNode(name string, newNode func(db *inodedb.DBHandler, origpath string) inodedb.ID) (inodedb.ID, error) {
	_, ok := dh.n.Entries[name]
	if ok {
		return 0, EEXIST
	}

	fullorigpath := fmt.Sprintf("%s/%s", dh.Path(), name)
	id := newNode(dh.fs.INodeDB, fullorigpath)
	dh.n.Entries[name] = id
	return id, nil
}

func (dh *DirHandle) CreateFile(name string) (inodedb.ID, error) {
	return dh.createNode(name, func(db *INodeDB, origpath string) inodedb.ID {
		n := NewFileNode(dh.fs.INodeDB, origpath)
		return n.ID()
	})
}

func (dh *DirHandle) CreateDir(name string) (inodedb.ID, error) {
	return dh.createNode(name, func(db *INodeDB, origpath string) inodedb.ID {
		n := NewDirNode(dh.fs.INodeDB, origpath)
		return n.ID()
	})
}

// FIXME: Multiple FileHandle may exist for same file at once. Support it!
type FileHandle struct {
	fs    *FileSystem
	nlock inodedb.NodeLock
	wc    *FileWriteCache
	cfio  blobstore.BlobHandle
}

func (fs *FileSystem) openFileNode(n *FileNode) (*FileHandle, error) {
	wc := fs.getOrCreateFileWriteCache(n.ID())
	cfio := fs.newChunkedFileIO(fs.bs, n, fs.c)
	h := &FileHandle{fs: fs, n: n, wc: wc, cfio: cfio}
	return h, nil
}

func (fs *FileSystem) OpenFile(id inodedb.ID) (*FileHandle, error) {
	node := fs.INodeDB.Get(id)
	if node == nil {
		return nil, ENOENT
	}
	if node.Type() != FileNodeT {
		return nil, ENOTDIR
	}
	filenode := node.(*FileNode)

	h, err := fs.openFileNode(filenode)
	if err != nil {
		return nil, err
	}
	return h, nil
}

type Attr struct {
	inodedb.ID
	INodeType
	Size int64
}

func (fs *FileSystem) Attr(id inodedb.ID) (Attr, error) {
	n := fs.INodeDB.Get(id)
	if n == nil {
		return Attr{}, ENOENT
	}

	size := int64(0)
	if fn, ok := n.(*FileNode); ok {
		size = fn.Size
	}

	a := Attr{
		inodedb.ID: n.ID(),
		INodeType:  n.Type(),
		Size:       size,
	}
	return a, nil
}

func (fs *FileSystem) IsDir(id inodedb.ID) (bool, error) {
	n := fs.INodeDB.Get(id)
	if n == nil {
		return false, ENOENT
	}

	return n.Type() == DirNodeT, nil
}

func (h *FileHandle) ID() inodedb.ID {
	return h.n.ID()
}

func (h *FileHandle) PWrite(offset int64, p []byte) error {
	if err := h.wc.PWrite(offset, p); err != nil {
		return err
	}

	if h.wc.NeedsFlush() {
		if err := h.wc.Flush(h.cfio); err != nil {
			return err
		}
	}

	right := offset + int64(len(p))
	if right > h.n.Size {
		h.n.Size = right
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
	return h.n.Size
}

func (h *FileHandle) Truncate(newsize int64) error {
	if newsize > h.n.Size {
		h.n.Size = newsize
		return nil
	}

	if newsize < h.n.Size {
		h.wc.Truncate(newsize)
		h.cfio.Truncate(newsize)
		h.n.Size = newsize
	}
	return nil
}
*/
