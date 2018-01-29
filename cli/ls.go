package cli

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"

	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/util"
)

func typeToR(t string) rune {
	switch t {
	case "file":
		return '-'
	case "dir":
		return 'd'
	default:
		return '?'
	}
}

func permTo3Letters(m uint32) string {
	s := ""

	if m&4 != 0 {
		s += "r"
	} else {
		s += "-"
	}
	if m&2 != 0 {
		s += "w"
	} else {
		s += "-"
	}
	if m&1 != 0 {
		s += "x"
	} else {
		s += "-"
	}

	return s
}

func formatSize(n int64, h bool) string {
	if !h {
		return strconv.FormatInt(n, 10)
	}

	return humanize.Bytes(uint64(n))
}

func formatDate(n int64) string {
	return time.Unix(n, 0).Format("Jan 2  2006")
}

func Ls(ctx context.Context, cfg *CliConfig, args []string) {
	fset := flag.NewFlagSet("ls", flag.ExitOnError)
	flagL := fset.Bool("l", false, "use a long listing format")
	flagH := fset.Bool("h", false, "print human readable sizes (e.g., 1K 234M 2G)")
	fset.Parse(args[1:])

	pathstr := fset.Arg(0)

	p, err := opath.Parse(pathstr)
	if err != nil {
		logger.Criticalf(Log, "Failed to parse: \"%s\". err: %v", pathstr, err)
		return
	}

	conn, err := DialGrpcVhost(cfg, p.Vhost)
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
	defer conn.Close()

	fsc := pb.NewFileSystemServiceClient(conn)
	resp, err := fsc.ListDir(ctx, &pb.ListDirRequest{Path: p.FsPath})
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
	if len(resp.Listing) != 1 {
		logger.Criticalf(Log, "Expected 1 listing, but got %d listings.", len(resp.Listing))
	}
	l := resp.Listing[0]
	for _, e := range l.Entry {
		if *flagL {
			// drwxr-xr-x  7 kouhei kouhei   4096 Feb 12  2017 processing-3.3

			fmt.Printf("%c%s%s%s %s %s %s %s %s\n",
				typeToR(e.Type),
				permTo3Letters(e.PermMode>>6),
				permTo3Letters(e.PermMode>>3),
				permTo3Letters(e.PermMode>>0),
				util.TryUserName(e.Uid),
				util.TryGroupName(e.Gid),
				formatSize(e.Size, *flagH),
				formatDate(e.ModifiedTime),
				e.Name)
		} else {
			fmt.Printf("%s\n", e.Name)
		}
	}
}
