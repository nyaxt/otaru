package cachedblobstore

import (
	"fmt"
	"math"
	"time"

	"context"

	"github.com/dustin/go-humanize"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
)

type ReduceCacheTask struct {
	CBS         *CachedBlobStore
	DesiredSize int64
	DryRun      bool
}

func (t *ReduceCacheTask) Run(ctx context.Context) scheduler.Result {
	err := t.CBS.ReduceCache(ctx, t.DesiredSize, t.DryRun)
	return scheduler.ErrorResult{err}
}

const autoReduceCachePeriod = 10 * time.Second

type AutoReduceCacheTask struct {
	CBS           *CachedBlobStore
	HighWatermark int64
	LowWatermark  int64
}

func (t AutoReduceCacheTask) main(ctx context.Context) error {
	cachebs := t.CBS.cachebs
	tsizer, ok := cachebs.(blobstore.TotalSizer)
	if !ok {
		return fmt.Errorf("Cache backend \"%s\" doesn't support TotalSize() method, required for AutoReduceCacheTask. aborting.", util.TryGetImplName(cachebs))
	}
	currentSize, err := tsizer.TotalSize()
	if err != nil {
		return fmt.Errorf("Failed to query current total cache size: %v", err)
	}

	if currentSize <= t.HighWatermark {
		logger.Infof(mylog, "AutoReduceCacheTask: Current size %s < high watermark %s. No-op.",
			humanize.Bytes(uint64(currentSize)),
			humanize.Bytes(uint64(t.HighWatermark)))
		return nil
	}

	return t.CBS.ReduceCache(ctx, t.LowWatermark, false)
}

func (t AutoReduceCacheTask) Run(ctx context.Context) scheduler.Result {
	return scheduler.ErrorResult{t.main(ctx)}
}

func (t AutoReduceCacheTask) String() string {
	return fmt.Sprintf("AutoReduceCacheTask{highwm: %s, lowwm: %s}",
		humanize.Bytes(uint64(t.HighWatermark)),
		humanize.Bytes(uint64(t.LowWatermark)))
}

func SetupAutoReduceCache(cbs *CachedBlobStore, r *scheduler.RepetitiveJobRunner, highwm, lowwm int64) scheduler.ID {
	if highwm == math.MaxInt64 || lowwm == math.MaxInt64 || lowwm > highwm {
		logger.Infof(mylog, "No automatic cache discards due to bad watermarks")
		return 0
	}

	logger.Infof(mylog, "Setting up automatic cache discards. high/low watermark: %s/%s",
		humanize.Bytes(uint64(highwm)), humanize.Bytes(uint64(lowwm)))

	return r.RunEveryPeriod(AutoReduceCacheTask{cbs, highwm, lowwm}, autoReduceCachePeriod)
}
