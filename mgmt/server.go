package mgmt

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	rtr     *mux.Router
	apirtr  *mux.Router
	httpsrv *http.Server
}

func NewServer() *Server {
	rtr := mux.NewRouter()

	rtr.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})

	apirtr := rtr.PathPrefix("/api").Subrouter()

	// FIXME: Migrate to github.com/elazarl/go-bindata-assetfs
	rtr.Handle("/", http.FileServer(http.Dir("../www")))

	httpsrv := &http.Server{
		Addr:    ":10246",
		Handler: rtr,
	}
	return &Server{rtr: rtr, apirtr: apirtr, httpsrv: httpsrv}
}

func (srv *Server) APIRouter() *mux.Router { return srv.apirtr }

func (srv *Server) Run() error {
	if err := srv.httpsrv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
