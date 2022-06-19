package blobstoregc

import (
	"fmt"
	"time"

	"context"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/metadata"
	"github.com/nyaxt/otaru/util"
	"go.uber.org/zap"
)

var mylog = logger.Registry().Category("blobstoregc")

type GCableBlobStore interface {
	blobstore.BlobLister
	blobstore.BlobRemover
}

func GC(ctx context.Context, bs GCableBlobStore, idb inodedb.DBFscker, dryrun bool) error {
	start := time.Now()

	zap.S().Infof("GC start. Dryrun: %t. Listing blobs.", dryrun)
	allbs, err := bs.ListBlobs()
	if err != nil {
		return fmt.Errorf("ListBlobs failed: %v", err)
	}
	zap.S().Infof("List blobs done. %d blobs found.", len(allbs))
	if err := ctx.Err(); err != nil {
		zap.S().Infof("Detected cancel. Bailing out.")
		return err
	}
	zap.S().Infof("Starting INodeDB fsck.")
	usedbs, errs := idb.Fsck()
	if len(errs) != 0 {
		return fmt.Errorf("Fsck returned err: %v", err)
	}
	zap.S().Infof("Fsck done. %d used blobs found.", len(usedbs))
	if err := ctx.Err(); err != nil {
		zap.S().Infof("Detected cancel. Bailing out.")
		return err
	}

	zap.S().Infof("Converting used blob list to a hashset")
	usedbset := make(map[string]struct{})
	for _, b := range usedbs {
		usedbset[b] = struct{}{}
	}
	zap.S().Infof("Convert used blob list to a hashset: Done.")

	if err := ctx.Err(); err != nil {
		zap.S().Infof("Detected cancel. Bailing out.")
		return err
	}

	zap.S().Infof("Listing unused blobpaths.")
	unusedbs := make([]string, 0, util.IntMax(len(allbs)-len(usedbs), 0))
	for _, b := range allbs {
		if _, ok := usedbset[b]; ok {
			continue
		}

		if metadata.IsMetadataBlobpath(b) {
			zap.S().Infof("Marking metadata blobpath as used: %s", b)
			continue
		}

		unusedbs = append(unusedbs, b)
	}

	traceend := time.Now()
	zap.S().Infof("GC Found %d unused blobpaths. (Trace took %v)", len(unusedbs), traceend.Sub(start))

	for _, b := range unusedbs {
		if err := ctx.Err(); err != nil {
			zap.S().Infof("Detected cancel. Bailing out.")
			return err
		}

		if dryrun {
			zap.S().Infof("Dryrun found unused blob: %s", b)
		} else {
			zap.S().Infof("Removing unused blob: %s", b)
			if err := bs.RemoveBlob(b); err != nil {
				return fmt.Errorf("Removing unused blob \"%s\" failed: %v", b, err)
			}
		}
	}
	sweepend := time.Now()
	zap.S().Infof("GC success. Dryrun: %t. (Sweep took %v. The whole GC took %v.)", dryrun, sweepend.Sub(traceend), sweepend.Sub(start))

	return err
}
