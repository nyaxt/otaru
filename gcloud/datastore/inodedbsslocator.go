package datastore

import (
	"errors"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/cloud/datastore"

	gcutil "github.com/nyaxt/otaru/gcloud/util"
	"github.com/nyaxt/otaru/logger"
)

var sslog = logger.Registry().Category("inodedbsslocator")

const kindINodeDBSS = "OtaruINodeDBSS"

var EEMPTY = errors.New("Failed to find any snapshot location entry.")

type INodeDBSSLocator struct {
	cfg     *Config
	rootKey *datastore.Key
}

func NewINodeDBSSLocator(cfg *Config) *INodeDBSSLocator {
	return &INodeDBSSLocator{
		cfg:     cfg,
		rootKey: datastore.NewKey(ctxNoNamespace, kindINodeDBSS, cfg.rootKeyStr, 0, nil),
	}
}

type sslocentry struct {
	BlobPath  string `datastore:,noindex`
	TxID      int64
	CreatedAt time.Time
}

func (loc *INodeDBSSLocator) tryLocateOnce(history int) (string, error) {
	start := time.Now()
	cli, err := loc.cfg.getClient(context.TODO())
	if err != nil {
		return "", err
	}
	dstx, err := cli.NewTransaction(context.TODO(), datastore.Serializable)
	if err != nil {
		return "", err
	}

	q := datastore.NewQuery(kindINodeDBSS).Ancestor(loc.rootKey).Order("-TxID").Offset(history).Limit(1).Transaction(dstx)
	it := cli.Run(context.TODO(), q)
	var e sslocentry
	if _, err := it.Next(&e); err != nil {
		dstx.Rollback()
		if err == datastore.Done {
			return "", EEMPTY
		}
		return "", err
	}

	if _, err := dstx.Commit(); err != nil {
		return "", err
	}

	logger.Infof(sslog, "LocateSnapshot(%d) took %s. Found entry: %+v", history, time.Since(start), e)
	return e.BlobPath, nil
}

func (loc *INodeDBSSLocator) Locate(history int) (bp string, err error) {
	err = gcutil.RetryIfNeeded(func() error {
		bp, err = loc.tryLocateOnce(history)
		return err
	}, sslog)
	return
}

func (loc *INodeDBSSLocator) tryPutOnce(blobpath string, txid int64) error {
	start := time.Now()
	e := sslocentry{BlobPath: blobpath, TxID: txid, CreatedAt: start}

	cli, err := loc.cfg.getClient(context.TODO())
	if err != nil {
		return err
	}
	dstx, err := cli.NewTransaction(context.TODO(), datastore.Serializable)
	if err != nil {
		return err
	}

	key := datastore.NewKey(ctxNoNamespace, kindINodeDBSS, "", int64(e.TxID), loc.rootKey)
	if _, err := dstx.Put(key, &e); err != nil {
		dstx.Rollback()
		return err
	}
	if _, err := dstx.Commit(); err != nil {
		return err
	}

	logger.Infof(sslog, "Put(%s, %d) took %s.", blobpath, txid, time.Since(start))
	return nil
}

func (loc *INodeDBSSLocator) Put(blobpath string, txid int64) error {
	return gcutil.RetryIfNeeded(func() error {
		return loc.tryPutOnce(blobpath, txid)
	}, sslog)
}

func (loc *INodeDBSSLocator) DeleteAll() ([]string, error) {
	start := time.Now()

	cli, err := loc.cfg.getClient(context.TODO())
	if err != nil {
		return nil, err
	}

	dstx, err := cli.NewTransaction(context.TODO(), datastore.Serializable)
	if err != nil {
		return nil, err
	}

	keys := make([]*datastore.Key, 0)
	blobpaths := make([]string, 0)
	q := datastore.NewQuery(kindINodeDBSS).Ancestor(loc.rootKey).Transaction(dstx)
	it := cli.Run(context.TODO(), q)
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

	logger.Debugf(sslog, "keys to delete: %v", keys)
	if err := dstx.DeleteMulti(keys); err != nil {
		dstx.Rollback()
		return nil, err
	}

	if _, err := dstx.Commit(); err != nil {
		return nil, err
	}
	logger.Infof(sslog, "DeleteAll() deleted %d entries. Took %s", len(keys), time.Since(start))
	return blobpaths, nil
}
