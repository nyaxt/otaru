package datastore

import (
	"errors"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"

	oflags "github.com/nyaxt/otaru/flags"
	gcutil "github.com/nyaxt/otaru/gcloud/util"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/metadata"
	"github.com/nyaxt/otaru/util"
)

var sslog = logger.Registry().Category("inodedbsslocator")

const kindINodeDBSS = "OtaruINodeDBSS"

var EEMPTY = errors.New("Failed to find any snapshot location entry.")

type INodeDBSSLocator struct {
	flags   int
	cfg     *Config
	rootKey *datastore.Key
}

func NewINodeDBSSLocator(cfg *Config, flags int) *INodeDBSSLocator {
	return &INodeDBSSLocator{
		flags:   flags,
		cfg:     cfg,
		rootKey: datastore.NameKey(kindINodeDBSS, cfg.rootKeyStr, nil),
	}
}

type sslocentry struct {
	BlobPath  string `datastore:,noindex`
	TxID      int64
	CreatedAt time.Time
}

func (loc *INodeDBSSLocator) tryLocateOnce(history int) (string, int64, error) {
	start := time.Now()
	cli, err := loc.cfg.getClient(context.TODO())
	if err != nil {
		return "", 0, err
	}
	defer cli.Close()

	dstx, err := cli.NewTransaction(context.TODO())
	if err != nil {
		return "", 0, err
	}

	q := datastore.NewQuery(kindINodeDBSS).Ancestor(loc.rootKey).Order("-TxID").Offset(history).Limit(1).Transaction(dstx)
	it := cli.Run(context.TODO(), q)
	var e sslocentry
	if _, err := it.Next(&e); err != nil {
		dstx.Rollback()
		if err == iterator.Done {
			return "", 0, EEMPTY
		}
		return "", 0, err
	}

	if _, err := dstx.Commit(); err != nil {
		return "", 0, err
	}

	logger.Infof(sslog, "LocateSnapshot(%d) took %s. Found entry: %+v", history, time.Since(start), e)
	return e.BlobPath, e.TxID, nil
}

func (loc *INodeDBSSLocator) Locate(history int) (bp string, txid int64, err error) {
	err = gcutil.RetryIfNeeded(func() error {
		bp, txid, err = loc.tryLocateOnce(history)
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
	defer cli.Close()

	dstx, err := cli.NewTransaction(context.TODO())
	if err != nil {
		return err
	}

	key := datastore.IDKey(kindINodeDBSS, int64(e.TxID), loc.rootKey)
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

func (*INodeDBSSLocator) GenerateBlobpath() string {
	return metadata.GenINodeDBSnapshotBlobpath()
}

func (loc *INodeDBSSLocator) Put(blobpath string, txid int64) error {
	if !oflags.IsWriteAllowed(loc.flags) {
		return util.EACCES
	}

	return gcutil.RetryIfNeeded(func() error {
		return loc.tryPutOnce(blobpath, txid)
	}, sslog)
}

func (loc *INodeDBSSLocator) DeleteOld(ctx context.Context, threshold int, dryRun bool) ([]string, error) {
	if !oflags.IsWriteAllowed(loc.flags) {
		return nil, util.EACCES
	}

	start := time.Now()

	cli, err := loc.cfg.getClient(context.TODO())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	blobpaths := make([]string, 0)
	ndel := 0
	for {
		needAnotherTx := false
		txStart := time.Now()
		dstx, err := cli.NewTransaction(context.TODO())
		if err != nil {
			return nil, err
		}

		keys := make([]*datastore.Key, 0)
		q := datastore.NewQuery(kindINodeDBSS).Ancestor(loc.rootKey).Order("-TxID").Offset(threshold).Transaction(dstx)
		it := cli.Run(ctx, q)
		for {
			var e sslocentry
			k, err := it.Next(&e)
			if err != nil {
				if err == iterator.Done {
					break
				}
				dstx.Rollback()
				return nil, err
			}

			keys = append(keys, k)
			blobpaths = append(blobpaths, e.BlobPath)
			if len(keys) == maxWriteEntriesPerTx {
				needAnotherTx = true
				break
			}
		}

		if !dryRun {
			if err := dstx.DeleteMulti(keys); err != nil {
				dstx.Rollback()
				return nil, err
			}
		}

		if _, err := dstx.Commit(); err != nil {
			return nil, err
		}
		ndel += len(keys)

		if needAnotherTx {
			logger.Infof(txlog, "DeleteOld(): A tx deleting %d entries took %s. Starting next tx to delete more.", len(keys), time.Since(txStart))
		} else {
			logger.Infof(txlog, "DeleteOld(): A tx deleting %d entries took %s.", len(keys), time.Since(txStart))
			break
		}
	}
	logger.Infof(sslog, "DeleteOld() deleted %d entries. Took %s", ndel, time.Since(start))
	return blobpaths, nil
}

func (loc *INodeDBSSLocator) DeleteAll(ctx context.Context, dryRun bool) ([]string, error) {
	return loc.DeleteOld(ctx, 0, dryRun)
}

func (*INodeDBSSLocator) ImplName() string { return "gcloud/datastore.INodeDBSSLocator" }
