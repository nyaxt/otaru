package apiserver

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/apiserver/jwt"
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
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleReadOnly); err != nil {
		return nil, err
	}

	return &pb.ListHostsResponse{Host: s.hnames}, nil
}

func (s *feService) ListLocalDir(ctx context.Context, req *pb.ListLocalDirRequest) (*pb.ListLocalDirResponse, error) {
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleReadOnly); err != nil {
		return nil, err
	}

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
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleAdmin); err != nil {
		return nil, err
	}

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

func (s *feService) CopyLocal(ctx context.Context, req *pb.CopyLocalRequest) (*pb.CopyLocalResponse, error) {
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleAdmin); err != nil {
		return nil, err
	}

	pathSrc, err := s.cfg.ResolveLocalPath(req.PathSrc)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}
	pathDest, err := s.cfg.ResolveLocalPath(req.PathDest)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}

	logger.Infof(mylog, "CopyLocal %q -> %q", pathSrc, pathDest)

	fiSrc, err := os.Stat(pathSrc)
	if err != nil {
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

	r, err := os.Open(pathSrc)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to os.Open(src): %v", err)
	}
	defer r.Close()

	mode := fiSrc.Mode()
	w, err := os.OpenFile(pathDest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to os.OpenFile(pathDest): %v", err)
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to io.Copy: %v", err)
	}

	return &pb.CopyLocalResponse{}, nil
}

func (s *feService) MoveLocal(ctx context.Context, req *pb.MoveLocalRequest) (*pb.MoveLocalResponse, error) {
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleAdmin); err != nil {
		return nil, err
	}

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

func (s *feService) Download(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadResponse, error) {
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleAdmin); err != nil {
		return nil, err
	}

	pathDest, err := s.cfg.ResolveLocalPath(req.PathDest)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}
	opathSrc := req.OpathSrc

	logger.Infof(mylog, "Download %q -> %q", opathSrc, pathDest)

	fi, err := os.Stat(pathDest)
	if err == nil {
		if !req.AllowOverwrite {
			return nil, grpc.Errorf(codes.FailedPrecondition, "Destination path already exists.")
		}
		if fi.IsDir() {
			return nil, grpc.Errorf(codes.FailedPrecondition, "Destination path is a directory.")
		}
	}
	if !os.IsNotExist(err) {
		return nil, grpc.Errorf(codes.Internal, "Error Stat()ing destination path: %v", err)
	}

	r, err := cli.NewReader(opathSrc, cli.WithCliConfig(s.cfg), cli.WithContext(ctx)) //, cli.WithTokenOverride(token))
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to init reader: %v", err)
	}
	defer r.Close()

	mode := os.FileMode(0644) // FIXME
	w, err := os.OpenFile(pathDest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to os.OpenFile(pathDest): %v", err)
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to io.Copy: %v", err)
	}

	return &pb.DownloadResponse{}, nil
}

func (s *feService) Upload(ctx context.Context, req *pb.UploadRequest) (*pb.UploadResponse, error) {
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleAdmin); err != nil {
		return nil, err
	}

	pathSrc, err := s.cfg.ResolveLocalPath(req.PathSrc)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}
	opathDest := req.OpathDest

	logger.Infof(mylog, "Upload %q -> %q", pathSrc, opathDest)

	if _, err = os.Stat(pathSrc); err != nil {
		if os.IsNotExist(err) {
			return nil, grpc.Errorf(codes.FailedPrecondition, "Source path doesn't exist.")
		}
		return nil, grpc.Errorf(codes.Internal, "Error Stat()ing source path: %v", err)
	}

	r, err := os.Open(pathSrc)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to os.Open(src): %v", err)
	}
	defer r.Close()

	w, err := cli.NewWriter(opathDest, cli.WithCliConfig(s.cfg), cli.WithContext(ctx), cli.AllowOverwrite(req.AllowOverwrite))
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to init writer: %v", err)
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return nil, grpc.Errorf(codes.Internal, "Failed to io.Copy: %v", err)
	}

	return &pb.UploadResponse{}, nil
}

func (s *feService) RemoteMove(ctx context.Context, req *pb.RemoteMoveRequest) (*pb.RemoteMoveResponse, error) {
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleAdmin); err != nil {
		return nil, err
	}

	return nil, grpc.Errorf(codes.Internal, "Not implemented!")

	return &pb.RemoteMoveResponse{}, nil
}

func (s *feService) RemoveLocal(ctx context.Context, req *pb.RemoveLocalRequest) (*pb.RemoveLocalResponse, error) {
	if err := jwt.RequireRoleGRPC(ctx, jwt.RoleAdmin); err != nil {
		return nil, err
	}

	path, err := s.cfg.ResolveLocalPath(req.Path)
	if err != nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Failed to resolve local path: %v", err)
	}

	logger.Infof(mylog, "Remove %q", path)
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, grpc.Errorf(codes.FailedPrecondition, "Destination path does not exist.")
		}
		return nil, grpc.Errorf(codes.Internal, "Error Stat()ing destination path: %v", err)
	}

	if fi.IsDir() && !req.RemoveChildren {
		d, err := os.Open(path)
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, "Failed to open dir: %v", err)
		}
		defer d.Close()

		if _, err := d.Readdirnames(1); err != io.EOF {
			if err != nil {
				return nil, grpc.Errorf(codes.Internal, "Failed to peek dir: %v", err)
			}
			return nil, grpc.Errorf(codes.FailedPrecondition, "Target dir not empty and RemoveChildren flag is not set.")
		}
	}

	if s.cfg.TrashDirPath != "" {
		trashPath := filepath.Join(s.cfg.TrashDirPath, filepath.Base(path))
		for {
			if _, err := os.Stat(trashPath); err != nil {
				if os.IsNotExist(err) {
					break
				}
				return nil, grpc.Errorf(codes.Internal, "Stat failed for unknown reason: %v", err)
			}

			trashBase := fmt.Sprintf("%s_%d", filepath.Base(path), rand.Intn(100))
			trashPath = filepath.Join(s.cfg.TrashDirPath, trashBase)
		}

		if err := os.Rename(path, trashPath); err != nil {
			return nil, grpc.Errorf(codes.Internal, "Error os.Rename(): %v", err)
		}
	} else {
		if req.RemoveChildren {
			if err := os.RemoveAll(path); err != nil {
				return nil, grpc.Errorf(codes.Internal, "Error os.RemoveAll(): %v", err)
			}
		} else {
			if err := os.Remove(path); err != nil {
				return nil, grpc.Errorf(codes.Internal, "Error os.Remove(): %v", err)
			}
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
