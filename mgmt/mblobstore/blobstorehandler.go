package mblobstore

import (
	"net/http"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/mgmt"
)

type BlobStoreHandler struct{}

func Install(srv *mgmt.Server) {
	rtr := srv.APIRouter().PathPrefix("/blobstore").Subrouter()

	rtr.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "json")
		w.Write([]byte("bs st\n"))
	})
}
