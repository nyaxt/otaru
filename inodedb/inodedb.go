package inode

import (
	"fmt"
)

type ID uint64
type Type int

const (
	FileNodeT = iota
	DirNodeT
	// SymlinkNode
)

type DBState struct {
	nodes  map[ID]INode
	lastId ID
}

func NewDBState() *DBState {
	return &DBState{
		nodes:  make(map[INodeID]INodeID),
		lastID: 0,
	}
}

type INode interface {
	GetID() ID
	GetType() Type

	//FIXME: SerializeSnapshot(enc *gob.Encoder) error
}

type INodeCommon struct {
	ID
	Type

	// OrigPath contains filepath passed to first create and does not necessary follow "rename" operations.
	// To be used for recovery/debug purposes only
	OrigPath string
}

func (n INodeCommon) GetID() ID {
	return n.ID
}

func (n INodeCommon) GetType() Type {
	return n.Type
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

type DirNode struct {
	INodeCommon
	Entries map[string]ID
}

/*
func NewFileNode(db *INodeDB, origpath string) *FileNode {
	id := db.GenerateNewID()
	fn := &FileNode{
		INodeCommon: INodeCommon{
			ID:   id,
			INodeType: FileNodeT,
			OrigPath:  origpath,
		},
		Size: 0,
	}
	db.PutMustSucceed(fn)
	return fn
}

func NewDirNode(db *INodeDB, origpath string) *DirNode {
	id := db.GenerateNewID()
	dn := &DirNode{
		INodeCommon: INodeCommon{
			ID:   id,
			INodeType: DirNodeT,
			OrigPath:  origpath,
		},
		Entries: make(map[string]ID),
	}
	db.PutMustSucceed(dn)
	return dn
}
*/

type DBStateTransfer interface {
	Apply(s *DBState) error
}

type DBOperation interface {
	JSONEncodable
	DBStateTransfer
}

type DBTransaction struct {
	// FIXME: IssuedAt Time   `json:"issuedat"`
	TxID uint64
	Ops  []*DBStateTransfer
}

type DB struct {
	state DBState

	// FIXME: mutex
}

func (db *DB) ApplyTransaction(tx DBTransaction) {

}

type OpMeta struct {
	Kind string `json:"kind"`
}

type CreateFileOp struct {
	OpMeta   `json:",inline"`
	ID       `json:"id"`
	Name     string `json:"name"`
	OrigPath string `json:"origpath"`
	DirID    ID     `json:"dirid"`
}

func (op *CreateFileOp) Apply(s *DBState) error {
	if _, ok := s.nodes[op.ID]; ok {
		return fmt.Errorf("Node already exists")
	}

	s.nodes[op.ID] = &FileNode{
		INodeCommon: INodeCommon{ID: op.ID, Type: FileNodeT, OrigPath: op.OrigPath},
		Size:        0,
	}
	return nil
}
