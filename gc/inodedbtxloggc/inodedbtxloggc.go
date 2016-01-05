package inodedbtxloggc

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
)

type UnneededTxIDThresholdFinder interface {
	FindUnneededTxIDThreshold() (int64, error)
}

type TransactionLogDeleter interface {
	DeleteTransactions(smallerThanID inodedb.TxID) error
}

var mylog = logger.Registry().Category("inodedbtxloggc")

func GC(ctx context.Context, thresfinder UnneededTxIDThresholdFinder, logdeleter TransactionLogDeleter, dryrun bool) error {
	start := time.Now()

	logger.Infof(mylog, "GC start. Dryrun: %t. Trying to find UnneededTxIDThreshold.", dryrun)

	ntxid, err := thresfinder.FindUnneededTxIDThreshold()
	if err != nil {
		return fmt.Errorf("Failed to find UnneededTxIDThreshold: %v", err)
	}
	txid := inodedb.TxID(ntxid)
	if txid == inodedb.AnyVersion {
		return fmt.Errorf("UnneededTxIDThreshold was AnyVersion. No TxID log to be deleted")
	}
	logger.Infof(mylog, "Found UnneededTxIDThreshold: %v", txid)

	if err := ctx.Err(); err != nil {
		logger.Infof(mylog, "Detected cancel. Bailing out.")
		return err
	}

	if dryrun {
		logger.Infof(mylog, "Dry run. Not actually deleting txlog.")
	} else {
		if err := logdeleter.DeleteTransactions(txid); err != nil {
			return err
		}
	}
	logger.Infof(mylog, "GC success. Dryrun: %t. The whole GC took %v.", dryrun, time.Since(start))
	return nil
}
