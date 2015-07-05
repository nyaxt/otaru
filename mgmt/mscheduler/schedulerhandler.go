package mscheduler

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/scheduler"
)

func Install(srv *mgmt.Server, s *scheduler.Scheduler) {
	rtr := srv.APIRouter().PathPrefix("/scheduler").Subrouter()

	rtr.HandleFunc("/stats", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return s.GetStats()
	}))
	rtr.HandleFunc("/all", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return s.QueryAll()
	}))
	rtr.HandleFunc("/{id:[0-9]+}", mgmt.JSONHandler(func(req *http.Request) interface{} {
		vars := mux.Vars(req)
		nid, err := strconv.ParseUint(vars["id"], 10, 32)
		if err != nil {
			return err
		}
		return s.Query(scheduler.ID(nid))
	}))
}
