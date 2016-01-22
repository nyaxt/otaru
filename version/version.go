package version

import (
	"fmt"
	"runtime"
	"time"
)

type buildInfo struct {
	GitCommit string    `json:"git_commit"`
	BuildHost string    `json:"build_host"`
	BuildTime time.Time `json:"build_time"`
}

var BuildInfo = buildInfo{
	GitCommit: GIT_COMMIT,
	BuildHost: BUILD_HOST,
	BuildTime: time.Unix(BUILD_TIME, 0),
}

func DumpBuildInfo() string {
	return fmt.Sprintf(""+
		"Git commit: %s\n"+
		"Build host: %s\n"+
		"Build time: %s\n"+
		"Go version: %s\n"+
		"OS/Arch:    %s/%s\n",
		GIT_COMMIT,
		BUILD_HOST,
		BuildInfo.BuildTime.Format("Mon Jan 2 15:04:05 -0700 MST 2006"),
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH,
	)
}
