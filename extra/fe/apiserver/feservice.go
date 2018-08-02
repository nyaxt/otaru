package apiserver

import (
	"io/ioutil"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/extra/fe/pb"
	"github.com/nyaxt/otaru/logger"
	otarupb "github.com/nyaxt/otaru/pb"
)

type feService struct {
	cfg    *cli.CliConfig
	hnames []string
}

func (s *feService) ListHosts(ctx context.Context, req *pb.ListHostsRequest) (*pb.ListHostsResponse, error) {
	return &pb.ListHostsResponse{Host: s.hnames}, nil
}

func (s *feService) ListLocalDir(ctx context.Context, req *pb.ListLocalDirRequest) (*pb.ListLocalDirResponse, error) {
	path, err := s.cfg.ResolveLocalPath(req.Path)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}

	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "ReadDir Err: %v", err)
	}
	es := make([]*pb.FileInfo, 0, len(fis))
	for _, fi := range fis {
		t := otarupb.INodeType_FILE
		if fi.IsDir() {
			t = otarupb.INodeType_DIR
		}
		e := &pb.FileInfo{
			Name:         fi.Name(),
			Type:         t,
			Size:         fi.Size(),
			PermMode:     uint32(fi.Mode()),
			ModifiedTime: fi.ModTime().Unix(),
		}
		es = append(es, e)
	}

	return &pb.ListLocalDirResponse{Entry: es}, nil
}

func (s *feService) MkdirLocal(ctx context.Context, req *pb.MkdirLocalRequest) (*pb.MkdirLocalResponse, error) {
	path, err := s.cfg.ResolveLocalPath(req.Path)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}

	logger.Infof(mylog, "Mkdir %q", path)
	_, err = os.Stat(path)
	if err == nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Destination path already exists.")
	}
	if !os.IsNotExist(err) {
		return nil, grpc.Errorf(codes.Internal, "Error Stat()ing destination path: %v", err)
	}

	if err := os.Mkdir(path, 0755); err != nil {
		return nil, grpc.Errorf(codes.Internal, "Error os.Mkdir(): %v", err)
	}

	return &pb.MkdirLocalResponse{}, nil
}

func (s *feService) MoveLocal(ctx context.Context, req *pb.MoveLocalRequest) (*pb.MoveLocalResponse, error) {
	pathSrc, err := s.cfg.ResolveLocalPath(req.PathSrc)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}
	pathDest, err := s.cfg.ResolveLocalPath(req.PathDest)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}

	logger.Infof(mylog, "MoveLocal %q -> %q", pathSrc, pathDest)

	if _, err = os.Stat(pathSrc); err != nil {
		if os.IsNotExist(err) {
			return nil, grpc.Errorf(codes.FailedPrecondition, "Source path doesn't exist.")
		}
		return nil, grpc.Errorf(codes.Internal, "Error Stat()ing source path: %v", err)
	}

	_, err = os.Stat(pathDest)
	if err == nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Destination path already exists.")
	}
	if !os.IsNotExist(err) {
		return nil, grpc.Errorf(codes.Internal, "Error Stat()ing destination path: %v", err)
	}

	if err := os.Rename(pathSrc, pathDest); err != nil {
		return nil, grpc.Errorf(codes.Internal, "Error os.Rename(): %v", err)
	}

	return &pb.MoveLocalResponse{}, nil
}

func (s *feService) RemoveLocal(ctx context.Context, req *pb.RemoveLocalRequest) (*pb.RemoveLocalResponse, error) {
	path, err := s.cfg.ResolveLocalPath(req.Path)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}

	logger.Infof(mylog, "Remove %q", path)
	if _, err = os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, grpc.Errorf(codes.FailedPrecondition, "Destination path does not exist.")
		}
		return nil, grpc.Errorf(codes.Internal, "Error Stat()ing destination path: %v", err)
	}

	if req.RemoveAll {
		if err := os.RemoveAll(path); err != nil {
			return nil, grpc.Errorf(codes.Internal, "Error os.RemoveAll(): %v", err)
		}
	} else {
		if err := os.Remove(path); err != nil {
			return nil, grpc.Errorf(codes.Internal, "Error os.Remove(): %v", err)
		}
	}

	return &pb.RemoveLocalResponse{}, nil
}

func genHostNames(cfg *cli.CliConfig) []string {
	hnames := make([]string, 0)
	id := uint32(0)
	for name, _ := range cfg.Host {
		hnames = append(hnames, name)
		id++
	}
	if cfg.LocalRootPath != "" {
		hnames = append(hnames, opath.VhostLocal)
	}
	return hnames
}

func InstallFeService(cfg *cli.CliConfig) apiserver.Option {
	fes := &feService{
		cfg:    cfg,
		hnames: genHostNames(cfg),
	}

	return apiserver.RegisterService(
		func(s *grpc.Server) { pb.RegisterFeServiceServer(s, fes) },
		pb.RegisterFeServiceHandlerFromEndpoint,
	)
}
