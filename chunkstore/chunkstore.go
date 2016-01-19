package chunkstore

import (
	"fmt"
	"io"
	"math"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/blobstore/version"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("chunkstore")

const (
	ContentFramePayloadLength = btncrypt.BtnFrameMaxPayload
)

var (
	ZeroContent = make([]byte, ContentFramePayloadLength)
)

func NewQueryChunkVersion(c *btncrypt.Cipher) version.QueryFunc {
	return func(r io.Reader) (version.Version, error) {
		var h ChunkHeader
		if err := h.ReadFrom(r, c); err != nil {
			if err == io.EOF {
				return 0, nil
			}
			return 0, err
		}

		return version.Version(h.PayloadVersion), nil
	}
}

func NewChunkWriter(w io.Writer, c *btncrypt.Cipher, h ChunkHeader) (io.WriteCloser, error) {
	if err := h.WriteTo(w, c); err != nil {
		return nil, fmt.Errorf("Failed to write header: %v", err)
	}

	return c.NewWriteCloser(w, int(h.PayloadLen))
}

type ChunkReader struct {
	r io.Reader
	c *btncrypt.Cipher

	header ChunkHeader

	bdr *btncrypt.Reader
}

func NewChunkReader(r io.Reader, c *btncrypt.Cipher) (*ChunkReader, error) {
	cr := &ChunkReader{r: r, c: c}

	if err := cr.header.ReadFrom(r, c); err != nil {
		return nil, fmt.Errorf("Failed to read header: %v", err)
	}

	var err error
	cr.bdr, err = cr.c.NewReader(cr.r, cr.Length())
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

func (cr *ChunkReader) Close() error {
	cr.bdr.Close()
	return nil
}

// ChunkIO provides RandomAccessIO for blobchunk
type ChunkIO struct {
	bh blobstore.BlobHandle
	c  *btncrypt.Cipher

	didReadHeader bool
	header        ChunkHeader

	needsHeaderUpdate bool
}

func NewChunkIO(bh blobstore.BlobHandle, c *btncrypt.Cipher) *ChunkIO {
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

func NewChunkIOWithMetadata(bh blobstore.BlobHandle, c *btncrypt.Cipher, h ChunkHeader) *ChunkIO {
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
		logger.Criticalf(mylog, "Failed to ensureHeader(): %v", err)
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
		logger.Warningf(mylog, "Failed to read header for payload len: %v", err)
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
	logger.Debugf(mylog, "ChunkIO expandLength +%d = %d", by, ch.header.PayloadLen)
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

	enc := ch.c.GetEncryptedFrameBuf()[:ch.c.EncryptedFrameSize(framePayloadLen)]
	defer ch.c.PutEncryptedFrameBuf(enc)
	if err := ch.bh.PRead(enc, int64(blobOffset)); err != nil {
		return nil, fmt.Errorf("Failed to read encrypted frame: %v", err)
	}

	dec := ch.c.GetDecryptedFrameBuf()
	dec, err := ch.c.DecryptFrame(dec, enc)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt frame idx: %d, err: %v", i, err)
	}

	logger.Debugf(mylog, "ChunkIO: Read content frame idx: %d", i)
	return &decryptedContentFrame{
		P: dec, Offset: offset,
		IsLastFrame: isLastFrame,
	}, nil
}

func (ch *ChunkIO) writeContentFrame(i int, f *decryptedContentFrame) error {
	// the offset of the start of the frame in blob
	blobOffset := ch.encryptedFrameOffset(i)

	enc := ch.c.GetEncryptedFrameBuf()
	defer ch.c.PutEncryptedFrameBuf(enc)
	enc = ch.c.EncryptFrame(enc, f.P)

	if err := ch.bh.PWrite(enc, int64(blobOffset)); err != nil {
		return fmt.Errorf("Failed to write encrypted frame: %v", err)
	}

	ch.header.PayloadVersion++
	ch.needsHeaderUpdate = true

	logger.Debugf(mylog, "ChunkIO: Wrote content frame idx: %d", i)
	return nil
}

func (ch *ChunkIO) PRead(p []byte, offset int64) error {
	if err := ch.ensureHeader(); err != nil {
		return err
	}

	if offset < 0 || math.MaxInt32 < offset {
		return fmt.Errorf("Offset out of int32 range: %d", offset)
	}

	logger.Debugf(mylog, "ChunkIO: PRead off %d len %d. Chunk payload len: %d", offset, len(p), ch.PayloadLen())

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

		n := len(remp)
		valid := len(f.P) - inframeOffset // valid payload after offset
		logger.Debugf(mylog, "ChunkIO: PRead n: %d. valid: %d", n, valid)
		if n > valid {
			if f.IsLastFrame {
				return fmt.Errorf("Attempted to read beyond written size: %d. inframeOffset: %d, framePayloadLen: %d", remo, inframeOffset, len(f.P))
			}

			n = valid
		}

		copy(remp[:n], f.P[inframeOffset:])
		logger.Debugf(mylog, "ChunkIO: PRead: read %d bytes for off %d len %d", n, remo, len(remp))

		remo += n
		remp = remp[n:]
	}
	return nil
}

func (ch *ChunkIO) PWrite(p []byte, offset int64) error {
	logger.Debugf(mylog, "PWrite: offset %d, len %d", offset, len(p))

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
			logger.Debugf(mylog, "PWrite zfoff: %d, zflen: %d", zfoff, zflen)
			i := zfoff / ContentFramePayloadLength
			fOffset := i * ContentFramePayloadLength

			inframeOffset := zfoff - fOffset
			if zfoff == ch.PayloadLen() && inframeOffset == 0 {
				logger.Debugf(mylog, "PWrite: write new zero fill frame")

				// FIXME: maybe skip writing pure 0 frame.
				//        Old sambad writes a byte of the end of the file instead of ftruncate, which is a nightmare in the current impl.

				n := util.IntMin(zflen, ContentFramePayloadLength)

				f := &decryptedContentFrame{
					P:           ZeroContent[:n],
					Offset:      fOffset,
					IsLastFrame: false,
				}

				zfoff += n
				zflen -= n
				ch.expandLengthBy(n)
				logger.Debugf(mylog, " len: %d", n)

				if err := ch.writeContentFrame(i, f); err != nil {
					return fmt.Errorf("failed to write zero fill frame: %v", err)
				}
			} else {
				n := util.IntMin(zflen, ContentFramePayloadLength-inframeOffset)
				logger.Debugf(mylog, "PWrite: zero fill last of existing content frame. len: %d f.P[%d:%d] = 0", n, inframeOffset, inframeOffset+n)

				// read the frame
				f, err := ch.readContentFrame(i)
				if err != nil {
					return err
				}
				if fOffset != f.Offset {
					panic("fOffset != f.Offset")
				}

				// expand & zero fill
				f.P = append(f.P[:inframeOffset], ZeroContent[:n]...)

				zfoff += n
				zflen -= n
				ch.expandLengthBy(n)

				// writeback the frame
				if err := ch.writeContentFrame(i, f); err != nil {
					return fmt.Errorf("failed to write back the encrypted frame: %v", err)
				}
				ch.c.PutDecryptedFrameBuf(f.P)
			}
		}
	}

	for len(remp) > 0 {
		i := remo / ContentFramePayloadLength
		fOffset := i * ContentFramePayloadLength

		var f *decryptedContentFrame
		if remo == ch.PayloadLen() && fOffset == remo {
			logger.Debugf(mylog, "PWrite: Preparing new frame to append")
			f = &decryptedContentFrame{
				P:           ch.c.GetDecryptedFrameBuf()[:0],
				Offset:      fOffset,
				IsLastFrame: true,
			}
		} else {
			logger.Debugf(mylog, "PWrite: Read existing frame %d to append/update", i)
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

			logger.Debugf(mylog, "PWrite: Expanding the last frame from %d to %d", len(f.P), newSize)

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
		logger.Debugf(mylog, "PWrite: wrote %d bytes for off %d len %d", n, offset, len(remp))

		ch.c.PutDecryptedFrameBuf(f.P)

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
		logger.Debugf(mylog, "Wrote chunk header: %+v", ch.header)
		ch.needsHeaderUpdate = false
	}
	return nil
}

func (ch *ChunkIO) Close() error {
	return ch.Sync()
}
