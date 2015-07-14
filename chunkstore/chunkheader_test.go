package chunkstore_test

import (
	"bytes"
	"testing"

	"github.com/nyaxt/otaru/chunkstore"
	. "github.com/nyaxt/otaru/testutils"
)

func TestChunkHeader_SerDes(t *testing.T) {
	// ser
	var b bytes.Buffer
	{
		h := chunkstore.ChunkHeader{
			FrameEncapsulation: 0x02,
			PayloadLen:         0x0dedbeef,
			PayloadVersion:     0x12345678,
			OrigFilename:       "/home/otaru/foobar.txt",
			OrigOffset:         0x0123456789abcdef,
		}

		if err := h.WriteTo(&b, TestCipher()); err != nil {
			t.Errorf("WriteTo failed: %v", err)
		}
		if !bytes.Equal(b.Bytes()[:3], []byte{0x05, 0xa6, chunkstore.CurrentFormat}) {
			t.Errorf("Unexpected ChunkHeader bytestream: %v", b)
		}
	}

	// des
	{
		var h chunkstore.ChunkHeader
		if err := h.ReadFrom(&b, TestCipher()); err != nil {
			t.Errorf("ReadFrom failed: %v", err)
		}

		if h.FrameEncapsulation != chunkstore.CurrentFrameEncapsulation {
			t.Errorf("Failed to unmarshal FrameEncapsulation")
		}
		if h.PayloadLen != 0x0dedbeef {
			t.Errorf("Failed to unmarshal PayloadLen")
		}
		if h.PayloadVersion != 0x12345678 {
			t.Errorf("Failed to unmarshal PayloadVersion")
		}
		if h.OrigFilename != "/home/otaru/foobar.txt" {
			t.Errorf("Failed to unmarshal OrigFilename")
		}
		if h.OrigOffset != 0x0123456789abcdef {
			t.Errorf("Failed to unmarshal OrigOffset")
		}
	}
}

func TestChunkHeader_Read_BadMagic(t *testing.T) {
	b := []byte{0xba, 0xad, chunkstore.CurrentFormat, 0x02, 0xcd, 0xab, 0x21, 0x43, 0x01, 0x02, 0x03, 0x04}
	var h chunkstore.ChunkHeader
	if err := h.ReadFrom(bytes.NewBuffer(b), TestCipher()); err == nil {
		t.Errorf("UnmarshalBinary passed on bad magic!", err)
	}
}
