package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/nyaxt/otaru/logger"
)

func Put(ctx context.Context, cfg *CliConfig, args []string) {
	fset := flag.NewFlagSet("put", flag.ExitOnError)
	fset.Usage = func() {
		fmt.Printf("Usage of %s put:\n", os.Args[0])
		fmt.Printf(" %s put OTARU_PATH LOCAL_PATH\n", os.Args[0])
		fset.PrintDefaults()
	}
	fset.Parse(args[1:])

	if fset.NArg() != 2 {
		logger.Criticalf(Log, "Invalid number of arguments")
		fset.Usage()
		return
	}
	pathstr, localpathstr := fset.Arg(0), fset.Arg(1)
	// FIXME: pathstr may end in /, in which case should join(pathstr, base(localpathstr))

	f, err := os.Open(localpathstr)
	if err != nil {
		logger.Criticalf(Log, "Failed to open source file: \"%s\". err: %v", localpathstr, err)
		return
	}
	defer f.Close()

	w, err := NewWriter(pathstr, WithCliConfig(cfg), WithContext(ctx))
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}

	if _, err := io.Copy(w, f); err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
}
