package otaru

import (
	"bytes"
	"fmt"
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

func TestChunkedFileIO_SingleChunk(t *testing.T) {
	fn := NewFileNode(456, "piyo.txt")
	bs := NewMockBlobStore()
	cfio := NewChunkedFileIO(bs, fn, testCipher())

	// Disable Chunk framing for testing
	cfio.OverrideNewChunkIOForTesting(func(bh BlobHandle, c Cipher) BlobHandle { return bh })

	if err := cfio.PWrite(123, HelloWorld); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}
	if err := cfio.PWrite(456, HelloWorld); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}

	if len(fn.Chunks) != 1 {
		t.Errorf("len(fn.Chunks) %d", len(fn.Chunks))
		return
	}
	if fn.Chunks[0].Offset != 0 {
		t.Errorf("Chunk at wierd offset: %d", fn.Chunks[1].Offset)
	}
	bh := bs.Paths[fn.Chunks[0].BlobPath]
	if bh.Log[0].Offset != 123 {
		t.Errorf("Chunk write at invalid offset: %d", bh.Log[0].Offset)
	}
	if bh.Log[0].Offset != 456 {
		t.Errorf("Chunk write at invalid offset: %d", bh.Log[0].Offset)
	}
}

func TestChunkedFileIO_MultiChunk(t *testing.T) {
	fn := NewFileNode(456, "piyo.txt")
	bs := NewMockBlobStore()
	cfio := NewChunkedFileIO(bs, fn, testCipher())

	// Disable Chunk framing for testing
	cfio.OverrideNewChunkIOForTesting(func(bh BlobHandle, c Cipher) BlobHandle { return bh })

	if err := cfio.PWrite(ChunkSplitSize+12345, HelloWorld); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}
	if err := cfio.PWrite(123, HelloWorld); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}

	if len(fn.Chunks) != 2 {
		t.Errorf("len(fn.Chunks) %d", len(fn.Chunks))
		return
	}
	if fn.Chunks[0].Offset != 0 {
		t.Errorf("Chunk at wierd offset: %d", fn.Chunks[1].Offset)
	}
	bh := bs.Paths[fn.Chunks[0].BlobPath]
	if bh.Log[0].Offset != 123 {
		t.Errorf("Chunk write at invalid offset: %d", bh.Log[0].Offset)
	}
	if fn.Chunks[1].Offset != ChunkSplitSize {
		t.Errorf("Split chunk at wierd offset: %d", fn.Chunks[1].Offset)
	}
	bh = bs.Paths[fn.Chunks[1].BlobPath]
	if bh.Log[0].Offset != 12345 {
		t.Errorf("Split chunk write at invalid offset: %d", bh.Log[0].Offset)
	}

	if err := cfio.PWrite(ChunkSplitSize-5, HelloWorld); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}
	bh = bs.Paths[fn.Chunks[1].BlobPath]
	fmt.Printf("? %v\n", bh.Log[1])
}
