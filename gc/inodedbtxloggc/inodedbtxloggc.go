package inodedbtxloggc

import (
	"fmt"
	"sync/atomic"
	"time"

	"context"

	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
)

type UnneededTxIDThresholdFinder interface {
	FindUnneededTxIDThreshold() (inodedb.TxID, error)
}

type TransactionLogDeleter interface {
	DeleteTransactions(smallerThanID inodedb.TxID) error
}

var mylog = logger.Registry().Category("inodedbtxloggc")

var gcRunning uint32

func GC(ctx context.Context, thresfinder UnneededTxIDThresholdFinder, logdeleter TransactionLogDeleter, dryrun bool) error {
	start := time.Now()

	if !atomic.CompareAndSwapUint32(&gcRunning, 0, 1) {
		return fmt.Errorf("Another inodedbtxloggc is already running.")
	}
	defer atomic.StoreUint32(&gcRunning, 0)

	logger.Infof(mylog, "GC start. Dryrun: %t. Trying to find UnneededTxIDThreshold.", dryrun)

	txid, err := thresfinder.FindUnneededTxIDThreshold()
	if err != nil {
		return fmt.Errorf("Failed to find UnneededTxIDThreshold: %v", err)
	}
	if txid == inodedb.AnyVersion {
		logger.Infof(mylog, "UnneededTxIDThreshold was AnyVersion. No TxID log to be deleted")
		return nil
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
