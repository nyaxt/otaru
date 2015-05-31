package inodedb_test

import (
	"reflect"
	"testing"

	idb "github.com/nyaxt/otaru/inodedb"
)

func testTx(id idb.TxID) idb.DBTransaction {
	return idb.DBTransaction{TxID: id, Ops: []idb.DBOperation{
		&idb.CreateNodeOp{},
	}}
}

func TestCachedDBTransactionLogIO_SingleTx(t *testing.T) {
	be := idb.NewSimpleDBTransactionLogIO()
	ctxio := idb.NewCachedDBTransactionLogIO(be)

	if err := ctxio.AppendTransaction(testTx(1)); err != nil {
		t.Errorf("AppendTransaction failed: %v", err)
	}

	txs, err := ctxio.QueryTransactions(1)
	if err != nil {
		t.Errorf("QueryTransactions failed: %v", err)
	}

	txs2, err := be.QueryTransactions(1)
	if err != nil {
		t.Errorf("be.QueryTransactions failed: %v", err)
	}

	if !reflect.DeepEqual(txs, txs2) {
		t.Errorf("mismatch %+v != %+v", txs, txs2)
	}
}

func TestCachedDBTransactionLogIO_1000Tx(t *testing.T) {
	be := idb.NewSimpleDBTransactionLogIO()
	ctxio := idb.NewCachedDBTransactionLogIO(be)

	for i := idb.TxID(1); i <= 1000; i++ {
		if err := ctxio.AppendTransaction(testTx(i)); err != nil {
			t.Errorf("AppendTransaction failed: %v", err)
		}
	}

	txs, err := ctxio.QueryTransactions(800)
	if err != nil {
		t.Errorf("QueryTransactions failed: %v", err)
	}

	txs2, err := be.QueryTransactions(800)
	if err != nil {
		t.Errorf("be.QueryTransactions failed: %v", err)
	}

	if !reflect.DeepEqual(txs, txs2) {
		t.Errorf("mismatch %+v != %+v", txs, txs2)
	}
}
