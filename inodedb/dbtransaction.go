package inodedb

import (
	"fmt"
)

type DBTransaction struct {
	TxID `json:"txid"`
	Ops  []DBOperation `json:"ops"`
}

func (tx DBTransaction) String() string {
	opsJson, err := EncodeDBOperationsToJson(tx.Ops)
	if err != nil {
		opsJson = []byte("*ENC_ERR*")
	}

	return fmt.Sprintf("{TxID: %s, Ops: %s}", tx.TxID, string(opsJson))
}
