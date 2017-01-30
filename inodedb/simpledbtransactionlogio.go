package inodedb

import (
	"github.com/nyaxt/otaru/util"
)

type SimpleDBTransactionLogIO struct {
	readOnly bool
	txs      []DBTransaction
}

var _ = DBTransactionLogIO(&SimpleDBTransactionLogIO{})

func NewSimpleDBTransactionLogIO() *SimpleDBTransactionLogIO {
	return &SimpleDBTransactionLogIO{readOnly: false}
}

func (io *SimpleDBTransactionLogIO) SetReadOnly(b bool) {
	io.readOnly = b
}

func (io *SimpleDBTransactionLogIO) AppendTransaction(tx DBTransaction) error {
	if io.readOnly {
		return util.EACCES
	}
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
