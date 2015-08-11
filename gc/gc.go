package gc

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/metadata"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("gc")

type GCableBlobStore interface {
	blobstore.BlobLister
	blobstore.BlobRemover
}

func GC(ctx context.Context, bs GCableBlobStore, idb inodedb.DBFscker, dryrun bool) error {
	start := time.Now()

	logger.Infof(mylog, "GC start. Dryrun: %t. Listing blobs.", dryrun)
	allbs, err := bs.ListBlobs()
	if err != nil {
		return fmt.Errorf("ListBlobs failed: %v", err)
	}
	logger.Infof(mylog, "List blobs done. %d blobs found.", len(allbs))
	if err := ctx.Err(); err != nil {
		logger.Infof(mylog, "Detected cancel. Bailing out.")
		return err
	}
	logger.Infof(mylog, "Starting INodeDB fsck.")
	usedbs, errs := idb.Fsck()
	if len(errs) != 0 {
		return fmt.Errorf("Fsck returned err: %v", err)
	}
	logger.Infof(mylog, "Fsck done. %d used blobs found.", len(usedbs))
	if err := ctx.Err(); err != nil {
		logger.Infof(mylog, "Detected cancel. Bailing out.")
		return err
	}

	logger.Infof(mylog, "Converting used blob list to a hashset")
	usedbset := make(map[string]struct{})
	for _, b := range usedbs {
		usedbset[b] = struct{}{}
	}
	logger.Infof(mylog, "Convert used blob list to a hashset: Done.")

	if err := ctx.Err(); err != nil {
		logger.Infof(mylog, "Detected cancel. Bailing out.")
		return err
	}

	logger.Infof(mylog, "Listing unused blobpaths.")
	unusedbs := make([]string, 0, util.IntMax(len(allbs)-len(usedbs), 0))
	for _, b := range allbs {
		if _, ok := usedbset[b]; ok {
			continue
		}

		if metadata.IsMetadataBlobpath(b) {
			logger.Infof(mylog, "Marking metadata blobpath as used: %s", b)
			continue
		}

		unusedbs = append(unusedbs, b)
	}

	traceend := time.Now()
	logger.Infof(mylog, "GC Found %d unused blobpaths. (Trace took %v)", len(unusedbs), traceend.Sub(start))

	for _, b := range unusedbs {
		if err := ctx.Err(); err != nil {
			logger.Infof(mylog, "Detected cancel. Bailing out.")
			return err
		}

		if dryrun {
			logger.Infof(mylog, "Dryrun found unused blob: %s", b)
		} else {
			logger.Infof(mylog, "Removing unused blob: %s", b)
			if err := bs.RemoveBlob(b); err != nil {
				return fmt.Errorf("Removing unused blob \"%s\" failed: %v", b, err)
			}
		}
	}
	sweepend := time.Now()
	logger.Infof(mylog, "GC success. Dryrun: %t. (Sweep took %v. The whole GC took %v.)", dryrun, sweepend.Sub(traceend), sweepend.Sub(start))

	return err
}
