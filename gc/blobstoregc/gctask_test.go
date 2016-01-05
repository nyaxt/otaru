package blobstoregc_test

import (
	"github.com/nyaxt/otaru/gc/blobstoregc"
	"github.com/nyaxt/otaru/scheduler"
)

var _ = scheduler.Task(&blobstoregc.GCTask{})
