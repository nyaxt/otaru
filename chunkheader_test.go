package otaru

import (
	"bytes"
	"testing"
)

func TestChunkHeader_MarshalBinary(t *testing.T) {
	h := ChunkHeader{
		Format:             0x01,
		FrameEncapsulation: 0x02,
	}
	b, err := h.MarshalBinary()
	if err != nil {
		t.Errorf("MarshalBinary failed: %v", err)
	}
	if !bytes.Equal(b, []byte{0x05, 0xa6, 0x01, 0x02}) {
		t.Errorf("Unexpected ChunkHeader bytestream: %v", b)
	}
}

func TestChunkHeader_UnmarshalBinary(t *testing.T) {
	b := []byte{0x05, 0xa6, 0x01, 0x02}
	var h ChunkHeader
	if err := h.UnmarshalBinary(b); err != nil {
		t.Errorf("UnmarshalBinary failed: %v", err)
	}

	if h.Format != 0x01 {
		t.Errorf("Failed to unmarshal Format")
	}
	if h.FrameEncapsulation != 0x02 {
		t.Errorf("Failed to unmarshal FrameEncapsulation")
	}
}

func TestChunkHeader_UnmarshalBinary_BadMagic(t *testing.T) {
	b := []byte{0xba, 0xad, 0x01, 0x02}
	var h ChunkHeader
	if err := h.UnmarshalBinary(b); err == nil {
		t.Errorf("UnmarshalBinary passed on bad magic!", err)
	}
}
