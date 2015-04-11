package otaru

import (
	"bytes"
	"testing"
)

var (
	Key        = []byte("0123456789abcdef")
	HelloWorld = []byte("Hello, world")
)

func testCipher() Cipher {
	c, err := NewCipher(Key)
	if err != nil {
		panic("Failed to init Cipher for testing")
	}
	return c
}

func genTestData(l int) []byte {
	ret := make([]byte, l)
	i := 0
	for i+3 < l {
		x := i / 4
		ret[i+0] = byte((x >> 24) & 0xff)
		ret[i+1] = byte((x >> 16) & 0xff)
		ret[i+2] = byte((x >> 8) & 0xff)
		ret[i+3] = byte((x >> 0) & 0xff)
		i += 4
	}
	if i < l {
		copy(ret[i:], []byte{0xab, 0xcd, 0xef})
	}
	return ret
}

func negateBits(p []byte) []byte {
	ret := make([]byte, len(p))
	for i, x := range p {
		ret[i] = 0xff ^ x
	}
	return ret
}

func Test_genTestData(t *testing.T) {
	td := genTestData(5)
	if !bytes.Equal(td, []byte{0, 0, 0, 0, 0xab}) {
		t.Errorf("unexp testdata: %v", td)
	}

	td = genTestData(11)
	if !bytes.Equal(td, []byte{0, 0, 0, 0, 0, 0, 0, 1, 0xab, 0xcd, 0xef}) {
		t.Errorf("unexp testdata: %v", td)
	}
}

func genFrameByChunkWriter(t *testing.T, p []byte) []byte {
	buf := new(bytes.Buffer)
	cw := NewChunkWriter(buf, testCipher())

	err := cw.WriteHeaderAndPrologue(&ChunkPrologue{
		PayloadLen:   len(p),
		OrigFilename: "testframe.dat",
		OrigOffset:   0,
	})
	if err != nil {
		t.Errorf("Failed to write chunk header: %v", err)
		return nil
	}

	if _, err := cw.Write(p); err != nil {
		t.Errorf("Failed to write chunk payload: %v", err)
		return nil
	}

	if err := cw.Close(); err != nil {
		t.Errorf("Failed to close ChunkWriter: %v", err)
		return nil
	}

	return buf.Bytes()
}

func TestChunkIO_Read_HelloWorld(t *testing.T) {
	b := genFrameByChunkWriter(t, HelloWorld)
	if b == nil {
		return
	}
	testbh := &TestBlobHandle{b}
	cio := NewChunkIO(testbh, testCipher())

	readtgt := make([]byte, len(HelloWorld))
	if err := cio.PRead(0, readtgt); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	if !bytes.Equal(readtgt, HelloWorld) {
		t.Errorf("Read content invalid")
		return
	}

	if err := cio.Close(); err != nil {
		t.Errorf("failed to Close ChunkIO: %v", err)
		return
	}
}

func TestChunkIO_Read_1MB(t *testing.T) {
	td := genTestData(1024*1024 + 123)
	b := genFrameByChunkWriter(t, td)
	if b == nil {
		return
	}
	testbh := &TestBlobHandle{b}
	cio := NewChunkIO(testbh, testCipher())

	// Full read
	readtgt := make([]byte, len(td))
	if err := cio.PRead(0, readtgt); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	if !bytes.Equal(readtgt, td) {
		t.Errorf("Read content invalid")
		return
	}

	// Partial read
	readtgt = readtgt[:321]
	if err := cio.PRead(1012345, readtgt); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	if !bytes.Equal(readtgt, td[1012345:1012345+321]) {
		t.Errorf("Read content invalid")
		return
	}

	if err := cio.Close(); err != nil {
		t.Errorf("failed to Close ChunkIO: %v", err)
		return
	}
}

func TestChunkIO_Write_UpdateHello(t *testing.T) {
	b := genFrameByChunkWriter(t, HelloWorld)
	if b == nil {
		return
	}
	testbh := &TestBlobHandle{b}
	cio := NewChunkIO(testbh, testCipher())

	upd := []byte("testin write")
	if err := cio.PWrite(0, upd); err != nil {
		t.Errorf("failed to PWrite to ChunkIO: %v", err)
		return
	}

	readtgt := make([]byte, len(upd))
	if err := cio.PRead(0, readtgt); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	if !bytes.Equal(readtgt, upd) {
		t.Errorf("Read content invalid")
		return
	}

	if err := cio.Close(); err != nil {
		t.Errorf("failed to Close ChunkIO: %v", err)
		return
	}
}

func TestChunkIO_Write_Update1MB(t *testing.T) {
	td := genTestData(1024*1024 + 123)
	b := genFrameByChunkWriter(t, td)
	if b == nil {
		return
	}
	testbh := &TestBlobHandle{b}
	cio := NewChunkIO(testbh, testCipher())

	// Full update
	td2 := negateBits(td)
	if err := cio.PWrite(0, td2); err != nil {
		t.Errorf("failed to PWrite into ChunkIO: %v", err)
		return
	}
	readtgt := make([]byte, len(td))
	if err := cio.PRead(0, readtgt); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	if !bytes.Equal(readtgt, td2) {
		t.Errorf("Read content invalid")
		return
	}

	// Partial update
	if err := cio.PWrite(1012345, td[1012345:1012345+321]); err != nil {
		t.Errorf("failed to PWrite into ChunkIO: %v", err)
		return
	}
	td3 := make([]byte, len(td2))
	copy(td3, td2)
	copy(td3[1012345:1012345+321], td[1012345:1012345+321])
	if err := cio.PRead(0, readtgt); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	if !bytes.Equal(readtgt, td3) {
		t.Errorf("Read content invalid")
		return
	}
	if err := cio.Close(); err != nil {
		t.Errorf("failed to Close ChunkIO: %v", err)
		return
	}
}

func TestChunkIO_Write_NewHello_MatchChunkWriter(t *testing.T) {
	exp := genFrameByChunkWriter(t, HelloWorld)
	if exp == nil {
		return
	}

	testbh := &TestBlobHandle{}
	cio := NewChunkIO(testbh, testCipher())
	if err := cio.PWrite(0, HelloWorld); err != nil {
		t.Errorf("failed to PWrite to ChunkIO: %v", err)
		return
	}
	readtgt := make([]byte, len(HelloWorld))
	if err := cio.PRead(0, readtgt); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	if !bytes.Equal(readtgt, HelloWorld) {
		t.Errorf("Read content invalid")
		return
	}
	if err := cio.Close(); err != nil {
		t.Errorf("failed to Close ChunkIO: %v", err)
		return
	}

	if !bytes.Equal(testbh.Buf, exp) {
		t.Errorf("Chunk written by ChunkIO and ChunkWriter don't match")
	}
}
