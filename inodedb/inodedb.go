package inodedb

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"syscall"
	"time"

	bfuse "github.com/nyaxt/fuse"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("inodedb")

type Errno syscall.Errno

func (e Errno) Errno() bfuse.Errno {
	return bfuse.Errno(e)
}

func (e Errno) Error() string {
	return syscall.Errno(e).Error()
}

var (
	ErrLockInvalid = errors.New("Invalid lock given.")
	ErrLockTaken   = errors.New("Lock is already acquired by someone else.")
)

func IsErrNotFound(err error) bool { return err == util.ENOENT }

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

func (id TxID) String() string {
	if id == LatestVersion {
		return "LatestVersion"
	}
	if id == AnyVersion {
		return "AnyVersion"
	}
	return strconv.FormatInt(int64(id), 10)
}

const (
	LatestVersion TxID = math.MaxInt64
	AnyVersion    TxID = 0
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

type FileNode struct {
	INodeCommon
	Size   int64
	Chunks []FileChunk
}

var _ = INode(&FileNode{})

func (fn *FileNode) GetType() Type { return FileNodeT }

func (fn *FileNode) View() NodeView {
	v := &FileNodeView{
		INodeCommon: fn.INodeCommon,
		Size:        fn.Size,
		Chunks:      make([]FileChunk, len(fn.Chunks)),
	}
	copy(v.Chunks, fn.Chunks)

	return v
}

type DirNode struct {
	INodeCommon
	ParentID ID
	Entries  map[string]ID
}

var _ = INode(&DirNode{})

func (dn *DirNode) GetType() Type { return DirNodeT }

func (dn *DirNode) View() NodeView {
	v := &DirNodeView{
		INodeCommon: dn.INodeCommon,
		ParentID:    dn.ParentID,
		Entries:     make(map[string]ID),
	}
	for name, id := range dn.Entries {
		v.Entries[name] = id
	}

	return v
}

type DBStateSnapshotIO interface {
	SaveSnapshot(s *DBState) <-chan error
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

	readOnly bool
	stats    DBServiceStats
}

var _ = DBHandler(&DB{})

func newDB(snapshotIO DBStateSnapshotIO, txLogIO DBTransactionLogIO, readOnly bool) *DB {
	return &DB{
		state:      NewDBState(),
		snapshotIO: snapshotIO,
		txLogIO:    txLogIO,
		readOnly:   readOnly,
	}
}

func NewEmptyDB(snapshotIO DBStateSnapshotIO, txLogIO DBTransactionLogIO) (*DB, error) {
	db := newDB(snapshotIO, txLogIO, false)
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

func NewDB(snapshotIO DBStateSnapshotIO, txLogIO DBTransactionLogIO, readOnly bool) (*DB, error) {
	db := newDB(snapshotIO, txLogIO, readOnly)
	if err := db.RestoreVersion(LatestVersion); err != nil {
		return nil, err
	}

	return db, nil
}

const (
	writeTxLog = true
	skipTxLog  = false
)

func (db *DB) RestoreVersion(version TxID) error {
	logger.Infof(mylog, "RestoreVersion(%s) start.", version)

	state, err := db.snapshotIO.RestoreSnapshot()
	if err != nil {
		return fmt.Errorf("Failed to restore snapshot: %v", err)
	}

	oldState := db.state
	db.state = state

	ssver := state.version
	logger.Infof(mylog, "Restored snapshot of ver %d.", ssver)

	if state.version > version {
		return fmt.Errorf("Can't rollback to old version %d which is older than snapshot version %d", version, state.version)
	}
	logger.Infof(mylog, "RestoreVersion(%s): restored ver: %s", version, ssver)

	txlog, err := db.txLogIO.QueryTransactions(ssver + 1)
	if txlog == nil || err != nil {
		db.state = oldState
		return fmt.Errorf("Failed to query txlog: %v", err)
	}

	for _, tx := range txlog {
		logger.Debugf(mylog, "RestoreVersion(%s): apply tx ver %s", version, tx.TxID)
		if _, err := db.applyTransactionInternal(tx, skipTxLog); err != nil {
			db.state = oldState
			return fmt.Errorf("Failed to replay tx: %v", err)
		}
	}

	logger.Infof(mylog, "Fast forward txlog from ver %d to %d", ssver, state.version)

	return nil
}

func (db *DB) applyTransactionInternal(tx DBTransaction, writeTxLogFlag bool) (TxID, error) {
	logger.Debugf(mylog, "applyTransactionInternal(%+v, writeTxLog: %t)", tx, writeTxLogFlag)

	if tx.TxID == AnyVersion {
		tx.TxID = db.state.version + 1
	} else if tx.TxID != db.state.version+1 {
		return 0, fmt.Errorf("Attempted to apply tx %d to dbver %d. Next accepted tx is %d", tx.TxID, db.state.version, db.state.version+1)
	}

	for _, op := range tx.Ops {
		if err := op.Apply(db.state); err != nil {
			if rerr := db.RestoreVersion(db.state.version); rerr != nil {
				logger.Panicf(mylog, "Following Error: %v. DB rollback failed!!!: %v", err, rerr)
			}
			return 0, err
		}
	}
	if writeTxLogFlag == writeTxLog {
		if err := db.txLogIO.AppendTransaction(tx); err != nil {
			if rerr := db.RestoreVersion(db.state.version); rerr != nil {
				logger.Panicf(mylog, "Failed to write txlog: %v. DB rollback failed!!!: %v", err, rerr)
			}
			return 0, fmt.Errorf("Failed to write txlog: %v", err)
		}
	}

	db.state.version = tx.TxID
	db.stats.LastTx = time.Now()
	return tx.TxID, nil
}

func (db *DB) ApplyTransaction(tx DBTransaction) (TxID, error) {
	if db.readOnly {
		return 0, util.EACCES
	}

	return db.applyTransactionInternal(tx, writeTxLog)
}

func (db *DB) QueryNode(id ID, tryLock bool) (NodeView, NodeLock, error) {
	n := db.state.nodes[id]
	if n == nil {
		return nil, NodeLock{}, util.ENOENT
	}
	v := n.View()

	if tryLock {
		nlock, err := db.LockNode(id)
		return v, nlock, err
	}

	return v, NodeLock{ID: id, Ticket: NoTicket}, nil
}

func (db *DB) LockNode(id ID) (NodeLock, error) {
	if db.readOnly {
		logger.Warningf(mylog, "Lock node readonlly")
		return NodeLock{}, util.EACCES
	}

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
	if db.readOnly {
		return util.EACCES
	}

	if err := db.state.checkLock(nlock, true); err != nil {
		logger.Warningf(mylog, "Unlock node failed: %v", err)
		return err
	}

	delete(db.state.nodeLocks, nlock.ID)
	return nil
}

func (db *DB) TriggerSync() <-chan error {
	errC := db.snapshotIO.SaveSnapshot(db.state)

	errCwrap := make(chan error, 1)
	if errC == nil {
		close(errCwrap)
		return errCwrap
	}

	go func() {
		err := <-errC
		if err == nil {
			db.stats.LastSync = time.Now()
		}

		errCwrap <- err
		close(errCwrap)
	}()
	return errCwrap
}

func (db *DB) Sync() error {
	if db.readOnly {
		return util.EACCES
	}

	return <-db.TriggerSync()
}

func (db *DB) fsckRecursive(id ID, foundblobpaths []string, errs []error) ([]string, []error) {
	n, ok := db.state.nodes[id]
	if !ok {
		errs = append(errs, fmt.Errorf("Node ID %d not found", id))
		return foundblobpaths, errs
	}
	switch n.GetType() {
	case FileNodeT:
		fn, ok := n.(*FileNode)
		if !ok {
			errs = append(errs, fmt.Errorf("Node ID %d said it is FileNodeT, but cast failed", id))
		} else {
			for _, fc := range fn.Chunks {
				foundblobpaths = append(foundblobpaths, fc.BlobPath)
			}
		}

	case DirNodeT:
		dn, ok := n.(*DirNode)
		if !ok {
			errs = append(errs, fmt.Errorf("Node ID %d said it is FileNodeT, but cast failed", id))
		} else {
			for _, cid := range dn.Entries {
				foundblobpaths, errs = db.fsckRecursive(cid, foundblobpaths, errs)
			}
		}

	default:
		errs = append(errs, fmt.Errorf("Node ID %d has unknown type %v", id, n.GetType()))
	}
	return foundblobpaths, errs
}

func (db *DB) Fsck() ([]string, []error) {
	if db.readOnly {
		return nil, []error{util.EACCES}
	}

	foundblobpaths := make([]string, 0)
	errs := make([]error, 0)
	return db.fsckRecursive(RootDirID, foundblobpaths, errs)
}

var _ = DBServiceStatsProvider(&DB{})

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
