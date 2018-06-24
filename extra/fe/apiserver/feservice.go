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
	his []*pb.HostInfo
}

func (s *feService) ListHosts(ctx context.Context, req *pb.ListHostsRequest) (*pb.ListHostsResponse, error) {
	return &pb.ListHostsResponse{Host: s.his}, nil
}

func genHostInfos(cfg *cli.CliConfig) []*pb.HostInfo {
	his := make([]*pb.HostInfo, 0, len(cfg.Host))
	id := uint32(0)
	for name, _ := range cfg.Host {
		his = append(his, &pb.HostInfo{
			Id:   id,
			Name: name,
		})
		id++
	}
	return his
}

func InstallFeService(cfg *cli.CliConfig) apiserver.Option {
	fes := &feService{
		his: genHostInfos(cfg),
	}

	return apiserver.RegisterService(
		func(s *grpc.Server) { pb.RegisterFeServiceServer(s, fes) },
		pb.RegisterFeServiceHandlerFromEndpoint,
	)
}
