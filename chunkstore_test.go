package otaru_test

import (
	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	. "github.com/nyaxt/otaru/testutils"

	"bytes"
	"io"
	"testing"
)

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
	cw, err := otaru.NewChunkWriter(buf, TestCipher(), otaru.ChunkHeader{PayloadLen: uint32(len(p))})
	if err != nil {
		t.Errorf("Failed to create chunk writer: %v", err)
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
	cio := otaru.NewChunkIO(testbh, TestCipher())

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
	cio := otaru.NewChunkIO(testbh, TestCipher())

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
	cio := otaru.NewChunkIO(testbh, TestCipher())

	if cio.Header().PayloadVersion != 0 {
		t.Errorf("Initial PayloadVersion != 0")
	}

	upd := []byte("testin write")
	if err := cio.PWrite(0, upd); err != nil {
		t.Errorf("failed to PWrite to ChunkIO: %v", err)
		return
	}

	if cio.Header().PayloadVersion != 1 {
		t.Errorf("PayloadVersion after PWrite != 1")
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

	queryFn := otaru.NewQueryChunkVersion(TestCipher())
	ver, err := queryFn(&blobstore.OffsetReader{testbh, 0})
	if err != nil {
		t.Errorf("PayloadVersion query failed")
	}
	if ver != 1 {
		t.Errorf("PayloadVersion read back after cio.Close != 1")
	}
}

func TestChunkIO_Write_Update1MB(t *testing.T) {
	td := genTestData(1024*1024 + 123)
	b := genFrameByChunkWriter(t, td)
	if b == nil {
		return
	}
	testbh := &TestBlobHandle{b}
	cio := otaru.NewChunkIO(testbh, TestCipher())
	origver := cio.Header().PayloadVersion

	// Full update
	td2 := negateBits(td)
	if err := cio.PWrite(0, td2); err != nil {
		t.Errorf("failed to PWrite into ChunkIO: %v", err)
		return
	}
	if cio.Header().PayloadVersion <= origver {
		t.Errorf("PayloadVersion after PWrite < origver: %d < %d", cio.Header().PayloadVersion, origver)
	}
	origver = cio.Header().PayloadVersion
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
	if cio.Header().PayloadVersion <= origver {
		t.Errorf("PayloadVersion after PWrite < origver: %d < %d", cio.Header().PayloadVersion, origver)
	}
	origver = cio.Header().PayloadVersion
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

func Test_ChunkIOWrite_NewHello_ChunkReaderRead(t *testing.T) {
	testbh := &TestBlobHandle{}
	cio := otaru.NewChunkIO(testbh, TestCipher())
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

	cr, err := otaru.NewChunkReader(bytes.NewBuffer(testbh.Buf), TestCipher())
	if err != nil {
		t.Errorf("failed to create chunk reader: %v", err)
		return
	}
	if cr.Length() != len(HelloWorld) {
		t.Errorf("failed to recover payload len")
	}
	readtgt2 := make([]byte, len(HelloWorld))
	if _, err := io.ReadFull(cr, readtgt2); err != nil {
		t.Errorf("failed to Read from ChunkReader: %v", err)
		return
	}
	if !bytes.Equal(readtgt2, HelloWorld) {
		t.Errorf("Read content invalid")
		return
	}
}

func checkZero(t *testing.T, p []byte, off int, length int) {
	i := 0
	for i < length {
		if p[off+i] != 0 {
			t.Errorf("Given slice non-0 at idx: %d", off+i)
		}
		i++
	}
}

func Test_ChunkIOWrite_ZeroFillPadding(t *testing.T) {
	testbh := &TestBlobHandle{}
	cio := otaru.NewChunkIO(testbh, TestCipher())

	// [ zero ][ hello ]
	//    10      12
	if err := cio.PWrite(10, HelloWorld); err != nil {
		t.Errorf("failed to PWrite to ChunkIO: %v", err)
		return
	}
	readtgt := make([]byte, len(HelloWorld))
	if err := cio.PRead(10, readtgt); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	if !bytes.Equal(readtgt, HelloWorld) {
		t.Errorf("Read content invalid")
		return
	}
	readtgt2 := make([]byte, 10+len(HelloWorld))
	if err := cio.PRead(0, readtgt2); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	checkZero(t, readtgt2, 0, 10)
	if !bytes.Equal(readtgt2[10:10+12], HelloWorld) {
		t.Errorf("Read content invalid: hello1 %v != %v", readtgt2[10:10+12], HelloWorld)
		return
	}

	// [ zero ][ hello ][ zero ][ hello ]
	//    10      12      512k      12
	if err := cio.PWrite(10+12+512*1024, HelloWorld); err != nil {
		t.Errorf("failed to PWrite to ChunkIO: %v", err)
		return
	}
	readtgt3 := make([]byte, 10+12+512*1024+12)
	if err := cio.PRead(0, readtgt3); err != nil {
		t.Errorf("failed to PRead from ChunkIO: %v", err)
		return
	}
	checkZero(t, readtgt3, 0, 10)
	checkZero(t, readtgt3, 10+12, 512*1024)
	if !bytes.Equal(readtgt3[10:10+12], HelloWorld) {
		t.Errorf("Read content invalid: hello1")
		return
	}
	if !bytes.Equal(readtgt3[10+12+512*1024:10+12+512*1024+12], HelloWorld) {
		t.Errorf("Read content invalid: hello2")
		return
	}

	if err := cio.Close(); err != nil {
		t.Errorf("failed to Close ChunkIO: %v", err)
		return
	}
}

func Test_ChunkIOWrite_OverflowUpdate(t *testing.T) {
	testbh := &TestBlobHandle{}
	cio := otaru.NewChunkIO(testbh, TestCipher())
	if err := cio.PWrite(0, HelloWorld); err != nil {
		t.Errorf("failed to PWrite to ChunkIO: %v", err)
		return
	}
	if err := cio.PWrite(7, HogeFugaPiyo); err != nil {
		t.Errorf("failed to PWrite to ChunkIO: %v", err)
		return
	}
	if err := cio.Close(); err != nil {
		t.Errorf("failed to Close ChunkIO: %v", err)
		return
	}

	cr, err := otaru.NewChunkReader(bytes.NewBuffer(testbh.Buf), TestCipher())
	if err != nil {
		t.Errorf("failed to create chunk reader: %v", err)
		return
	}
	exp := []byte("Hello, hogefugapiyo")
	if cr.Length() != len(exp) {
		t.Errorf("failed to recover payload len")
	}
	readtgt := make([]byte, len(exp))
	if _, err := io.ReadFull(cr, readtgt); err != nil {
		t.Errorf("failed to Read from ChunkReader: %v", err)
		return
	}
	if !bytes.Equal(readtgt, exp) {
		t.Errorf("Read content invalid")
		return
	}
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

func Test_QueryChunkVersion_EOF(t *testing.T) {
	queryFn := otaru.NewQueryChunkVersion(TestCipher())
	ver, err := queryFn(&eofReader{})
	if err != nil {
		t.Errorf("NewQueryChunkVersion should return no err on EOF")
	}
	if ver != 0 {
		t.Errorf("NewQueryChunkVersion should return 0 on EOF")
	}
}
