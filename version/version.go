package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var BuildVersion string = "<unknown>"
var BuildSum string = "<unknown>"

func init() {
	if bi, ok := debug.ReadBuildInfo(); ok {
		BuildVersion = bi.Main.Version
		BuildSum = bi.Main.Sum
	}
}

func DumpBuildInfo() string {
	return fmt.Sprintf(""+
		"Version: %s\n"+
		"Sum: %s\n"+
		"Go version: %s\n"+
		"OS/Arch:    %s/%s\n",
		BuildVersion,
		BuildSum,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH,
	)
}
