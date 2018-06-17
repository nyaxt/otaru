package apiserver

import (
	"fmt"
	"path"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/util"
)

const MaxReadLen = 1024 * 1024

type fileSystemService struct {
	fs *filesystem.FileSystem
}

func type2pb(t inodedb.Type) pb.INodeType {
	switch t {
	case inodedb.FileNodeT:
		return pb.INodeType_FILE
	case inodedb.DirNodeT:
		return pb.INodeType_DIR
	default:
		logger.Panicf(mylog, "unknown inodedb.Type(%v)", t)
		return pb.INodeType_FILE
	}
}

func attrToINodeView(id inodedb.ID, name string, a filesystem.Attr) *pb.INodeView {
	return &pb.INodeView{
		Id:           uint64(id),
		Name:         name,
		Type:         type2pb(a.Type),
		Size:         a.Size,
		Uid:          a.Uid,
		Gid:          a.Gid,
		PermMode:     uint32(a.PermMode),
		ModifiedTime: a.ModifiedT.Unix(),
	}
}

func (svc *fileSystemService) ListDir(ctx context.Context, req *pb.ListDirRequest) (*pb.ListDirResponse, error) {
	ids := req.Id
	if len(ids) == 0 {
		id, err := svc.fs.FindNodeFullPath(req.Path)
		if err != nil {
			if util.IsNotExist(err) {
				return nil, grpc.Errorf(codes.NotFound, "Specified path not found.")
			}
			return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("FindNodeFullPath failed: %v", err))
		}
		ids = []uint64{uint64(id)}
	}

	ls := make([]*pb.ListDirResponse_Listing, 0, len(ids))
	for _, nid := range ids {
		id := inodedb.ID(nid)
		isDir, err := svc.fs.IsDir(id)
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("IsDir(%d) failed: %v", id, err))
		}
		if !isDir {
			return nil, grpc.Errorf(codes.FailedPrecondition, "ListDir target %d is non-dir.", id)
		}

		entriesMap, err := svc.fs.DirEntries(id)
		es := make([]*pb.INodeView, 0, len(entriesMap))
		for name, cid := range entriesMap {
			attr, err := svc.fs.Attr(cid)
			if err != nil {
				return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("Child node Attr(%d) failed: %v", cid, err))
			}

			inv := attrToINodeView(cid, name, attr)
			es = append(es, inv)
		}

		ls = append(ls, &pb.ListDirResponse_Listing{
			DirId: nid,
			Entry: es,
		})
	}

	return &pb.ListDirResponse{Listing: ls}, nil
}

func (svc *fileSystemService) FindNodeFullPath(ctx context.Context, req *pb.FindNodeFullPathRequest) (*pb.FindNodeFullPathResponse, error) {
	id, err := svc.fs.FindNodeFullPath(req.Path)
	if err != nil {
		if util.IsNotExist(err) {
			return nil, grpc.Errorf(codes.NotFound, "Specified path not found.")
		}
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("FindNodeFullPath failed: %v", err))
	}
	return &pb.FindNodeFullPathResponse{Id: uint64(id)}, nil
}

func (svc *fileSystemService) Attr(ctx context.Context, req *pb.AttrRequest) (*pb.AttrResponse, error) {
	id := inodedb.ID(req.Id)
	if id == 0 {
		var err error
		id, err = svc.fs.FindNodeFullPath(req.Path)
		if err != nil {
			if util.IsNotExist(err) {
				return nil, grpc.Errorf(codes.NotFound, "Specified path not found.")
			}
			return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("FindNodeFullPath failed: %v", err))
		}
	}

	attr, err := svc.fs.Attr(id)
	if err != nil {
		if util.IsNotExist(err) {
			return nil, grpc.Errorf(codes.NotFound, "Specified path not found.")
		}
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("Attr(%d) failed: %v", id, err))
	}

	inv := attrToINodeView(id, path.Base(attr.OrigPath), attr)
	return &pb.AttrResponse{Entry: inv}, nil
}

func (svc *fileSystemService) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error) {
	dirId := inodedb.ID(req.DirId)
	permMode := uint16(req.PermMode & 0777)
	var modifiedT time.Time
	if req.ModifiedTime > 0 {
		modifiedT = time.Unix(req.ModifiedTime, 0)
	} else {
		modifiedT = time.Now()
	}
	if dirId == 0 {
		// Fullpath mode.
		fullpath := req.Name
		var id inodedb.ID
		var err error
		if req.Type == pb.INodeType_FILE {
			id, err = svc.fs.CreateFileFullPath(fullpath, permMode, req.Uid, req.Gid, modifiedT)
		} else if req.Type == pb.INodeType_DIR {
			id, err = svc.fs.CreateDirFullPath(fullpath, permMode, req.Uid, req.Gid, modifiedT)
		} else {
			return nil, grpc.Errorf(codes.InvalidArgument, "invalid Type %d given.", req.Type)
		}
		if err != nil {
			if util.IsExist(err) {
				id, err := svc.fs.FindNodeFullPath(fullpath)
				if err != nil {
					return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("FindNodeFullPath for existing file failed: %v", err))
				}
				return &pb.CreateResponse{Id: uint64(id), IsNew: false}, nil
			}
			return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("CreateFileFullPath failed: %v", err))
		}
		return &pb.CreateResponse{Id: uint64(id), IsNew: true}, nil
	}

	var id inodedb.ID
	var err error
	if req.Type == pb.INodeType_FILE {
		id, err = svc.fs.CreateFile(dirId, req.Name, permMode, req.Uid, req.Gid, modifiedT)
	} else if req.Type == pb.INodeType_DIR {
		id, err = svc.fs.CreateDir(dirId, req.Name, permMode, req.Uid, req.Gid, modifiedT)
	} else {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid Type %d given.", req.Type)
	}
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
			return &pb.CreateResponse{Id: uint64(id), IsNew: false}, nil
		}
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("CreateFile failed: %v", err))
	}

	return &pb.CreateResponse{Id: uint64(id), IsNew: true}, nil
}

func (svc *fileSystemService) ReadFile(ctx context.Context, req *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	id := inodedb.ID(req.Id)

	h, err := svc.fs.OpenFile(id, flags.O_RDONLY)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("OpenFile failed: %v", err))
	}
	defer h.Close()

	if req.Length > MaxReadLen {
		req.Length = MaxReadLen
	}
	body := make([]byte, req.Length)
	n, err := h.ReadAt(body, int64(req.Offset))
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("ReadAt failed: %v", err))
	}

	return &pb.ReadFileResponse{Body: body[:n]}, nil
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
