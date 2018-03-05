package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

func typeToR(t pb.INodeType) rune {
	switch t {
	case pb.INodeType_FILE:
		return '-'
	case pb.INodeType_DIR:
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
		e.Id, e.Type.String(), e.Name, e.Uid, e.Gid,
		e.PermMode, permModeStr,
		time.Unix(e.ModifiedTime, 0),
	}
	j, err := json.Marshal(eobj)
	if err != nil {
		log.Panicf("json.Marshal failure: %v", err)
	}
	return j
}

func Attr(ctx context.Context, cfg *CliConfig, args []string) error {
	fset := flag.NewFlagSet("attr", flag.ExitOnError)
	fset.Usage = func() {
		fmt.Printf("Usage of %s attr:\n", os.Args[0])
		fmt.Printf(" %s attr OTARU_PATH [nodeid]\n", os.Args[0])
		fset.PrintDefaults()
	}
	fset.Parse(args[1:])

	if fset.NArg() < 1 || 2 < fset.NArg() {
		fset.Usage()
		return fmt.Errorf("Invalid number of arguments")
	}
	pathstr := fset.Arg(0)

	id := uint64(0)
	if fset.NArg() == 2 {
		var err error
		id, err = strconv.ParseUint(fset.Arg(1), 10, 64)
		if err != nil {
			fset.Usage()
			return fmt.Errorf("Failed to parse id from \"%s\"", fset.Arg(1))
		}
	}

	p, err := opath.Parse(pathstr)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	conn, err := DialGrpcVhost(cfg, p.Vhost)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()

	fsc := pb.NewFileSystemServiceClient(conn)
	resp, err := fsc.Attr(ctx, &pb.AttrRequest{Id: id, Path: p.FsPath})
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	j := invToJson(resp.Entry)
	if _, err := os.Stdout.Write(j); err != nil {
		return fmt.Errorf("failed to Write: %v", err)
	}
	fmt.Printf("\n")
	return nil
}

func Ls(ctx context.Context, w io.Writer, cfg *CliConfig, args []string) error {
	fset := flag.NewFlagSet("ls", flag.ExitOnError)
	flagL := fset.Bool("l", false, "use a long listing format")
	flagH := fset.Bool("h", false, "print human readable sizes (e.g., 1K 234M 2G)")
	flagJson := fset.Bool("json", false, "format output using json")
	fset.Parse(args[1:])

	paths := fset.Args()
	if fset.NArg() == 0 {
		paths = []string{"/"}
	}

	var conn *grpc.ClientConn
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	var connectedVhost string
	if *flagJson {
		if _, err := w.Write([]byte("{\n")); err != nil {
			return fmt.Errorf("failed to Write: %v", err)
		}
	}
	for _, s := range paths {
		logger.Debugf(Log, "pathstr: %s", s)
		p, err := opath.Parse(s)
		if err != nil {
			return fmt.Errorf("Failed to parse: \"%s\". err: %v", s, err)
		}

		if connectedVhost != p.Vhost {
			// Close previous connection if needed.
			if conn != nil {
				conn.Close()
			}

			conn, err = DialGrpcVhost(cfg, p.Vhost)
			if err != nil {
				return fmt.Errorf("%v", err)
			}
			connectedVhost = p.Vhost
		}

		fsc := pb.NewFileSystemServiceClient(conn)
		resp, err := fsc.ListDir(ctx, &pb.ListDirRequest{Path: p.FsPath})
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		if len(resp.Listing) != 1 {
			return fmt.Errorf("Expected 1 listing, but got %d listings.", len(resp.Listing))
		}
		l := resp.Listing[0]
		if *flagJson {
			fmt.Fprintf(w, "\"%s\": [\n", s)
		}
		for i, e := range l.Entry {
			if *flagJson {
				j := invToJson(e)
				if _, err := w.Write(j); err != nil {
					return fmt.Errorf("failed to Write: %v", err)
				}
				if i != len(l.Entry)-1 {
					if _, err := w.Write([]byte(",\n")); err != nil {
						return fmt.Errorf("failed to Write: %v", err)
					}
				}
			} else if *flagL {
				fmt.Fprintf(w, "%c%s%s%s %s %s %s %s %s\n",
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
				fmt.Fprintf(w, "%s\n", e.Name)
			}
		}
		if *flagJson {
			if _, err := w.Write([]byte("]\n")); err != nil {
				return fmt.Errorf("failed to Write: %v", err)
			}
		}
	}
	if *flagJson {
		if _, err := w.Write([]byte("}\n")); err != nil {
			return fmt.Errorf("failed to Write: %v", err)
		}
	}
	return nil
}
