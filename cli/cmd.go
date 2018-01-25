package cli

import (
	"context"
	"flag"
	"io"
	"os"

	"github.com/nyaxt/otaru/logger"
)

var Log = logger.Registry().Category("cli")

const (
	BufLen = 32 * 1024
)

func Get(ctx context.Context, cfg *CliConfig, args []string) {
	fset := flag.NewFlagSet("get", flag.ExitOnError)
	fset.Parse(args[1:])

	pathstr := fset.Arg(0)
	w := os.Stdout // FIXME

	r, err := NewReader(pathstr, WithCliConfig(cfg), WithContext(ctx))
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
	defer r.Close()

	if _, err := io.Copy(w, r); err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
}

func Put(ctx context.Context, cfg *CliConfig, args []string) {
	fset := flag.NewFlagSet("put", flag.ExitOnError)
	fset.Parse(args[1:])

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
