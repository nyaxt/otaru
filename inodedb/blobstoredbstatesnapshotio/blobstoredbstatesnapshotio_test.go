package blobstoredbstatesnapshotio_test

import (
	"testing"

	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/inodedb/blobstoredbstatesnapshotio"
	tu "github.com/nyaxt/otaru/testutils"
)

func testRootKey() string { return authtu.TestBucketName() + "-blobstoredbstatesnapshotio_test" }

func TestSS_SaveRestore(t *testing.T) {
	loc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(testRootKey()))
	if _, err := loc.DeleteAll(); err != nil {
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

	_, err = inodedb.NewDB(sio, inodedb.NewSimpleDBTransactionLogIO())
	if err != nil {
		t.Errorf("Failed to NewDB: %v", err)
	}
}

func TestSS_AutoAvoidCorruptedSnapshot(t *testing.T) {
	loc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(testRootKey()))
	if _, err := loc.DeleteAll(); err != nil {
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
		nlock, err := db.LockNode(inodedb.AllocateNewNodeID)
		if err != nil {
			t.Errorf("Failed to LockNode: %v", err)
			return
		}

		tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
			&inodedb.CreateNodeOp{NodeLock: nlock, OrigPath: "/hoge.txt", Type: inodedb.FileNodeT},
			&inodedb.HardLinkOp{NodeLock: inodedb.NodeLock{1, inodedb.NoTicket}, Name: "hoge.txt", TargetID: nlock.ID},
		}}
		if _, err := db.ApplyTransaction(tx); err != nil {
			t.Errorf("Failed to apply tx: %v", err)
			return
		}

		if err := db.UnlockNode(nlock); err != nil {
			t.Errorf("Failed to UnlockNode: %v", err)
			return
		}

		// create 2nd snapshot
		if err := db.Sync(); err != nil {
			t.Errorf("Failed to Sync DB (2): %v", err)
			return
		}
	}

	if _, err := inodedb.NewDB(sio, txlogio); err != nil {
		t.Errorf("Failed to NewDB (uncorrupted): %v", err)
		return
	}

	// destroy latest snapshot (corrupt data)
	ssbp, err := loc.Locate(0)
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
		_, err = inodedb.NewDB(sio, txlogio)
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
		_, err = inodedb.NewDB(sio, txlogio)
		if err != nil {
			t.Errorf("Failed to NewDB (ss blob removed): %v", err)
			return
		}
	}
}
