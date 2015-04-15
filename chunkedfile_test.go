package otaru

import (
	"bytes"
	"testing"
)

func TestChunkedFileIO_FileBlobStore(t *testing.T) {
	fn := NewFileNode(123, "hoge/fuga.txt")
	cfio := NewChunkedFileIO(&FileBlobStore{}, fn, testCipher())

	if err := cfio.PWrite(0, HelloWorld); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}
	readtgt := make([]byte, len(HelloWorld))
	if err := cfio.PRead(0, readtgt); err != nil {
		t.Errorf("PRead failed: %v", err)
		return
	}
	if !bytes.Equal(HelloWorld, readtgt) {
		t.Errorf("read content invalid: %v", readtgt)
	}

	if int64(len(HelloWorld)) != cfio.Size() {
		t.Errorf("len invalid: %v", cfio.Size())
	}
}
