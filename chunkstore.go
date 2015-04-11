package otaru

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	ContentFramePayloadLength = BtnFrameMaxPayload
	MaxMarshaledPrologueLen   = 65000
)

type ChunkPrologue struct {
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

func WriteHeaderAndPrologue(w io.Writer, c Cipher, payloadLen int, p *ChunkPrologue) error {
	pjson, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("Serializing header failed: %v", err)
	}
	if len(pjson) > MaxMarshaledPrologueLen {
		panic("marshaled prologue too long!")
	}

	if payloadLen > math.MaxInt32 {
		return fmt.Errorf("PayloadLen too long: %d", payloadLen)
	}
	hdr := ChunkHeader{
		Format:             0x02,
		FrameEncapsulation: 0x01,
		PrologueLen:        uint16(len(pjson)),
		EpilogueLen:        0,
		PayloadLen:         uint32(payloadLen),
	}
	bhdr, err := hdr.MarshalBinary()
	if err != nil {
		return fmt.Errorf("Failed to marshal ChunkHeader: %v", err)
	}
	if _, err := w.Write(bhdr); err != nil {
		return fmt.Errorf("Header write failed: %v", err)
	}

	bew, err := NewBtnEncryptWriteCloser(w, c, len(pjson))
	if err != nil {
		return fmt.Errorf("Failed to initialize frame encryptor: %v", err)
	}
	if _, err := bew.Write(pjson); err != nil {
		return fmt.Errorf("Prologue frame write failed: %v", err)
	}
	if err := bew.Close(); err != nil {
		return fmt.Errorf("Prologue frame close failed: %v", err)
	}
	return nil
}

func (cw *ChunkWriter) WriteHeaderAndPrologue(payloadLen int, p *ChunkPrologue) error {
	if cw.wroteHeader {
		return errors.New("Already wrote header")
	}

	cw.lenTotal = payloadLen
	if err := WriteHeaderAndPrologue(cw.w, cw.c, payloadLen, p); err != nil {
		return err
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

	bdr, err := NewBtnDecryptReader(cr.r, cr.c, int(cr.header.PrologueLen))
	if err != nil {
		return err
	}

	mpro := make([]byte, cr.header.PrologueLen)
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
	return int(cr.header.PayloadLen)
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

// ChunkIO provides RandomAccessIO for blobchunk
type ChunkIO struct {
	bh BlobHandle
	c  Cipher

	// FIXME: fn *FileNode or something for header debug info

	didReadHeader   bool
	header          ChunkHeader
	didReadPrologue bool
	prologue        ChunkPrologue
}

func NewChunkIO(bh BlobHandle, c Cipher) *ChunkIO {
	return &ChunkIO{
		bh:              bh,
		c:               c,
		didReadHeader:   false,
		didReadPrologue: false,
	}
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
	bdr, err := NewBtnDecryptReader(rd, ch.c, int(ch.header.PrologueLen))
	if err != nil {
		return err
	}

	mpro := make([]byte, ch.header.PrologueLen)
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

func (ch *ChunkIO) encryptedFrameOffset(i int) int {
	o := MarshaledChunkHeaderLength + ch.c.EncryptedFrameSize(int(ch.header.PrologueLen))

	encryptedFrameSize := ch.c.EncryptedFrameSize(ContentFramePayloadLength)
	o += encryptedFrameSize * i

	return o
}

type decryptedContentFrame struct {
	P      []byte
	Offset int
}

func (ch *ChunkIO) readContentFrame(i int) (*decryptedContentFrame, error) {
	// the frame carries a part of the content at offset
	offset := i * BtnFrameMaxPayload
	// payload length of the encrypted frame
	framePayloadLen := IntMin(ContentFramePayloadLength, int(ch.header.PayloadLen)-offset)

	// the offset of the start of the frame in blob
	blobOffset := ch.encryptedFrameOffset(i)

	rd := &OffsetReader{ch.bh, int64(blobOffset)}
	bdr, err := NewBtnDecryptReader(rd, ch.c, framePayloadLen)
	if err != nil {
		return nil, fmt.Errorf("Failed to create BtnDecryptReader: %v", err)
	}

	p := make([]byte, framePayloadLen)
	if _, err := io.ReadFull(bdr, p); err != nil {
		return nil, fmt.Errorf("Failed to decrypt frame idx: %d, err: %v", i, err)
	}
	if !bdr.HasReadAll() {
		panic("Incomplete frame read")
	}

	fmt.Printf("Read content frame idx: %d\n", i)
	return &decryptedContentFrame{P: p, Offset: offset}, nil
}

func (ch *ChunkIO) writeContentFrame(i int, f *decryptedContentFrame) error {
	// the offset of the start of the frame in blob
	blobOffset := ch.encryptedFrameOffset(i)

	wr := &OffsetWriter{ch.bh, int64(blobOffset)}
	bew, err := NewBtnEncryptWriteCloser(wr, ch.c, len(f.P))
	if err != nil {
		return fmt.Errorf("Failed to create BtnEncryptWriteCloser: %v", err)
	}
	if _, err := bew.Write(f.P); err != nil {
		return fmt.Errorf("Failed to encrypt frame: %v", err)
	}
	if err := bew.Close(); err != nil {
		return fmt.Errorf("Failed to Close BtnEncryptWriteCloser: %v", err)
	}

	fmt.Printf("Wrote content frame idx: %d\n", i)
	return nil
}

func (ch *ChunkIO) ensurePrologue() error {
	if !ch.didReadPrologue {
		if ch.bh.Size() == 0 {
			return fmt.Errorf("FIXME: write header/prologue")
		}

		if !ch.didReadHeader {
			if err := ch.readHeader(); err != nil {
				return err
			}
		}
		if err := ch.readPrologue(); err != nil {
			return err
		}
	}

	return nil
}

func (ch *ChunkIO) PRead(offset int64, p []byte) error {
	if err := ch.ensurePrologue(); err != nil {
		return err
	}

	if offset < 0 || math.MaxInt32 < offset {
		return fmt.Errorf("Offset out of int32 range: %d", offset)
	}
	offset32 := int(offset)

	i := offset32 / BtnFrameMaxPayload
	f, err := ch.readContentFrame(i)
	if err != nil {
		return err
	}
	inframeOffset := offset32 - f.Offset
	if inframeOffset < 0 {
		panic("ASSERT: inframeOffset must be non-negative here")
	}
	if inframeOffset >= len(f.P) {
		return fmt.Errorf("Attempted to read beyond written size: %d. inframeOffset: %d, framePayloadLen: %d", offset, inframeOffset, len(f.P))
	}

	n := IntMin(len(p), len(f.P)-inframeOffset)
	copy(p[:n], f.P[inframeOffset:])

	if n < len(p) {
		fmt.Printf("read %d bytes for off %d len %d, continuing on off %d len %d\n", n, offset, len(p), offset+int64(n), len(p[n:]))
		return ch.PRead(offset+int64(n), p[n:])
	}
	return nil
}

func (ch *ChunkIO) PWrite(offset int64, p []byte) error {
	if err := ch.ensurePrologue(); err != nil {
		return err
	}

	if offset < 0 || math.MaxInt32 < offset {
		return fmt.Errorf("Offset out of range: %d", offset)
	}
	offset32 := int(offset)

	// read the encrypted frame
	i := offset32 / BtnFrameMaxPayload
	f, err := ch.readContentFrame(i)
	if err != nil {
		return err
	}

	// modify the payload
	inframeOffset := offset32 - f.Offset
	if inframeOffset < 0 {
		panic("ASSERT: inframeOffset must be non-negative here")
	}

	// FIXME: may need to expand the buffer here

	n := IntMin(len(p), len(f.P)-inframeOffset)
	copy(f.P[inframeOffset:], p[:n])

	// writeback the updated encrypted frame
	if err := ch.writeContentFrame(i, f); err != nil {
		return fmt.Errorf("failed to write back the encrypted frame: %v", err)
	}

	if n < len(p) {
		return ch.PWrite(offset+int64(n), p[n:])
	}
	return nil

	return errors.New("Not Implemented")
}

func (ch *ChunkIO) Close() error {
	return nil
}
