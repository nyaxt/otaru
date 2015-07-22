package facade

import (
	"log"

	"github.com/nyaxt/otaru/gcloud/gcs"
	"github.com/nyaxt/otaru/mgmt/mblobstore"
	"github.com/nyaxt/otaru/mgmt/mgc"
	"github.com/nyaxt/otaru/mgmt/mgcsblobstore"
	"github.com/nyaxt/otaru/mgmt/minodedb"
	"github.com/nyaxt/otaru/mgmt/mscheduler"
)

func (o *Otaru) setupMgmtAPIs() error {
	mblobstore.Install(o.MGMT, o.DefaultBS, o.CBS)
	if gcsbs, ok := o.DefaultBS.(*gcs.GCSBlobStore); ok {
		mgcsblobstore.Install(o.MGMT, gcsbs)
	}
	minodedb.Install(o.MGMT, o.IDBS)
	mscheduler.Install(o.MGMT, o.S)
	mgc.Install(o.MGMT, o.S, o.CBS, o.IDBS)

	return nil
}

func (o *Otaru) runMgmtServer() error {
	if err := o.setupMgmtAPIs(); err != nil {
		return err
	}

	go func() {
		if err := o.MGMT.Run(); err != nil {
			log.Fatalf("mgmt httpd died")
		}
	}()
	return nil
}
