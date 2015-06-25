package inodedb

import (
	"errors"
	"fmt"
	"log"
	"math"
	"syscall"
	"time"

	bfuse "bazil.org/fuse"
)

type Errno syscall.Errno

func (e Errno) Errno() bfuse.Errno {
	return bfuse.Errno(e)
}

func (e Errno) Error() string {
	return syscall.Errno(e).Error()
}

var (
	EEXIST         = Errno(syscall.EEXIST)
	ENOENT         = Errno(syscall.ENOENT)
	ENOTDIR        = Errno(syscall.ENOTDIR)
	ENOTEMPTY      = Errno(syscall.ENOTEMPTY)
	ErrLockInvalid = errors.New("Invalid lock given.")
	ErrLockTaken   = errors.New("Lock is already acquired by someone else.")
)

func IsErrNotFound(err error) bool { return err == ENOENT }

type ID uint64

const (
	AllocateNewNodeID = 0
)

type Type int

const (
	FileNodeT = iota
	DirNodeT
	// SymlinkNode
)

type INode interface {
	GetID() ID
	GetType() Type

	GobEncodable

	View() NodeView
}

type TxID int64

const (
	LatestVersion = math.MaxInt64
	AnyVersion    = 0
)

type DBState struct {
	nodes   map[ID]INode
	lastID  ID
	version TxID

	lastTicket Ticket
	nodeLocks  map[ID]NodeLock
}

func NewDBState() *DBState {
	return &DBState{
		nodes:   make(map[ID]INode),
		lastID:  0,
		version: 0,

		lastTicket: 1,
		nodeLocks:  make(map[ID]NodeLock),
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

func (s *DBState) checkLock(nlock NodeLock, requireTicket bool) error {
	// if ticket is required, make sure we have one
	if requireTicket && nlock.Ticket == NoTicket {
		return ErrLockInvalid
	}

	// if node is locked, and nlock isn't the lock, return error
	existing, ok := s.nodeLocks[nlock.ID]
	if ok && nlock != existing {
		return ErrLockTaken
	}

	return nil
}

func (s *DBState) Version() TxID {
	return s.version
}

type INodeCommon struct {
	ID

	// OrigPath contains filepath passed to first create and does not necessary follow "rename" operations.
	// To be used for recovery/debug purposes only
	OrigPath string
}

func (n INodeCommon) GetID() ID {
	return n.ID
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

func (fn *FileNode) GetType() Type { return FileNodeT }

type fileNodeView struct {
	ss FileNode
}

func (fn *FileNode) View() NodeView {
	v := &fileNodeView{
		ss: FileNode{
			INodeCommon: fn.INodeCommon,
			Size:        fn.Size,
			Chunks:      make([]FileChunk, len(fn.Chunks)),
		},
	}
	copy(v.ss.Chunks, fn.Chunks)

	return v
}

func (v fileNodeView) GetID() ID              { return v.ss.GetID() }
func (v fileNodeView) GetType() Type          { return v.ss.GetType() }
func (v fileNodeView) GetSize() int64         { return v.ss.Size }
func (v fileNodeView) GetChunks() []FileChunk { return v.ss.Chunks }

type DirNode struct {
	INodeCommon
	Entries map[string]ID
}

var _ = INode(&DirNode{})

func (dn *DirNode) GetType() Type { return DirNodeT }

type dirNodeView struct {
	ss DirNode
}

func (dn *DirNode) View() NodeView {
	v := &dirNodeView{
		ss: DirNode{
			INodeCommon: dn.INodeCommon,
			Entries:     make(map[string]ID),
		},
	}
	for name, id := range dn.Entries {
		v.ss.Entries[name] = id
	}

	return v
}

func (v dirNodeView) GetID() ID                 { return v.ss.GetID() }
func (v dirNodeView) GetType() Type             { return v.ss.GetType() }
func (v dirNodeView) GetEntries() map[string]ID { return v.ss.Entries }

type DBTransaction struct {
	// FIXME: IssuedAt Time   `json:"issuedat"`
	TxID `json:"txid"`
	Ops  []DBOperation `json:'ops'`
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

	stats DBServiceStats
}

var _ = DBHandler(&DB{})

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
			&InitializeFileSystemOp{},
		},
	}
	if _, err := db.ApplyTransaction(tx); err != nil {
		return nil, fmt.Errorf("Failed to initilaize db: %v", err)
	}
	if err := db.Sync(); err != nil {
		return nil, fmt.Errorf("Failed to sync db: %v", err)
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

	ssver := state.version

	if state.version > version {
		return fmt.Errorf("Can't rollback to old version %d which is older than snapshot version %d", version, state.version)
	}

	txlog, err := db.txLogIO.QueryTransactions(ssver + 1)
	if txlog == nil || err != nil {
		db.state = oldState
		return fmt.Errorf("Failed to query txlog: %v", err)
	}

	for _, tx := range txlog {
		if _, err := db.ApplyTransaction(tx); err != nil {
			db.state = oldState
			return fmt.Errorf("Failed to replay tx: %v", err)
		}
	}

	log.Printf("Fast forward txlog from ver %d to %d", ssver, state.version)

	return nil
}

func (db *DB) ApplyTransaction(tx DBTransaction) (TxID, error) {
	if tx.TxID == AnyVersion {
		tx.TxID = db.state.version + 1
	} else if tx.TxID != db.state.version+1 {
		return 0, fmt.Errorf("Skipped tx %d", db.state.version+1)
	}

	for _, op := range tx.Ops {
		if err := op.Apply(db.state); err != nil {
			if rerr := db.RestoreVersion(db.state.version); rerr != nil {
				log.Fatalf("Following Error: %v. DB rollback failed!!!: %v", err, rerr)
			}
			return 0, err
		}
	}
	if err := db.txLogIO.AppendTransaction(tx); err != nil {
		if rerr := db.RestoreVersion(db.state.version); rerr != nil {
			log.Fatalf("Failed to write txlog: %v. DB rollback failed!!!: %v", err, rerr)
		}
		return 0, fmt.Errorf("Failed to write txlog: %v", err)
	}

	db.state.version = tx.TxID
	return tx.TxID, nil
}

func (db *DB) QueryNode(id ID, tryLock bool) (NodeView, NodeLock, error) {
	n := db.state.nodes[id]
	if n == nil {
		return nil, NodeLock{}, ENOENT
	}
	v := n.View()

	if tryLock {
		nlock, err := db.LockNode(id)
		return v, nlock, err
	}

	return v, NodeLock{ID: id, Ticket: NoTicket}, nil
}

func (db *DB) LockNode(id ID) (NodeLock, error) {
	if id == AllocateNewNodeID {
		id = db.state.lastID + 1
		db.state.lastID = id
	}

	if _, ok := db.state.nodeLocks[id]; ok {
		return NodeLock{}, ErrLockTaken
	}

	ticket := db.state.lastTicket + 1
	db.state.lastTicket = ticket

	nlock := NodeLock{ID: id, Ticket: ticket}

	db.state.nodeLocks[id] = nlock
	return nlock, nil
}

func (db *DB) UnlockNode(nlock NodeLock) error {
	if err := db.state.checkLock(nlock, true); err != nil {
		return err
	}

	delete(db.state.nodeLocks, nlock.ID)
	return nil
}

func (db *DB) Sync() error {
	if err := db.snapshotIO.SaveSnapshot(db.state); err != nil {
		return err
	}

	db.stats.LastSync = time.Now()
	return nil
}

func (db *DB) GetStats() DBServiceStats {
	stats := db.stats
	stats.LastID = db.state.lastID
	stats.Version = db.state.version
	stats.LastTicket = db.state.lastTicket
	stats.NumberOfNodeLocks = len(db.state.nodeLocks)

	return stats
}

var _ = QueryRecentTransactionsProvider(&DB{})

func (db *DB) QueryRecentTransactions() ([]DBTransaction, error) {
	ctxio, ok := db.txLogIO.(*CachedDBTransactionLogIO)
	if !ok {
		return nil, fmt.Errorf("TxLogIO backend isn't CachedDBTransactionLogIO")
	}

	var minID TxID
	if db.state.version > 17 {
		minID = db.state.version - 16
	} else {
		minID = 1
	}
	return ctxio.QueryCachedTransactions(minID)
}
