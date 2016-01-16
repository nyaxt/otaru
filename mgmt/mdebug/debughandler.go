package mdebug

import (
	"net/http"
	"net/http/pprof"
	"runtime"

	"github.com/gorilla/mux"

	"github.com/nyaxt/otaru/mgmt"
)

func Install(srv *mgmt.Server) {
	rtr := srv.APIRouter().PathPrefix("/debug/pprof").Subrouter()

	rtr.HandleFunc("/", http.HandlerFunc(pprof.Index))
	rtr.HandleFunc("/set_block_profile_rate", func(w http.ResponseWriter, req *http.Request) {
		runtime.SetBlockProfileRate(1)

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok"))
	})
	rtr.HandleFunc("/cmdline", http.HandlerFunc(pprof.Cmdline))
	rtr.HandleFunc("/profile", http.HandlerFunc(pprof.Profile))
	rtr.HandleFunc("/symbol", http.HandlerFunc(pprof.Symbol))
	rtr.HandleFunc("/trace", http.HandlerFunc(pprof.Trace))
	rtr.HandleFunc("/{cmd}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		cmd := vars["cmd"]
		pprof.Handler(cmd).ServeHTTP(w, r)
	}))
}
