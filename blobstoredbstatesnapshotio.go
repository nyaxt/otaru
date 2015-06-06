package otaru

import (
	"bufio"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"io"
	"log"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/util"
)

const (
	INodeDBSnapshotBlobpath = "INODEDB_SNAPSHOT"
)

type BlobStoreDBStateSnapshotIO struct {
	bs blobstore.RandomAccessBlobStore
	c  btncrypt.Cipher

	snapshotVer inodedb.TxID
}

var _ = inodedb.DBStateSnapshotIO(&BlobStoreDBStateSnapshotIO{})

func NewBlobStoreDBStateSnapshotIO(bs blobstore.RandomAccessBlobStore, c btncrypt.Cipher) *BlobStoreDBStateSnapshotIO {
	return &BlobStoreDBStateSnapshotIO{bs: bs, c: c, snapshotVer: -1}
}

func (sio *BlobStoreDBStateSnapshotIO) SaveSnapshot(s *inodedb.DBState) error {
	currVer := s.Version()
	if sio.snapshotVer > currVer {
		log.Printf("SaveSnapshot: ASSERT fail: snapshot version %d newer than current ver %d", sio.snapshotVer, currVer)
	} else if sio.snapshotVer == currVer {
		log.Printf("SaveSnapshot: Current ver %d is already snapshotted. No-op.", sio.snapshotVer)
		return nil
	}

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

	es := []error{}
	if err := s.EncodeToGob(enc); err != nil {
		es = append(es, fmt.Errorf("Failed to encode DBState: %v", err))
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
	sio.snapshotVer = s.Version()
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

	es := []error{}
	state, err := inodedb.DecodeDBStateFromGob(dec)
	if err != nil {
		es = append(es, fmt.Errorf("Failed to decode dbstate: %v", err))
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
		return nil, err
	}
	sio.snapshotVer = state.Version()
	return state, nil
}
