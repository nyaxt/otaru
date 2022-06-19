package deleteallblobs

import (
	"bufio"
	"fmt"
	"os"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/facade"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/gcloud/gcs"
)

const notReadOnly = false

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

var Command = &cli.Command{
	Name:  "deleteallblobs",
	Usage: "delete all blobs from specified blobstorage",
	Action: func(c *cli.Context) error {
		s := zap.S().Named("deleteallblobs")

		cfg, err := facade.NewConfig(c.Path("configDir"))
		if err != nil {
			return err
		}

		tsrc, err := auth.GetGCloudTokenSource(cfg.CredentialsFilePath)
		if err != nil {
			return fmt.Errorf("Failed to init GCloudClientSource: %w", err)
		}
		key := btncrypt.KeyFromPassword(cfg.Password)
		cipher, err := btncrypt.NewCipher(key)
		if err != nil {
			return fmt.Errorf("Failed to init *btncrypt.Cipher: %w", err)
		}

		fmt.Printf("Do you really want to proceed with deleting all blobs in gs://%s{,-meta} and its cache in %s?\n", cfg.BucketName, cfg.CacheDir)
		fmt.Printf("Type \"deleteall\" to proceed: ")
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			return fmt.Errorf("Failed to scan line input from stdin")
		}
		if sc.Text() != "deleteall" {
			zap.S().Infof("Cancelled.\n")
			os.Exit(1)
		}

		dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, cipher, tsrc)
		l := datastore.NewGlobalLocker(dscfg, "otaru-deleteallblobs", facade.GenHostName())
		if err := l.Lock(c.Context, notReadOnly); err != nil {
			return fmt.Errorf("Failed to acquire global lock: %w", err)
		}
		defer l.Unlock(c.Context)

		if err := clearGCS(cfg.ProjectName, cfg.BucketName, tsrc); err != nil {
			return fmt.Errorf("Failed to clear bucket \"%s\": %w", cfg.BucketName, err)
		}
		if cfg.UseSeparateBucketForMetadata {
			metabucketname := fmt.Sprintf("%s-meta", cfg.BucketName)
			if err := clearGCS(cfg.ProjectName, metabucketname, tsrc); err != nil {
				return fmt.Errorf("Failed to clear metadata bucket \"%s\": %w", metabucketname, err)
			}
		}
		if err := clearCache(cfg.CacheDir); err != nil {
			return fmt.Errorf("Failed to clear cache \"%s\": %w", cfg.CacheDir, err)
		}

		s.Infof("otaru-deleteallblobs: Successfully completed!")
		s.Infof("Hint: You might also want to run \"otaru-txlogio purge\" to delete inodedb txlogs.")
		return nil
	},
}
