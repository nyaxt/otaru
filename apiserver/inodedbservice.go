package apiserver

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/pb"
)

type inodedbService struct {
	h inodedb.DBHandler
}

func (svc *inodedbService) GetINodeDBStats(ctx context.Context, req *pb.GetINodeDBStatsRequest) (*pb.GetINodeDBStatsResponse, error) {
	prov, ok := svc.h.(inodedb.DBServiceStatsProvider)
	if !ok {
		return nil, grpc.Errorf(codes.Unimplemented, "inodedb doesn't support providing stats.")
	}
	stats := prov.GetStats()

	return &pb.GetINodeDBStatsResponse{
		LastSync:          stats.LastSync.Unix(),
		LastTx:            stats.LastTx.Unix(),
		LastId:            uint64(stats.LastID),
		Version:           uint64(stats.Version),
		LastTicket:        uint64(stats.LastTicket),
		NumberOfNodeLocks: uint32(stats.NumberOfNodeLocks),
	}, nil
}

func InstallINodeDBService(h inodedb.DBHandler) Option {
	return func(o *options) {
		o.serviceRegistry = append(o.serviceRegistry, serviceRegistryEntry{
			registerServiceServer: func(s *grpc.Server) {
				pb.RegisterINodeDBServiceServer(s, &inodedbService{h})
			},
			registerProxy: pb.RegisterINodeDBServiceHandlerFromEndpoint,
		})
	}
}
