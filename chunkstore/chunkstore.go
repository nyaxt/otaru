package chunkstore

import (
	"fmt"
	"io"
	"log"
	"math"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/util"
)

const (
	ContentFramePayloadLength = btncrypt.BtnFrameMaxPayload
)

var (
	ZeroContent = make([]byte, ContentFramePayloadLength)
)

func NewQueryChunkVersion(c btncrypt.Cipher) blobstore.QueryVersionFunc {
	return func(r io.Reader) (blobstore.BlobVersion, error) {
		var h ChunkHeader
		if err := h.ReadFrom(r, c); err != nil {
			if err == io.EOF {
				return 0, nil
			}
			return 0, err
		}

		return blobstore.BlobVersion(h.PayloadVersion), nil
	}
}

func NewChunkWriter(w io.Writer, c btncrypt.Cipher, h ChunkHeader) (io.WriteCloser, error) {
	if err := h.WriteTo(w, c); err != nil {
		return nil, fmt.Errorf("Failed to write header: %v", err)
	}

	return btncrypt.NewWriteCloser(w, c, int(h.PayloadLen))
}

type ChunkReader struct {
	r io.Reader
	c btncrypt.Cipher

	header ChunkHeader

	bdr *btncrypt.Reader
}

func NewChunkReader(r io.Reader, c btncrypt.Cipher) (*ChunkReader, error) {
	cr := &ChunkReader{r: r, c: c}

	if err := cr.header.ReadFrom(r, c); err != nil {
		return nil, fmt.Errorf("Failed to read header: %v", err)
	}

	var err error
	cr.bdr, err = btncrypt.NewReader(cr.r, cr.c, cr.Length())
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
	bh blobstore.BlobHandle
	c  btncrypt.Cipher

	didReadHeader bool
	header        ChunkHeader

	needsHeaderUpdate bool
}

func NewChunkIO(bh blobstore.BlobHandle, c btncrypt.Cipher) *ChunkIO {
	return &ChunkIO{
		bh:            bh,
		c:             c,
		didReadHeader: false,
		header: ChunkHeader{
			OrigFilename:   "<unknown>",
			OrigOffset:     -1,
			PayloadVersion: 1,
		},
		needsHeaderUpdate: false,
	}
}

func NewChunkIOWithMetadata(bh blobstore.BlobHandle, c btncrypt.Cipher, h ChunkHeader) *ChunkIO {
	ch := NewChunkIO(bh, c)
	ch.header = h
	ch.header.PayloadVersion = 1
	return ch
}

func (ch *ChunkIO) ensureHeader() error {
	if ch.didReadHeader {
		return nil
	}

	if ch.bh.Size() == 0 {
		ch.didReadHeader = true
		return nil
	}

	if err := ch.header.ReadFrom(&blobstore.OffsetReader{ch.bh, 0}, ch.c); err != nil {
		return fmt.Errorf("Failed to read header: %v", err)
	}

	ch.didReadHeader = true
	return nil
}

func (ch *ChunkIO) Header() ChunkHeader {
	if err := ch.ensureHeader(); err != nil {
		log.Printf("Failed to ensureHeader(): %v", err)
	}
	return ch.header
}

func (ch *ChunkIO) Size() int64 {
	return int64(ch.PayloadLen())
}

func (ch *ChunkIO) Truncate(size int64) error {
	return fmt.Errorf("FIXME: implement ChunkIO.Truncate!")
}

func (ch *ChunkIO) PayloadLen() int {
	if err := ch.ensureHeader(); err != nil {
		log.Printf("Failed to read header for payload len: %v", err)
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
	log.Printf("ChunkIO expandLength +%d = %d", by, ch.header.PayloadLen)
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

	rd := &blobstore.OffsetReader{ch.bh, int64(blobOffset)}
	bdr, err := btncrypt.NewReader(rd, ch.c, framePayloadLen)
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

	log.Printf("ChunkIO: Read content frame idx: %d", i)
	return &decryptedContentFrame{
		P: p, Offset: offset,
		IsLastFrame: isLastFrame,
	}, nil
}

func (ch *ChunkIO) writeContentFrame(i int, f *decryptedContentFrame) error {
	// the offset of the start of the frame in blob
	blobOffset := ch.encryptedFrameOffset(i)

	wr := &blobstore.OffsetWriter{ch.bh, int64(blobOffset)}
	bew, err := btncrypt.NewWriteCloser(wr, ch.c, len(f.P))
	if err != nil {
		return fmt.Errorf("Failed to create BtnEncryptWriteCloser: %v", err)
	}
	defer func() {
		if err := bew.Close(); err != nil {
			log.Printf("Failed to Close BtnEncryptWriteCloser: %v", err)
		}
	}()
	if _, err := bew.Write(f.P); err != nil {
		return fmt.Errorf("Failed to encrypt frame: %v", err)
	}
	ch.header.PayloadVersion++
	ch.needsHeaderUpdate = true

	log.Printf("ChunkIO: Wrote content frame idx: %d", i)
	return nil
}

func (ch *ChunkIO) PRead(offset int64, p []byte) error {
	if err := ch.ensureHeader(); err != nil {
		return err
	}

	if offset < 0 || math.MaxInt32 < offset {
		return fmt.Errorf("Offset out of int32 range: %d", offset)
	}

	log.Printf("ChunkIO: PRead off %d len %d. Chunk payload len: %d", offset, len(p), ch.PayloadLen())

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
		// log.Printf("ChunkIO: PRead: Decoded content frame. %+v", f)

		n := len(remp)
		valid := len(f.P) - inframeOffset // valid payload after offset
		log.Printf("ChunkIO: PRead n: %d. valid: %d", n, valid)
		if n > valid {
			if f.IsLastFrame {
				return fmt.Errorf("Attempted to read beyond written size: %d. inframeOffset: %d, framePayloadLen: %d", remo, inframeOffset, len(f.P))
			}

			n = valid
		}

		copy(remp[:n], f.P[inframeOffset:])
		log.Printf("ChunkIO: PRead: read %d bytes for off %d len %d", n, remo, len(remp))

		remo += n
		remp = remp[n:]
	}
	return nil
}

func (ch *ChunkIO) PWrite(offset int64, p []byte) error {
	log.Printf("PWrite: offset %d, len %d", offset, len(p))
	// log.Printf("PWrite: p=%v", p)

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
			log.Printf("PWrite zfoff: %d, zflen: %d", zfoff, zflen)
			i := zfoff / ContentFramePayloadLength
			fOffset := i * ContentFramePayloadLength

			var f *decryptedContentFrame

			inframeOffset := zfoff - fOffset
			if zfoff == ch.PayloadLen() && inframeOffset == 0 {
				log.Printf("PWrite: write new zero fill frame")

				// FIXME: maybe skip writing pure 0 frame.
				//        Old sambad writes a byte of the end of the file instead of ftruncate, which is a nightmare in the current impl.

				n := util.IntMin(zflen, ContentFramePayloadLength)

				f = &decryptedContentFrame{
					P:           ZeroContent[:n],
					Offset:      fOffset,
					IsLastFrame: false,
				}

				zfoff += n
				zflen -= n
				ch.expandLengthBy(n)
				log.Printf(" len: %d", n)
			} else {
				n := util.IntMin(zflen, ContentFramePayloadLength-inframeOffset)
				log.Printf("PWrite: zero fill last of existing content frame. len: %d f.P[%d:%d] = 0", n, inframeOffset, inframeOffset+n)

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
			log.Printf("PWrite: Preparing new frame to append")
			f = &decryptedContentFrame{
				P:           make([]byte, 0, ContentFramePayloadLength),
				Offset:      fOffset,
				IsLastFrame: true,
			}
		} else {
			log.Printf("PWrite: Read existing frame %d to append/update", i)
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

			log.Printf("PWrite: Expanding the last frame from %d to %d", len(f.P), newSize)

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
		n = util.IntMin(n, valid)

		copy(f.P[inframeOffset:inframeOffset+n], remp)

		// writeback the updated encrypted frame
		if err := ch.writeContentFrame(i, f); err != nil {
			return fmt.Errorf("failed to write back the encrypted frame: %v", err)
		}
		log.Printf("PWrite: wrote %d bytes for off %d len %d", n, offset, len(remp))

		remo += n
		remp = remp[n:]
	}

	return nil
}

func (ch *ChunkIO) Sync() error {
	if ch.needsHeaderUpdate {
		if err := ch.header.WriteTo(&blobstore.OffsetWriter{ch.bh, 0}, ch.c); err != nil {
			return fmt.Errorf("Header write failed: %v", err)
		}
		log.Printf("Wrote chunk header: %+v", ch.header)
		ch.needsHeaderUpdate = false
	}
	return nil
}

func (ch *ChunkIO) Close() error {
	return ch.Sync()
}
