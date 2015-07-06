package mgc

import (
	"net/http"

	"github.com/nyaxt/otaru/gc"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/scheduler"
)

func Install(srv *mgmt.Server, s *scheduler.Scheduler, bs gc.GCableBlobStore, idb inodedb.DBFscker) {
	rtr := srv.APIRouter().PathPrefix("/gc").Subrouter()

	rtr.HandleFunc("/trigger", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			http.Error(w, "GC should be triggered with POST method.", http.StatusMethodNotAllowed)
			return
		}

		dryrunp := req.URL.Query().Get("dryrun")
		dryrun := len(dryrunp) > 0

		jv := s.RunImmediatelyBlock(&gc.GCTask{bs, idb, dryrun})
		if err := jv.Result.Err(); err != nil {
			http.Error(w, "GC task failed with error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok"))
	})
}
