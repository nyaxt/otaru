package facade

import (
	"fmt"
	"os"
	"path"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/chunkstore"
	"github.com/nyaxt/otaru/filesystem"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/fuse"
	"github.com/nyaxt/otaru/gc/blobstoregc"
	"github.com/nyaxt/otaru/gc/inodedbssgc"
	"github.com/nyaxt/otaru/gc/inodedbtxloggc"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/inodedb/blobstoredbstatesnapshotio"
	"github.com/nyaxt/otaru/inodedb/inodedbsyncer"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/metadata"
	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
	"github.com/nyaxt/otaru/webdav"
)

var mylog = logger.Registry().Category("facade")

type Otaru struct {
	ReadOnly bool

	C *btncrypt.Cipher

	S *scheduler.Scheduler
	R *scheduler.RepetitiveJobRunner

	Tsrc  oauth2.TokenSource
	DSCfg *datastore.Config
	GL    *datastore.GlobalLocker

	MetadataBS blobstore.BlobStore
	DefaultBS  blobstore.BlobStore

	BackendBS blobstore.BlobStore

	CacheTgtBS         *blobstore.FileBlobStore
	CBS                *cachedblobstore.CachedBlobStore
	AutoReduceCacheJob scheduler.ID
	SaveStateJob       scheduler.ID

	SSLoc blobstoredbstatesnapshotio.SSLocator
	SIO   *blobstoredbstatesnapshotio.DBStateSnapshotIO

	TxIO        inodedb.DBTransactionLogIO
	CTxIO       inodedb.DBTransactionLogIO
	TxIOSyncJob scheduler.ID

	IDBBE      *inodedb.DB
	IDBS       *inodedb.DBService
	IDBSyncJob scheduler.ID

	FS *otaru.FileSystem

	AutoBlobstoreGCJob    scheduler.ID
	AutoINodeDBTxLogGCJob scheduler.ID
	AutoINodeDBSSGCJob    scheduler.ID
}

func BootstrapLogger() {
	logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
}

func Serve(cfg *Config, oneshotcfg *OneshotConfig, closeC <-chan error) error {
	o := &Otaru{}
	defer o.Close()

	if err := SetupFluentLogger(cfg); err != nil {
		return fmt.Errorf("Failed to setup fluentd logger: %v", err)
	}

	o.ReadOnly = cfg.ReadOnly

	flags := oflags.O_RDWRCREATE
	if o.ReadOnly {
		logger.Infof(mylog, "Otaru in read only mode.")
		flags = oflags.O_RDONLY
	}

	var err error

	key := btncrypt.KeyFromPassword(cfg.Password)
	o.C, err = btncrypt.NewCipher(key)
	if err != nil {
		return fmt.Errorf("Failed to init Cipher: %v", err)
	}

	o.S = scheduler.NewScheduler()
	o.R = scheduler.NewRepetitiveJobRunner(o.S)

	if !cfg.LocalDebug {
		o.Tsrc, err = auth.GetGCloudTokenSource(context.TODO(), cfg.CredentialsFilePath, cfg.TokenCacheFilePath, false)
		if err != nil {
			return fmt.Errorf("Failed to init GCloudClientSource: %v", err)
		}
		o.DSCfg = datastore.NewConfig(cfg.ProjectName, cfg.BucketName, o.C, o.Tsrc)
		o.GL = datastore.NewGlobalLocker(o.DSCfg, GenHostName(), "FIXME: fill info")
		if err := o.GL.Lock(o.ReadOnly); err != nil {
			return err
		}
	}

	o.CacheTgtBS, err = blobstore.NewFileBlobStore(cfg.CacheDir, oflags.O_RDWRCREATE)
	if err != nil {
		return fmt.Errorf("Failed to init FileBlobStore: %v", err)
	}

	if !cfg.LocalDebug {
		o.DefaultBS, err = gcs.NewGCSBlobStore(cfg.ProjectName, cfg.BucketName, o.Tsrc, flags)
		if err != nil {
			return fmt.Errorf("Failed to init GCSBlobStore: %v", err)
		}
		if !cfg.UseSeparateBucketForMetadata {
			o.BackendBS = o.DefaultBS
		} else {
			metabucketname := fmt.Sprintf("%s-meta", cfg.BucketName)
			o.MetadataBS, err = gcs.NewGCSBlobStore(cfg.ProjectName, metabucketname, o.Tsrc, flags)
			if err != nil {
				return fmt.Errorf("Failed to init GCSBlobStore (metadata): %v", err)
			}

			o.BackendBS = blobstore.Mux{
				blobstore.MuxEntry{metadata.IsMetadataBlobpath, o.MetadataBS},
				blobstore.MuxEntry{nil, o.DefaultBS},
			}
		}
	} else {
		o.BackendBS, err = blobstore.NewFileBlobStore(path.Join(DefaultConfigDir(), "bbs"), flags)
		if err != nil {
			return fmt.Errorf("Failed to init FileBlobStore (backend for local debugging): %v", err)
		}
	}

	queryFn := chunkstore.NewQueryChunkVersion(o.C)
	o.CBS, err = cachedblobstore.New(o.BackendBS, o.CacheTgtBS, o.S, flags, queryFn)
	if err != nil {
		return fmt.Errorf("Failed to init CachedBlobStore: %v", err)
	}
	if err := o.CBS.RestoreState(o.C); err != nil {
		logger.Warningf(mylog, "Attempted to restore cachedblobstore state but failed: %v", err)
	}
	o.AutoReduceCacheJob = cachedblobstore.SetupAutoReduceCache(o.CBS, o.R, cfg.CacheHighWatermarkInBytes, cfg.CacheLowWatermarkInBytes)
	if !o.ReadOnly {
		o.SaveStateJob = o.R.RunEveryPeriod(cachedblobstore.SaveStateTask{o.CBS, o.C}, 30*time.Second)
	}

	if !cfg.LocalDebug {
		o.SSLoc = datastore.NewINodeDBSSLocator(o.DSCfg, flags)
	} else {
		o.SSLoc = blobstoredbstatesnapshotio.SimpleSSLocator{}
	}
	o.SIO = blobstoredbstatesnapshotio.New(o.CBS, o.C, o.SSLoc)

	if !cfg.LocalDebug {
		txio := datastore.NewDBTransactionLogIO(o.DSCfg, flags)
		o.TxIO = txio
		if !cfg.ReadOnly {
			o.TxIOSyncJob = o.R.SyncEveryPeriod(txio, 300*time.Millisecond)
		}
	} else {
		o.TxIO = inodedb.NewSimpleDBTransactionLogIO()
	}
	o.CTxIO = inodedb.NewCachedDBTransactionLogIO(o.TxIO)

	if oneshotcfg.Mkfs {
		o.IDBBE, err = inodedb.NewEmptyDB(o.SIO, o.CTxIO)
		if err != nil {
			return fmt.Errorf("NewEmptyDB failed: %v", err)
		}
	} else {
		o.IDBBE, err = inodedb.NewDB(o.SIO, o.CTxIO, cfg.ReadOnly)
		if err != nil {
			return fmt.Errorf("NewDB failed: %v", err)
		}
	}

	o.IDBS = inodedb.NewDBService(o.IDBBE)
	if !cfg.ReadOnly {
		o.IDBSyncJob = o.R.RunEveryPeriod(inodedbsyncer.NewSyncTask(o.IDBS), 30*time.Second)
	}

	o.FS = otaru.NewFileSystem(o.IDBS, o.CBS, o.C)

	if o.ReadOnly {
		logger.Infof(mylog, "No GC tasks are scheduled in read only mode.")
	} else if cfg.GCPeriod <= 0 {
		logger.Infof(mylog, "GCPeriod %d <= 0. No GC tasks are scheduled automatically.", cfg.GCPeriod)
	} else {
		const NoDryRun = false
		if t := o.GetBlobstoreGCTask(NoDryRun); t != nil {
			o.AutoBlobstoreGCJob = o.R.RunEveryPeriod(t, time.Duration(cfg.GCPeriod)*time.Second)
		}
		if t := o.GetINodeDBTxLogGCTask(NoDryRun); t != nil {
			o.AutoINodeDBTxLogGCJob = o.R.RunEveryPeriod(t, time.Duration(cfg.GCPeriod)*time.Second)
		}
		if t := o.GetINodeDBSSGCTask(NoDryRun); t != nil {
			o.AutoINodeDBSSGCJob = o.R.RunEveryPeriod(t, time.Duration(cfg.GCPeriod)*time.Second)
		}
	}

	apiCloseC := make(chan struct{})
	defer close(apiCloseC)
	apiErrC := make(chan error)
	apiopts := append(o.buildApiServerOptions(cfg), apiserver.CloseChannel(apiCloseC))
	go func() {
		if err := apiserver.Serve(apiopts...); err != nil {
			apiErrC <- err
		}
		close(apiErrC)
	}()

	fuseErrC := make(chan error)
	if cfg.FuseMountPoint != "" {
		fuseCloseC := make(chan struct{})
		defer close(fuseCloseC)

		go func() {
			if err := fuse.Serve(cfg.BucketName, cfg.FuseMountPoint, o.FS, nil, fuseCloseC); err != nil {
				fuseErrC <- err
			}
			close(fuseErrC)
		}()
	}

	webdavErrC := make(chan error)
	// FIXME: webdavCloseC
	if cfg.WebdavAddr != "" {
		go func() {
			if err := webdav.Serve(cfg.WebdavAddr, o.FS); err != nil {
				webdavErrC <- err
			}
			close(webdavErrC)
		}()
	}

	select {
	case err := <-apiErrC:
		if err == nil {
			logger.Infof(mylog, "Apiserver shutdown detected.")
		} else {
			return fmt.Errorf("Apiserver error: %v", err)
		}

	case err := <-fuseErrC:
		if err == nil {
			logger.Infof(mylog, "Fuse shutdown detected.")
		} else {
			return fmt.Errorf("Fuse error: %v", err)
		}

	case err := <-webdavErrC:
		if err == nil {
			logger.Infof(mylog, "WebDav shutdown detected.")
		} else {
			return fmt.Errorf("WebDav error: %v", err)
		}

	case err := <-closeC:
		if err == nil {
			logger.Infof(mylog, "Shutdown requested.")
		} else {
			return fmt.Errorf("Shutdown requested. Cause: %v", err)
		}
	}

	return nil
}

func (o *Otaru) Close() error {
	errs := []error{}

	if o.R != nil {
		o.R.Stop()
	}

	if o.S != nil {
		o.S.AbortAllAndStop()
	}

	if o.FS != nil && !o.ReadOnly {
		if err := o.FS.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IDBS != nil {
		o.IDBS.Quit()
	}

	if o.IDBBE != nil && !o.ReadOnly {
		if err := o.IDBBE.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.CBS != nil {
		if !o.ReadOnly {
			if err := o.CBS.SaveState(o.C); err != nil {
				errs = append(errs, err)
			}
		}
		if err := o.CBS.Quit(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.GL != nil && !o.ReadOnly {
		if err := o.GL.Unlock(); err != nil {
			errs = append(errs, err)
		}
	}

	return util.ToErrors(errs)
}

func (o *Otaru) GetBlobstoreGCTask(dryrun bool) scheduler.Task {
	return &blobstoregc.Task{o.CBS, o.IDBS, dryrun}
}

func (o *Otaru) GetINodeDBTxLogGCTask(dryrun bool) scheduler.Task {
	logdeleter, ok := o.TxIO.(inodedbtxloggc.TransactionLogDeleter)
	if ok {
		return &inodedbtxloggc.Task{o.SIO, logdeleter, dryrun}
	} else {
		logger.Infof(mylog, "DBTransactionLogIO backend %s doesn't support log deletion. Not scheduling txlog GC task.", util.TryGetImplName(o.TxIO))
		return nil
	}
}

func (o *Otaru) GetINodeDBSSGCTask(dryrun bool) scheduler.Task {
	return &inodedbssgc.Task{o.SIO, dryrun}
}
