package mscheduler

import (
	"strconv"

	"net/http"

	"github.com/gorilla/mux"

	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/scheduler"
)

func Install(srv *mgmt.Server, s *scheduler.Scheduler, r *scheduler.RepetitiveJobRunner) {
	rtr := srv.APIRouter().PathPrefix("/scheduler").Subrouter()

	rtr.HandleFunc("/stats", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return s.GetStats()
	}))
	rtr.HandleFunc("/job/all", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return s.QueryAll()
	}))
	rtr.HandleFunc("/job/{id:[0-9]+}", mgmt.JSONHandler(func(req *http.Request) interface{} {
		vars := mux.Vars(req)
		nid, err := strconv.ParseUint(vars["id"], 10, 32)
		if err != nil {
			return err
		}
		return s.Query(scheduler.ID(nid))
	}))

	if r != nil {
		rtr.HandleFunc("/rep/all", mgmt.JSONHandler(func(req *http.Request) interface{} {
			return r.QueryAll()
		}))
		rtr.HandleFunc("/rep/{id:[0-9]+}", mgmt.JSONHandler(func(req *http.Request) interface{} {
			vars := mux.Vars(req)
			nid, err := strconv.ParseUint(vars["id"], 10, 32)
			if err != nil {
				return err
			}
			return r.Query(scheduler.ID(nid))
		}))
	}
}
