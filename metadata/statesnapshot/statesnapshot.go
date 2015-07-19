package statesnapshot

import (
	"bufio"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"io"
	"log"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/chunkstore"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/util"
)

type EncodeCallback func(enc *gob.Encoder) error

func Save(blobpath string, c btncrypt.Cipher, bs blobstore.RandomAccessBlobStore, cb EncodeCallback) error {
	raw, err := bs.Open(blobpath, fl.O_RDWR|fl.O_CREATE)
	if err != nil {
		return err
	}
	if err := raw.Truncate(0); err != nil {
		return err
	}

	cio := chunkstore.NewChunkIOWithMetadata(raw, c, chunkstore.ChunkHeader{
		OrigFilename: blobpath,
		OrigOffset:   0,
	})
	bufio := bufio.NewWriter(&blobstore.OffsetWriter{cio, 0})
	zw := zlib.NewWriter(bufio)
	enc := gob.NewEncoder(zw)

	es := []error{}
	if err := cb(enc); err != nil {
		es = append(es, fmt.Errorf("Failed to encode state: %v", err))
	}
	if err := zw.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close zlib Writer: %v", err))
	}
	if err := bufio.Flush(); err != nil {
		es = append(es, fmt.Errorf("Failed to close bufio: %v", err))
	}
	if err := cio.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close ChunkIO: %v", err))
	}
	if err := raw.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close blobhandle: %v", err))
	}

	if err := util.ToErrors(es); err != nil {
		return err
	}
	return nil
}

type DecodeCallback func(dec *gob.Decoder) error

func Restore(blobpath string, c btncrypt.Cipher, bs blobstore.RandomAccessBlobStore, cb DecodeCallback) error {
	raw, err := bs.Open(blobpath, fl.O_RDONLY)
	if err != nil {
		return err
	}

	cio := chunkstore.NewChunkIO(raw, c)
	log.Printf("serialized blob size: %d", cio.Size())
	zr, err := zlib.NewReader(&io.LimitedReader{&blobstore.OffsetReader{cio, 0}, cio.Size()})
	if err != nil {
		return err
	}
	log.Printf("LoadINodeDBFromBlobStore: zlib init success!")
	dec := gob.NewDecoder(zr)

	es := []error{}
	if err := cb(dec); err != nil {
		es = append(es, fmt.Errorf("Failed to decode state: %v", err))
	}
	if err := zr.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close zlib Reader: %v", err))
	}
	if err := cio.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close ChunkIO: %v", err))
	}
	if err := raw.Close(); err != nil {
		es = append(es, fmt.Errorf("Failed to close BlobHandle: %v", err))
	}

	if err := util.ToErrors(es); err != nil {
		return err
	}
	return nil
}
