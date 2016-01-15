package chunkstore

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/logger"
)

const (
	ChunkSignatureMagic1 = 0x05 // o t
	ChunkSignatureMagic2 = 0xa6 // a ru

	SignatureLength   = 2
	ChunkHeaderLength = 4096

	MaxChunkPayloadLen = math.MaxInt32
	MaxOrigFilenameLen = 1024

	CurrentFormat             byte = 0x03
	CurrentFrameEncapsulation byte = 0x02
)

type ChunkHeader struct {
	FrameEncapsulation byte
	PayloadLen         uint32
	PayloadVersion     int64
	OrigFilename       string
	OrigOffset         int64
}

func (h ChunkHeader) WriteTo(w io.Writer, c *btncrypt.Cipher) error {
	h.FrameEncapsulation = CurrentFrameEncapsulation

	if h.PayloadLen > MaxChunkPayloadLen {
		return fmt.Errorf("payload length too big: %d", h.PayloadLen)
	}

	if len(h.OrigFilename) > MaxOrigFilenameLen {
		h.OrigFilename = filepath.Base(h.OrigFilename)
		if len(h.OrigFilename) > MaxOrigFilenameLen {
			h.OrigFilename = "<filename_too_long>"
		}
	}

	if _, err := w.Write([]byte{ChunkSignatureMagic1, ChunkSignatureMagic2}); err != nil {
		return fmt.Errorf("Failed to write signature magic: %v", err)
	}
	if _, err := w.Write([]byte{CurrentFormat}); err != nil {
		return fmt.Errorf("Failed to write format byte: %v", err)
	}

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(h); err != nil {
		return err
	}
	framelen := ChunkHeaderLength - c.FrameOverhead() - SignatureLength - 1
	paddinglen := framelen - b.Len()
	if paddinglen < 0 {
		logger.Panicf(mylog, "SHOULD NOT BE REACHED: Marshaled ChunkHeader size too large")
	}

	bew, err := c.NewWriteCloser(w, framelen)
	if _, err := b.WriteTo(bew); err != nil {
		return fmt.Errorf("Failed to initialize frame encryptor: %v", err)
	}
	if err != nil {
		return fmt.Errorf("Header frame gob payload write failed: %v", err)
	}
	// zero padding
	if _, err := bew.Write(make([]byte, paddinglen)); err != nil {
		return fmt.Errorf("Header frame zero padding write failed: %v", err)
	}
	if err := bew.Close(); err != nil {
		return fmt.Errorf("Header frame close failed: %v", err)
	}
	return nil
}

func (h *ChunkHeader) ReadFrom(r io.Reader, c *btncrypt.Cipher) error {
	magic := make([]byte, SignatureLength+1)
	if _, err := r.Read(magic); err != nil {
		if err == io.EOF {
			return err
		}
		return fmt.Errorf("Failed to read signature magic / format bytes: %v", err)
	}
	if magic[0] != ChunkSignatureMagic1 ||
		magic[1] != ChunkSignatureMagic2 {
		return errors.New("signature magic mismatch")
	}
	if magic[2] != CurrentFormat {
		return fmt.Errorf("Expected format version %x but got %x", CurrentFormat, magic[2])
	}

	framelen := ChunkHeaderLength - c.FrameOverhead() - SignatureLength - 1
	bdr, err := c.NewReader(r, framelen)
	if err != nil {
		return err
	}
	defer bdr.Close()

	encoded := make([]byte, framelen)
	if _, err := io.ReadFull(bdr, encoded); err != nil {
		return fmt.Errorf("Failed to read header frame: %v", err)
	}
	if !bdr.HasReadAll() {
		panic("Incomplete read in prologue frame !?!?")
	}

	dec := gob.NewDecoder(bytes.NewBuffer(encoded))
	if err := dec.Decode(h); err != nil {
		return err
	}

	return nil
}
