package mgmt

import (
	"net/http"
)

type Server struct {
	mux     *http.ServeMux
	httpsrv *http.Server
}

func NewServer() *Server {
	mux := http.NewServeMux()
	httpsrv := &http.Server{
		Addr:    ":10246",
		Handler: mux,
	}

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})

	// FIXME: Migrate to github.com/elazarl/go-bindata-assetfs
	mux.Handle("/", http.FileServer(http.Dir("../www")))

	return &Server{mux: mux, httpsrv: httpsrv}
}

func (srv *Server) Run() error {
	if err := srv.httpsrv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
