package apiserver

import (
	"os"
	"runtime"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/util/countfds"
	"github.com/nyaxt/otaru/version"
)

type systemService struct{}

func (*systemService) GetSystemInfo(context.Context, *pb.GetSystemInfoRequest) (*pb.SystemInfoResponse, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "<os.Hostname failed!>"
	}

	return &pb.SystemInfoResponse{
		GoVersion: runtime.Version(),
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,

		NumGoroutine: uint32(runtime.NumGoroutine()),

		Hostname: hostname,
		Pid:      uint64(os.Getpid()),
		Uid:      uint64(os.Getuid()),

		MemAlloc: m.Alloc,
		MemSys:   m.Sys,

		NumGc: m.NumGC,

		NumFds: uint32(countfds.CountFds()),
	}, nil
}

func (*systemService) GetVersion(ctx context.Context, in *pb.GetVersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		GitCommit: version.GIT_COMMIT,
		BuildHost: version.BUILD_HOST,
		BuildTime: version.BuildTimeString,
	}, nil
}

func InstallSystemService() Option {
	return func(o *options) {
		o.serviceRegistry = append(o.serviceRegistry, serviceRegistryEntry{
			registerServiceServer: func(s *grpc.Server) {
				pb.RegisterSystemInfoServiceServer(s, &systemService{})
			},
			registerProxy: pb.RegisterSystemInfoServiceHandlerFromEndpoint,
		})
	}
}
