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
}
