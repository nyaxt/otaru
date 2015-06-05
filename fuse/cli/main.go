package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	bfuse "bazil.org/fuse"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/fuse"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/util"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	flagMkfs         = flag.Bool("mkfs", false, "Reset metadata if no existing metadata exists")
	flagPasswordFile = flag.String("passwordFile", path.Join(os.Getenv("HOME"), ".otaru", "password.txt"), "Path of a text file storing password")
	flagProjectName  = flag.String("projectName", "", "google cloud project name")
	flagBucketName   = flag.String("bucketName", "", "google cloud storage bucket name")
	flagCacheDir     = flag.String("cachedir", "/var/cache/otaru", "Path to blob cache dir")
)

type Otaru struct {
	C btncrypt.Cipher

	Clisrc auth.ClientSource

	FBS *blobstore.FileBlobStore
	GBS *gcs.GCSBlobStore
	CBS *blobstore.CachedBlobStore

	FS *otaru.FileSystem
}

func NewOtaru(mkfs bool, password string, projectName string, bucketName string, cacheDir string) (*Otaru, error) {
	o := &Otaru{}

	var err error

	key := btncrypt.KeyFromPassword(password)
	o.C, err = btncrypt.NewCipher(key)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init Cipher: %v", err)
	}

	o.Clisrc, err = auth.GetGCloudClientSource(
		path.Join(os.Getenv("HOME"), ".otaru", "credentials.json"),
		path.Join(os.Getenv("HOME"), ".otaru", "tokencache.json"),
		false)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init GCloudClientSource: %v", err)
	}

	o.FBS, err = blobstore.NewFileBlobStore(cacheDir, oflags.O_RDWRCREATE)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init FileBlobStore: %v", err)
	}

	o.GBS, err = gcs.NewGCSBlobStore(projectName, bucketName, clisrc)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init GCSBlobStore: %v", err)
	}

	o.CBS, err = blobstore.NewCachedBlobStore(o.GBS, o.FBS, oflags.O_RDWRCREATE /* FIXME */, otaru.QueryChunkVersion)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init CachedBlobStore: %v", err)
	}

	c.FS, err = otaru.NewFileSystemFromSnapshot(o.CBS, o.C)
	if err != nil {
		if err == otaru.ENOENT && mkfs {
			c.FS, err = otaru.NewFileSystemEmpty(o.CBS, o.C)
			if err != nil {
				o.Close()
				return nil, fmt.Errorf("NewFileSystemEmpty failed: %v", err)
			}
		} else {
			o.Close()
			return nil, fmt.Errorf("NewFileSystemFromSnapshot failed: %v", err)
		}
	}

	return o, nil
}

type OtaruCloseErrors []error

func (errs OtaruCloseErrors) Error() string {
	errstrs := make([]string, len(errs))
	for i, e := range errs {
		errstrs[i] = errs[i].Error()
	}

	return "Errors: [" + strings.Join(errstrs, ", ") + "]"
}

func (o *Otaru) Close() error {
	errs := []error{}

	if o.FS != nil {
		if err := o.FS.Close(); err != nil {
			errs := append(errs, err)
		}
	}

	if o.CBS != nil {
		if err := o.CBS.Close(); err != nil {
			errs := append(errs, err)
		}
	}

	if o.GBS != nil {
		if err := o.GBS.Close(); err != nil {
			errs := append(errs, err)
		}
	}

	if o.FBS != nil {
		if err := o.FBS.Close(); err != nil {
			errs := append(errs, err)
		}
	}

	return nil
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	password := util.StringFromFile(*flagPasswordFile, "password")
	if *flagProjectName == "" {
		log.Printf("Please specify a valid project name")
		Usage()
		os.Exit(2)
	}
	if *flagBucketName == "" {
		log.Printf("Please specify a valid bucket name")
		Usage()
		os.Exit(2)
	}
	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	otaru := NewOtaru(*flagMkfs, password, *flagProjectName, *flagBucketName, *flagCacheDir)

	bfuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}
	if err := fuse.ServeFUSE(mountpoint, o.FS, nil); err != nil {
		log.Fatalf("ServeFUSE failed: %v", err)
	}
}
