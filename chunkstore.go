package otaru

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	PayloadOffset              = 1024
	ContentFramePayloadLength  = BtnFrameMaxPayload
	MaxEncryptedPrologueLength = PayloadOffset - MarshaledChunkHeaderLength
	MaxMarshaledPrologueLength = 1000
)

type ChunkPrologue struct {
	PayloadLen   int
	OrigFilename string
	OrigOffset   int64
}

type ChunkWriter struct {
	w             io.Writer
	c             Cipher
	bew           *BtnEncryptWriteCloser
	wroteHeader   bool
	wroteEpilogue bool
	lenTotal      int
}

func NewChunkWriter(w io.Writer, c Cipher) *ChunkWriter {
	return &ChunkWriter{
		w: w, c: c,
		wroteHeader:   false,
		wroteEpilogue: false,
	}
}

func (cw *ChunkWriter) WriteHeaderAndPrologue(p *ChunkPrologue) error {
	if cw.wroteHeader {
		return errors.New("Already wrote header")
	}

	cw.lenTotal = p.PayloadLen

	hdr := ChunkHeader{
		Format:             0x01,
		FrameEncapsulation: 0x01,
	}
	bhdr, err := hdr.MarshalBinary()
	if err != nil {
		return fmt.Errorf("Failed to marshal ChunkHeader: %v", err)
	}
	if _, err := cw.w.Write(bhdr); err != nil {
		return fmt.Errorf("Header write failed: %v", err)
	}

	pjson, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("Serializing header failed: %v", err)
	}
	if len(pjson) > MaxMarshaledPrologueLength {
		panic("marshaled prologue too long!")
	}
	bew, err := NewBtnEncryptWriteCloser(cw.w, cw.c, MaxMarshaledPrologueLength)
	if err != nil {
		return fmt.Errorf("Failed to initialize frame encryptor: %v", err)
	}
	if _, err := bew.Write(pjson); err != nil {
		return fmt.Errorf("Prologue frame write failed: %v", err)
	}
	zeros := make([]byte, MaxMarshaledPrologueLength-len(pjson))
	if _, err := bew.Write(zeros); err != nil {
		return fmt.Errorf("Prologue frame write failed: %v", err)
	}
	if err := bew.Close(); err != nil {
		return fmt.Errorf("Prologue frame close failed: %v", err)
	}

	cw.wroteHeader = true
	return nil
}

func (cw *ChunkWriter) Write(p []byte) (int, error) {
	if !cw.wroteHeader {
		return 0, errors.New("Header is not yet written to chunk")
	}

	if cw.bew == nil {
		var err error
		cw.bew, err = NewBtnEncryptWriteCloser(cw.w, cw.c, cw.lenTotal)
		if err != nil {
			return 0, fmt.Errorf("Failed to initialize frame encryptor: %v", err)
		}
	}

	if _, err := cw.bew.Write(p); err != nil {
		return 0, fmt.Errorf("Failed to write encrypted frame: %v", err)
	}

	return len(p), nil
}

func (cw *ChunkWriter) Close() error {
	if cw.bew != nil {
		if err := cw.bew.Close(); err != nil {
			return err
		}
	}

	// FIXME: Write epilogue
	return nil
}

type ChunkReader struct {
	r io.ReadCloser
	c Cipher

	bdr *BtnDecryptReader

	didReadHeader   bool
	header          ChunkHeader
	didReadPrologue bool
	prologue        ChunkPrologue

	lenTotal int
}

func NewChunkReader(r io.ReadCloser, c Cipher) *ChunkReader {
	return &ChunkReader{
		r: r, c: c,
		didReadHeader: false, didReadPrologue: false,
	}
}

func (cr *ChunkReader) ReadHeader() error {
	if cr.didReadHeader {
		return errors.New("Already read header.")
	}

	b := make([]byte, MarshaledChunkHeaderLength)
	if _, err := io.ReadFull(cr.r, b); err != nil {
		return fmt.Errorf("Failed to read ChunkHeader: %v", err)
	}

	if err := cr.header.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("Failed to unmarshal ChunkHeader: %v", err)
	}

	cr.didReadHeader = true
	return nil
}

func (cr *ChunkReader) Header() ChunkHeader {
	if !cr.didReadHeader {
		panic("Tried to access header before reading it.")
	}
	return cr.header
}

func (cr *ChunkReader) ReadPrologue() error {
	if cr.didReadPrologue {
		return errors.New("Already read prologue.")
	}
	if !cr.didReadHeader {
		return errors.New("Tried to read prologue before reading header.")
	}

	bdr, err := NewBtnDecryptReader(cr.r, cr.c, int(MaxMarshaledPrologueLength))
	if err != nil {
		return err
	}

	mpro := make([]byte, MaxMarshaledPrologueLength)
	if _, err := io.ReadFull(bdr, mpro); err != nil {
		return fmt.Errorf("Failed to read prologue frame: %v", err)
	}
	if !bdr.HasReadAll() {
		panic("Incomplete read in prologue frame !?!?")
	}

	if err := json.Unmarshal(mpro, &cr.prologue); err != nil {
		return fmt.Errorf("Failed to unmarshal prologue: %v", err)
	}

	cr.didReadPrologue = true
	return nil
}

func (cr *ChunkReader) Prologue() ChunkPrologue {
	if !cr.didReadPrologue {
		panic("Tried to access prologue before reading it.")
	}
	return cr.prologue
}

// Length returns length of content.
func (cr *ChunkReader) Length() int {
	return cr.prologue.PayloadLen
}

func (cr *ChunkReader) Read(p []byte) (int, error) {
	if !cr.didReadPrologue {
		return 0, errors.New("Tried to read content before reading prologue.")
	}

	if cr.bdr == nil {
		var err error
		cr.bdr, err = NewBtnDecryptReader(cr.r, cr.c, cr.Length())
		if err != nil {
			return 0, err
		}
	}

	nr, err := cr.bdr.Read(p)
	return nr, err
}

type ChunkIO struct {
	bh RandomAccessIO
	c  Cipher

	// FIXME: fn *FileNode or something for header debug info

	didReadHeader   bool
	header          ChunkHeader
	didReadPrologue bool
	prologue        ChunkPrologue
}

func NewChunkIO(bh RandomAccessIO, c Cipher) *ChunkIO {
	return &ChunkIO{
		bh:              bh,
		c:               c,
		didReadHeader:   false,
		didReadPrologue: false,
	}
}

func (ch *ChunkIO) PWrite(offset int64, p []byte) error {
	return errors.New("Not Implemented")
}

func (ch *ChunkIO) readHeader() error {
	if ch.didReadHeader {
		return errors.New("Already read header.")
	}

	b := make([]byte, MarshaledChunkHeaderLength)
	if err := ch.bh.PRead(0, b); err != nil {
		return fmt.Errorf("Failed to read ChunkHeader: %v", err)
	}

	if err := ch.header.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("Failed to unmarshal ChunkHeader: %v", err)
	}

	ch.didReadHeader = true
	return nil
}

func (ch *ChunkIO) readPrologue() error {
	if ch.didReadPrologue {
		return errors.New("Already read prologue.")
	}

	rd := &OffsetReader{ch.bh, MarshaledChunkHeaderLength}
	bdr, err := NewBtnDecryptReader(rd, ch.c, int(MaxMarshaledPrologueLength))
	if err != nil {
		return err
	}

	mpro := make([]byte, MaxMarshaledPrologueLength)
	if _, err := io.ReadFull(bdr, mpro); err != nil {
		return fmt.Errorf("Failed to read prologue frame: %v", err)
	}
	if !bdr.HasReadAll() {
		panic("Incomplete read in prologue frame !?!?")
	}

	if err := json.Unmarshal(mpro, &ch.prologue); err != nil {
		return fmt.Errorf("Failed to unmarshal prologue: %v", err)
	}

	ch.didReadPrologue = true
	return nil
}

func (ch *ChunkIO) PRead(offset int64, p []byte) error {
	if !ch.didReadPrologue {
		if !ch.didReadHeader {
			if err := ch.readHeader(); err != nil {
				return err
			}
		}
		if err := ch.readPrologue(); err != nil {
			return err
		}
	}

	if offset < 0 || math.MaxInt32 < offset {
		return fmt.Errorf("Offset out of range: %d", offset)
	}
	offset32 := int(offset)

	encryptedContentFrameSize := ch.c.EncryptedFrameSize(ContentFramePayloadLength)

	payloadOffset := offset32 / encryptedContentFrameSize
	payloadLen := ch.prologue.PayloadLen

	frameOffset := PayloadOffset + payloadOffset
	frameLen := IntMin(ContentFramePayloadLength, payloadLen-payloadOffset)

	rd := &OffsetReader{ch.bh, int64(frameOffset)}
	bdr, err := NewBtnDecryptReader(rd, ch.c, frameLen)
	if err != nil {
		return fmt.Errorf("Failed to create BtnDecryptReader: %v", err)
	}

	decrypted := make([]byte, frameLen)
	if _, err := io.ReadFull(bdr, decrypted); err != nil {
		return fmt.Errorf("Failed to decrypt frame: %v", err)
	}
	if !bdr.HasReadAll() {
		panic("Incomplete frame read")
	}

	return errors.New("Not Implemented")
}

func (ch *ChunkIO) Close() error {
	return errors.New("Not Implemented")
}
