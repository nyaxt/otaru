package mscheduler

import (
	"net/http"

	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/scheduler"
)

func Install(srv *mgmt.Server, s *scheduler.Scheduler) {
	rtr := srv.APIRouter().PathPrefix("/scheduler").Subrouter()

	rtr.HandleFunc("/stats", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return s.GetStats()
	}))
}
