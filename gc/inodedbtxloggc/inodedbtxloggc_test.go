package inodedbtxloggc_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/gc/inodedbtxloggc"
	"github.com/nyaxt/otaru/inodedb"
	tu "github.com/nyaxt/otaru/testutils"
)

func init() { tu.EnsureLogger() }

type MockUnneededTxIDThresholdFinder int64

func (n MockUnneededTxIDThresholdFinder) FindUnneededTxIDThreshold() (int64, error) {
	return int64(n), nil
}

type MockTransactionLogDeleter struct {
	called bool
	id     inodedb.TxID
}

func (logdeleter *MockTransactionLogDeleter) DeleteTransactions(id inodedb.TxID) error {
	logdeleter.called = true
	logdeleter.id = id
	return nil
}

func TestINodeDBTxLogGC_DryRun(t *testing.T) {
	thresfinder := MockUnneededTxIDThresholdFinder(345)
	logdeleter := &MockTransactionLogDeleter{called: false, id: inodedb.AnyVersion}

	if err := inodedbtxloggc.GC(context.TODO(), thresfinder, logdeleter, true); err != nil {
		t.Errorf("GC err: %v", err)
	}

	if logdeleter.called {
		t.Errorf("Dry run invoked log deleter!")
	}
}

func TestINodeDBTxLogGC_RealRun(t *testing.T) {
	thresfinder := MockUnneededTxIDThresholdFinder(345)
	logdeleter := &MockTransactionLogDeleter{called: false, id: inodedb.AnyVersion}

	if err := inodedbtxloggc.GC(context.TODO(), thresfinder, logdeleter, false); err != nil {
		t.Errorf("GC err: %v", err)
	}

	if !logdeleter.called {
		t.Errorf("Log deleter was not invoked!")
	}
	if logdeleter.id != inodedb.TxID(345) {
		t.Errorf("Log deleter was invoked with unexpected txid: %v", logdeleter.id)
	}
}

func TestINodeDBTxLogGC_AnyVersion(t *testing.T) {
	thresfinder := MockUnneededTxIDThresholdFinder(inodedb.AnyVersion)
	logdeleter := &MockTransactionLogDeleter{called: false, id: 123}

	if err := inodedbtxloggc.GC(context.TODO(), thresfinder, logdeleter, false); err != nil {
		t.Errorf("GC err: %v", err)
	}

	if logdeleter.called {
		t.Errorf("Log deleter should not be invoked!")
	}
}
