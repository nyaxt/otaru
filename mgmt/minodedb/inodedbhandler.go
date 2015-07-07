package minodedb

import (
	"fmt"
	"strconv"

	"net/http"

	"github.com/gorilla/mux"

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
	rtr.HandleFunc("/inode/{id:[0-9]+}", mgmt.JSONHandler(func(req *http.Request) interface{} {
		vars := mux.Vars(req)
		nid, err := strconv.ParseUint(vars["id"], 10, 32)
		if err != nil {
			return err
		}
		id := inodedb.ID(nid)
		v, _, err := h.QueryNode(id, false)
		if err != nil {
			return err
		}

		return v
	}))
}
