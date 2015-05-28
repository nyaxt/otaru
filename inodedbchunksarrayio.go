package otaru

import (
	"fmt"
	"log"

	"github.com/nyaxt/otaru/inodedb"
)

type INodeDBChunksArrayIO struct {
	db    inodedb.DBHandler
	nlock inodedb.NodeLock
	fn    inodedb.FileNodeView
}

var _ = ChunksArrayIO(&INodeDBChunksArrayIO{})

func NewINodeDBChunksArrayIO(db inodedb.DBHandler, nlock inodedb.NodeLock, fn inodedb.FileNodeView) *INodeDBChunksArrayIO {
	if nlock.Ticket == inodedb.NoTicket {
		log.Fatalf("NewINodeDBChunksArrayIO requires valid node lock. Use NewReadOnlyINodeDBChunksArrayIO if need read access")
	}

	return &INodeDBChunksArrayIO{db: db, nlock: nlock, fn: fn}
}

func NewReadOnlyINodeDBChunksArrayIO(db inodedb.DBHandler, nlock inodedb.NodeLock) *INodeDBChunksArrayIO {
	if nlock.Ticket != inodedb.NoTicket {
		log.Fatalf("Valid node lock w/ ticket passed to NewReadOnlyINodeDBChunksArrayIO")
	}

	return &INodeDBChunksArrayIO{db: db, nlock: nlock, fn: nil}
}

func (caio *INodeDBChunksArrayIO) Read() ([]inodedb.FileChunk, error) {
	if caio.nlock.Ticket != inodedb.NoTicket {
		return caio.fn.GetChunks(), nil
	}

	v, _, err := caio.db.QueryNode(caio.nlock.ID, false)
	if err != nil {
		return nil, err
	}

	fn, ok := v.(inodedb.FileNodeView)
	if !ok {
		return nil, fmt.Errorf("Target node view is not a file.")
	}

	return fn.GetChunks(), nil
}

func (caio *INodeDBChunksArrayIO) Write(cs []inodedb.FileChunk) error {
	if caio.nlock.Ticket == inodedb.NoTicket {
		return fmt.Errorf("No ticket lock is acquired.")
	}

	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.UpdateChunksOp{NodeLock: caio.nlock, Chunks: cs},
	}}
	if _, err := caio.db.ApplyTransaction(tx); err != nil {
		return fmt.Errorf("Failed to apply tx for updating cs: %v", err)
	}
	return nil
}

func (caio *INodeDBChunksArrayIO) Close() error {
	if caio.nlock.Ticket != inodedb.NoTicket {
		if err := caio.db.UnlockNode(caio.nlock); err != nil {
			return err
		}
	}
	return nil
}
