package inodedb

import (
	"log"
)

type CachedDBTransactionLogIO struct {
	be DBTransactionLogIO

	ringbuf    []DBTransaction
	next       int
	oldestTxID TxID
}

var _ = DBTransactionLogIO(&CachedDBTransactionLogIO{})

const ringbufLen = 256

func NewCachedDBTransactionLogIO(be DBTransactionLogIO) *CachedDBTransactionLogIO {
	txio := &CachedDBTransactionLogIO{
		be:         be,
		ringbuf:    make([]DBTransaction, ringbufLen),
		next:       0,
		oldestTxID: LatestVersion,
	}

	for i, _ := range txio.ringbuf {
		txio.ringbuf[i].TxID = 0
	}

	return txio
}

func (txio *CachedDBTransactionLogIO) AppendTransaction(tx DBTransaction) error {
	if err := txio.be.AppendTransaction(tx); err != nil {
		return err
	}

	txidToBeDeleted := txio.ringbuf[txio.next].TxID
	if txidToBeDeleted == txio.oldestTxID {
		txio.oldestTxID = txidToBeDeleted + 1
	}

	txio.ringbuf[txio.next] = tx
	if txio.oldestTxID == LatestVersion {
		txio.oldestTxID = tx.TxID
	}
	txio.next++
	if txio.next == ringbufLen {
		txio.next = 0
	}
	return nil
}

func (txio *CachedDBTransactionLogIO) QueryTransactions(minID TxID) ([]DBTransaction, error) {
	if minID < txio.oldestTxID {
		log.Printf("Queried id range of \">= %d\" is not cached. Falling back to backend.", minID)
		return txio.be.QueryTransactions(minID)
	}

	return txio.QueryCachedTransactions(minID)
}

func (txio *CachedDBTransactionLogIO) QueryCachedTransactions(minID TxID) ([]DBTransaction, error) {
	result := []DBTransaction{}
	for _, tx := range txio.ringbuf[txio.next:] {
		if tx.TxID >= minID {
			result = append(result, tx)
		}
	}
	for _, tx := range txio.ringbuf[:txio.next] {
		if tx.TxID >= minID {
			result = append(result, tx)
		}
	}
	return result, nil
}
