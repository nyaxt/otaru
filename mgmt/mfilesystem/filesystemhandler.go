package mfilesystem

import (
	"net/http"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/mgmt"
)

func Install(srv *mgmt.Server, fs *otaru.FileSystem) {
	rtr := srv.APIRouter().PathPrefix("/filesystem").Subrouter()

	rtr.HandleFunc("/stats", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return fs.GetStats()
	}))
}
