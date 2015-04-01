package otaru

import (
	"testing"
)

func TestFile(t *testing.T) {
	fs := NewFileSystem()
	h, err := fs.CreateFile("hello.txt")
	if err != nil {
		t.Errorf("CreateFile failed")
		return
	}

	err = h.PWrite(0, []byte("hello world!\n"))
	if err != nil {
		t.Errorf("PWrite failed")
	}
}
