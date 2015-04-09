package otaru

import (
	"errors"
)

const (
	ChunkSignatureMagic1 = 0x05 // o t
	ChunkSignatureMagic2 = 0xa6 // a ru

	MarshaledChunkHeaderLength = 8
)

type ChunkHeader struct {
	Format             byte
	FrameEncapsulation byte
	// FIXME: use fixed sized pro/epi to make this seekable
	PrologueLength uint16
	EpilogueLength uint16
}

func (h ChunkHeader) MarshalBinary() ([]byte, error) {
	b := make([]byte, MarshaledChunkHeaderLength)
	b[0] = ChunkSignatureMagic1
	b[1] = ChunkSignatureMagic2
	b[2] = h.Format
	b[3] = h.FrameEncapsulation
	b[4] = byte((h.PrologueLength >> 0) & 0xff)
	b[5] = byte((h.PrologueLength >> 8) & 0xff)
	b[6] = byte((h.EpilogueLength >> 0) & 0xff)
	b[7] = byte((h.EpilogueLength >> 8) & 0xff)
	return b, nil
}

func (h *ChunkHeader) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return errors.New("data length too short")
	}

	if data[0] != ChunkSignatureMagic1 ||
		data[1] != ChunkSignatureMagic2 {
		return errors.New("signature magic mismatch")
	}

	h.Format = data[2]
	h.FrameEncapsulation = data[3]
	h.PrologueLength = uint16(data[5])<<8 | uint16(data[4])
	h.EpilogueLength = uint16(data[7])<<8 | uint16(data[6])

	return nil
}
