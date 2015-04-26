package otaru

import (
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
	DirectoryNodeT
	// SymlinkNode
)

type INode interface {
	ID() INodeID
	Type() INodeType
}

type INodeCommon struct {
	INodeID
	INodeType
}

func (n INodeCommon) ID() INodeID {
	return n.INodeID
}

func (n INodeCommon) Type() INodeType {
	return n.INodeType
}

type FileNode struct {
	INodeCommon

	// OrigPath contains filepath passed to first create and does not necessary follow "rename" operations.
	// To be used for recovery/debug purposes only
	OrigPath string

	Chunks []FileChunk
}

func NewFileNode(id INodeID, origpath string) *FileNode {
	return &FileNode{
		INodeCommon: INodeCommon{INodeID: id, INodeType: FileNodeT},
		OrigPath:    origpath,
	}
}

type INodeDB struct {
	nodes map[INodeID]INode
}

func NewINodeDB() *INodeDB {
	return &INodeDB{
		nodes: make(map[INodeID]INode),
	}
}

func (idb *INodeDB) Get(id INodeID) INode {
	return idb.nodes[id]
}

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

type FileSystem struct {
	*INodeDB
	lastID INodeID

	bs RandomAccessBlobStore
	c  Cipher

	newChunkedFileIO func(bs RandomAccessBlobStore, fn *FileNode, c Cipher) BlobHandle

	wcmap map[INodeID]*FileWriteCache
}

func NewFileSystem(bs RandomAccessBlobStore, c Cipher) *FileSystem {
	return &FileSystem{
		INodeDB: NewINodeDB(),
		lastID:  0,

		bs: bs,
		c:  c,

		newChunkedFileIO: func(bs RandomAccessBlobStore, fn *FileNode, c Cipher) BlobHandle {
			return NewChunkedFileIO(bs, fn, c)
		},

		wcmap: make(map[INodeID]*FileWriteCache),
	}
}

func (fs *FileSystem) NewINodeID() INodeID {
	id := fs.lastID + 1
	fs.lastID = id
	return id
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

type FileHandle struct {
	fs   *FileSystem
	n    *FileNode
	wc   *FileWriteCache
	cfio BlobHandle
}

func (fs *FileSystem) CreateFile(otarupath string) (*FileHandle, error) {
	id := fs.NewINodeID()
	n := NewFileNode(id, otarupath)
	wc := fs.getOrCreateFileWriteCache(id)
	cfio := fs.newChunkedFileIO(fs.bs, n, fs.c)
	h := &FileHandle{fs: fs, n: n, wc: wc, cfio: cfio}

	return h, nil
}

func (h *FileHandle) PWrite(offset int64, p []byte) error {
	return h.wc.PWrite(offset, p)
}

func (h *FileHandle) PRead(offset int64, p []byte) error {
	return h.wc.PReadThrough(offset, p, ZeroFillPReader{})
}
