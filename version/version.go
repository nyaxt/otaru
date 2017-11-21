package version

import (
	"fmt"
	"runtime"
	"time"
)

var BuildTime time.Time
var BuildTimeString string

func init() {
	BuildTime = time.Unix(BUILD_TIME, 0)
	BuildTimeString = BuildTime.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
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
		BuildTimeString,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH,
	)
}
