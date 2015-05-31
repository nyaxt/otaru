package datastore

import (
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"

	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/inodedb"
)

type DBTransactionLogIO struct {
	projectName string
	rootKey     *datastore.Key
	clisrc      auth.ClientSource
}

const (
	kindTransaction = "OtaruINodeDBTx"
)

var _ = inodedb.DBTransactionLogIO(&DBTransactionLogIO{})

func NewDBTransactionLogIO(projectName, rootKeyStr string, clisrc auth.ClientSource) (*DBTransactionLogIO, error) {
	txio := &DBTransactionLogIO{
		projectName: projectName,
		clisrc:      clisrc,
	}
	ctx := txio.getContext()
	txio.rootKey = datastore.NewKey(ctx, kindTransaction, rootKeyStr, 0, nil)

	return txio, nil
}

func (txio *DBTransactionLogIO) getContext() context.Context {
	return cloud.NewContext(txio.projectName, txio.clisrc(context.TODO()))
}

type storedbtx struct {
	TxID    int64
	OpsJSON []string
}

func encode(tx inodedb.DBTransaction) (*storedbtx, error) {
	ops := make([]string, 0, len(tx.Ops))
	for _, op := range tx.Ops {
		jsonop, err := inodedb.EncodeDBOperationToJson(op)
		if err != nil {
			return nil, err
		}
		ops = append(ops, string(jsonop))
	}
	return &storedbtx{TxID: int64(tx.TxID), OpsJSON: ops}, nil
}

func decode(stx *storedbtx) (inodedb.DBTransaction, error) {
	ops := make([]inodedb.DBOperation, 0, len(stx.OpsJSON))
	for _, jsonnop := range stx.OpsJSON {
		op, err := inodedb.DecodeDBOperationFromJson([]byte(jsonnop))
		if err != nil {
			return inodedb.DBTransaction{}, err
		}
		ops = append(ops, op)
	}

	return inodedb.DBTransaction{TxID: inodedb.TxID(stx.TxID), Ops: ops}, nil
}

func (txio *DBTransactionLogIO) AppendTransaction(tx inodedb.DBTransaction) error {
	start := time.Now()
	ctx := txio.getContext()

	key := datastore.NewKey(ctx, kindTransaction, "", int64(tx.TxID), txio.rootKey)

	stx, err := encode(tx)
	if err != nil {
		return err
	}
	if _, err := datastore.Put(ctx, key, stx); err != nil {
		return err
	}
	log.Printf("AppendTransaction(%v) took %s", tx.TxID, time.Since(start))
	return nil
}

func (txio *DBTransactionLogIO) QueryTransactions(minID inodedb.TxID) ([]inodedb.DBTransaction, error) {
	start := time.Now()
	ctx := txio.getContext()

	result := []inodedb.DBTransaction{}
	q := datastore.NewQuery(kindTransaction).Ancestor(txio.rootKey).Filter("TxID >=", int64(minID)).Order("TxID")
	it := q.Run(ctx)
	for {
		var stx storedbtx
		_, err := it.Next(&stx)
		if err != nil {
			if err == datastore.Done {
				break
			}
			return []inodedb.DBTransaction{}, err
		}

		tx, err := decode(&stx)
		if err != nil {
			return []inodedb.DBTransaction{}, err
		}

		result = append(result, tx)
	}
	log.Printf("QueryTransactions(%v) took %s", minID, time.Since(start))
	return result, nil
}

func (txio *DBTransactionLogIO) DeleteTransactions(smallerThanID inodedb.TxID) error {
	start := time.Now()

	ctx := txio.getContext()

	keys := []*datastore.Key{}
	q := datastore.NewQuery(kindTransaction).Ancestor(txio.rootKey).Filter("TxID <", int64(smallerThanID)).KeysOnly()
	it := q.Run(ctx)
	for {
		k, err := it.Next(nil)
		if err != nil {
			if err == datastore.Done {
				break
			}
			return err
		}

		keys = append(keys, k)
	}

	if err := datastore.DeleteMulti(ctx, keys); err != nil {
		return err
	}

	log.Printf("DeleteTransactions(%v) took %s", smallerThanID, time.Since(start))
	return nil
}

func (txio *DBTransactionLogIO) DeleteAllTransactions() error {
	return txio.DeleteTransactions(inodedb.LatestVersion)
}
