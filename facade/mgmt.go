package facade

import (
	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/mgmt/mblobstore"
	"github.com/nyaxt/otaru/mgmt/mdebug"
	"github.com/nyaxt/otaru/mgmt/mfilesystem"
	"github.com/nyaxt/otaru/mgmt/mgc"
	"github.com/nyaxt/otaru/mgmt/mgcsblobstore"
	"github.com/nyaxt/otaru/mgmt/minodedb"
	"github.com/nyaxt/otaru/mgmt/mlogger"
	"github.com/nyaxt/otaru/mgmt/mscheduler"
	"github.com/nyaxt/otaru/mgmt/msystem"
)

func (o *Otaru) setupMgmtAPIs(cfg *Config) error {
	mlogger.Install(o.MGMT)
	if cfg.InstallDebugApi {
		logger.Infof(mylog, "Installing Debug APIs.")
		mdebug.Install(o.MGMT)
	}
	msystem.Install(o.MGMT)
	mblobstore.Install(o.MGMT, o.S, o.DefaultBS, o.CBS)
	if gcsbs, ok := o.DefaultBS.(*gcs.GCSBlobStore); ok {
		mgcsblobstore.Install(o.MGMT, gcsbs)
	}
	minodedb.Install(o.MGMT, o.IDBS)
	mscheduler.Install(o.MGMT, o.S, o.R)
	mfilesystem.Install(o.MGMT, o.FS)
	mgc.Install(o.MGMT, o.S, o.CBS, o.IDBS)

	return nil
}

func (o *Otaru) runMgmtServer(cfg *Config) error {
	if err := o.setupMgmtAPIs(cfg); err != nil {
		return err
	}

	go func() {
		if err := o.MGMT.Run(); err != nil {
			logger.Criticalf(mylog, "mgmt httpd died")
		}
	}()
	return nil
}
