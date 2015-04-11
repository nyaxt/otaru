package otaru

import (
	"errors"
)

const (
	ChunkSignatureMagic1 = 0x05 // o t
	ChunkSignatureMagic2 = 0xa6 // a ru

	MarshaledChunkHeaderLength = 4
)

type ChunkHeader struct {
	Format             byte
	FrameEncapsulation byte
}

func (h ChunkHeader) MarshalBinary() ([]byte, error) {
	b := make([]byte, MarshaledChunkHeaderLength)
	b[0] = ChunkSignatureMagic1
	b[1] = ChunkSignatureMagic2
	b[2] = h.Format
	b[3] = h.FrameEncapsulation
	return b, nil
}

func (h *ChunkHeader) UnmarshalBinary(data []byte) error {
	if len(data) < MarshaledChunkHeaderLength {
		return errors.New("data length too short")
	}

	if data[0] != ChunkSignatureMagic1 ||
		data[1] != ChunkSignatureMagic2 {
		return errors.New("signature magic mismatch")
	}

	h.Format = data[2]
	h.FrameEncapsulation = data[3]

	return nil
}
