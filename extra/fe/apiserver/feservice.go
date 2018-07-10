package apiserver

import (
	"io/ioutil"
	"path/filepath"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/extra/fe/pb"
	otarupb "github.com/nyaxt/otaru/pb"
)

type feService struct {
	hnames        []string
	localRootPath string
}

func (s *feService) ListHosts(ctx context.Context, req *pb.ListHostsRequest) (*pb.ListHostsResponse, error) {
	return &pb.ListHostsResponse{Host: s.hnames}, nil
}

func (s *feService) ListLocalDir(ctx context.Context, req *pb.ListLocalDirRequest) (*pb.ListLocalDirResponse, error) {
	if s.localRootPath == "" {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Local filesystem operations disabled.")
	}

	path := filepath.Join(s.localRootPath, filepath.Clean("/"+req.Path))

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

func genHostNames(cfg *cli.CliConfig) []string {
	hnames := make([]string, 0)
	id := uint32(0)
	for name, _ := range cfg.Host {
		hnames = append(hnames, name)
		id++
	}
	if cfg.Fe.LocalRootPath != "" {
		hnames = append(hnames, "[local]")
	}
	return hnames
}

func InstallFeService(cfg *cli.CliConfig) apiserver.Option {
	fes := &feService{
		hnames:        genHostNames(cfg),
		localRootPath: filepath.Clean(cfg.Fe.LocalRootPath),
	}

	return apiserver.RegisterService(
		func(s *grpc.Server) { pb.RegisterFeServiceServer(s, fes) },
		pb.RegisterFeServiceHandlerFromEndpoint,
	)
}
