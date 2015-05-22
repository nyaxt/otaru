package inodedb

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

type DBHandler interface {
	// ApplyTransaction applies DBTransaction to db.state, and returns applied transaction's TxID. If it fails to apply the transaction, it rollbacks intermediate state and returns error.
	ApplyTransaction(tx DBTransaction) (TxID, error)

	// QueryNode returns read-only snapshot of INode id, with a lock if specified
	QueryNode(id ID, tryLock bool) (NodeView, NodeLock, error)

	LockNode(id ID) (NodeLock, error)
	UnlockNode(nlock NodeLock) error
}

/*

File write:
- Acquire lock when opened with write perm
{
  - get old chunks
  - cs <- add new chunk ** not cancellable **
  - save new cs
}
- keep renewing the lock
- release the lock when done

Rename:
atomic {
  - link new dir
  - unlink old dir
}

CreateFile:
atomic {
  - create new file node
  - link new dir
}

*/
