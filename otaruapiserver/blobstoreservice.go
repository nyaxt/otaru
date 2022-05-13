package otaruapiserver

import (
	"context"
	"math"

	"github.com/dustin/go-humanize"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/apiserver/clientauth"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
)

type blobstoreService struct {
	s   *scheduler.Scheduler
	bbs blobstore.BlobStore
	cbs *cachedblobstore.CachedBlobStore

	pb.UnimplementedBlobstoreServiceServer
}

func (svc *blobstoreService) GetConfig(ctx context.Context, in *pb.GetBlobstoreConfigRequest) (*pb.GetBlobstoreConfigResponse, error) {
	if err := clientauth.RequireRoleGRPC(ctx, clientauth.RoleAdmin); err != nil {
		return nil, err
	}

	beFlags := "unknown"
	if reader, ok := svc.bbs.(flags.FlagsReader); ok {
		beFlags = flags.FlagsToString(reader.Flags())
	}

	return &pb.GetBlobstoreConfigResponse{
		BackendImplName: util.TryGetImplName(svc.bbs),
		BackendFlags:    beFlags,
		CacheImplName:   util.TryGetImplName(svc.cbs),
		CacheFlags:      flags.FlagsToString(svc.cbs.Flags()),
	}, nil
}

func (svc *blobstoreService) GetEntries(ctx context.Context, in *pb.GetEntriesRequest) (*pb.GetEntriesResponse, error) {
	if err := clientauth.RequireRoleGRPC(ctx, clientauth.RoleAdmin); err != nil {
		return nil, err
	}

	oes := svc.cbs.DumpEntriesInfo()

	es := make([]*pb.GetEntriesResponse_Entry, 0, len(oes))
	for _, oe := range oes {
		e := &pb.GetEntriesResponse_Entry{
			BlobPath:              oe.BlobPath,
			State:                 oe.State,
			BlobLen:               oe.BlobLen,
			ValidLen:              oe.ValidLen,
			SyncCount:             int64(oe.SyncCount),
			LastUsed:              oe.LastUsed.Unix(),
			LastWrite:             oe.LastWrite.Unix(),
			LastSync:              oe.LastSync.Unix(),
			NumberOfWriterHandles: int64(oe.NumberOfWriterHandles),
			NumberOfHandles:       int64(oe.NumberOfHandles),
		}
		es = append(es, e)
	}

	return &pb.GetEntriesResponse{Entry: es}, nil
}

func (svc *blobstoreService) ReduceCache(ctx context.Context, req *pb.ReduceCacheRequest) (*pb.ReduceCacheResponse, error) {
	if err := clientauth.RequireRoleGRPC(ctx, clientauth.RoleAdmin); err != nil {
		return nil, err
	}

	desiredSizeP := req.DesiredSize
	desiredSize, err := humanize.ParseBytes(desiredSizeP)
	if desiredSize > math.MaxInt64 || err != nil {
		return &pb.ReduceCacheResponse{
			Success:      false,
			ErrorMessage: "Invalid desired size given.",
		}, nil
	}

	jv := svc.s.RunImmediatelyBlock(&cachedblobstore.ReduceCacheTask{
		svc.cbs, int64(desiredSize), req.DryRun})
	if err := jv.Result.Err(); err != nil {
		return &pb.ReduceCacheResponse{
			Success:      false,
			ErrorMessage: "Reduce cache task failed with error",
		}, nil
	}
	return &pb.ReduceCacheResponse{Success: true, ErrorMessage: "ok"}, nil
}

func InstallBlobstoreService(s *scheduler.Scheduler, bbs blobstore.BlobStore, cbs *cachedblobstore.CachedBlobStore) apiserver.Option {
	svc := &blobstoreService{
		s: s, bbs: bbs, cbs: cbs,
	}

	return apiserver.RegisterService(
		func(s *grpc.Server) { pb.RegisterBlobstoreServiceServer(s, svc) },
		pb.RegisterBlobstoreServiceHandlerFromEndpoint,
	)
}
