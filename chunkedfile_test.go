package otaru_test

import (
	. "github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	. "github.com/nyaxt/otaru/testutils"

	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func TestChunkedFileIO_FileBlobStore(t *testing.T) {
	fn := NewFileNode(NewINodeDBEmpty(), "/hoge/fuga.txt")
	fbs := TestFileBlobStore()
	cfio := NewChunkedFileIO(fbs, fn, TestCipher())

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
	fn := NewFileNode(NewINodeDBEmpty(), "/piyo.txt")
	bs := blobstore.NewMockBlobStore()
	cfio := NewChunkedFileIO(bs, fn, TestCipher())

	// Disable Chunk framing for testing
	cfio.OverrideNewChunkIOForTesting(func(bh blobstore.BlobHandle, c Cipher) blobstore.BlobHandle { return bh })

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
		t.Errorf("Chunk at invalid offset: %d", fn.Chunks[1].Offset)
	}
	bh := bs.Paths[fn.Chunks[0].BlobPath]
	if bh.Log[0].Offset != 123 {
		t.Errorf("Chunk write at invalid offset: %d", bh.Log[0].Offset)
	}
	if bh.Log[1].Offset != 456 {
		t.Errorf("Chunk write at invalid offset: %d", bh.Log[0].Offset)
	}
}

func TestChunkedFileIO_MultiChunk(t *testing.T) {
	fn := NewFileNode(NewINodeDBEmpty(), "/piyo.txt")
	bs := blobstore.NewMockBlobStore()
	cfio := NewChunkedFileIO(bs, fn, TestCipher())

	// Disable Chunk framing for testing
	cfio.OverrideNewChunkIOForTesting(func(bh blobstore.BlobHandle, c Cipher) blobstore.BlobHandle { return bh })

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
		t.Errorf("Chunk at invalid offset: %d", fn.Chunks[1].Offset)
	}
	bh := bs.Paths[fn.Chunks[0].BlobPath]
	if bh.Log[0].Offset != 123 {
		t.Errorf("Chunk write at invalid offset: %d", bh.Log[0].Offset)
	}
	if fn.Chunks[1].Offset != ChunkSplitSize {
		t.Errorf("Split chunk at invalid offset: %d", fn.Chunks[1].Offset)
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
	if !reflect.DeepEqual(bh.Log[1], blobstore.MockBlobStoreOperation{'W', 0, 7, HelloWorld[5]}) {
		fmt.Printf("? %+v\n", bh.Log[1])
	}
}
