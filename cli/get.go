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

func Get(ctx context.Context, cfg *CliConfig, args []string) error {
	fset := flag.NewFlagSet("get", flag.ExitOnError)
	flagO := fset.String("o", "", "destination file path")
	flagC := fset.String("C", "", "destination dir path")
	fset.Usage = func() {
		fmt.Printf("Usage of %s get:\n", os.Args[0])
		fmt.Printf(" %s get OTARU_PATH...\n", os.Args[0])
		fset.PrintDefaults()
	}
	fset.Parse(args[1:])

	if fset.NArg() == 0 {
		fset.Usage()
		return fmt.Errorf("Invalid number of arguments.")
	}
	if *flagO != "" && fset.NArg() != 1 {
		fset.Usage()
		return fmt.Errorf("Only one path is allowed when specified -o option.")
	}

	var destdir string
	if *flagC != "" {
		destdir = *flagC
	} else {
		var err error
		destdir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("Failed to query current dir: %v", err)
		}
	}
	fi, err := os.Stat(destdir)
	if err != nil {
		return fmt.Errorf("Failed to stat target dir: %v", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("Specified destination is not a dir")
	}

	for _, srcstr := range fset.Args() {
		r, err := NewReader(srcstr, WithCliConfig(cfg), WithContext(ctx))
		if err != nil {
			return fmt.Errorf("%v", err)
		}

		var dest string
		if *flagO != "" {
			dest = *flagO
		} else {
			dest = path.Join(destdir, path.Base(srcstr))
		}
		w, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0644) // FIXME: r.Stat().FileMode
		if err != nil {
			r.Close()
			return fmt.Errorf("Failed to open dest file: %v", err)
		}
		logger.Infof(Log, "Remote %s -> Local %s", srcstr, dest)

		if _, err := io.Copy(w, r); err != nil {
			r.Close()
			w.Close()
			return fmt.Errorf("io.Copy failed: %v", err)
		}
		if err := r.Close(); err != nil {
			return fmt.Errorf("Failed to Close(src): %v", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("Failed to Close(dest): %v", err)
		}
	}
	return nil
}
