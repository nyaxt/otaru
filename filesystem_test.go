package otaru_test

import (
	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	tu "github.com/nyaxt/otaru/testutils"

	"bytes"
	"testing"
)

func TestFileWriteRead(t *testing.T) {
	snapshotio := inodedb.NewSimpleDBStateSnapshotIO()
	txio := inodedb.NewSimpleDBTransactionLogIO()
	idb, err := inodedb.NewEmptyDB(snapshotio, txio)
	if err != nil {
		t.Errorf("NewEmptyDB failed: %v", err)
		return
	}

	bs := tu.TestFileBlobStore()
	fs := otaru.NewFileSystem(idb, bs, tu.TestCipher())
	h, err := fs.OpenFileFullPath("/hello.txt", flags.O_CREATE|flags.O_RDWR, 0666)
	if err != nil {
		t.Errorf("OpenFileFullPath failed: %v", err)
		return
	}

	err = h.PWrite(0, tu.HelloWorld)
	if err != nil {
		t.Errorf("PWrite failed: %v", err)
	}

	buf := make([]byte, 32)
	n, err := h.ReadAt(buf, 0)
	if err != nil {
		t.Errorf("PRead failed: %v", err)
	}
	buf = buf[:n]
	if n != len(tu.HelloWorld) {
		t.Errorf("n: %d", n)
	}
	if !bytes.Equal(tu.HelloWorld, buf) {
		t.Errorf("PRead content != PWrite content: %v", buf)
	}
}
