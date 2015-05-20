package inodedb

import (
	"fmt"
	"log"
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
		nodes:   make(map[ID]INode),
		lastID:  0,
		version: 0,
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
			log.Fatalf("nodes map key (%d) != node.GetID() result (%d)", id, node.GetID())
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

func deserializeCommon(dec *gob.Decoder, t Type, c *INodeCommon) error {
	c.Type = t

	if err := dec.Decode(&c.ID); err != nil {
		return fmt.Errorf("Failed to decode ID: %v", err)
	}

	if err := dec.Decode(&c.OrigPath); err != nil {
		return fmt.Errorf("Failed to decode OrigPath: %v", err)
	}

	return nil
}

func deserializeFileNodeSnapshot(dec *gob.Decoder) (*FileNode, error) {
	fn := &FileNode{}
	if err := deserializeCommon(dec, FileNodeT, &fn.INodeCommon); err != nil {
		return nil, err
	}

	if err := dec.Decode(&fn.Size); err != nil {
		return nil, fmt.Errorf("Failed to decode Size: %v", err)
	}

	if err := dec.Decode(&fn.Chunks); err != nil {
		return nil, fmt.Errorf("Failed to decode Chunks: %v", err)
	}

	return fn, nil
}

func deserializeDirNodeSnapshot(dec *gob.Decoder) (*DirNode, error) {
	dn := &DirNode{}
	if err := deserializeCommon(dec, DirNodeT, &dn.INodeCommon); err != nil {
		return nil, err
	}

	if err := dec.Decode(&dn.Entries); err != nil {
		return nil, fmt.Errorf("Failed to decode Entries: %v", err)
	}

	return dn, nil
}

func DecodeNodeFromGob(dec *gob.Decoder) (INode, error) {
	var t Type
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("Failed to decode Type: %v", err)
	}

	switch t {
	case FileNodeT:
		fn, err := deserializeFileNodeSnapshot(dec)
		return fn, err

	case DirNodeT:
		dn, err := deserializeDirNodeSnapshot(dec)
		return dn, err

	default:
	}
	return nil, fmt.Errorf("Invalid Type: %d", t)
}

func DecodeDBStateFromGob(dec *gob.Decoder) (*DBState, error) {
	s := &DBState{}

	var numNodes uint64
	if err := dec.Decode(&numNodes); err != nil {
		return nil, fmt.Errorf("failed to decode numNodes: %v", err)
	}
	for i := uint64(0); i < numNodes; i++ {
		n, err := DecodeNodeFromGob(dec)
		if err != nil {
			return nil, fmt.Errorf("failed to decode node: %v", err)
		}
		s.nodes[n.GetID()] = n
	}

	if err := dec.Decode(&s.lastID); err != nil {
		return nil, fmt.Errorf("Failed to decode lastID: %v", err)
	}
	if err := dec.Decode(&s.version); err != nil {
		return nil, fmt.Errorf("Failed to decode version: %v", err)
	}

	return s, nil
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

type FileNode struct {
	INodeCommon
	Size   int64
	Chunks []FileChunk
}

var _ = INode(&FileNode{})

func serializeCommon(enc *gob.Encoder, c INodeCommon) error {
	if err := enc.Encode(c.Type); err != nil {
		return fmt.Errorf("Failed to encode Type: %v", err)
	}

	if err := enc.Encode(c.ID); err != nil {
		return fmt.Errorf("Failed to encode ID: %v", err)
	}

	if err := enc.Encode(c.OrigPath); err != nil {
		return fmt.Errorf("Failed to encode OrigPath: %v", err)
	}

	return nil
}

func (fn *FileNode) EncodeToGob(enc *gob.Encoder) error {
	if err := serializeCommon(enc, fn.INodeCommon); err != nil {
		return err
	}

	if err := enc.Encode(fn.Size); err != nil {
		return fmt.Errorf("Failed to encode Size: %v", err)
	}

	if err := enc.Encode(fn.Chunks); err != nil {
		return fmt.Errorf("Failed to encode Chunks: %v", err)
	}

	return nil
}

type DirNode struct {
	INodeCommon
	Entries map[string]ID
}

var _ = INode(&DirNode{})

func (dn *DirNode) EncodeToGob(enc *gob.Encoder) error {
	if err := serializeCommon(enc, dn.INodeCommon); err != nil {
		return err
	}

	if err := enc.Encode(dn.Entries); err != nil {
		return fmt.Errorf("Failed to encode Entries: %v", err)
	}

	return nil
}

type DBOperation interface {
	Apply(s *DBState) error
}

type DBTransaction struct {
	// FIXME: IssuedAt Time   `json:"issuedat"`
	TxID
	Ops []DBOperation
}

type DBStateSnapshotIO interface {
	SaveSnapshot(s *DBState) error
	RestoreSnapshot() (*DBState, error)
}

type DBTransactionLogIO interface {
	AppendTransaction(tx DBTransaction) error
	QueryTransactions(minID TxID) ([]DBTransaction, error)
}

type DB struct {
	state *DBState

	snapshotIO DBStateSnapshotIO
	txLogIO    DBTransactionLogIO
}

func newDB(snapshotIO DBStateSnapshotIO, txLogIO DBTransactionLogIO) *DB {
	return &DB{
		state:      NewDBState(),
		snapshotIO: snapshotIO,
		txLogIO:    txLogIO,
	}
}

func NewEmptyDB(snapshotIO DBStateSnapshotIO, txLogIO DBTransactionLogIO) (*DB, error) {
	db := newDB(snapshotIO, txLogIO)
	if _, err := snapshotIO.RestoreSnapshot(); err == nil {
		return nil, fmt.Errorf("NewEmptyDB: Refusing to use non-empty snapshotIO")
	}
	if txs, err := txLogIO.QueryTransactions(0); err == nil && len(txs) > 0 {
		return nil, fmt.Errorf("NewEmptyDB: Refusing to use non-empty txLogIO")
	}

	tx := DBTransaction{
		TxID: 1,
		Ops: []DBOperation{
			&AssertEmptyFileSystemOp{},
			&CreateDirOp{ID: 1, OrigPath: "/", DirID: 0},
		},
	}
	if err := db.ApplyTransaction(tx); err != nil {
		return nil, fmt.Errorf("Failed to initilaize db: %v", err)
	}
	return db, nil
}

func NewDB(snapshotIO DBStateSnapshotIO, txLogIO DBTransactionLogIO) (*DB, error) {
	db := newDB(snapshotIO, txLogIO)
	if err := db.RestoreVersion(LatestVersion); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) RestoreVersion(version TxID) error {
	state, err := db.snapshotIO.RestoreSnapshot()
	if err != nil {
		return fmt.Errorf("Failed to restore snapshot: %v", err)
	}

	oldState := db.state
	db.state = state

	if state.version > version {
		return fmt.Errorf("Can't rollback to old version %d which is older than snapshot version %d", version, state.version)
	}

	txlog, err := db.txLogIO.QueryTransactions(state.version + 1)
	if txlog == nil || err != nil {
		db.state = oldState
		return fmt.Errorf("Failed to query txlog: %v", err)
	}

	for _, tx := range txlog {
		if err := db.ApplyTransaction(tx); err != nil {
			db.state = oldState
			return fmt.Errorf("Failed to replay tx: %v", err)
		}
	}

	return nil
}

func (db *DB) ApplyTransaction(tx DBTransaction) error {
	if tx.TxID != db.state.version+1 {
		return fmt.Errorf("Skipped tx %d", db.state.version+1)
	}

	for _, op := range tx.Ops {
		if err := op.Apply(db.state); err != nil {
			if rerr := db.RestoreVersion(db.state.version); rerr != nil {
				log.Fatalf("Following Error: %v. DB rollback failed!!!: %v", err, rerr)
			}
			return err
		}
	}

	db.state.version = tx.TxID
	return nil
}

func (db *DB) ReadNode(id ID) INode {
	return db.state.nodes[id]
}

type OpMeta struct {
	Kind string `json:"kind"`
}

type AssertEmptyFileSystemOp struct {
	OpMeta `json:",inline"`
}

var _ = DBOperation(&AssertEmptyFileSystemOp{})

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

var _ = DBOperation(&CreateDirOp{})

func (op *CreateDirOp) Apply(s *DBState) error {
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
