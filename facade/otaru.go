package facade

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/chunkstore"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/inodedb/blobstoredbstatesnapshotio"
	"github.com/nyaxt/otaru/metadata"
	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
)

type Otaru struct {
	C btncrypt.Cipher

	S *scheduler.Scheduler

	Clisrc auth.ClientSource
	DSCfg  *datastore.Config
	GL     *datastore.GlobalLocker

	MetadataBS blobstore.BlobStore
	DefaultBS  blobstore.BlobStore

	BackendBS blobstore.BlobStore

	CacheTgtBS *blobstore.FileBlobStore
	CBS        *cachedblobstore.CachedBlobStore
	CSS        *util.PeriodicRunner

	SSLoc blobstoredbstatesnapshotio.SSLocator
	SIO   *blobstoredbstatesnapshotio.DBStateSnapshotIO

	TxIO   inodedb.DBTransactionLogIO
	TxIOSS *util.PeriodicRunner

	IDBBE *inodedb.DB
	IDBS  *inodedb.DBService
	IDBSS *util.PeriodicRunner

	FS   *otaru.FileSystem
	MGMT *mgmt.Server
}

func NewOtaru(cfg *Config, oneshotcfg *OneshotConfig) (*Otaru, error) {
	o := &Otaru{}

	var err error

	key := btncrypt.KeyFromPassword(cfg.Password)
	o.C, err = btncrypt.NewCipher(key)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init Cipher: %v", err)
	}

	o.S = scheduler.NewScheduler()

	if !cfg.LocalDebug {
		o.Clisrc, err = auth.GetGCloudClientSource(cfg.CredentialsFilePath, cfg.TokenCacheFilePath, false)
		if err != nil {
			o.Close()
			return nil, fmt.Errorf("Failed to init GCloudClientSource: %v", err)
		}
		o.DSCfg = datastore.NewConfig(cfg.ProjectName, cfg.BucketName, o.C, o.Clisrc)
		o.GL = datastore.NewGlobalLocker(o.DSCfg, genHostName(), "FIXME: fill info")
		if err := o.GL.Lock(); err != nil {
			return nil, err
		}
	}

	o.CacheTgtBS, err = blobstore.NewFileBlobStore(cfg.CacheDir, oflags.O_RDWRCREATE)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init FileBlobStore: %v", err)
	}

	if !cfg.LocalDebug {
		o.DefaultBS, err = gcs.NewGCSBlobStore(cfg.ProjectName, cfg.BucketName, o.Clisrc, oflags.O_RDWRCREATE)
		if err != nil {
			o.Close()
			return nil, fmt.Errorf("Failed to init GCSBlobStore: %v", err)
		}
		if !cfg.UseSeparateBucketForMetadata {
			o.BackendBS = o.DefaultBS
		} else {
			metabucketname := fmt.Sprintf("%s-meta", cfg.BucketName)
			o.MetadataBS, err = gcs.NewGCSBlobStore(cfg.ProjectName, metabucketname, o.Clisrc, oflags.O_RDWRCREATE)
			if err != nil {
				o.Close()
				return nil, fmt.Errorf("Failed to init GCSBlobStore (metadata): %v", err)
			}

			o.BackendBS = blobstore.Mux{
				blobstore.MuxEntry{metadata.IsMetadataBlobpath, o.MetadataBS},
				blobstore.MuxEntry{nil, o.DefaultBS},
			}
		}
	} else {
		o.BackendBS, err = blobstore.NewFileBlobStore(path.Join(os.Getenv("HOME"), ".otaru", "bbs"), oflags.O_RDWRCREATE)
	}

	queryFn := chunkstore.NewQueryChunkVersion(o.C)
	o.CBS, err = cachedblobstore.New(o.BackendBS, o.CacheTgtBS, oflags.O_RDWRCREATE /* FIXME */, queryFn)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("Failed to init CachedBlobStore: %v", err)
	}
	if err := o.CBS.RestoreState(o.C); err != nil {
		log.Printf("Warning: Attempted to restore cachedblobstore state but failed: %v", err)
	}
	o.CSS = cachedblobstore.NewCacheSyncScheduler(o.CBS)

	if !cfg.LocalDebug {
		o.SSLoc = datastore.NewINodeDBSSLocator(o.DSCfg)
	} else {
		panic("Implement mock sslocator that doesn't depend on gcloud/datastore")
	}
	o.SIO = blobstoredbstatesnapshotio.New(o.CBS, o.C, o.SSLoc)

	if !cfg.LocalDebug {
		txio := datastore.NewDBTransactionLogIO(o.DSCfg)
		o.TxIO = o.TxIO
		o.TxIOSS = util.NewSyncScheduler(txio, 300*time.Millisecond)
	} else {
		o.TxIO = inodedb.NewSimpleDBTransactionLogIO()
	}

	if oneshotcfg.Mkfs {
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
	o.IDBSS = util.NewSyncScheduler(o.IDBS, 30*time.Second)

	o.FS = otaru.NewFileSystem(o.IDBS, o.CBS, o.C)
	o.MGMT = mgmt.NewServer()
	if err := o.runMgmtServer(); err != nil {
		o.Close()
		return nil, fmt.Errorf("Mgmt server run failed: %v", err)
	}

	return o, nil
}

func (o *Otaru) Close() error {
	errs := []error{}

	if o.S != nil {
		o.S.AbortAllAndStop()
	}

	if o.FS != nil {
		if err := o.FS.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IDBSS != nil {
		o.IDBSS.Stop()
	}

	if o.IDBS != nil {
		o.IDBS.Quit()
	}

	if o.IDBBE != nil {
		if err := o.IDBBE.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.CSS != nil {
		o.CSS.Stop()
	}

	if o.CBS != nil {
		if err := o.CBS.SaveState(o.C); err != nil {
			errs = append(errs, err)
		}
	}

	if o.GL != nil {
		if err := o.GL.Unlock(); err != nil {
			errs = append(errs, err)
		}
	}

	return util.ToErrors(errs)
}
