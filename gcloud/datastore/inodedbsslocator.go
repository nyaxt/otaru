package datastore

/*

import (
	"errors"
	"log"
	"time"

	"google.golang.org/cloud/datastore"
)

const kindINodeDBSS = "OtaruINodeDBSS"

var EEMPTY = errors.New("Failed to find any snapshot location entry.")

type INodeDBSSLocator struct {
	cfg     *Config
	rootKey *datastore.Key
}

func NewINodeDBSSLocator(cfg *Config) *INodeDBSSLocator {
	return &INodeDBSSLocator{
		cfg:     cfg,
		rootKey: datastore.NewKey(cfg.getContext(), kindINodeDBSS, cfg.rootKeyStr, 0, nil),
	}
}

type sslocentry struct {
	BlobPath  string `datastore:,noindex`
	TxID      int64
	CreatedAt time.Time
}

func (loc *INodeDBSSLocator) Locate(history int) (string, error) {
	start := time.Now()
	ctx := loc.cfg.getContext()
	dstx, err := datastore.NewTransaction(ctx, datastore.Serializable)
	if err != nil {
		return "", err
	}
	defer dstx.Commit()

	q := datastore.NewQuery(kindINodeDBSS).Ancestor(loc.rootKey).Order("-TxID").Offset(history).Limit(1).Transaction(dstx)
	it := q.Run(ctx)
	var e sslocentry
	if _, err := it.Next(&e); err != nil {
		if err == datastore.Done {
			return "", EEMPTY
		}
		return "", err
	}

	log.Printf("LocateSnapshot(%d) took %s. Found entry: %+v", history, time.Since(start), e)
	return e.BlobPath, nil
}

func (loc *INodeDBSSLocator) Put(blobpath string, txid int64) error {
	start := time.Now()
	e := sslocentry{BlobPath: blobpath, TxID: txid, CreatedAt: start}

	ctx := loc.cfg.getContext()
	dstx, err := datastore.NewTransaction(ctx, datastore.Serializable)
	if err != nil {
		return err
	}

	key := datastore.NewKey(ctx, kindINodeDBSS, "", int64(e.TxID), loc.rootKey)
	if _, err := dstx.Put(key, &e); err != nil {
		dstx.Rollback()
		return err
	}
	if _, err := dstx.Commit(); err != nil {
		return err
	}

	log.Printf("Put(%s, %d) took %s.", blobpath, txid, time.Since(start))
	return nil
}

func (loc *INodeDBSSLocator) DeleteAll() ([]string, error) {
	start := time.Now()

	ctx := loc.cfg.getContext()

	dstx, err := datastore.NewTransaction(ctx, datastore.Serializable)
	if err != nil {
		return nil, err
	}

	keys := make([]*datastore.Key, 0)
	blobpaths := make([]string, 0)
	q := datastore.NewQuery(kindINodeDBSS).Ancestor(loc.rootKey).Transaction(dstx)
	it := q.Run(ctx)
	for {
		var e sslocentry
		k, err := it.Next(&e)
		if err != nil {
			if err == datastore.Done {
				break
			}
			dstx.Rollback()
			return nil, err
		}

		keys = append(keys, k)
		blobpaths = append(blobpaths, e.BlobPath)
	}

	log.Printf("keys to delete: %v", keys)
	if err := dstx.DeleteMulti(keys); err != nil {
		dstx.Rollback()
		return nil, err
	}

	if _, err := dstx.Commit(); err != nil {
		return nil, err
	}
	log.Printf("DeleteAll() deleted %d entries. Took %s", len(keys), time.Since(start))
	return blobpaths, nil
}
*/
