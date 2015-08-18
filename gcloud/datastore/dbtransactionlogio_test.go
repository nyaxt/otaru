package datastore_test

import (
	"reflect"
	"testing"
	"time"

	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/testutils"
)

func init() { testutils.EnsureLogger() }

func testDBTransactionIOWithRootKey(rootKeyStr string) *datastore.DBTransactionLogIO {
	return datastore.NewDBTransactionLogIO(authtu.TestDSConfig(rootKeyStr))
}

func testDBTransactionIO() *datastore.DBTransactionLogIO {
	return testDBTransactionIOWithRootKey(authtu.TestBucketName())
}

var stableT = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

func TestDBTransactionIO_PutQuery(t *testing.T) {
	txio := testDBTransactionIO()

	if err := txio.DeleteAllTransactions(); err != nil {
		t.Errorf("DeleteTransactions failed: %v", err)
	}

	tx := inodedb.DBTransaction{TxID: 123, Ops: []inodedb.DBOperation{
		&inodedb.CreateNodeOp{NodeLock: inodedb.NodeLock{2, 123456}, OrigPath: "/hoge.txt", Type: inodedb.FileNodeT, ModifiedT: stableT},
		&inodedb.HardLinkOp{NodeLock: inodedb.NodeLock{1, inodedb.NoTicket}, Name: "hoge.txt", TargetID: 2},
	}}

	if err := txio.AppendTransaction(tx); err != nil {
		t.Errorf("AppendTransaction failed: %v", err)
		return
	}

	// query before sync
	{
		txs, err := txio.QueryTransactions(123)
		if err != nil {
			t.Errorf("QueryTransactions failed: %v", err)
			return
		}
		if len(txs) != 1 {
			t.Errorf("QueryTransactions >=123 result invalid len: %+v", txs)
			return
		}

		if !reflect.DeepEqual(txs[0], tx) {
			t.Errorf("serdes mismatch:\nExpected: %+v\nGot     : %+v", txs[0], tx)
		}
	}

	// query after sync
	if err := txio.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}
	{
		txs, err := txio.QueryTransactions(123)
		if err != nil {
			t.Errorf("QueryTransactions failed: %v", err)
			return
		}
		if len(txs) != 1 {
			t.Errorf("QueryTransactions >=123 result invalid len: %+v", txs)
			return
		}

		if !reflect.DeepEqual(txs[0], tx) {
			t.Errorf("serdes mismatch:\nExpected: %+v\nGot     : %+v", txs[0], tx)
		}
	}

	{
		txs, err := txio.QueryTransactions(124)
		if err != nil {
			t.Errorf("QueryTransactions failed: %v", err)
		}
		if len(txs) != 0 {
			t.Errorf("QueryTransactions >=124 should be noent but got: %+v", txs)
		}
	}
}

func TestDBTransactionIO_DeleteAll_IsIsolated(t *testing.T) {
	key1 := authtu.TestBucketName()
	key2 := authtu.TestBucketName() + "-2"

	txio := testDBTransactionIOWithRootKey(key1)
	txio2 := testDBTransactionIOWithRootKey(key2)

	if err := txio.DeleteAllTransactions(); err != nil {
		t.Errorf("DeleteTransactions failed: %v", err)
	}
	if err := txio2.DeleteAllTransactions(); err != nil {
		t.Errorf("DeleteTransactions failed: %v", err)
	}

	tx := inodedb.DBTransaction{TxID: 100, Ops: []inodedb.DBOperation{
		&inodedb.CreateNodeOp{NodeLock: inodedb.NodeLock{2, 123456}, OrigPath: "/hoge.txt", Type: inodedb.FileNodeT},
		&inodedb.HardLinkOp{NodeLock: inodedb.NodeLock{1, inodedb.NoTicket}, Name: "hoge.txt", TargetID: 2},
	}}
	if err := txio.AppendTransaction(tx); err != nil {
		t.Errorf("AppendTransaction failed: %v", err)
	}
	tx2 := inodedb.DBTransaction{TxID: 200, Ops: []inodedb.DBOperation{
		&inodedb.CreateNodeOp{NodeLock: inodedb.NodeLock{2, 123456}, OrigPath: "/fuga.txt", Type: inodedb.FileNodeT},
		&inodedb.HardLinkOp{NodeLock: inodedb.NodeLock{1, inodedb.NoTicket}, Name: "fuga.txt", TargetID: 2},
	}}
	if err := txio2.AppendTransaction(tx2); err != nil {
		t.Errorf("AppendTransaction failed: %v", err)
	}

	if err := txio.DeleteAllTransactions(); err != nil {
		t.Errorf("DeleteTransactions failed: %v", err)
	}

	{
		txs, err := txio.QueryTransactions(inodedb.AnyVersion)
		if err != nil {
			t.Errorf("QueryTransactions failed: %v", err)
			return
		}
		if len(txs) != 0 {
			t.Errorf("Tx queried after DeleteAllTransactions")
			return
		}
	}
	{
		txs, err := txio2.QueryTransactions(inodedb.AnyVersion)
		if err != nil {
			t.Errorf("QueryTransactions failed: %v", err)
			return
		}
		if len(txs) != 1 {
			t.Errorf("DeleteAllTransactions deleted txlog on separate rootKey")
			return
		}

		if !reflect.DeepEqual(txs[0], tx2) {
			t.Errorf("serdes mismatch: %+v", txs)
		}
	}
}

func TestDBTransactionIO_BigTx(t *testing.T) {
	txio := testDBTransactionIO()

	if err := txio.DeleteAllTransactions(); err != nil {
		t.Errorf("DeleteTransactions failed: %v", err)
	}

	ops := make([]inodedb.DBOperation, 0, 300)
	for i := 0; i < 300; i++ {
		ops = append(ops, &inodedb.CreateNodeOp{NodeLock: inodedb.NodeLock{inodedb.ID(i), inodedb.Ticket(1000 + i)}, OrigPath: "/hoge.txt", Type: inodedb.FileNodeT, ModifiedT: stableT})
	}
	tx := inodedb.DBTransaction{TxID: 567, Ops: ops}

	if err := txio.AppendTransaction(tx); err != nil {
		t.Errorf("AppendTransaction failed: %v", err)
		return
	}
	if err := txio.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}
