package inodedb

import (
	"time"
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

type INodeCommon struct {
	ID

	// OrigPath contains filepath passed to first create and does not necessary follow "rename" operations.
	// To be used for recovery/debug purposes only
	OrigPath string
}

func (n INodeCommon) GetID() ID {
	return n.ID
}

type NodeView interface {
	// GetVersion() TxID

	GetID() ID
	GetType() Type
}

type FileNodeView struct {
	INodeCommon `json:",inline"`
	Size        int64       `json:"size"`
	Chunks      []FileChunk `json:"chunks"`
}

var _ = NodeView(&FileNodeView{})

func (v FileNodeView) GetType() Type { return FileNodeT }

type DirNodeView struct {
	INodeCommon `json:",inline"`
	Entries     map[string]ID `json:"entries"`
}

var _ = NodeView(&DirNodeView{})

func (v DirNodeView) GetType() Type { return DirNodeT }

type Ticket uint64

const NoTicket = Ticket(0)

type NodeLock struct {
	ID
	Ticket

	// FIXME: add Expire
}

func (nlock NodeLock) HasTicket() bool { return nlock.Ticket != NoTicket }

type DBHandler interface {
	// ApplyTransaction applies DBTransaction to db.state, and returns applied transaction's TxID. If it fails to apply the transaction, it rollbacks intermediate state and returns error.
	ApplyTransaction(tx DBTransaction) (TxID, error)

	// QueryNode returns read-only snapshot of INode id, with a lock if specified
	QueryNode(id ID, tryLock bool) (NodeView, NodeLock, error)

	// FIXME: this should actually take NodeLock for renew ticket operation
	LockNode(id ID) (NodeLock, error)
	UnlockNode(nlock NodeLock) error
}

type DBServiceStats struct {
	// === Fields that are kept up to date by DBHandler ===

	LastSync time.Time `json:"last_sync"`
	LastTx   time.Time `json:"last_tx"`

	// === Fields dynamically filled in on GetStats() ===

	LastID            ID     `json:"last_id"`
	Version           TxID   `json:"version"`
	LastTicket        Ticket `json:"last_ticket"`
	NumberOfNodeLocks int    `json:"number_of_node_locks"`
}

type DBServiceStatsProvider interface {
	GetStats() DBServiceStats
}

type QueryRecentTransactionsProvider interface {
	QueryRecentTransactions() ([]DBTransaction, error)
}

type DBFscker interface {
	Fsck() ([]string, []error)
}
