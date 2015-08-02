package datastore

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/cloud/datastore"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/inodedb"
)

type DBTransactionLogIO struct {
	cfg     *Config
	rootKey *datastore.Key

	mu        sync.Mutex
	nextbatch []inodedb.DBTransaction

	muSync     sync.Mutex
	committing []inodedb.DBTransaction

	cli *datastore.Client
}

const kindTransaction = "OtaruINodeDBTx"

var _ = inodedb.DBTransactionLogIO(&DBTransactionLogIO{})

func NewDBTransactionLogIO(cfg *Config) (*DBTransactionLogIO, error) {
	cli, err := cfg.getClient(context.Background())
	if err != nil {
		return nil, err
	}

	return &DBTransactionLogIO{
		cfg:       cfg,
		rootKey:   datastore.NewKey(ctxNoNamespace, kindTransaction, cfg.rootKeyStr, 0, nil),
		nextbatch: make([]inodedb.DBTransaction, 0),
		cli:       cli,
	}, nil
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

	keys := make([]*datastore.Key, 0, len(batch))
	stxs := make([]*storedbtx, 0, len(batch))
	for _, tx := range batch {
		keys = append(keys, datastore.NewKey(ctxNoNamespace, kindTransaction, "", int64(tx.TxID), txio.rootKey))
		stx, err := encode(txio.cfg.c, tx)
		if err != nil {
			rollback()
			return err
		}
		stxs = append(stxs, stx)
	}

	dstx, err := txio.cli.NewTransaction(context.Background(), datastore.Serializable)
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

type txsorter []inodedb.DBTransaction

func (s txsorter) Len() int           { return len(s) }
func (s txsorter) Less(i, j int) bool { return s[i].TxID < s[j].TxID }
func (s txsorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (txio *DBTransactionLogIO) QueryTransactions(minID inodedb.TxID) ([]inodedb.DBTransaction, error) {
	start := time.Now()
	txs := []inodedb.DBTransaction{}

	txio.mu.Lock()
	for _, tx := range txio.committing {
		if tx.TxID >= minID {
			txs = append(txs, tx)
		}
	}
	for _, tx := range txio.nextbatch {
		if tx.TxID >= minID {
			txs = append(txs, tx)
		}
	}
	txio.mu.Unlock()

	dstx, err := txio.cli.NewTransaction(context.Background(), datastore.Serializable)
	if err != nil {
		return nil, err
	}

	q := datastore.NewQuery(kindTransaction).Ancestor(txio.rootKey).Filter("TxID >=", int64(minID)).Order("TxID").Transaction(dstx)
	it := txio.cli.Run(context.Background(), q)
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

		txs = append(txs, tx)
	}

	// FIXME: not sure if Rollback() is better
	if _, err := dstx.Commit(); err != nil {
		return nil, err
	}

	sort.Sort(txsorter(txs))
	uniqed := make([]inodedb.DBTransaction, 0, len(txs))
	var prevId inodedb.TxID
	for _, tx := range txs {
		if tx.TxID == prevId {
			continue
		}

		uniqed = append(uniqed, tx)
		prevId = tx.TxID
	}

	log.Printf("QueryTransactions(%v) took %s", minID, time.Since(start))
	return uniqed, nil
}

const maxWriteEntriesPerTx = 500 // Google Cloud Datastore limit on number of write entries per tx

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

	ndel := 0
	for {
		needAnotherTx := false
		txStart := time.Now()
		dstx, err := txio.cli.NewTransaction(context.Background(), datastore.Serializable)
		if err != nil {
			return err
		}

		keys := []*datastore.Key{}
		q := datastore.NewQuery(kindTransaction).Ancestor(txio.rootKey).Filter("TxID <", int64(smallerThanID)).KeysOnly().Transaction(dstx)
		it := txio.cli.Run(context.Background(), q)
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
			if len(keys) == maxWriteEntriesPerTx {
				needAnotherTx = true
				break
			}
		}

		//log.Printf("keys to delete: %v", keys)
		if err := dstx.DeleteMulti(keys); err != nil {
			dstx.Rollback()
			return err
		}

		if _, err := dstx.Commit(); err != nil {
			return err
		}
		ndel += len(keys)

		if needAnotherTx {
			log.Printf("DeleteTransactions(%v): A tx deleting %d entries took %s. Starting next tx to delete more.", smallerThanID, len(keys), time.Since(txStart))
		} else {
			break
		}
	}
	log.Printf("DeleteTransactions(%v) deleted %d entries. tx took %s", smallerThanID, ndel, time.Since(start))
	return nil
}

func (txio *DBTransactionLogIO) DeleteAllTransactions() error {
	return txio.DeleteTransactions(inodedb.LatestVersion)
}
