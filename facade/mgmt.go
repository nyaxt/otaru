package facade

import (
	"log"

	"github.com/nyaxt/otaru/mgmt/mblobstore"
)

func (o *Otaru) setupMgmtAPIs() error {
	mblobstore.Install(o.MGMT, o.CBS)

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
