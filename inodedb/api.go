package inodedb

import (
	"time"
)

type NodeView interface {
	// GetVersion() TxID

	GetID() ID
	GetType() Type
}

type FileNodeView interface {
	NodeView
	GetSize() int64
	GetChunks() []FileChunk
}

type DirNodeView interface {
	NodeView
	GetEntries() map[string]ID
}

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
