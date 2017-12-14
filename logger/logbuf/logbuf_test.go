package logbuf

import (
	"testing"
	"time"

	"github.com/nyaxt/otaru/logger"
)

func TestLogBuf_WillAccept(t *testing.T) {
	lb := New(3)
	if !lb.WillAccept(logger.Debug) {
		t.Errorf("LogBuf should accept all log levels")
	}
}

type expectedEntry struct {
	Id       int
	Log      string
	Category string
	logger.Level
}

func helperDumpEntries(t *testing.T, es []*Entry) {
	t.Helper()
	for i, e := range es {
		t.Errorf("es[%d]: %v", i, e)
	}
}

func helperAssertEntries(t *testing.T, as []*Entry, es []expectedEntry) {
	t.Helper()

	if len(as) != len(es) {
		helperDumpEntries(t, as)
		t.Fatalf("expected entries len: %d, got len: %d", len(es), len(as))
	}
	for i, e := range es {
		a := as[i]
		if a.Id != e.Id {
			t.Fatalf("es[%d] expected id: %d, got %d", i, e.Id, a.Id)
		}
		if a.Log != e.Log {
			t.Fatalf("es[%d] expected log: %s, got %s", i, e.Log, a.Log)
		}
		if a.Category != e.Category {
			t.Fatalf("es[%d] expected category: %s, got %s", i, e.Category, a.Category)
		}
		if a.Level != e.Level {
			t.Fatalf("es[%d] expected level: %v, got %v", i, e.Level, a.Level)
		}
	}
}

func helperWaitFlush(t *testing.T, lb *LogBuf, id int) {
	t.Helper()

	ticker := time.NewTicker(100 * time.Microsecond)
	defer ticker.Stop()

	for i := 0; i < 3; i++ {
		if lb.LatestEntryId() >= id {
			return
		}
		<-ticker.C
	}

	t.Fatalf("lb.LatestEntryId() never became >= %d", id)
}

func TestLogBuf_Query(t *testing.T) {
	lb := New(300)
	logger.Debugf(lb, "msg1")
	helperWaitFlush(t, lb, 1)

	es := lb.Query(0, []string{}, 100)
	helperAssertEntries(t, es, []expectedEntry{
		{1, "msg1", "", logger.Debug},
	})
	if lb.LatestEntryId() != 1 {
		t.Errorf("LatestEntryId")
	}

	logger.Infof(lb, "msg2")
	logger.Warningf(lb, "msg3")
	helperWaitFlush(t, lb, 3)

	es = lb.Query(0, []string{}, 100)
	helperAssertEntries(t, es, []expectedEntry{
		{1, "msg1", "", logger.Debug},
		{2, "msg2", "", logger.Info},
		{3, "msg3", "", logger.Warning},
	})
	if lb.LatestEntryId() != 3 {
		t.Errorf("LatestEntryId")
	}

	es = lb.Query(2, []string{}, 100)
	helperAssertEntries(t, es, []expectedEntry{
		{2, "msg2", "", logger.Info},
		{3, "msg3", "", logger.Warning},
	})

	es = lb.Query(2, []string{}, 1)
	helperAssertEntries(t, es, []expectedEntry{
		{2, "msg2", "", logger.Info},
	})
}
