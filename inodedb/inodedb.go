package inodedb

import (
	"fmt"
	"math"

	"encoding/gob"
)

type ID uint64
type Type int

const (
	FileNodeT = iota
	DirNodeT
	// SymlinkNode
)

type INode interface {
	GetID() ID
	GetType() Type

	EncodeToGob(enc *gob.Encoder) error
}

type TxID uint64

const (
	LatestVersion = math.MaxUint64
)

type DBState struct {
	nodes   map[ID]INode
	lastID  ID
	version TxID
}

func NewDBState() *DBState {
	return &DBState{
		nodes:  make(map[ID]INode),
		lastID: 0,
	}
}

// addNewNode must be only used from Create*Op.Apply implementations.
func (s *DBState) addNewNode(node INode) error {
	id := node.GetID()

	if _, ok := s.nodes[id]; ok {
		return fmt.Errorf("Node already exists")
	}
	if id < s.lastID {
		return fmt.Errorf("ID may be being reused")
	}

	s.nodes[id] = node
	s.lastID = id
	return nil
}

func (s *DBState) EncodeToGob(enc *gob.Encoder) error {
	numNodes := uint64(len(s.nodes))
	if err := enc.Encode(numNodes); err != nil {
		return fmt.Errorf("Failed to encode numNodes: %v", err)
	}
	for id, node := range s.nodes {
		if id != node.GetID() {
			fmt.Fatalf("nodes map key (%d) != node.GetID() result (%d)", id, node.GetID())
		}

		if err := node.EncodeToGob(enc); err != nil {
			return fmt.Errorf("Failed to encode node: %v", err)
		}
	}

	if err := enc.Encode(s.lastID); err != nil {
		return fmt.Errorf("Failed to encode lastID: %v", err)
	}
	if err := enc.Encode(s.version); err != nil {
		return fmt.Errorf("Failed to encode version: %v", err)
	}
	return nil
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
	TxID
	Ops []*DBOperation
}

type DBStateSnapshotIO interface {
	SaveSnapshot(s *DBState) error
	RestoreSnapshot() (*DBState, error)
}

type DBTransactionLogIO interface {
	AppendTransaction(tx *DBTransaction) error
	QueryTransactions(minID TxID) ([]*DBTransaction, error)
}

type DB struct {
	state *DBState

	snapshotIO DBStateSnapshotIO
	txLogIO    DBTransactionLogIO
}

func NewEmptyDB(snapshotIO DBStateSnapshotIO, txLogIO DBTransactionLogIO) *DB {
	return &DB{
		state:      NewDBState(),
		snapshotIO: DBStateSnapshotIO,
		txLogIO:    txLogIO,
	}
}

func NewDB(snapshotIO DBStateSnapshotIO, txLogIO DBTransactionLogIO) (*DB, error) {
	db := NewEmptyDB(snapshotIO, txLogIO)
	if err := db.RestoreVersion(LatestVersion); err != nil {
		return err
	}
}

func (db *DB) RestoreVersion(version TxID) error {
	state, err := db.snapshotIO.RestoreSnapshot()
	if err != nil {
		return fmt.Errorf("Failed to restore snapshot: %v")
	}

	oldState := db.state
	db.state = state

	if state.version > version {
		return fmt.Errorf("Can't rollback to old version %d which is older than snapshot version %d", version, db.snapshotState.version)
	}

	txlog, err := db.txLogIO.QueryTransactions(state.version + 1)
	if txlog == nil || err != nil {
		db.state = oldState
		return fmt.Errorf("Failed to query txlog: %v", err)
	}

	for i, tx := range txlog {
		if err := db.ApplyTransaction(tx); err != nil {
			db.state = oldState
			return fmt.Errorf("Failed to replay tx: %v", err)
		}
	}

	return nil
}

func (db *DB) ApplyTransaction(tx DBTransaction) error {
	if tx.TxID != db.version+1 {
		return fmt.Errorf("Skipped tx %d", db.version+1)
	}

	for _, op := range db.Ops {
		if err := db.rollbackToVersionLocked(db.version); err != nil {
			log.Fatalf("DB rollback failed!!!: %v", err)
		}
		return err
	}

	db.version = tx.TxID
	return nil
}

type OpMeta struct {
	Kind string `json:"kind"`
}

type AssertEmptyFileSystemOp struct {
	OpMeta `json:",inline"`
}

func (op *AssertEmptyFileSystemOp) Apply(s *DBState) error {
	if len(s.nodes) != 0 {
		return fmt.Errorf("DB not empty. Already contains %d nodes!", len(s.nodes))
	}
	if s.lastID != 0 {
		return fmt.Errorf("DB lastId != 0")
	}

	return nil
}

type CreateDirOp struct {
	OpMeta   `json:",inline"`
	ID       `json:"id"`
	Name     string `json:"name"`
	OrigPath string `json:"origpath"`
	DirID    ID     `json:"dirid"`
}

func (op *InitializeFileSystemOp) Apply(s *DBState) error {
	n := &DirNode{
		INodeCommon: INodeCommon{ID: op.ID, Type: DirNodeT, OrigPath: op.OrigPath},
		Entries:     make(map[string]ID),
	}

	if err := s.addNewNode(n); err != nil {
		return fmt.Errorf("Failed to create new FileNode: %v", err)
	}

	return nil
}

type CreateFileOp struct {
	OpMeta   `json:",inline"`
	ID       `json:"id"`
	Name     string `json:"name"`
	OrigPath string `json:"origpath"`
	DirID    ID     `json:"dirid"`
}

func (op *CreateFileOp) Apply(s *DBState) error {
	n := &FileNode{
		INodeCommon: INodeCommon{ID: op.ID, Type: FileNodeT, OrigPath: op.OrigPath},
		Size:        0,
	}

	if err := s.addNewNode(n); err != nil {
		return fmt.Errorf("Failed to create new FileNode: %v", err)
	}

	return nil
}
