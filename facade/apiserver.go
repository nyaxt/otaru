package facade

import (
	"github.com/nyaxt/otaru/apiserver"
	/*
		  "github.com/nyaxt/otaru/logger"
			"github.com/nyaxt/otaru/gcloud/gcs"
			"github.com/nyaxt/otaru/mgmt/mblobstore"
			"github.com/nyaxt/otaru/mgmt/mdebug"
			"github.com/nyaxt/otaru/mgmt/mfilesystem"
			"github.com/nyaxt/otaru/mgmt/mgc"
			"github.com/nyaxt/otaru/mgmt/mgcsblobstore"
			"github.com/nyaxt/otaru/mgmt/minodedb"
			"github.com/nyaxt/otaru/mgmt/mlogger"
			"github.com/nyaxt/otaru/mgmt/mscheduler"
			"github.com/nyaxt/otaru/mgmt/msystem"
	*/)

func (o *Otaru) buildApiServerOptions(cfg *Config) []apiserver.Option {
	options := []apiserver.Option{
		apiserver.ListenAddr(cfg.HttpApiAddr),
		apiserver.InstallSystemService(),
	}
	/*
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
		mgc.Install(o.MGMT, o.S, o)
	*/

	return options
}
