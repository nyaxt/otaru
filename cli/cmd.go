package cli

import (
	"context"
	"io"
	"os"

	"github.com/nyaxt/otaru/logger"
)

var Log = logger.Registry().Category("cli")

const (
	BufLen = 32 * 1024
)

func Get(ctx context.Context, cfg *CliConfig, args []string) {
	pathstr := args[1]
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
	pathstr, localpathstr := args[1], args[2] // FIXME
	// FIXME: pathstr may end in /, in which case should join(pathstr, base(localpathstr))

	f, err := os.Open(localpathstr)
	if err != nil {
		logger.Criticalf(Log, "Failed to open source file: \"%s\". err: %v", localpathstr, err)
		return
	}
	defer f.Close()

	w, err := NewWriter(pathstr, WithCliConfig(cfg), WithContext(ctx), ForceGrpc())
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}

	if _, err := io.Copy(w, f); err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
}
