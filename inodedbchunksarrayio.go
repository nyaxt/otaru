package otaru

import (
	"fmt"

	"github.com/nyaxt/otaru/inodedb"
)

type INodeDBChunksArrayIO struct {
	db    inodedb.DBHandler
	nlock inodedb.NodeLock
}

var _ = ChunksArrayIO(&INodeDBChunksArrayIO{})

func NewINodeDBChunksArrayIO(db inodedb.DBHandler, nlock inodedb.NodeLock) *INodeDBChunksArrayIO {
	return &INodeDBChunksArrayIO{db: db, nlock: nlock}
}

func (caio *INodeDBChunksArrayIO) Read() ([]inodedb.FileChunk, error) {
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
	if !caio.nlock.HasTicket() {
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
