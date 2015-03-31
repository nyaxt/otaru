package otaru

import (
	"testing"
)

func TestFile(t *testing.T) {
	fs := NewFileSystem()
	f := fs.CreateFile("hello.txt")

	f.PWrite(0, []byte("hello world!\n"))
}
