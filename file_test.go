package otaru_test

import (
	"github.com/nyaxt/otaru"
	. "github.com/nyaxt/otaru/testutils"

	"bytes"
	"testing"
)

func TestFileWriteRead(t *testing.T) {
	bs := TestFileBlobStore()
	fs, err := otaru.NewFileSystemEmpty(bs, TestCipher())
	if err != nil {
		t.Errorf("NewFileSystemEmpty failed: %v", err)
		return
	}
	h, err := fs.OpenFileFullPath("/hello.txt", otaru.O_CREATE|otaru.O_RDWR, 0666)
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
