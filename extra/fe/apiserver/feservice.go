package apiserver

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	//"google.golang.org/grpc/codes"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/extra/fe/pb"
)

type feService struct {
	hnames []string
}

func (s *feService) ListHosts(ctx context.Context, req *pb.ListHostsRequest) (*pb.ListHostsResponse, error) {
	return &pb.ListHostsResponse{Host: s.hnames}, nil
}

func genHostNames(cfg *cli.CliConfig) []string {
	hnames := make([]string, 0, len(cfg.Host))
	id := uint32(0)
	for name, _ := range cfg.Host {
		hnames = append(hnames, name)
		id++
	}
	return hnames
}

func InstallFeService(cfg *cli.CliConfig) apiserver.Option {
	fes := &feService{
		hnames: genHostNames(cfg),
	}

	return apiserver.RegisterService(
		func(s *grpc.Server) { pb.RegisterFeServiceServer(s, fes) },
		pb.RegisterFeServiceHandlerFromEndpoint,
	)
}
