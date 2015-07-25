package mblobstore

import (
	"net/http"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
)

func Install(srv *mgmt.Server, s *scheduler.Scheduler, bbs blobstore.BlobStore, cbs *cachedblobstore.CachedBlobStore) {
	rtr := srv.APIRouter().PathPrefix("/blobstore").Subrouter()

	rtr.HandleFunc("/config", mgmt.JSONHandler(func(req *http.Request) interface{} {
		type Config struct {
			Flags           string `json:"flags"`
			BackendImplName string `json:"backend_impl_name"`
			CacheImplName   string `json:"cache_impl_name"`
		}
		return Config{
			Flags:           flags.FlagsToString(cbs.Flags()),
			BackendImplName: util.TryGetImplName(bbs),
			CacheImplName:   util.TryGetImplName(cbs),
		}
	}))
	rtr.HandleFunc("/entries", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return cbs.DumpEntriesInfo()
	}))
	rtr.HandleFunc("/reduce_cache", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			http.Error(w, "reduce_cache should be triggered with POST method.", http.StatusMethodNotAllowed)
			return
		}

		dryrunp := req.URL.Query().Get("dryrun")
		dryrun := len(dryrunp) > 0

		jv := s.RunImmediatelyBlock(&cachedblobstore.ReduceCacheTask{cbs, dryrun})
		if err := jv.Result.Err(); err != nil {
			http.Error(w, "Reduce cache task failed with error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok"))
	})
}
