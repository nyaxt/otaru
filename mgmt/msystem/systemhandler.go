package msystem

import (
	"os"
	"runtime"

	"net/http"

	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/util/countfds"
)

type SystemInfo struct {
	GoVersion string `json:"goversion"`
	Os        string `json:"os"`
	Arch      string `json:"arch"`

	NumGoroutine int `json:"num_goroutine"`

	Hostname string `json:"hostname"`
	Pid      int    `json:"pid"`
	Uid      int    `json:"uid"`

	MemAlloc uint64 `json:"mem_alloc"`
	MemSys   uint64 `json:"mem_sys"`

	NumGC uint32 `json:"num_gc"`

	Fds int `json:"fds"`
}

func GetSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "<os.Hostname failed!>"
	}

	ret := SystemInfo{
		GoVersion: runtime.Version(),
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,

		NumGoroutine: runtime.NumGoroutine(),

		Hostname: hostname,
		Pid:      os.Getpid(),
		Uid:      os.Getuid(),

		MemAlloc: m.Alloc,
		MemSys:   m.Sys,

		NumGC: m.NumGC,

		Fds: countfds.CountFds(),
	}

	return ret
}

func Install(srv *mgmt.Server) {
	rtr := srv.APIRouter().PathPrefix("/system").Subrouter()

	rtr.HandleFunc("/info", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return GetSystemInfo()
	}))
	rtr.HandleFunc("/meminfo", mgmt.JSONHandler(func(req *http.Request) interface{} {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m
	}))
}
