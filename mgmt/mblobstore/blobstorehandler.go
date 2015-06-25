package mblobstore

import (
	"net/http"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/util"
)

func Install(srv *mgmt.Server, bbs blobstore.BlobStore, cbs *blobstore.CachedBlobStore) {
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
}
