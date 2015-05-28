package inodedb

import ()

type SimpleDBTransactionLogIO struct {
	txs []DBTransaction
}

var _ = DBTransactionLogIO(&SimpleDBTransactionLogIO{})

func NewSimpleDBTransactionLogIO() *SimpleDBTransactionLogIO {
	return &SimpleDBTransactionLogIO{}
}

func (io *SimpleDBTransactionLogIO) AppendTransaction(tx DBTransaction) error {
	io.txs = append(io.txs, tx)
	return nil
}

func (io *SimpleDBTransactionLogIO) QueryTransactions(minID TxID) ([]DBTransaction, error) {
	result := []DBTransaction{}
	for _, tx := range io.txs {
		if tx.TxID >= minID {
			result = append(result, tx)
		}
	}
	return result, nil
}
