package facade

import (
	"fmt"
	"os"
	"path"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/util"
)

type Otaru struct {
	C btncrypt.Cipher

	Clisrc auth.ClientSource

	FBS *blobstore.FileBlobStore
	BBS blobstore.BlobStore
	CBS *blobstore.CachedBlobStore

	SIO   *otaru.BlobStoreDBStateSnapshotIO
	TxIO  inodedb.DBTransactionLogIO
	IDBBE *inodedb.DB
	IDBS  *inodedb.DBService

	FS *otaru.FileSystem
}

func NewOtaru(mkfs bool, password string, projectName string, bucketName string, cacheDir string, localDebug bool) (*Otaru, error) {
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

	if !localDebug {
		o.BBS, err = gcs.NewGCSBlobStore(projectName, bucketName, o.Clisrc, oflags.O_RDWRCREATE)
	} else {
		o.BBS, err = blobstore.NewFileBlobStore(path.Join(os.Getenv("HOME"), ".otaru", "bbs"), oflags.O_RDWRCREATE)
	}
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init GCSBlobStore: %v", err)
	}

	queryFn := otaru.NewQueryChunkVersion(o.C)
	o.CBS, err = blobstore.NewCachedBlobStore(o.BBS, o.FBS, oflags.O_RDWRCREATE /* FIXME */, queryFn)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init CachedBlobStore: %v", err)
	}

	o.SIO = otaru.NewBlobStoreDBStateSnapshotIO(o.CBS, o.C)

	if !localDebug {
		o.TxIO, err = datastore.NewDBTransactionLogIO(projectName, bucketName, o.C, o.Clisrc)
	} else {
		o.TxIO = inodedb.NewSimpleDBTransactionLogIO()
		err = nil
	}
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init gcloud DBTransactionLogIO: %v", err)
	}

	if mkfs {
		o.IDBBE, err = inodedb.NewEmptyDB(o.SIO, o.TxIO)
		if err != nil {
			o.Close()
			return nil, fmt.Errorf("NewEmptyDB failed: %v", err)
		}
	} else {
		o.IDBBE, err = inodedb.NewDB(o.SIO, o.TxIO)
		if err != nil {
			o.Close()
			return nil, fmt.Errorf("NewDB failed: %v", err)
		}
	}

	o.IDBS = inodedb.NewDBService(o.IDBBE)
	o.FS = otaru.NewFileSystem(o.IDBS, o.CBS, o.C)

	return o, nil
}

func (o *Otaru) Close() error {
	errs := []error{}

	if o.FS != nil {
		if err := o.FS.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IDBS != nil {
		o.IDBS.Quit()
	}

	if o.IDBBE != nil {
		if err := o.IDBBE.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	return util.ToErrors(errs)
}
