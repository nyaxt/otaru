package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/nyaxt/otaru/logger"
)

var Log = logger.Registry().Category("cli")

const (
	BufLen = 32 * 1024
)

func Get(ctx context.Context, cfg *CliConfig, args []string) {
	fset := flag.NewFlagSet("get", flag.ExitOnError)
	flagO := fset.String("o", "", "destination file path")
	flagC := fset.String("C", "", "destination dir path")
	fset.Usage = func() {
		fmt.Printf("Usage of %s get:\n", os.Args[0])
		fmt.Printf(" %s get [OTARU_PATH]...\n", os.Args[0])
		fset.PrintDefaults()
	}
	fset.Parse(args[1:])

	if fset.NArg() == 0 {
		fset.Usage()
		return
	}
	if *flagO != "" && fset.NArg() != 1 {
		logger.Criticalf(Log, "Only one path is allowed when specified -o option.")
		fset.Usage()
		return
	}

	var destdir string
	if *flagC != "" {
		destdir = *flagC
	} else {
		var err error
		destdir, err = os.Getwd()
		if err != nil {
			logger.Criticalf(Log, "Failed to query current dir: %v", err)
			return
		}
	}
	fi, err := os.Stat(destdir)
	if err != nil {
		logger.Criticalf(Log, "Failed to stat target dir: %v", err)
		return
	}
	if !fi.IsDir() {
		logger.Criticalf(Log, "Specified destination is not a dir")
		return
	}

	for _, srcstr := range fset.Args() {
		r, err := NewReader(srcstr, WithCliConfig(cfg), WithContext(ctx))
		if err != nil {
			logger.Criticalf(Log, "%v", err)
			return
		}

		var dest string
		if *flagO != "" {
			dest = *flagO
		} else {
			dest = path.Join(destdir, path.Base(srcstr))
		}
		w, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0644) // FIXME: r.Stat().FileMode
		if err != nil {
			logger.Criticalf(Log, "%v", err)
			r.Close()
			return
		}
		logger.Infof(Log, "Remote %s -> Local %s", srcstr, dest)

		if _, err := io.Copy(w, r); err != nil {
			logger.Criticalf(Log, "%v", err)
			r.Close()
			w.Close()
			return
		}
		r.Close()
		w.Close()
	}
}

func Put(ctx context.Context, cfg *CliConfig, args []string) {
	fset := flag.NewFlagSet("put", flag.ExitOnError)
	fset.Usage = func() {
		fmt.Printf("Usage of %s put:\n", os.Args[0])
		fmt.Printf(" %s put [OTARU_PATH] [LOCAL_PATH]\n", os.Args[0])
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
