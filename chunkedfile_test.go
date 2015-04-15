package otaru

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"
)

func testFileBlobStore() *FileBlobStore {
	tempdir, err := ioutil.TempDir("", "otarutest")
	if err != nil {
		log.Fatalf("failed to create tmpdir: %v", err)
	}
	fbs, err := NewFileBlobStore(tempdir)
	if err != nil {
		log.Fatalf("failed to create blobstore: %v", err)
	}
	return fbs
}

func TestChunkedFileIO_FileBlobStore(t *testing.T) {
	fn := NewFileNode(123, "hoge/fuga.txt")
	fbs := testFileBlobStore()
	cfio := NewChunkedFileIO(fbs, fn, testCipher())

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
