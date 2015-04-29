package otaru

import (
	"fmt"

	"github.com/nyaxt/otaru/intn"
)

type FileChunk struct {
	Offset   int64
	Length   int64
	BlobPath string
}

func (fc FileChunk) Left() int64 {
	return fc.Offset
}

func (fc FileChunk) Right() int64 {
	return fc.Offset + fc.Length
}

type INodeID uint32

type INodeType int

const (
	FileNodeT = iota
	DirNodeT
	// SymlinkNode
)

type INode interface {
	ID() INodeID
	Type() INodeType
}

type INodeCommon struct {
	INodeID
	INodeType

	// OrigPath contains filepath passed to first create and does not necessary follow "rename" operations.
	// To be used for recovery/debug purposes only
	OrigPath string
}

func (n INodeCommon) ID() INodeID {
	return n.INodeID
}

func (n INodeCommon) Type() INodeType {
	return n.INodeType
}

type FileNode struct {
	INodeCommon
	Size   int64
	Chunks []FileChunk
}

func NewFileNode(db *INodeDB, origpath string) *FileNode {
	id := db.GenerateNewID()
	fn := &FileNode{
		INodeCommon: INodeCommon{
			INodeID:   id,
			INodeType: FileNodeT,
			OrigPath:  origpath,
		},
		Size: 0,
	}
	db.PutMustSucceed(fn)
	return fn
}

type DirNodeEntry struct {
	INodeID
	Name string
}

type DirNode struct {
	INodeCommon
	Entries []DirNodeEntry
}

func NewDirNode(db *INodeDB, origpath string) *DirNode {
	id := db.GenerateNewID()
	dn := &DirNode{
		INodeCommon: INodeCommon{
			INodeID:   id,
			INodeType: DirNodeT,
			OrigPath:  origpath,
		},
	}
	db.PutMustSucceed(dn)
	return dn
}

type INodeDB struct {
	nodes  map[INodeID]INode
	lastID INodeID
}

func NewINodeDB() *INodeDB {
	return &INodeDB{
		nodes:  make(map[INodeID]INode),
		lastID: 0,
	}
}

func (idb *INodeDB) Put(n INode) error {
	_, ok := idb.nodes[n.ID()]
	if ok {
		return fmt.Errorf("INodeID collision: %v", n)
	}
	idb.nodes[n.ID()] = n
	return nil
}

func (idb *INodeDB) PutMustSucceed(n INode) {
	if err := idb.Put(n); err != nil {
		panic(fmt.Sprintf("Failed to put node: %v", err))
	}
}

func (idb *INodeDB) Get(id INodeID) INode {
	node := idb.nodes[id]
	if node.ID() != id {
		panic("INodeDB is corrupt!")
	}
	return node
}

func (idb *INodeDB) GenerateNewID() INodeID {
	id := idb.lastID + 1
	idb.lastID = id
	return id
}

const (
	FileWriteCacheMaxPatches         = 32
	FileWriteCacheMaxPatchContentLen = 256 * 1024
)

type FileWriteCache struct {
	ps intn.Patches
}

func NewFileWriteCache() *FileWriteCache {
	return &FileWriteCache{ps: intn.NewPatches()}
}

func (wc *FileWriteCache) PWrite(offset int64, p []byte) error {
	newp := intn.Patch{Offset: offset, P: p}
	wc.ps = wc.ps.Merge(newp)
	return nil
}

func (wc *FileWriteCache) PReadThrough(offset int64, p []byte, r PReader) error {
	nr := int64(len(p))
	remo := offset
	remp := p

	for _, patch := range wc.ps {
		if nr <= 0 {
			return nil
		}

		if remo > patch.Right() {
			continue
		}

		if remo < patch.Left() {
			fallbackLen := Int64Min(nr, patch.Left()-remo)

			if err := r.PRead(remo, remp[:fallbackLen]); err != nil {
				return err
			}

			remp = remp[fallbackLen:]
			nr -= fallbackLen
			remo += fallbackLen
		}

		if nr <= 0 {
			return nil
		}

		applyOffset := remo - patch.Offset
		applyLen := Int64Min(int64(len(patch.P))-applyOffset, nr)
		copy(remp[:applyLen], patch.P[applyOffset:])

		remp = remp[applyLen:]
		nr -= applyLen
		remo += applyLen
	}

	if err := r.PRead(remo, remp); err != nil {
		return err
	}
	return nil
}

func (wc *FileWriteCache) ContentLen() int64 {
	l := int64(0)
	for _, p := range wc.ps {
		l += int64(len(p.P))
	}
	return l
}

func (wc *FileWriteCache) NeedsFlush() bool {
	if len(wc.ps) > FileWriteCacheMaxPatches {
		return true
	}
	if wc.ContentLen() > FileWriteCacheMaxPatchContentLen {
		return true
	}

	return false
}

func (wc *FileWriteCache) Flush(bh BlobHandle) error {
	for _, p := range wc.ps {
		if err := bh.PWrite(p.Offset, p.P); err != nil {
			return err
		}
	}
	wc.ps = wc.ps[:0]

	return nil
}

func (wc *FileWriteCache) Right() int64 {
	if len(wc.ps) == 0 {
		return 0
	}

	return wc.ps[0].Right()
}

func (wc *FileWriteCache) Truncate(size int64) {
	wc.ps = wc.ps.Truncate(size)
}

type FileSystem struct {
	*INodeDB
	lastID INodeID

	bs RandomAccessBlobStore
	c  Cipher

	newChunkedFileIO func(bs RandomAccessBlobStore, fn *FileNode, c Cipher) BlobHandle

	wcmap map[INodeID]*FileWriteCache
}

func NewFileSystem(bs RandomAccessBlobStore, c Cipher) *FileSystem {
	fs := &FileSystem{
		INodeDB: NewINodeDB(),
		lastID:  0,

		bs: bs,
		c:  c,

		newChunkedFileIO: func(bs RandomAccessBlobStore, fn *FileNode, c Cipher) BlobHandle {
			return NewChunkedFileIO(bs, fn, c)
		},

		wcmap: make(map[INodeID]*FileWriteCache),
	}

	rootdir := NewDirNode(fs.INodeDB, "/")
	if rootdir.ID() != 1 {
		panic("rootdir must have INodeID 1")
	}

	return fs
}

func (fs *FileSystem) getOrCreateFileWriteCache(id INodeID) *FileWriteCache {
	wc := fs.wcmap[id]
	if wc == nil {
		wc = NewFileWriteCache()
		fs.wcmap[id] = wc
	}
	return wc
}

func (fs *FileSystem) OverrideNewChunkedFileIOForTesting(newChunkedFileIO func(RandomAccessBlobStore, *FileNode, Cipher) BlobHandle) {
	fs.newChunkedFileIO = newChunkedFileIO
}

type DirHandle struct {
	fs *FileSystem
	n  *DirNode
}

func (fs *FileSystem) OpenDir() (*DirHandle, error) {
	rootnode := fs.INodeDB.Get(1)
	rootdir := rootnode.(*DirNode)

	h := &DirHandle{fs: fs, n: rootdir}
	return h, nil
}

func (dh *DirHandle) FileSystem() *FileSystem {
	return dh.fs
}

func (dh *DirHandle) INodeID() INodeID {
	return dh.n.ID()
}

func (dh *DirHandle) Entries() []DirNodeEntry {
	return dh.n.Entries
}

type FileHandle struct {
	fs   *FileSystem
	n    *FileNode
	wc   *FileWriteCache
	cfio BlobHandle
}

func (fs *FileSystem) CreateFile(otarupath string) (*FileHandle, error) {
	n := NewFileNode(fs.INodeDB, otarupath)
	wc := fs.getOrCreateFileWriteCache(n.ID())
	cfio := fs.newChunkedFileIO(fs.bs, n, fs.c)
	h := &FileHandle{fs: fs, n: n, wc: wc, cfio: cfio}

	return h, nil
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
