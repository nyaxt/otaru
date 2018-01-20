package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"

	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/util"
)

var Log = logger.Registry().Category("cli")

const (
	BufLen = 32 * 1024
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
	for _, e := range resp.Entry {
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

func Get(ctx context.Context, cfg *CliConfig, args []string) {
	pathstr := args[1]
	w := os.Stdout // FIXME

	r, err := NewReader(pathstr, WithCliConfig(cfg))
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

	p, err := opath.Parse(pathstr)
	if err != nil {
		logger.Criticalf(Log, "Failed to parse: \"%s\". err: %v", pathstr, err)
		return
	}

	f, err := os.Open(localpathstr)
	if err != nil {
		logger.Criticalf(Log, "Failed to open source file: \"%s\". err: %v", localpathstr, err)
		return
	}
	defer f.Close()

	conn, err := DialGrpcVhost(cfg, p.Vhost)
	if err != nil {
		logger.Criticalf(Log, "%v", err)
		return
	}
	defer conn.Close()

	fsc := pb.NewFileSystemServiceClient(conn)

	cfresp, err := fsc.CreateFile(ctx, &pb.CreateFileRequest{
		DirId:        0, // Fullpath mode
		Name:         p.FsPath,
		Uid:          uint32(os.Geteuid()),
		Gid:          uint32(os.Getegid()),
		PermMode:     uint32(0644), // FIXME
		ModifiedTime: time.Now().Unix(),
	})
	if err != nil {
		logger.Criticalf(Log, "CreateFile: %v", err)
		return
	}

	id := cfresp.Id
	logger.Infof(Log, "Target file \"%s\" inode id: %v", p.FsPath, id)
	if !cfresp.IsNewFile {
		logger.Infof(Log, "Target file \"%s\" already exists. Overwriting.", p.FsPath)
	}

	offset := uint64(0)
	buf := make([]byte, BufLen)
	for {
		n, err := f.Read(buf)
		if n == 0 {
			break
		}
		_, err = fsc.WriteFile(ctx, &pb.WriteFileRequest{
			Id: id, Offset: offset, Body: buf[:n],
		})
		if err != nil {
			logger.Criticalf(Log, "WriteFile(id=%v, offset=%d, len=%d). err: %v", id, offset, n, err)
			return
		}
		offset += uint64(n)
	}

	// FIXME: set modified time again
}
