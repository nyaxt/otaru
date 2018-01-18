package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

var Log = logger.Registry().Category("cli")

const (
	BufLen = 32 * 1024
)

func Ls(ctx context.Context, cfg *CliConfig, args []string) {
	pathstr := args[1] // FIXME

	p, err := opath.Parse(pathstr)
	if err != nil {
		logger.Criticalf(Log, "Failed to parse: \"%s\". err: %v", pathstr, err)
		return
	}

	conn, err := DialVhost(cfg, p.Vhost)
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
		fmt.Printf("%s\n", e.Name)
	}
}

func Get(ctx context.Context, cfg *CliConfig, args []string) {
	pathstr := args[1]
	useHttpApi := true
	w := os.Stdout // FIXME

	p, err := opath.Parse(pathstr)
	if err != nil {
		logger.Criticalf(Log, "Failed to parse: \"%s\". err: %v", pathstr, err)
		return
	}

	if useHttpApi {
		id := 20

		ep, tc, err := ConnectionInfo(cfg, p.Vhost)
		if err != nil {
			logger.Criticalf(Log, "%v", err)
			return
		}
		cli := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tc,
			},
		}
		url := &url.URL{
			Scheme: "https",
			Host:   ep,
			Path:   fmt.Sprintf("/file/%d/bin", id),
		}
		logger.Debugf(Log, "requrl %v", url.String())
		req := &http.Request{
			Method: "GET",
			Header: map[string][]string{
			// "Accept-Encoding": {"gzip"}, // FIXME
			},
			URL: url,
		}
		resp, err := cli.Do(req)
		if err != nil {
			logger.Criticalf(Log, "%v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			logger.Criticalf(Log, "server responded w/ status code %d", resp.StatusCode)
			return
		}

		nw, err := io.Copy(w, resp.Body)
		if err != nil {
			logger.Criticalf(Log, "io copy %v", err)
			return
		}
		logger.Debugf(Log, "Received %d bytes.", nw)
	} else {
		conn, err := DialVhost(cfg, p.Vhost)
		if err != nil {
			logger.Criticalf(Log, "%v", err)
			return
		}
		defer conn.Close()

		fsc := pb.NewFileSystemServiceClient(conn)
		resp, err := fsc.FindNodeFullPath(ctx, &pb.FindNodeFullPathRequest{Path: p.FsPath})
		if err != nil {
			logger.Criticalf(Log, "FindNodeFullPath failed: %v", err)
			return
		}
		id := resp.Id
		logger.Infof(Log, "Got id %d for path \"%s\"", id, p.FsPath)

		offset := uint64(0)
		for {
			n := BufLen
			resp, err := fsc.ReadFile(ctx, &pb.ReadFileRequest{Id: id, Offset: offset, Length: uint32(n)})
			if err != nil {
				logger.Criticalf(Log, "ReadFile(id=%d, offset=%d, len=%d) failed: %v", id, offset, n, err)
				return
			}
			nr := len(resp.Body)
			// logger.Debugf(Log, "ReadFile(id=%d, offset=%d, len=%d) -> len=%d", id, offset, n, nr)
			if nr == 0 {
				break
			}
			offset += uint64(nr)

			if _, err := w.Write(resp.Body); err != nil {
				logger.Criticalf(Log, "Write(len=%d) err: %v", len(resp.Body), err)
			}
		}
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

	conn, err := DialVhost(cfg, p.Vhost)
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
