package apiserver

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/util"
)

type fileSystemService struct {
	fs *otaru.FileSystem
}

func (svc *fileSystemService) ListDir(ctx context.Context, req *pb.ListDirRequest) (*pb.ListDirResponse, error) {
	id, err := svc.fs.FindNodeFullPath(req.Path)
	if err != nil {
		if util.IsNotExist(err) {
			return nil, grpc.Errorf(codes.NotFound, "Specified path not found.")
		}
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("FindNodeFullPath failed: %v", err))
	}

	isDir, err := svc.fs.IsDir(id)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("IsDir failed: %v", err))
	}
	if !isDir {
		return nil, grpc.Errorf(codes.FailedPrecondition, "ListDir target is non-dir.")
	}

	entriesMap, err := svc.fs.DirEntries(id)
	es := make([]*pb.ListDirResponse_Entry, 0, len(entriesMap))
	for name, cid := range entriesMap {
		attr, err := svc.fs.Attr(cid)
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("Child node Attr(%d) failed: %v", cid, err))
		}

		es = append(es, &pb.ListDirResponse_Entry{
			Id:           uint64(cid),
			Name:         name,
			Type:         inodedb.TypeName(attr.Type),
			Size:         attr.Size,
			Uid:          attr.Uid,
			Gid:          attr.Gid,
			PermMode:     uint32(attr.PermMode),
			ModifiedTime: attr.ModifiedT.Unix(),
		})
	}

	return &pb.ListDirResponse{Entry: es}, nil
}

func InstallFileSystemService(fs *otaru.FileSystem) Option {
	svc := &fileSystemService{fs}

	return func(o *options) {
		o.serviceRegistry = append(o.serviceRegistry, serviceRegistryEntry{
			registerServiceServer: func(s *grpc.Server) {
				pb.RegisterFileSystemServiceServer(s, svc)
			},
			registerProxy: pb.RegisterFileSystemServiceHandlerFromEndpoint,
		})
	}
}
