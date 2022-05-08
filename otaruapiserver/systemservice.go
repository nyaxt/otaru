package otaruapiserver

import (
	"os"
	"runtime"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/apiserver/clientauth"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/util/countfds"
	"github.com/nyaxt/otaru/version"
)

type systemService struct{}

func (*systemService) GetSystemInfo(ctx context.Context, in *pb.GetSystemInfoRequest) (*pb.SystemInfoResponse, error) {
	if err := clientauth.RequireRoleGRPC(ctx, clientauth.RoleAdmin); err != nil {
		return nil, err
	}

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
	}, nil
}

func (*systemService) Whoami(ctx context.Context, in *pb.WhoamiRequest) (*pb.WhoamiResponse, error) {
	ui := clientauth.UserInfoFromContext(ctx)

	return &pb.WhoamiResponse{
		Role: ui.Role.String(),
		User: ui.User,
	}, nil
}

func (*systemService) AuthTestAnonymous(ctx context.Context, in *pb.AuthTestRequest) (*pb.AuthTestResponse, error) {
	if err := clientauth.RequireRoleGRPC(ctx, clientauth.RoleAnonymous); err != nil {
		return nil, err
	}

	return &pb.AuthTestResponse{}, nil
}

func (*systemService) AuthTestReadOnly(ctx context.Context, in *pb.AuthTestRequest) (*pb.AuthTestResponse, error) {
	if err := clientauth.RequireRoleGRPC(ctx, clientauth.RoleReadOnly); err != nil {
		return nil, err
	}

	return &pb.AuthTestResponse{}, nil
}

func (*systemService) AuthTestAdmin(ctx context.Context, in *pb.AuthTestRequest) (*pb.AuthTestResponse, error) {
	if err := clientauth.RequireRoleGRPC(ctx, clientauth.RoleAdmin); err != nil {
		return nil, err
	}

	return &pb.AuthTestResponse{}, nil
}

func InstallSystemService() apiserver.Option {
	return apiserver.RegisterService(
		func(s *grpc.Server) { pb.RegisterSystemInfoServiceServer(s, &systemService{}) },
		pb.RegisterSystemInfoServiceHandlerFromEndpoint,
	)
}
