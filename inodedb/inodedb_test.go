package inodedb_test

import (
	"testing"

	"github.com/nyaxt/otaru/inodedb"
)

func TestInitialState(t *testing.T) {
	NewDB(NewSimpleDBStateSnapshotIO(), NewSimpleDBTransactionLogIO())
}
