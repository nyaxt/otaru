package cli

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"
	"google.golang.org/grpc"

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

	if fset.NArg() == 0 {
		logger.Criticalf(Log, "No path given.")
		return
	}

	var conn *grpc.ClientConn
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	var connectedVhost string
	for _, s := range fset.Args() {
		logger.Debugf(Log, "pathstr: %s", s)
		p, err := opath.Parse(s)
		if err != nil {
			logger.Criticalf(Log, "Failed to parse: \"%s\". err: %v", s, err)
			return
		}

		if connectedVhost != p.Vhost {
			// Close previous connection if needed.
			if conn != nil {
				conn.Close()
			}

			conn, err = DialGrpcVhost(cfg, p.Vhost)
			if err != nil {
				logger.Criticalf(Log, "%v", err)
				return
			}
			connectedVhost = p.Vhost
		}

		fsc := pb.NewFileSystemServiceClient(conn)
		resp, err := fsc.ListDir(ctx, &pb.ListDirRequest{Path: p.FsPath})
		if err != nil {
			logger.Criticalf(Log, "%v", err)
			return
		}
		for _, e := range resp.Entry {
			if *flagL {
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
}
