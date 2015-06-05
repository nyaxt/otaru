package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	bfuse "bazil.org/fuse"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/fuse"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/inodedb"
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

	SIO  *otaru.BlobStoreDBStateSnapshotIO
	TxIO *datastore.DBTransactionLogIO
	IDB  *inodedb.DB

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

	o.GBS, err = gcs.NewGCSBlobStore(projectName, bucketName, o.Clisrc, oflags.O_RDWRCREATE)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init GCSBlobStore: %v", err)
	}

	queryFn := otaru.NewQueryChunkVersion(o.C)
	o.CBS, err = blobstore.NewCachedBlobStore(o.GBS, o.FBS, oflags.O_RDWRCREATE /* FIXME */, queryFn)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init CachedBlobStore: %v", err)
	}

	o.SIO = otaru.NewBlobStoreDBStateSnapshotIO(o.CBS, o.C)

	o.TxIO, err = datastore.NewDBTransactionLogIO(projectName, bucketName, o.C, o.Clisrc)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init gcloud DBTransactionLogIO: %v", err)
	}

	if mkfs {
		o.IDB, err = inodedb.NewEmptyDB(o.SIO, o.TxIO)
		if err != nil {
			o.Close()
			return nil, fmt.Errorf("NewEmptyDB failed: %v", err)
		}
	} else {
		o.IDB, err = inodedb.NewDB(o.SIO, o.TxIO)
		if err != nil {
			o.Close()
			return nil, fmt.Errorf("NewDB failed: %v", err)
		}
	}

	o.FS = otaru.NewFileSystem(o.IDB, o.CBS, o.C)

	return o, nil
}

type OtaruCloseErrors []error

func (errs OtaruCloseErrors) Error() string {
	errstrs := make([]string, len(errs))
	for i, e := range errs {
		errstrs[i] = e.Error()
	}

	return "Errors: [" + strings.Join(errstrs, ", ") + "]"
}

func (o *Otaru) Close() error {
	errs := []error{}

	if o.FS != nil {
		if err := o.FS.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IDB != nil {
		if err := o.IDB.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return OtaruCloseErrors(errs)
	}
	return nil
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	password := util.StringFromFileOrDie(*flagPasswordFile, "password")
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

	o, err := NewOtaru(*flagMkfs, password, *flagProjectName, *flagBucketName, *flagCacheDir)
	if err != nil {
		log.Printf("NewOtaru failed: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := o.Close(); err != nil {
			log.Printf("Otaru.Close() returned errs: %v", err)
		}
	}()

	bfuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}
	if err := fuse.ServeFUSE(mountpoint, o.FS, nil); err != nil {
		log.Fatalf("ServeFUSE failed: %v", err)
	}
}
