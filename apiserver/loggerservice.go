package apiserver

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

type loggerService struct{}

func (loggerService) GetCategories(ctx context.Context, req *pb.GetCategoriesRequest) (*pb.GetCategoriesResponse, error) {
	cs := logger.Registry().Categories()

	pcs := make([]*pb.LoggerCategory, 0, len(cs))
	for _, c := range cs {
		pcs = append(pcs, &pb.LoggerCategory{
			Category: c.Category,
			Level:    uint32(c.Level),
		})
	}

	return &pb.GetCategoriesResponse{Category: pcs}, nil
}

func InstallLoggerService() Option {
	return func(o *options) {
		o.serviceRegistry = append(o.serviceRegistry, serviceRegistryEntry{
			registerServiceServer: func(s *grpc.Server) {
				pb.RegisterLoggerServiceServer(s, loggerService{})
			},
			registerProxy: pb.RegisterLoggerServiceHandlerFromEndpoint,
		})
	}
}
