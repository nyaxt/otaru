package otaru_test

import (
	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	. "github.com/nyaxt/otaru/testutils"

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

	bs := TestFileBlobStore()
	fs := otaru.NewFileSystem(idb, bs, TestCipher())
	h, err := fs.OpenFileFullPath("/hello.txt", flags.O_CREATE|flags.O_RDWR, 0666)
	if err != nil {
		t.Errorf("OpenFileFullPath failed: %v", err)
		return
	}

	err = h.PWrite(0, []byte("hello world!\n"))
	if err != nil {
		t.Errorf("PWrite failed: %v", err)
	}

	buf := make([]byte, 13)
	err = h.PRead(0, buf)
	if err != nil {
		t.Errorf("PRead failed: %v", err)
	}
	if !bytes.Equal([]byte("hello world!\n"), buf) {
		t.Errorf("PRead content != PWrite content")
	}
}
