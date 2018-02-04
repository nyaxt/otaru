package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
)

func Put(ctx context.Context, cfg *CliConfig, args []string) error {
	fset := flag.NewFlagSet("put", flag.ExitOnError)
	fset.Usage = func() {
		fmt.Printf("Usage of %s put:\n", os.Args[0])
		fmt.Printf(" %s put LOCAL_PATH OTARU_PATH\n", os.Args[0])
		fset.PrintDefaults()
	}
	fset.Parse(args[1:])

	if fset.NArg() != 2 {
		fset.Usage()
		return fmt.Errorf("Invalid number of arguments")
	}
	localpathstr, pathstr := fset.Arg(0), fset.Arg(1)
	// FIXME: pathstr may end in /, in which case should join(pathstr, base(localpathstr))

	f, err := os.Open(localpathstr)
	if err != nil {
		return fmt.Errorf("Failed to open source file: \"%s\". err: %v", localpathstr, err)
	}
	defer f.Close()

	w, err := NewWriter(pathstr, WithCliConfig(cfg), WithContext(ctx))
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, f); err != nil {
		return err
	}
	return nil
}
