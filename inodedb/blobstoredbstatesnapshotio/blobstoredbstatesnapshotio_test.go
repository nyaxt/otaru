package blobstoredbstatesnapshotio_test

import (
	"testing"

	"context"

	"github.com/nyaxt/otaru/flags"
	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/inodedb/blobstoredbstatesnapshotio"
	tu "github.com/nyaxt/otaru/testutils"
)

func init() { tu.EnsureLogger() }

func testRootKey() string { return authtu.TestBucketName + "-blobstoredbstatesnapshotio_test" }

func TestSS_SaveRestore(t *testing.T) {
	loc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(testRootKey()), flags.O_RDWRCREATE)
	if _, err := loc.DeleteAll(context.Background(), false); err != nil {
		t.Errorf("Failed to loc.DeleteAll: %v", err)
	}

	bs := tu.TestFileBlobStore()
	sio := blobstoredbstatesnapshotio.New(bs, tu.TestCipher(), loc)

	db, err := inodedb.NewEmptyDB(sio, inodedb.NewSimpleDBTransactionLogIO())
	if err != nil {
		t.Errorf("Failed to NewEmptyDB: %v", err)
		return
	}
	if err := db.Sync(); err != nil {
		t.Errorf("Failed to Sync DB: %v", err)
		return
	}

	_, err = inodedb.NewDB(sio, inodedb.NewSimpleDBTransactionLogIO(), false)
	if err != nil {
		t.Errorf("Failed to NewDB: %v", err)
	}
}

func mockFileOp(t *testing.T, db inodedb.DBHandler, filename string) bool {
	nlock, err := db.LockNode(inodedb.AllocateNewNodeID)
	if err != nil {
		t.Errorf("Failed to LockNode: %v", err)
		return false
	}

	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.CreateNodeOp{NodeLock: nlock, OrigPath: "/" + filename, Type: inodedb.FileNodeT},
		&inodedb.HardLinkOp{NodeLock: inodedb.NodeLock{inodedb.RootDirID, inodedb.NoTicket}, Name: filename, TargetID: nlock.ID},
	}}
	if _, err := db.ApplyTransaction(tx); err != nil {
		t.Errorf("Failed to apply tx: %v", err)
		return false
	}

	if err := db.UnlockNode(nlock); err != nil {
		t.Errorf("Failed to UnlockNode: %v", err)
		return false
	}
	return true
}

func TestSS_AutoAvoidCorruptedSnapshot(t *testing.T) {
	loc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(testRootKey()), flags.O_RDWRCREATE)
	if _, err := loc.DeleteAll(context.Background(), false); err != nil {
		t.Errorf("Failed to loc.DeleteAll: %v", err)
	}

	bs := tu.TestFileBlobStore()
	sio := blobstoredbstatesnapshotio.New(bs, tu.TestCipher(), loc)
	txlogio := inodedb.NewSimpleDBTransactionLogIO()

	{
		db, err := inodedb.NewEmptyDB(sio, txlogio)
		if err != nil {
			t.Errorf("Failed to NewEmptyDB: %v", err)
			return
		}

		// create 1st snapshot
		if err := db.Sync(); err != nil {
			t.Errorf("Failed to Sync DB: %v", err)
			return
		}

		// apply some mod to inodedb
		if !mockFileOp(t, db, "hoge.txt") {
			return
		}

		// create 2nd snapshot
		if err := db.Sync(); err != nil {
			t.Errorf("Failed to Sync DB (2): %v", err)
			return
		}
	}

	if _, err := inodedb.NewDB(sio, txlogio, false); err != nil {
		t.Errorf("Failed to NewDB (uncorrupted): %v", err)
		return
	}

	// destroy latest snapshot (corrupt data)
	ssbp, _, err := loc.Locate(0)
	if err != nil {
		t.Errorf("Failed to locate latest ssbp: %v", err)
		return
	}
	{
		wc, err := bs.OpenWriter(ssbp)
		if err != nil {
			t.Errorf("Failed to OpenWriter: %v", err)
			return
		}
		if _, err := wc.Write([]byte("hoge")); err != nil {
			t.Errorf("Failed to Write: %v", err)
		}
		wc.Close()
	}

	{
		_, err = inodedb.NewDB(sio, txlogio, false)
		if err != nil {
			t.Errorf("Failed to NewDB (corrupted): %v", err)
			return
		}
	}

	// destroy latest snapshot (remove ss blob)
	if err := bs.RemoveBlob(ssbp); err != nil {
		t.Errorf("Failed to RemoveBlob: %v", err)
	}

	{
		_, err = inodedb.NewDB(sio, txlogio, false)
		if err != nil {
			t.Errorf("Failed to NewDB (ss blob removed): %v", err)
			return
		}
	}
}

func TestSS_DeleteOldSnapshots(t *testing.T) {
	loc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(testRootKey()), flags.O_RDWRCREATE)
	if _, err := loc.DeleteAll(context.Background(), false); err != nil {
		t.Errorf("Failed to loc.DeleteAll: %v", err)
	}

	bs := tu.TestFileBlobStore()
	sio := blobstoredbstatesnapshotio.New(bs, tu.TestCipher(), loc)

	{
		db, err := inodedb.NewEmptyDB(sio, inodedb.NewSimpleDBTransactionLogIO())
		if err != nil {
			t.Errorf("Failed to NewEmptyDB: %v", err)
			return
		}
		if err := db.Sync(); err != nil {
			t.Errorf("Failed to Sync DB: %v", err)
			return
		}
		if !mockFileOp(t, db, "1.txt") {
			return
		}
		if err := db.Sync(); err != nil {
			t.Errorf("Failed to Sync DB: %v", err)
			return
		}
		if !mockFileOp(t, db, "2.txt") {
			return
		}
		if err := db.Sync(); err != nil {
			t.Errorf("Failed to Sync DB: %v", err)
			return
		}
		if !mockFileOp(t, db, "3.txt") {
			return
		}
		if err := db.Sync(); err != nil {
			t.Errorf("Failed to Sync DB: %v", err)
			return
		}
	}

	if err := sio.DeleteOldSnapshots(context.Background(), false); err != nil {
		t.Errorf("DeleteOldSnapshots. err: %v", err)
		return
	}

	{
		db, err := inodedb.NewDB(sio, inodedb.NewSimpleDBTransactionLogIO(), false)
		if err != nil {
			t.Errorf("Failed to NewDB: %v", err)
			return
		}

		v, _, err := db.QueryNode(inodedb.RootDirID, false)
		dv := v.(*inodedb.DirNodeView)
		if _, ok := dv.Entries["3.txt"]; !ok {
			t.Errorf("Failed to find 3.txt")
		}
	}
}
