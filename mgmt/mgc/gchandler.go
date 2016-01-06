package mgc

import (
	"net/http"

	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/scheduler"
)

type GCTaskFactory interface {
	GetBlobstoreGCTask(dryrun bool) scheduler.Task
	GetINodeDBTxLogGCTask(dryrun bool) scheduler.Task
}

func Install(srv *mgmt.Server, s *scheduler.Scheduler, factory GCTaskFactory) {
	rtr := srv.APIRouter().PathPrefix("/gc").Subrouter()

	triggerHandler := func(factory func(dryrun bool) scheduler.Task) func(w http.ResponseWriter, req *http.Request) {
		return func(w http.ResponseWriter, req *http.Request) {
			if req.Method != "POST" {
				http.Error(w, "GC should be triggered with POST method.", http.StatusMethodNotAllowed)
				return
			}

			dryrunp := req.URL.Query().Get("dryrun")
			dryrun := len(dryrunp) > 0

			t := factory(dryrun)
			if t == nil {
				http.Error(w, "GC task factory failed", http.StatusInternalServerError)
				return
			}

			jv := s.RunImmediatelyBlock(t)
			if err := jv.Result.Err(); err != nil {
				http.Error(w, "GC task failed with error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("ok"))
		}
	}

	rtr.HandleFunc("/blobstore/trigger", triggerHandler(func(dryrun bool) scheduler.Task {
		return factory.GetBlobstoreGCTask(dryrun)
	}))
	rtr.HandleFunc("/inodedbtxlog/trigger", triggerHandler(func(dryrun bool) scheduler.Task {
		return factory.GetINodeDBTxLogGCTask(dryrun)
	}))
}
