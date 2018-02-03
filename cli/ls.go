package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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

func invToJson(e *pb.INodeView) []byte {
	permModeStr := fmt.Sprintf("%s%s%s",
		permTo3Letters(e.PermMode>>6),
		permTo3Letters(e.PermMode>>3),
		permTo3Letters(e.PermMode>>0))
	eobj := struct {
		Id           uint64    `json:"id"`
		Type         string    `json:"type"`
		Name         string    `json:"name"`
		Uid          uint32    `json:"uid"`
		Gid          uint32    `json:"gid"`
		PermMode     uint32    `json:"perm_mode"`
		PermModeStr  string    `json:"perm_mode_str"`
		ModifiedTime time.Time `json:"modified_time"`
	}{
		e.Id, e.Type, e.Name, e.Uid, e.Gid,
		e.PermMode, permModeStr,
		time.Unix(e.ModifiedTime, 0),
	}
	j, err := json.Marshal(eobj)
	if err != nil {
		log.Panicf("json.Marshal failure: %v", err)
	}
	return j
}

func Attr(ctx context.Context, cfg *CliConfig, args []string) {
	fset := flag.NewFlagSet("attr", flag.ExitOnError)
	fset.Usage = func() {
		fmt.Printf("Usage of %s attr:\n", os.Args[0])
		fmt.Printf(" %s attr OTARU_PATH [nodeid]\n", os.Args[0])
		fset.PrintDefaults()
	}
	fset.Parse(args[1:])

	if fset.NArg() < 1 || 2 < fset.NArg() {
		logger.Criticalf(Log, "Invalid number of arguments")
		fset.Usage()
		return
	}
	pathstr := fset.Arg(0)

	id := uint64(0)
	if fset.NArg() == 2 {
		var err error
		id, err = strconv.ParseUint(fset.Arg(1), 10, 64)
		if err != nil {
			logger.Criticalf(Log, "Failed to parse id from \"%s\"", fset.Arg(1))
			fset.Usage()
			return
		}
	}

	p, err := opath.Parse(pathstr)
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}

	conn, err := DialGrpcVhost(cfg, p.Vhost)
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
	defer conn.Close()

	fsc := pb.NewFileSystemServiceClient(conn)
	resp, err := fsc.Attr(ctx, &pb.AttrRequest{Id: id, Path: p.FsPath})
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
	j := invToJson(resp.Entry)
	if _, err := os.Stdout.Write(j); err != nil {
		logger.Criticalf(Log, "failed to Write: %v", err)
		return
	}
	fmt.Printf("\n")
}

func Ls(ctx context.Context, cfg *CliConfig, args []string) {
	fset := flag.NewFlagSet("ls", flag.ExitOnError)
	flagL := fset.Bool("l", false, "use a long listing format")
	flagH := fset.Bool("h", false, "print human readable sizes (e.g., 1K 234M 2G)")
	flagJson := fset.Bool("json", false, "format output using json")
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
	if _, err := os.Stdout.WriteString("{\n"); err != nil {
		logger.Criticalf(Log, "failed to Write: %v", err)
		return
	}
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
		if len(resp.Listing) != 1 {
			logger.Criticalf(Log, "Expected 1 listing, but got %d listings.", len(resp.Listing))
		}
		l := resp.Listing[0]
		if *flagJson {
			fmt.Printf("\"%s\": [\n", s)
		}
		for i, e := range l.Entry {
			if *flagJson {
				j := invToJson(e)
				if _, err := os.Stdout.Write(j); err != nil {
					logger.Criticalf(Log, "failed to Write: %v", err)
					return
				}
				if i != len(l.Entry)-1 {
					if _, err := os.Stdout.WriteString(",\n"); err != nil {
						logger.Criticalf(Log, "failed to Write: %v", err)
						return
					}
				}
			} else if *flagL {
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
		if *flagJson {
			if _, err := os.Stdout.WriteString("]\n"); err != nil {
				logger.Criticalf(Log, "failed to Write: %v", err)
				return
			}
		}
	}
	if *flagJson {
		if _, err := os.Stdout.WriteString("}\n"); err != nil {
			logger.Criticalf(Log, "failed to Write: %v", err)
			return
		}
	}
}
