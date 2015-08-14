package msystem

import (
	"runtime"

	"net/http"

	"github.com/nyaxt/otaru/mgmt"
)

type SystemInfo struct {
	GoVersion string `json:"goversion"`
	Os        string `json:"os"`
	Arch      string `json:"arch"`
}

func GetSystemInfo() SystemInfo {
	return SystemInfo{
		GoVersion: runtime.Version(),
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

func Install(srv *mgmt.Server) {
	rtr := srv.APIRouter().PathPrefix("/system").Subrouter()

	rtr.HandleFunc("/info", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return GetSystemInfo()
	}))
}
