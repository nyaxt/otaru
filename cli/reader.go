package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"google.golang.org/grpc"

	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

const GrpcChunkLen = 32 * 1024

type httpReader struct {
	body io.ReadCloser
	size int64
	left int64
}

type Reader interface {
	io.ReadCloser
	Size() int64
}

var _ = Reader(&httpReader{})

func newReaderHttp(ep string, tc *tls.Config, id uint64) (Reader, error) {
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
		return nil, fmt.Errorf("Failed to issue Http GET request: %v", err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("server responded w/ status code %d", resp.StatusCode)
	}

	clstr := resp.Header.Get("Content-Length")
	cl, err := strconv.ParseInt(clstr, 10, 64)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("server responded w/ invalid Content-Length header: %q", clstr)
	}

	return &httpReader{
		body: resp.Body,
		size: cl,
		left: cl,
	}, nil
}

func (r *httpReader) Read(p []byte) (int, error) {
	n, err := r.body.Read(p)
	if err != nil {
		return n, err
	}

	if r.left < int64(n) {
		// Receiving more data than we expected at first.
		// This indicates that the source file may have been appended since started reading.
		return n, nil
	}

	if n == 0 {
		if r.left > 0 {
			return 0, fmt.Errorf("Unexpected end of HTTP response body. Expected to have %d bytes more", r.left)
		}
	}
	r.left -= int64(n)
	return n, nil
}

func (r *httpReader) Close() error {
	return r.body.Close()
}

func (r *httpReader) Size() int64 {
	return r.size
}

type grpcReader struct {
	ctx    context.Context
	conn   *grpc.ClientConn
	id     uint64
	size   int64
	offset uint64
}

var _ = Reader(&grpcReader{})

func (r *grpcReader) Read(p []byte) (int, error) {
	fsc := pb.NewFileSystemServiceClient(r.conn)

	if len(p) > GrpcChunkLen {
		p = p[:GrpcChunkLen]
	}

	resp, err := fsc.ReadFile(r.ctx, &pb.ReadFileRequest{
		Id:     r.id,
		Offset: r.offset,
		Length: uint32(len(p)),
	})
	if err != nil {
		return 0, fmt.Errorf("ReadFile(id=%d, offset=%d, len=%d) failed: %v", r.id, r.offset, len(p), err)
	}

	nr := len(resp.Body)
	// logger.Debugf(Log, "ReadFile(id=%d, offset=%d, len=%d) -> len=%d", id, offset, n, nr)
	if nr == 0 {
		return 0, nil
	}
	if nr > len(p) {
		nr = len(p)
	}
	r.offset += uint64(nr)

	copy(p, resp.Body[:nr])
	return nr, nil
}

func (r *grpcReader) Close() error {
	return r.conn.Close()
}

func (r *grpcReader) Size() int64 {
	return r.size
}

type fileReader os.File

var _ = Reader((*fileReader)(nil))

func (r *fileReader) Read(p []byte) (int, error) {
	return (*os.File)(r).Read(p)
}
func (r *fileReader) Close() error {
	return (*os.File)(r).Close()
}

func (r *fileReader) Size() int64 {
	var osf *os.File
	osf = (*os.File)(r)

	fi, err := osf.Stat()
	if err != nil {
		return -1
	}
	return fi.Size()
}

func NewReader(pathstr string, options ...Option) (Reader, error) {
	opts := defaultOptions
	for _, o := range options {
		o(&opts)
	}

	p, err := opath.Parse(pathstr)
	if err != nil {
		return nil, err
	}

	if p.Vhost == opath.VhostLocal {
		lpath, err := opts.cfg.ResolveLocalPath(p.FsPath)
		if err != nil {
			return nil, err
		}

		f, err := os.Open(lpath)
		if err != nil {
			return nil, err
		}
		return (*fileReader)(f), nil
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
	resp, err := fsc.FindNodeFullPath(opts.ctx, &pb.FindNodeFullPathRequest{Path: p.FsPath})
	if err != nil {
		return nil, fmt.Errorf("FindNodeFullPath failed: %v", err)
	}
	id := resp.Id
	logger.Infof(Log, "Got id %d for path \"%s\"", id, p.FsPath)

	if opts.forceGrpc {
		respA, err := fsc.Attr(opts.ctx, &pb.AttrRequest{Id: resp.Id})
		if err != nil {
			return nil, fmt.Errorf("Attr failed: %v", err)
		}
		size := respA.Entry.Size

		return &grpcReader{ctx: opts.ctx, conn: conn, id: id, size: size, offset: 0}, nil
	} else {
		conn.Close()
		return newReaderHttp(ep, tc, id)
	}
}
