package datastore

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/inodedb"
)

type DBTransactionLogIO struct {
	projectName string
	rootKey     *datastore.Key
	c           btncrypt.Cipher
	clisrc      auth.ClientSource
}

const (
	kindTransaction = "OtaruINodeDBTx"
)

var _ = inodedb.DBTransactionLogIO(&DBTransactionLogIO{})

func NewDBTransactionLogIO(projectName, rootKeyStr string, c btncrypt.Cipher, clisrc auth.ClientSource) (*DBTransactionLogIO, error) {
	txio := &DBTransactionLogIO{
		projectName: projectName,
		c:           c,
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
	OpsJSON []byte
}

func encode(c btncrypt.Cipher, tx inodedb.DBTransaction) (*storedbtx, error) {
	jsonops, err := inodedb.EncodeDBOperationsToJson(tx.Ops)
	if err != nil {
		return nil, fmt.Errorf("Failed to encode dbtx: %v", err)
	}

	env, err := btncrypt.Encrypt(c, jsonops)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt OpsJSON: %v", err)
	}

	return &storedbtx{TxID: int64(tx.TxID), OpsJSON: env}, nil
}

func decode(c btncrypt.Cipher, stx *storedbtx) (inodedb.DBTransaction, error) {
	jsonop, err := btncrypt.Decrypt(c, stx.OpsJSON, len(stx.OpsJSON)-c.FrameOverhead())
	if err != nil {
		return inodedb.DBTransaction{}, fmt.Errorf("Failed to decrypt OpsJSON: %v", err)
	}

	ops, err := inodedb.DecodeDBOperationsFromJson(jsonop)
	if err != nil {
		return inodedb.DBTransaction{}, err
	}

	return inodedb.DBTransaction{TxID: inodedb.TxID(stx.TxID), Ops: ops}, nil
}

func (txio *DBTransactionLogIO) AppendTransaction(tx inodedb.DBTransaction) error {
	start := time.Now()
	ctx := txio.getContext()

	key := datastore.NewKey(ctx, kindTransaction, "", int64(tx.TxID), txio.rootKey)

	stx, err := encode(txio.c, tx)
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

		tx, err := decode(txio.c, &stx)
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
