package statesnapshot

import (
	"bytes"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"io"

	"go.uber.org/multierr"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/chunkstore"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("statess")

func SaveBytes(w io.Writer, c *btncrypt.Cipher, p []byte) error {
	cw, err := chunkstore.NewChunkWriter(w, c, chunkstore.ChunkHeader{
		PayloadLen:     uint32(len(p)),
		PayloadVersion: 1,
		OrigFilename:   "statesnapshot",
		OrigOffset:     0,
	})
	if err != nil {
		return err
	}

	var me error
	if _, err := cw.Write(p); err != nil {
		me = multierr.Append(me, fmt.Errorf("Failed to write to ChunkWriter: %w", err))
	}
	if err := cw.Close(); err != nil {
		me = multierr.Append(me, fmt.Errorf("Failed to close ChunkWriter: %w", err))
	}
	return me
}

type EncodeCallback func(enc *gob.Encoder) error

func EncodeBytes(cb EncodeCallback) ([]byte, error) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	enc := gob.NewEncoder(zw)

	var me error
	if err := cb(enc); err != nil {
		me = multierr.Append(me, fmt.Errorf("Failed to encode state: %w", err))
	}
	if err := zw.Close(); err != nil {
		me = multierr.Append(me, fmt.Errorf("Failed to close zlib Writer: %w", err))
	}

	if me != nil {
		return nil, me
	}
	return buf.Bytes(), nil
}

func Save(w io.Writer, c *btncrypt.Cipher, cb EncodeCallback) error {
	p, err := EncodeBytes(cb)
	if err != nil {
		return err
	}

	return SaveBytes(w, c, p)
}

type DecodeCallback func(dec *gob.Decoder) error

func Restore(r io.Reader, c *btncrypt.Cipher, cb DecodeCallback) error {
	cr, err := chunkstore.NewChunkReader(r, c)
	if err != nil {
		return err
	}
	defer cr.Close()

	logger.Debugf(mylog, "serialized blob size: %d", cr.Length())
	zr, err := zlib.NewReader(&io.LimitedReader{cr, int64(cr.Length())})
	if err != nil {
		return err
	}
	logger.Debugf(mylog, "statesnapshot.Restore: zlib init success!")
	dec := gob.NewDecoder(zr)

	var me error
	if err := cb(dec); err != nil {
		me = multierr.Append(me, fmt.Errorf("Failed to decode state: %w", err))
	}
	if err := zr.Close(); err != nil {
		me = multierr.Append(me, fmt.Errorf("Failed to close zlib Reader: %w", err))
	}
	return err
}
