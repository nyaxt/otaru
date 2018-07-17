package logger

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("apilogger_fixme")

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		timeStart := time.Now()

		resp, err := handler(ctx, req)

		processTime := time.Since(timeStart)
		mylog.Log(
			logger.Info,
			map[string]interface{}{
				"log":         fmt.Sprintf("req: %v err: %v", req, err),
				"location":    info.FullMethod,
				"time":        time.Now(),
				"processTime": processTime,
				"req":         req,
			},
		)

		return resp, err
	}
}
