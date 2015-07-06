package gc_test

import (
	"github.com/nyaxt/otaru/gc"
	"github.com/nyaxt/otaru/scheduler"
)

var _ = scheduler.Task(&gc.GCTask{})
