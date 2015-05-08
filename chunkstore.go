package otaru

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
)

const (
	ContentFramePayloadLength = BtnFrameMaxPayload
)

var (
	ZeroContent = make([]byte, ContentFramePayloadLength)
)

type ChunkWriter struct {
	w           io.Writer
	c           Cipher
	bew         *BtnEncryptWriteCloser
	wroteHeader bool
}

func NewChunkWriter(w io.Writer, c Cipher, h ChunkHeader) (*ChunkWriter, error) {
	cw := &ChunkWriter{
		w: w, c: c,
		wroteHeader: false,
	}

	var err error
	cw.bew, err = NewBtnEncryptWriteCloser(cw.w, cw.c, int(h.PayloadLen))
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize frame encryptor: %v", err)
	}

	if err := h.WriteTo(cw.w, cw.c); err != nil {
		return nil, fmt.Errorf("Failed to write header: %v", err)
	}

	return cw, nil
}

func (cw *ChunkWriter) Write(p []byte) (int, error) {
	if !cw.wroteHeader {
		return 0, errors.New("Header is not yet written to chunk")
	}

	if _, err := cw.bew.Write(p); err != nil {
		return 0, fmt.Errorf("Failed to write encrypted frame: %v", err)
	}

	return len(p), nil
}

func (cw *ChunkWriter) Close() error {
	if err := cw.bew.Close(); err != nil {
		return err
	}

	return nil
}

type ChunkReader struct {
	r io.Reader
	c Cipher

	header ChunkHeader

	bdr *BtnDecryptReader
}

func NewChunkReader(r io.Reader, c Cipher) (*ChunkReader, error) {
	cr := &ChunkReader{r: r, c: c}

	if err := cr.header.ReadFrom(r, c); err != nil {
		return nil, fmt.Errorf("Failed to read header: %v", err)
	}

	var err error
	cr.bdr, err = NewBtnDecryptReader(cr.r, cr.c, cr.Length())
	if err != nil {
		return nil, err
	}

	return cr, nil
}

func (cr *ChunkReader) Header() ChunkHeader {
	return cr.header
}

// Length returns length of content.
func (cr *ChunkReader) Length() int {
	return int(cr.header.PayloadLen)
}

func (cr *ChunkReader) Read(p []byte) (int, error) {
	return cr.bdr.Read(p)
}

// ChunkIO provides RandomAccessIO for blobchunk
type ChunkIO struct {
	bh BlobHandle
	c  Cipher

	didReadHeader bool
	header        ChunkHeader

	needsHeaderUpdate bool
}

func NewChunkIO(bh BlobHandle, c Cipher) *ChunkIO {
	return &ChunkIO{
		bh:            bh,
		c:             c,
		didReadHeader: false,
		header: ChunkHeader{
			OrigFilename: "<unknown>",
			OrigOffset:   -1,
		},
		needsHeaderUpdate: false,
	}
}

func NewChunkIOWithMetadata(bh BlobHandle, c Cipher, h ChunkHeader) *ChunkIO {
	ch := NewChunkIO(bh, c)
	ch.header = h
	return ch
}

func (ch *ChunkIO) ensureHeader() error {
	if ch.didReadHeader {
		return errors.New("Already read header.")
	}

	if ch.bh.Size() == 0 {
		w := &OffsetWriter{ch.bh, 0}
		if err := ch.header.WriteTo(w, ch.c); err != nil {
			return fmt.Errorf("Failed to init header/prologue: %v", err)
		}

		ch.didReadHeader = true
		return nil
	}

	if err := ch.header.ReadFrom(&OffsetReader{ch.bh, 0}, ch.c); err != nil {
		return fmt.Errorf("Failed to read header: %v", err)
	}

	ch.didReadHeader = true
	return nil
}

func (ch *ChunkIO) Size() int64 {
	return int64(ch.PayloadLen())
}

func (ch *ChunkIO) Truncate(size int64) error {
	return fmt.Errorf("FIXME: implement ChunkIO.Truncate!")
}

func (ch *ChunkIO) PayloadLen() int {
	if err := ch.ensureHeader(); err != nil {
		log.Printf("Failed to read iheader for payload len: %v", err)
		return 0
	}

	return int(ch.header.PayloadLen)
}

func (ch *ChunkIO) expandLengthBy(by int) error {
	if by < 0 {
		panic("Tried to expand by negative length")
	}

	if by == 0 {
		return nil
	}

	len64 := int64(ch.PayloadLen())
	if len64+int64(by) > MaxChunkPayloadLen {
		return fmt.Errorf("Payload length out of range. Current: %d += %d", len64, by)
	}

	ch.header.PayloadLen = uint32(ch.PayloadLen() + by)
	ch.needsHeaderUpdate = true

	return nil
}

func (ch *ChunkIO) encryptedFrameOffset(i int) int {
	encryptedFrameSize := ch.c.EncryptedFrameSize(ContentFramePayloadLength)
	return ChunkHeaderLength + encryptedFrameSize*i
}

type decryptedContentFrame struct {
	P      []byte
	Offset int

	IsLastFrame bool
}

func (ch *ChunkIO) readContentFrame(i int) (*decryptedContentFrame, error) {
	// the frame carries a part of the content at offset
	offset := i * ContentFramePayloadLength

	// payload length of the encrypted frame
	framePayloadLen := ContentFramePayloadLength
	isLastFrame := false
	distToLast := ch.PayloadLen() - offset
	if distToLast <= ContentFramePayloadLength {
		framePayloadLen = distToLast
		isLastFrame = true
	}

	// the offset of the start of the frame in blob
	blobOffset := ch.encryptedFrameOffset(i)

	rd := &OffsetReader{ch.bh, int64(blobOffset)}
	bdr, err := NewBtnDecryptReader(rd, ch.c, framePayloadLen)
	if err != nil {
		return nil, fmt.Errorf("Failed to create BtnDecryptReader: %v", err)
	}

	p := make([]byte, framePayloadLen, ContentFramePayloadLength)
	if _, err := io.ReadFull(bdr, p); err != nil {
		return nil, fmt.Errorf("Failed to decrypt frame idx: %d, err: %v", i, err)
	}
	if !bdr.HasReadAll() {
		panic("Incomplete frame read")
	}

	fmt.Printf("Read content frame idx: %d\n", i)
	return &decryptedContentFrame{
		P: p, Offset: offset,
		IsLastFrame: isLastFrame,
	}, nil
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

func (ch *ChunkIO) PRead(offset int64, p []byte) error {
	if err := ch.ensureHeader(); err != nil {
		return err
	}

	if offset < 0 || math.MaxInt32 < offset {
		return fmt.Errorf("Offset out of int32 range: %d", offset)
	}

	fmt.Printf("PRead off %d len %d. Chunk payload len: %d\n", offset, len(p), ch.PayloadLen())

	remo := int(offset)
	remp := p
	for len(remp) > 0 {
		i := remo / ContentFramePayloadLength
		f, err := ch.readContentFrame(i)
		if err != nil {
			return err
		}
		inframeOffset := remo - f.Offset
		if inframeOffset < 0 {
			panic("ASSERT: inframeOffset must be non-negative here")
		}
		// fmt.Printf("Decoded content frame. %+v\n", f)

		n := len(remp)
		valid := len(f.P) - inframeOffset // valid payload after offset
		// fmt.Printf("n: %d. valid: %d\n", n, valid)
		if n > valid {
			if f.IsLastFrame {
				return fmt.Errorf("Attempted to read beyond written size: %d. inframeOffset: %d, framePayloadLen: %d", remo, inframeOffset, len(f.P))
			}

			n = valid
		}

		copy(remp[:n], f.P[inframeOffset:])
		fmt.Printf("read %d bytes for off %d len %d\n", n, remo, len(remp))

		remo += n
		remp = remp[n:]
	}
	return nil
}

func (ch *ChunkIO) PWrite(offset int64, p []byte) error {
	fmt.Printf("PWrite: offset %d, len %d\n", offset, len(p))

	if err := ch.ensureHeader(); err != nil {
		return err
	}

	if len(p) == 0 {
		return nil
	}
	if offset < 0 || math.MaxInt32 < offset {
		return fmt.Errorf("Offset out of range: %d", offset)
	}

	remo := int(offset)
	remp := p

	if remo > ch.PayloadLen() {
		// if expanding, zero fill content frames up to write offset

		zfoff := ch.PayloadLen()
		zflen := remo - ch.PayloadLen()

		for zflen > 0 {
			fmt.Printf("zfoff: %d, zflen: %d\n", zfoff, zflen)
			i := zfoff / ContentFramePayloadLength
			fOffset := i * ContentFramePayloadLength

			var f *decryptedContentFrame

			inframeOffset := zfoff - fOffset
			if zfoff == ch.PayloadLen() && inframeOffset == 0 {
				fmt.Printf("PWrite: write new zero fill frame")

				// FIXME: maybe skip writing pure 0 frame.
				//        Old sambad writes a byte of the end of the file instead of ftruncate, which is a nightmare in the current impl.

				n := IntMin(zflen, ContentFramePayloadLength)

				f = &decryptedContentFrame{
					P:           ZeroContent[:n],
					Offset:      fOffset,
					IsLastFrame: false,
				}

				zfoff += n
				zflen -= n
				ch.expandLengthBy(n)
				fmt.Printf(" len: %d\n", n)
			} else {
				n := IntMin(zflen, ContentFramePayloadLength-inframeOffset)
				fmt.Printf("PWrite: zero fill last of existing content frame. len: %d f.P[%d:%d] = 0\n", n, inframeOffset, inframeOffset+n)

				// read the frame
				var err error
				f, err = ch.readContentFrame(i)
				if err != nil {
					return err
				}
				if fOffset != f.Offset {
					panic("fOffset != f.Offset")
				}

				// expand & zero fill
				f.P = f.P[:inframeOffset+n]
				j := 0
				for j < n {
					f.P[inframeOffset+j] = 0
					j++
				}

				zfoff += n
				zflen -= n
				ch.expandLengthBy(n)
			}

			// writeback the frame
			if err := ch.writeContentFrame(i, f); err != nil {
				return fmt.Errorf("failed to write back the encrypted frame: %v", err)
			}
		}
	}

	for len(remp) > 0 {
		i := remo / ContentFramePayloadLength
		fOffset := i * ContentFramePayloadLength

		var f *decryptedContentFrame
		if remo == ch.PayloadLen() && fOffset == remo {
			fmt.Printf("PWrite: Preparing new frame to append\n")
			f = &decryptedContentFrame{
				P:           make([]byte, 0, ContentFramePayloadLength),
				Offset:      fOffset,
				IsLastFrame: true,
			}
		} else {
			fmt.Printf("PWrite: Read existing frame %d to append/update\n", i)
			var err error
			f, err = ch.readContentFrame(i)
			if err != nil {
				return err
			}
			if fOffset != f.Offset {
				panic("fOffset != f.Offset")
			}
		}

		// modify the payload
		inframeOffset := remo - f.Offset
		if inframeOffset < 0 {
			panic("ASSERT: inframeOffset must be non-negative here")
		}

		n := len(remp)
		valid := len(f.P) - inframeOffset // valid payload after offset
		if len(remp) > valid && f.IsLastFrame {
			// expand the last frame as needed
			newSize := inframeOffset + n
			if newSize > ContentFramePayloadLength {
				f.IsLastFrame = false
				newSize = ContentFramePayloadLength
			}

			fmt.Printf("PWrite: Expanding the last frame from %d to %d\n", len(f.P), newSize)

			expandLen := newSize - len(f.P)
			if err := ch.expandLengthBy(expandLen); err != nil {
				return err
			}

			f.P = f.P[:newSize]
			valid = newSize - inframeOffset
		}
		if valid == 0 {
			panic("Inf loop")
		}
		n = IntMin(n, valid)

		copy(f.P[inframeOffset:inframeOffset+n], remp)

		// writeback the updated encrypted frame
		if err := ch.writeContentFrame(i, f); err != nil {
			return fmt.Errorf("failed to write back the encrypted frame: %v", err)
		}
		fmt.Printf("wrote %d bytes for off %d len %d\n", n, offset, len(remp))

		remo += n
		remp = remp[n:]
	}

	return nil
}

func (ch *ChunkIO) Flush() error {
	if ch.needsHeaderUpdate {
		if err := ch.header.WriteTo(&OffsetWriter{ch.bh, 0}, ch.c); err != nil {
			return fmt.Errorf("Header write failed: %v", err)
		}
		log.Printf("Wrote chunk header: %+v", ch.header)
		ch.needsHeaderUpdate = false
	}
	return nil
}

func (ch *ChunkIO) Close() error {
	return ch.Flush()
}
