package inodedb_test

import (
	"testing"

	i "github.com/nyaxt/otaru/inodedb"
)

func TestInitialState(t *testing.T) {
	db, err := i.NewEmptyDB(i.NewSimpleDBStateSnapshotIO(), i.NewSimpleDBTransactionLogIO())
	if err != nil {
		t.Errorf("Failed to NewEmptyDB: %v", err)
		return
	}

	nv, _, err := db.QueryNode(1, false)
	if err != nil {
		t.Errorf("Failed to query root dir")
		return
	}
	if nv.GetType() != i.DirNodeT {
		t.Errorf("root dir not found!")
	}
}

func TestCreateFile(t *testing.T) {
	db, err := i.NewEmptyDB(i.NewSimpleDBStateSnapshotIO(), i.NewSimpleDBTransactionLogIO())
	if err != nil {
		t.Errorf("Failed to NewEmptyDB: %v", err)
		return
	}

	nlock, err := db.LockNode(i.AllocateNewNodeID)
	if err != nil {
		t.Errorf("Failed to LockNode: %v", err)
		return
	}

	tx := i.DBTransaction{Ops: []i.DBOperation{
		&i.CreateFileOp{NodeLock: nlock, OrigPath: "/hoge.txt"},
		&i.HardLinkOp{NodeLock: i.NodeLock{1, i.NoTicket}, Name: "hoge.txt", TargetID: nlock.ID},
	}}
	if _, err := db.ApplyTransaction(tx); err != nil {
		t.Errorf("Failed to apply tx: %v", err)
	}

	if err := db.UnlockNode(nlock); err != nil {
		t.Errorf("Failed to UnlockNode: %v", err)
	}
}
