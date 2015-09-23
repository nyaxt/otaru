package mdebug

import (
	"net/http"
	"net/http/pprof"

	"github.com/nyaxt/otaru/mgmt"
)

func Install(srv *mgmt.Server) {
	rtr := srv.APIRouter().PathPrefix("/debug/pprof").Subrouter()

	rtr.HandleFunc("/", http.HandlerFunc(pprof.Index))
	rtr.HandleFunc("/cmdline", http.HandlerFunc(pprof.Cmdline))
	rtr.HandleFunc("/profile", http.HandlerFunc(pprof.Profile))
	rtr.HandleFunc("/symbol", http.HandlerFunc(pprof.Symbol))
	rtr.HandleFunc("/trace", http.HandlerFunc(pprof.Trace))
}
