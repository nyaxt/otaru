package filesystem_test

import (
	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/testutils"
	"go.uber.org/zap"

	"bytes"
	"testing"
)

func init() { testutils.EnsureLogger() }

func TestFileWriteRead(t *testing.T) {
	snapshotio := inodedb.NewSimpleDBStateSnapshotIO()
	txio := inodedb.NewSimpleDBTransactionLogIO()
	idb, err := inodedb.NewEmptyDB(snapshotio, txio)
	if err != nil {
		t.Errorf("NewEmptyDB failed: %v", err)
		return
	}

	bs := testutils.TestFileBlobStore()
	fs := filesystem.NewFileSystem(idb, bs, testutils.TestCipher(), zap.L())
	h, err := fs.OpenFileFullPath("/hello.txt", flags.O_RDWRCREATE, 0666)
	if err != nil {
		t.Errorf("OpenFileFullPath failed: %v", err)
		return
	}
	defer h.Close()

	err = h.PWrite(testutils.HelloWorld, 0)
	if err != nil {
		t.Errorf("PWrite failed: %v", err)
	}

	buf := make([]byte, 32)
	n, err := h.ReadAt(buf, 0)
	if err != nil {
		t.Errorf("PRead failed: %v", err)
	}
	buf = buf[:n]
	if n != len(testutils.HelloWorld) {
		t.Errorf("n: %d", n)
	}
	if !bytes.Equal(testutils.HelloWorld, buf) {
		t.Errorf("PRead content != PWrite content: %v", buf)
	}
}

func TestFile_CloseOpenFile(t *testing.T) {
	snapshotio := inodedb.NewSimpleDBStateSnapshotIO()
	txio := inodedb.NewSimpleDBTransactionLogIO()
	idb, err := inodedb.NewEmptyDB(snapshotio, txio)
	if err != nil {
		t.Errorf("NewEmptyDB failed: %v", err)
		return
	}

	bs := testutils.TestFileBlobStore()
	fs := filesystem.NewFileSystem(idb, bs, testutils.TestCipher(), zap.L())
	h, err := fs.OpenFileFullPath("/hello.txt", flags.O_CREATE|flags.O_RDWR, 0666)
	if err != nil {
		t.Errorf("OpenFileFullPath failed: %v", err)
		return
	}

	if err = h.PWrite(testutils.HelloWorld, 0); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}

	if stats := fs.GetStats(); stats.NumOpenFiles != 1 {
		t.Errorf("NumOpenFiles %d != 1", stats.NumOpenFiles)
	}

	h.Close()

	if stats := fs.GetStats(); stats.NumOpenFiles != 0 {
		t.Errorf("NumOpenFiles %d != 0", stats.NumOpenFiles)
	}
}
