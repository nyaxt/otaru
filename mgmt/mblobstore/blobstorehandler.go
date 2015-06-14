package mblobstore

import (
	"net/http"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/mgmt"
)

type Status struct {
	Flags string `json:"flags"`
}

func Install(srv *mgmt.Server, cbs *blobstore.CachedBlobStore) {
	rtr := srv.APIRouter().PathPrefix("/blobstore").Subrouter()

	rtr.HandleFunc("/status", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return Status{flags.FlagsToString(cbs.Flags())}
	}))
	rtr.HandleFunc("/entries", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return cbs.DumpEntriesInfo()
	}))
}
