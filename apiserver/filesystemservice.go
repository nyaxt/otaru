package apiserver

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/util"
)

type fileSystemService struct {
	fs *filesystem.FileSystem
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

func (svc *fileSystemService) CreateFile(ctx context.Context, req *pb.CreateFileRequest) (*pb.CreateFileResponse, error) {
	dirId := inodedb.ID(req.DirId)
	permMode := uint16(req.PermMode & 0777)
	modifiedT := time.Unix(req.ModifiedTime, 0)
	if dirId == 0 {
		// Fullpath mode.
		fullpath := req.Name
		id, err := svc.fs.CreateFileFullPath(fullpath, permMode, req.Uid, req.Gid, modifiedT)
		if err != nil {
			if util.IsExist(err) {
				id, err := svc.fs.FindNodeFullPath(fullpath)
				if err != nil {
					return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("FindNodeFullPath for existing file failed: %v", err))
				}
				return &pb.CreateFileResponse{Id: uint64(id), IsNewFile: false}, nil
			}
			return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("CreateFileFullPath failed: %v", err))
		}
		return &pb.CreateFileResponse{Id: uint64(id), IsNewFile: true}, nil
	}
	id, err := svc.fs.CreateFile(dirId, req.Name, permMode, req.Uid, req.Gid, modifiedT)
	if err != nil {
		if util.IsExist(err) {
			entriesMap, err := svc.fs.DirEntries(dirId)
			if err != nil {
				return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("DirEntries for existing file parent dir failed: %v", err))
			}
			id, ok := entriesMap[req.Name]
			if !ok {
				return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("Existing file's parent dir doesn't have the existing file !??"))
			}
			return &pb.CreateFileResponse{Id: uint64(id), IsNewFile: false}, nil
		}
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("CreateFile failed: %v", err))
	}

	return &pb.CreateFileResponse{Id: uint64(id), IsNewFile: true}, nil
}

func (svc *fileSystemService) WriteFile(ctx context.Context, req *pb.WriteFileRequest) (*pb.WriteFileResponse, error) {
	id := inodedb.ID(req.Id)

	h, err := svc.fs.OpenFile(id, flags.O_RDWR)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("OpenFile failed: %v", err))
	}
	defer h.Close()

	err = h.PWrite(req.Body, int64(req.Offset))
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("PWrite failed: %v", err))
	}

	return &pb.WriteFileResponse{}, nil
}

func InstallFileSystemService(fs *filesystem.FileSystem) Option {
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
