package datastore

import (
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/cloud/datastore"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/util"
)

type DBTransactionLogIO struct {
	cfg     *Config
	rootKey *datastore.Key

	mu        sync.Mutex
	nextbatch []inodedb.DBTransaction

	muSync     sync.Mutex
	committing []inodedb.DBTransaction
}

const kindTransaction = "OtaruINodeDBTx"

var _ = inodedb.DBTransactionLogIO(&DBTransactionLogIO{})

func NewDBTransactionLogIO(cfg *Config) *DBTransactionLogIO {
	return &DBTransactionLogIO{
		cfg:       cfg,
		rootKey:   datastore.NewKey(cfg.getContext(), kindTransaction, cfg.rootKeyStr, 0, nil),
		nextbatch: make([]inodedb.DBTransaction, 0),
	}
}

type storedbtx struct {
	TxID    int64
	OpsJSON []byte `datastore:,noindex`
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
	txio.mu.Lock()
	defer txio.mu.Unlock()

	txio.nextbatch = append(txio.nextbatch, tx)
	return nil
}

func (txio *DBTransactionLogIO) Sync() error {
	start := time.Now()

	txio.muSync.Lock()
	defer txio.muSync.Unlock()

	txio.mu.Lock()
	if len(txio.committing) != 0 {
		panic("I should be the only one committing.")
	}
	txio.committing = txio.nextbatch
	batch := txio.committing
	txio.nextbatch = make([]inodedb.DBTransaction, 0)
	txio.mu.Unlock()
	rollback := func() {
		txio.mu.Lock()
		txio.nextbatch = append(txio.committing, txio.nextbatch...)
		txio.committing = []inodedb.DBTransaction{}
		txio.mu.Unlock()
	}

	if len(batch) == 0 {
		return nil
	}

	ctx := txio.cfg.getContext()
	keys := make([]*datastore.Key, 0, len(batch))
	stxs := make([]*storedbtx, 0, len(batch))
	for _, tx := range batch {
		keys = append(keys, datastore.NewKey(ctx, kindTransaction, "", int64(tx.TxID), txio.rootKey))
		stx, err := encode(txio.cfg.c, tx)
		if err != nil {
			rollback()
			return err
		}
		stxs = append(stxs, stx)
	}

	dstx, err := datastore.NewTransaction(ctx, datastore.Serializable)
	if err != nil {
		rollback()
		return err
	}

	if _, err := dstx.PutMulti(keys, stxs); err != nil {
		rollback()
		dstx.Rollback()
		return err
	}

	if _, err := dstx.Commit(); err != nil {
		rollback()
		return err
	}

	txio.mu.Lock()
	txio.committing = []inodedb.DBTransaction{}
	txio.mu.Unlock()

	log.Printf("Sync() took %s. Committed %d txs", time.Since(start), len(stxs))
	return nil
}

func (txio *DBTransactionLogIO) QueryTransactions(minID inodedb.TxID) ([]inodedb.DBTransaction, error) {
	start := time.Now()
	result := []inodedb.DBTransaction{}

	txio.mu.Lock()
	for _, tx := range txio.committing {
		if tx.TxID >= minID {
			result = append(result, tx)
		}
	}
	for _, tx := range txio.nextbatch {
		if tx.TxID >= minID {
			result = append(result, tx)
		}
	}
	txio.mu.Unlock()

	ctx := txio.cfg.getContext()

	dstx, err := datastore.NewTransaction(ctx, datastore.Serializable)
	if err != nil {
		return nil, err
	}

	q := datastore.NewQuery(kindTransaction).Ancestor(txio.rootKey).Filter("TxID >=", int64(minID)).Order("TxID").Transaction(dstx)
	it := q.Run(ctx)
	for {
		var stx storedbtx
		_, err := it.Next(&stx)
		if err != nil {
			if err == datastore.Done {
				break
			}
			dstx.Commit()
			return []inodedb.DBTransaction{}, err
		}

		tx, err := decode(txio.cfg.c, &stx)
		if err != nil {
			dstx.Commit()
			return []inodedb.DBTransaction{}, err
		}

		result = append(result, tx)
	}

	// FIXME: not sure if Rollback() is better
	if _, err := dstx.Commit(); err != nil {
		return nil, err
	}
	log.Printf("QueryTransactions(%v) took %s", minID, time.Since(start))
	return result, nil
}

func (txio *DBTransactionLogIO) DeleteTransactions(smallerThanID inodedb.TxID) error {
	start := time.Now()

	txio.mu.Lock()
	batch := make([]inodedb.DBTransaction, 0, len(txio.nextbatch))
	for _, tx := range txio.nextbatch {
		if tx.TxID < smallerThanID {
			continue
		}
		batch = append(batch, tx)
	}
	txio.nextbatch = batch
	txio.mu.Unlock()

	ctx := txio.cfg.getContext()

	dstx, err := datastore.NewTransaction(ctx, datastore.Serializable)
	if err != nil {
		return err
	}

	keys := []*datastore.Key{}
	q := datastore.NewQuery(kindTransaction).Ancestor(txio.rootKey).Filter("TxID <", int64(smallerThanID)).KeysOnly().Transaction(dstx)
	it := q.Run(ctx)
	for {
		k, err := it.Next(nil)
		if err != nil {
			if err == datastore.Done {
				break
			}
			dstx.Rollback()
			return err
		}

		keys = append(keys, k)
	}

	//log.Printf("keys to delete: %v", keys)
	if err := dstx.DeleteMulti(keys); err != nil {
		dstx.Rollback()
		return err
	}

	if _, err := dstx.Commit(); err != nil {
		return err
	}
	log.Printf("DeleteTransactions(%v) deleted %d entries. took %s", smallerThanID, len(keys), time.Since(start))
	return nil
}

func (txio *DBTransactionLogIO) DeleteAllTransactions() error {
	return txio.DeleteTransactions(inodedb.LatestVersion)
}
