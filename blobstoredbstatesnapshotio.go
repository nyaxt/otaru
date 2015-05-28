package otaru

import (
	"bufio"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"io"
	"log"

	"github.com/nyaxt/otaru/blobstore"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
)

const (
	INodeDBSnapshotBlobpath = "INODEDB_SNAPSHOT"
)

type BlobStoreDBStateSnapshotIO struct {
	bs blobstore.RandomAccessBlobStore
	c  Cipher
}

var _ = inodedb.DBStateSnapshotIO(&BlobStoreDBStateSnapshotIO{})

func NewBlobStoreDBStateSnapshotIO(bs blobstore.RandomAccessBlobStore, c Cipher) *BlobStoreDBStateSnapshotIO {
	return &BlobStoreDBStateSnapshotIO{bs, c}
}

func (sio *BlobStoreDBStateSnapshotIO) SaveSnapshot(s *inodedb.DBState) error {
	raw, err := sio.bs.Open(INodeDBSnapshotBlobpath, fl.O_RDWR|fl.O_CREATE)
	if err != nil {
		return err
	}
	if err := raw.Truncate(0); err != nil {
		return err
	}

	cio := NewChunkIOWithMetadata(raw, sio.c, ChunkHeader{
		OrigFilename: "*INODEDB_SNAPSHOT*",
		OrigOffset:   0,
	})
	bufio := bufio.NewWriter(&blobstore.OffsetWriter{cio, 0})
	zw := zlib.NewWriter(bufio)
	enc := gob.NewEncoder(zw)
	if err := s.EncodeToGob(enc); err != nil {
		return fmt.Errorf("Failed to encode DBState: %v", err)
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := bufio.Flush(); err != nil {
		return err
	}
	if err := cio.Close(); err != nil {
		return err
	}
	if err := raw.Close(); err != nil {
		return err
	}

	return nil
}

func (sio *BlobStoreDBStateSnapshotIO) RestoreSnapshot() (*inodedb.DBState, error) {
	raw, err := sio.bs.Open(INodeDBSnapshotBlobpath, fl.O_RDONLY)
	if err != nil {
		return nil, err
	}

	cio := NewChunkIO(raw, sio.c)
	log.Printf("serialized blob size: %d", cio.Size())
	zr, err := zlib.NewReader(&io.LimitedReader{&blobstore.OffsetReader{cio, 0}, cio.Size()})
	if err != nil {
		return nil, err
	}
	log.Printf("LoadINodeDBFromBlobStore: zlib init success!")
	dec := gob.NewDecoder(zr)
	state, err := inodedb.DecodeDBStateFromGob(dec)
	if err != nil {
		return nil, err
	}
	if err := zr.Close(); err != nil {
		return nil, err
	}
	if err := cio.Close(); err != nil {
		return nil, err
	}
	if err := raw.Close(); err != nil {
		return nil, err
	}

	return state, nil
}
