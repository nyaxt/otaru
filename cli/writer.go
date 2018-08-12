package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"google.golang.org/grpc"

	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

type httpWriter struct {
	pw   io.WriteCloser
	errC <-chan error
}

var _ = io.WriteCloser(&httpWriter{})

func newWriterHttp(ep string, tc *tls.Config, id uint64) (io.WriteCloser, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to create pipe: %v", err)
	}

	errC := make(chan error)
	go func() {
		defer pr.Close()
		defer close(errC)
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
			Method: "PUT",
			Header: map[string][]string{
				// "Accept-Encoding": {"gzip"}, // FIXME
			},
			URL:           url,
			Body:          pr,
			ContentLength: -1, // FIXME
		}
		resp, err := cli.Do(req)
		if err != nil {
			errC <- err
			return
		}
		if resp.StatusCode != 200 {
			errC <- fmt.Errorf("server responded w/ status code %d", resp.StatusCode)
			return
		}
	}()

	return &httpWriter{pw, errC}, nil
}

func (w *httpWriter) Write(p []byte) (int, error) {
	nw, perr := w.pw.Write(p)
	if perr != nil {
		rerr := <-w.errC
		if rerr != nil {
			return nw, rerr
		}
		return nw, fmt.Errorf("httpWriter pipe write failed: %v", perr)
	}
	return nw, nil
}

func (w *httpWriter) Close() error {
	w.pw.Close()
	return <-w.errC
}

type grpcWriter struct {
	ctx    context.Context
	conn   *grpc.ClientConn
	id     uint64
	offset uint64
}

var _ = io.WriteCloser(&grpcWriter{})

func (w *grpcWriter) Write(p []byte) (int, error) {
	fsc := pb.NewFileSystemServiceClient(w.conn)
	nw := 0

	for len(p) > 0 {
		pw := p
		if len(pw) > GrpcChunkLen {
			pw = pw[:GrpcChunkLen]
		}

		if _, err := fsc.WriteFile(w.ctx, &pb.WriteFileRequest{
			Id: w.id, Offset: w.offset, Body: pw,
		}); err != nil {
			return nw, fmt.Errorf("WriteFile(id=%v, offset=%d, len=%d). err: %v", w.id, w.offset, len(pw), err)
		}

		nw += len(pw)
		w.offset += uint64(len(pw))
		p = p[len(pw):]
	}

	return nw, nil
}

func (w *grpcWriter) Close() error {
	// FIXME: set modified time again

	return w.conn.Close()
}

func NewWriter(pathstr string, ofs ...Option) (io.WriteCloser, error) {
	opts := defaultOptions
	for _, o := range ofs {
		o(&opts)
	}

	p, err := opath.Parse(pathstr)
	if err != nil {
		return nil, err
	}

	ep, tc, err := ConnectionInfo(opts.cfg, p.Vhost)
	if err != nil {
		return nil, err
	}
	conn, err := DialGrpc(ep, tc)
	if err != nil {
		return nil, err
	}

	fsc := pb.NewFileSystemServiceClient(conn)

	resp, err := fsc.Create(opts.ctx, &pb.CreateRequest{
		DirId:        0, // Fullpath mode
		Name:         p.FsPath,
		Uid:          uint32(os.Geteuid()),
		Gid:          uint32(os.Getegid()),
		PermMode:     uint32(0644), // FIXME
		ModifiedTime: time.Now().Unix(),
		Type:         pb.INodeType_FILE,
	})
	if err != nil {
		return nil, fmt.Errorf("Create: %v", err)
	}

	id := resp.Id
	logger.Infof(Log, "Target file \"%s\" inode id: %v", p.FsPath, id)
	if !resp.IsNew {
		if !opts.allowOverwrite {
			return nil, fmt.Errorf("Target file already exists and overwriting is prohivited.")
		}

		logger.Infof(Log, "Target file \"%s\" already exists. Overwriting.", p.FsPath)
	}

	if opts.forceGrpc {
		return &grpcWriter{ctx: opts.ctx, conn: conn, id: id, offset: 0}, nil
	} else {
		conn.Close()
		return newWriterHttp(ep, tc, id)
	}
}
