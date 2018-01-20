package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"google.golang.org/grpc"

	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

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

	resp, err := fsc.CreateFile(opts.ctx, &pb.CreateFileRequest{
		DirId:        0, // Fullpath mode
		Name:         p.FsPath,
		Uid:          uint32(os.Geteuid()),
		Gid:          uint32(os.Getegid()),
		PermMode:     uint32(0644), // FIXME
		ModifiedTime: time.Now().Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("CreateFile: %v", err)
	}

	id := resp.Id
	logger.Infof(Log, "Target file \"%s\" inode id: %v", p.FsPath, id)
	if !resp.IsNewFile {
		logger.Infof(Log, "Target file \"%s\" already exists. Overwriting.", p.FsPath)
	}

	if opts.forceGrpc {
		return &grpcWriter{ctx: opts.ctx, conn: conn, id: id, offset: 0}, nil
	} else {
		conn.Close()
		// return newWriterHttp(ep, tc, id)
		return nil, fmt.Errorf("FIXME")
	}
}
