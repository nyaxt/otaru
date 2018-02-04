package cli_test

import (
	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	tu "github.com/nyaxt/otaru/testutils"

	"bytes"
	"testing"
)

func init() { tu.EnsureLogger() }

func TestFileWriteRead(t *testing.T) {
	snapshotio := inodedb.NewSimpleDBStateSnapshotIO()
	txio := inodedb.NewSimpleDBTransactionLogIO()
	idb, err := inodedb.NewEmptyDB(snapshotio, txio)
	if err != nil {
		t.Errorf("NewEmptyDB failed: %v", err)
		return
	}

	bs := tu.TestFileBlobStore()
	fs := filesystem.NewFileSystem(idb, bs, tu.TestCipher())
}
