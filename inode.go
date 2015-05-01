package otaru

import (
	"io"
)

type INodeType int

const (
	FileNodeT = iota
	DirNodeT
	// SymlinkNode
)

type INode interface {
	ID() INodeID
	Type() INodeType

	SerializeSnapshot(w io.Writer) error
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

type DirNode struct {
	INodeCommon
	Entries map[string]INodeID
}

func NewDirNode(db *INodeDB, origpath string) *DirNode {
	id := db.GenerateNewID()
	dn := &DirNode{
		INodeCommon: INodeCommon{
			INodeID:   id,
			INodeType: DirNodeT,
			OrigPath:  origpath,
		},
		Entries: make(map[string]INodeID),
	}
	db.PutMustSucceed(dn)
	return dn
}
