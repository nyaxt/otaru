package mgmt

import (
	"net/http"
	"os"

	"github.com/nyaxt/otaru/webui"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	rtr     *mux.Router
	apirtr  *mux.Router
	httpsrv *http.Server
}

func NewServer(addr string) *Server {
	rtr := mux.NewRouter()

	rtr.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("OK"))
	})
	assetDirWrap := func(path string) ([]string, error) {
		children, err := webui.AssetDir(path)
		if err != nil {
			return nil, os.ErrNotExist
		}
		return children, nil
	}
	rtr.NotFoundHandler = http.FileServer(
		&assetfs.AssetFS{
			Asset:     webui.Asset,
			AssetDir:  assetDirWrap,
			AssetInfo: webui.AssetInfo,
			Prefix:    "dist",
		})

	apirtr := rtr.PathPrefix("/api").Subrouter()
	apirtr.NotFoundHandler = http.NotFoundHandler()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:9000"}, // gulp devsrv
	})

	httpsrv := &http.Server{
		Addr:    addr,
		Handler: c.Handler(rtr),
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
