package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/facade"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/logger"
)

const notReadOnly = false

var mylog = logger.Registry().Category("otaru-globallock")

var (
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options]\n", os.Args[0])
	flag.PrintDefaults()
}

type BlobListerRemover interface {
	blobstore.BlobLister
	blobstore.BlobRemover
}

func clearBlobStore(bs BlobListerRemover) error {
	bps, err := bs.ListBlobs()
	if err != nil {
		return fmt.Errorf("Failed to ListBlobs(): %v", err)
	}
	zap.S().Infof("Found %d blobs!", len(bps))

	for i, bp := range bps {
		zap.S().Infof("Removing blob %d/%d: %s", i+1, len(bps), bp)
		if err := bs.RemoveBlob(bp); err != nil {
			return fmt.Errorf("Failed to RemoveBlob(%s): %v", bp, err)
		}
	}
	return nil
}

func clearCache(cacheDir string) error {
	bs, err := blobstore.NewFileBlobStore(cacheDir, oflags.O_RDWRCREATE)
	if err != nil {
		return fmt.Errorf("Failed to init FileBlobStore: %v", err)
	}

	return clearBlobStore(bs)
}

func clearGCS(projectName, bucketName string, tsrc oauth2.TokenSource) error {
	bs, err := gcs.NewGCSBlobStore(projectName, bucketName, tsrc, oflags.O_RDWRCREATE)
	if err != nil {
		return fmt.Errorf("Failed to init GCSBlobStore: %v", err)
	}

	return clearBlobStore(bs)
}

func main() {
	logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
	logger.Registry().AddOutput(logger.HandleCritical(func() { os.Exit(1) }))

	flag.Usage = Usage
	flag.Parse()
	if flag.NArg() != 0 {
		Usage()
		os.Exit(1)
	}

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		zap.S().Infof("%v", err)
		Usage()
		os.Exit(1)
	}

	tsrc, err := auth.GetGCloudTokenSource(cfg.CredentialsFilePath)
	if err != nil {
		zap.S().Errorf("Failed to init GCloudClientSource: %v", err)
	}
	key := btncrypt.KeyFromPassword(cfg.Password)
	c, err := btncrypt.NewCipher(key)
	if err != nil {
		zap.S().Errorf("Failed to init *btncrypt.Cipher: %v", err)
	}

	fmt.Printf("Do you really want to proceed with deleting all blobs in gs://%s{,-meta} and its cache in %s?\n", cfg.BucketName, cfg.CacheDir)
	fmt.Printf("Type \"deleteall\" to proceed: ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return
	}
	if sc.Text() != "deleteall" {
		zap.S().Infof("Cancelled.\n")
		os.Exit(1)
	}

	ctx := context.Background()

	dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, c, tsrc)
	l := datastore.NewGlobalLocker(dscfg, "otaru-deleteallblobs", facade.GenHostName())
	if err := l.Lock(ctx, notReadOnly); err != nil {
		zap.S().Infof("Failed to acquire global lock: %v", err)
		return
	}
	defer l.Unlock(ctx)

	if err := clearGCS(cfg.ProjectName, cfg.BucketName, tsrc); err != nil {
		zap.S().Infof("Failed to clear bucket \"%s\": %v", cfg.BucketName, err)
		return
	}
	if cfg.UseSeparateBucketForMetadata {
		metabucketname := fmt.Sprintf("%s-meta", cfg.BucketName)
		if err := clearGCS(cfg.ProjectName, metabucketname, tsrc); err != nil {
			zap.S().Infof("Failed to clear metadata bucket \"%s\": %v", metabucketname, err)
			return
		}
	}
	if err := clearCache(cfg.CacheDir); err != nil {
		zap.S().Infof("Failed to clear cache \"%s\": %v", cfg.CacheDir, err)
		return
	}

	zap.S().Infof("otaru-deleteallblobs: Successfully completed!")
	zap.S().Infof("Hint: You might also want to run \"otaru-txlogio purge\" to delete inodedb txlogs.")
}
