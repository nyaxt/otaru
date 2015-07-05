package gc

import (
	"fmt"
	"log"
	"time"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/util"
)

type GCableBlobStore interface {
	blobstore.BlobLister
	blobstore.BlobRemover
}

func GC(bs GCableBlobStore, idb inodedb.DBFscker, dryrun bool) error {
	start := time.Now()

	log.Printf("GC start. Dryrun: %t. Listing blobs.", dryrun)
	allbs, err := bs.ListBlobs()
	if err != nil {
		return fmt.Errorf("ListBlobs failed: %v", err)
	}
	log.Printf("List blobs done. %d blobs found. Starting INodeDB fsck.", len(allbs))
	usedbs, errs := idb.Fsck()
	if len(errs) != 0 {
		return fmt.Errorf("Fsck returned err: %v", err)
	}
	log.Printf("Fsck done. %d used blobs found.", len(usedbs))

	log.Printf("Converting used blob list to a hashset")
	usedbset := make(map[string]struct{})
	for _, b := range usedbs {
		usedbset[b] = struct{}{}
	}
	log.Printf("Convert used blob list to a hashset: Done.")

	log.Printf("Listing unused blobpaths.")
	unusedbs := make([]string, 0, util.IntMin(len(allbs)-len(usedbs), 0))
	for _, b := range allbs {
		if _, ok := usedbset[b]; ok {
			continue
		}
		unusedbs = append(unusedbs, b)
	}

	traceend := time.Now()
	log.Printf("GC Found %d unused blobpaths. (Trace took %v)", len(unusedbs), traceend.Sub(start))

	for _, b := range unusedbs {
		if dryrun {
			log.Printf("Dryrun found unused blob: %s", b)
		} else {
			log.Printf("Removing unused blob: %s", b)
			if err := bs.RemoveBlob(b); err != nil {
				return fmt.Errorf("Removing unused blob \"%s\" failed: %v", b, err)
			}
		}
	}
	sweepend := time.Now()
	log.Printf("GC success. Dryrun: %t. (Sweep took %v. The whole GC took %v.)", dryrun, sweepend.Sub(traceend), sweepend.Sub(start))

	return err
}
