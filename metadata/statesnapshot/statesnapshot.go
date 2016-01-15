package statesnapshot

import (
	"bytes"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/chunkstore"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("statess")

func SaveBytes(blobpath string, c *btncrypt.Cipher, bs blobstore.BlobStore, p []byte) error {
	w, err := bs.OpenWriter(blobpath)
	if err != nil {
		return err
	}
	cw, err := chunkstore.NewChunkWriter(w, c, chunkstore.ChunkHeader{
		PayloadLen:     uint32(len(p)),
		PayloadVersion: 1,
		OrigFilename:   blobpath,
		OrigOffset:     0,
	})
	if err != nil {
		return err
	}

	es := []error{}
	if _, err := cw.Write(p); err != nil {
		es = append(es, fmt.Errorf("Failed to write to ChunkWriter: %v", err))
	}

	if err := cw.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close ChunkWriter: %v", err))
	}
	if err := w.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close blobhandle: %v", err))
	}

	if err := util.ToErrors(es); err != nil {
		return err
	}
	return nil
}

type EncodeCallback func(enc *gob.Encoder) error

func EncodeBytes(cb EncodeCallback) ([]byte, error) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	enc := gob.NewEncoder(zw)

	es := []error{}
	if err := cb(enc); err != nil {
		es = append(es, fmt.Errorf("Failed to encode state: %v", err))
	}
	if err := zw.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close zlib Writer: %v", err))
	}

	if err := util.ToErrors(es); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Save(blobpath string, c *btncrypt.Cipher, bs blobstore.BlobStore, cb EncodeCallback) error {
	p, err := EncodeBytes(cb)
	if err != nil {
		return err
	}

	return SaveBytes(blobpath, c, bs, p)
}

type DecodeCallback func(dec *gob.Decoder) error

func Restore(blobpath string, c *btncrypt.Cipher, bs blobstore.BlobStore, cb DecodeCallback) error {
	r, err := bs.OpenReader(blobpath)
	if err != nil {
		return err
	}

	cr, err := chunkstore.NewChunkReader(r, c)
	if err != nil {
		return err
	}
	logger.Debugf(mylog, "serialized blob size: %d", cr.Length())
	zr, err := zlib.NewReader(&io.LimitedReader{cr, int64(cr.Length())})
	if err != nil {
		return err
	}
	logger.Debugf(mylog, "statesnapshot.Restore: zlib init success!")
	dec := gob.NewDecoder(zr)

	es := []error{}
	if err := cb(dec); err != nil {
		es = append(es, fmt.Errorf("Failed to decode state: %v", err))
	}
	if err := zr.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close zlib Reader: %v", err))
	}
	if err := r.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close BlobHandle: %v", err))
	}

	if err := util.ToErrors(es); err != nil {
		return err
	}
	return nil
}
