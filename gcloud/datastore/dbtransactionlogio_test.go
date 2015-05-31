package datastore_test

import (
	"log"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/inodedb"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
)

func testClientSource() auth.ClientSource {
	homedir := os.Getenv("HOME")
	clisrc, err := auth.GetGCloudClientSource(
		path.Join(homedir, ".otaru", "credentials.json"),
		path.Join(homedir, ".otaru", "tokencache.json"),
		false)
	if err != nil {
		log.Fatalf("Failed to create testClientSource: %v", err)
	}
	return clisrc
}

func testDBTransactionIO() *datastore.DBTransactionLogIO {
	homedir := os.Getenv("HOME")
	projectName := util.StringFromFileOrDie(path.Join(homedir, ".otaru", "projectname.txt"))
	bs, err := datastore.NewDBTransactionLogIO(
		projectName,
		"otaru-test",
		tu.TestCipher(),
		testClientSource(),
	)
	if err != nil {
		log.Fatalf("Failed to create DBTransactionLogIO: %v", err)
	}
	return bs
}

func TestDBTransactionIO_PutQuery(t *testing.T) {
	txio := testDBTransactionIO()

	if err := txio.DeleteAllTransactions(); err != nil {
		t.Errorf("DeleteTransactions failed: %v", err)
	}

	tx := inodedb.DBTransaction{TxID: 123, Ops: []inodedb.DBOperation{
		&inodedb.CreateNodeOp{NodeLock: inodedb.NodeLock{2, 123456}, OrigPath: "/hoge.txt", Type: inodedb.FileNodeT},
		&inodedb.HardLinkOp{NodeLock: inodedb.NodeLock{1, inodedb.NoTicket}, Name: "hoge.txt", TargetID: 2},
	}}

	if err := txio.AppendTransaction(tx); err != nil {
		t.Errorf("AppendTransaction failed: %v", err)
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
			t.Errorf("serdes mismatch: %+v", txs)
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
