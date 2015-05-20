package inodedb_test

import (
	"testing"

	"github.com/nyaxt/otaru/inodedb"
)

func TestInitialState(t *testing.T) {
	db, err := inodedb.NewEmptyDB(inodedb.NewSimpleDBStateSnapshotIO(), inodedb.NewSimpleDBTransactionLogIO())
	if err != nil {
		t.Errorf("Failed to NewEmptyDB: %v", err)
		return
	}

	if db.ReadNode(1).GetType() != inodedb.DirNodeT {
		t.Errorf("root dir not found!")
	}
}
