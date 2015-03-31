package otaru

import (
	"math"
)

type FileChunk struct {
	Offset   int
	Length   int
	BlobPath string
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
		INodeCommon{INodeID: id, INodeType: FileNodeT},
		OrigPath: origpath,
	}
}

type INodeDB struct {
	nodes map[INodeID]INode
}

func NewINodeDB() *INodeDB {
	return &INodeDB{
		nodes: make(map[InodeId]INode),
	}
}

func (idb *INodeDB) Get(id INodeID) INode {
	return idb.nodes[id]
}

type FileWriteCache struct {
	intn.Patches
}

func NewFileWriteCache() *FileWriteCache {
	return &FileWriteCache{Patches: Patches{patchSentinel}}
}

func (wc *FileWriteCache) PWrite(offset int, p []byte) error {
	if len(p) == 0 {
		return nil
	}

	/*
	  for i, patch := range wc.patches {
	    pleft := patch.offset
	    if right < pleft {
	      //         <patch>
	      // <new>

	      // insert new patch at index i
	      wc.patches = append(wc.patches, patchSentinel)
	      copy(wc.patches[i+1:], wc.patches[i:])
	      wc.patches[i] = patch{offset: remo, p: remp}
	      return nil
	    }
	  }
	*/
	panic("should not be reached!")
}

type FileSystem struct {
	*INodeDB
	lastID INodeID

	wcmap map[INodeID]*WriteCache
}

func NewFileSystem() *FileSystem {
	return &FileSystem{
		INodeDB: NewINodeDB(),
		lastID:  0,
		wcmap:   make(map[INodeID]*WriteCache),
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

func (fs *FileSystem) CreateFile(otarupath string) (*FileHandle, error) {
	id := fs.NewINodeID()
	n := NewFileNode(id, otarupath)
	wc := fs.getOrCreateFileWriteCache(id)
	h := &FileHandle{fs: fs, n: n}

	return h, nil
}

type FileHandle struct {
	fs *FileSystem
	n  *FileNode
	wc *FileWriteCache
}

func (h *FileHandle) PWrite(offset int, p []byte) error {
	return h.wc.PWrite(offset, p)
}

// func PRead(f File, offset int, p []byte) error
//   trivial
// func PWrite(f File, offset int, p []byte) error
//   ???
