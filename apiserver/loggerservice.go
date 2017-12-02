package apiserver

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

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

func (loggerService) SetCategory(ctx context.Context, req *pb.SetCategoryRequest) (*pb.SetCategoryResponse, error) {
	c := logger.Registry().CategoryIfExist(req.Category)
	if c == nil {
		return nil, grpc.Errorf(codes.NotFound, "Specified category not found")
	}

	c.Level = logger.Level(req.Level)

	return &pb.SetCategoryResponse{}, nil
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
