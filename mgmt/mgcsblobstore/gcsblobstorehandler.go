package mgcsblobstore

import (
	"log"
	"net/http"

	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/mgmt"
)

func Install(srv *mgmt.Server, bs *gcs.GCSBlobStore) {
	rtr := srv.APIRouter().PathPrefix("/gcsblobstore").Subrouter()

	rtr.HandleFunc("/stats", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return bs.GetStats()
	}))

	log.Printf("Installed /api/gcsblobstore")
}
