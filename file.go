package otaru

import (
	"fmt"
	"syscall"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/inodedb"
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
	*INodeDB
	lastID INodeID

	bs blobstore.RandomAccessBlobStore
	c  Cipher

	newChunkedFileIO func(bs blobstore.RandomAccessBlobStore, fn *FileNode, c Cipher) blobstore.BlobHandle

	wcmap map[INodeID]*FileWriteCache
}

func newFileSystemCommon(idb *INodeDB, bs blobstore.RandomAccessBlobStore, c Cipher) *FileSystem {
	fs := &FileSystem{
		INodeDB: idb,
		lastID:  0,

		bs: bs,
		c:  c,

		newChunkedFileIO: func(bs blobstore.RandomAccessBlobStore, fn *FileNode, c Cipher) blobstore.BlobHandle {
			return NewChunkedFileIO(bs, fn, c)
		},

		wcmap: make(map[INodeID]*FileWriteCache),
	}

	return fs
}

func NewFileSystemEmpty(bs blobstore.RandomAccessBlobStore, c Cipher) *FileSystem {
	idb := NewINodeDBEmpty()
	rootdir := NewDirNode(idb, "/")
	if rootdir.ID() != 1 {
		panic("rootdir must have INodeID 1")
	}

	return newFileSystemCommon(idb, bs, c)
}

func NewFileSystemFromSnapshot(bs blobstore.RandomAccessBlobStore, c Cipher) (*FileSystem, error) {
	idb, err := LoadINodeDBFromBlobStore(bs, c)
	if err != nil {
		return nil, err
	}

	return newFileSystemCommon(idb, bs, c), nil
}

func (fs *FileSystem) getOrCreateFileWriteCache(id INodeID) *FileWriteCache {
	wc := fs.wcmap[id]
	if wc == nil {
		wc = NewFileWriteCache()
		fs.wcmap[id] = wc
	}
	return wc
}

func (fs *FileSystem) OverrideNewChunkedFileIOForTesting(newChunkedFileIO func(blobstore.RandomAccessBlobStore, *FileNode, Cipher) blobstore.BlobHandle) {
	fs.newChunkedFileIO = newChunkedFileIO
}

type DirHandle struct {
	fs *FileSystem
	n  *DirNode
}

func (fs *FileSystem) OpenDir(id INodeID) (*DirHandle, error) {
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

func (dh *DirHandle) ID() INodeID {
	return dh.n.ID()
}

func (dh *DirHandle) Entries() map[string]INodeID {
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

func (dh *DirHandle) createNode(name string, newNode func(db *INodeDB, origpath string) INodeID) (INodeID, error) {
	_, ok := dh.n.Entries[name]
	if ok {
		return 0, EEXIST
	}

	fullorigpath := fmt.Sprintf("%s/%s", dh.Path(), name)
	id := newNode(dh.fs.INodeDB, fullorigpath)
	dh.n.Entries[name] = id
	return id, nil
}

func (dh *DirHandle) CreateFile(name string) (INodeID, error) {
	return dh.createNode(name, func(db *INodeDB, origpath string) INodeID {
		n := NewFileNode(dh.fs.INodeDB, origpath)
		return n.ID()
	})
}

func (dh *DirHandle) CreateDir(name string) (INodeID, error) {
	return dh.createNode(name, func(db *INodeDB, origpath string) INodeID {
		n := NewDirNode(dh.fs.INodeDB, origpath)
		return n.ID()
	})
}

// FIXME: Multiple FileHandle may exist for same file at once. Support it!
type FileHandle struct {
	fs   *FileSystem
	n    *FileNode
	wc   *FileWriteCache
	cfio blobstore.BlobHandle
}

func (fs *FileSystem) openFileNode(n *FileNode) (*FileHandle, error) {
	wc := fs.getOrCreateFileWriteCache(n.ID())
	cfio := fs.newChunkedFileIO(fs.bs, n, fs.c)
	h := &FileHandle{fs: fs, n: n, wc: wc, cfio: cfio}
	return h, nil
}

func (fs *FileSystem) OpenFile(id INodeID) (*FileHandle, error) {
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
	INodeID
	INodeType
	Size int64
}

func (fs *FileSystem) Attr(id INodeID) (Attr, error) {
	n := fs.INodeDB.Get(id)
	if n == nil {
		return Attr{}, ENOENT
	}

	size := int64(0)
	if fn, ok := n.(*FileNode); ok {
		size = fn.Size
	}

	a := Attr{
		INodeID:   n.ID(),
		INodeType: n.Type(),
		Size:      size,
	}
	return a, nil
}

func (fs *FileSystem) IsDir(id INodeID) (bool, error) {
	n := fs.INodeDB.Get(id)
	if n == nil {
		return false, ENOENT
	}

	return n.Type() == DirNodeT, nil
}

func (h *FileHandle) ID() INodeID {
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

func (fs *FileSystem) Sync() error {
	if err := fs.INodeDB.SaveToBlobStore(fs.bs, fs.c); err != nil {
		return fmt.Errorf("Failed to save INodeDB: %v", err)
	}

	return nil
}
