package minodedb

import (
	"fmt"
	"net/http"

	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/mgmt"
)

func Install(srv *mgmt.Server, h inodedb.DBHandler) {
	rtr := srv.APIRouter().PathPrefix("/inodedb").Subrouter()

	rtr.HandleFunc("/stats", mgmt.JSONHandler(func(req *http.Request) interface{} {
		prov, ok := h.(inodedb.DBServiceStatsProvider)
		if !ok {
			return fmt.Errorf("Active inodedb doesn't support /stats")
		}
		return prov.GetStats()
	}))
	rtr.HandleFunc("/recenttxs", mgmt.JSONHandler(func(req *http.Request) interface{} {
		prov, ok := h.(inodedb.QueryRecentTransactionsProvider)
		if !ok {
			return fmt.Errorf("Active inodedb doesn't support /recenttxs")
		}
		txs, err := prov.QueryRecentTransactions()
		if err != nil {
			return fmt.Errorf("QueryRecentTransactions failed: %v", err)
		}
		for _, tx := range txs {
			if err := inodedb.SetOpMetas(tx.Ops); err != nil {
				return fmt.Errorf("SetOpMetas failed: %v", err)
			}
		}

		return txs
	}))
}
