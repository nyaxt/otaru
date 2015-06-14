package mblobstore

import (
	"encoding/json"
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

	rtr.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "json")
		s := Status{
			Flags: flags.FlagsToString(cbs.Flags()),
		}
		b, err := json.Marshal(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Write(b)
	})

	rtr.HandleFunc("/entries", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "json")
		entries := cbs.DumpEntriesInfo()
		b, err := json.Marshal(entries)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Write(b)
	})
}
