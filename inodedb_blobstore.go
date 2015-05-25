package otaru

import (
	"bufio"
	"compress/zlib"
	"io"
	"log"

	"github.com/nyaxt/otaru/blobstore"
)

const (
	INodeDBSnapshotBlobpath = "INODEDB_SNAPSHOT"
)

func (idb *INodeDB) SaveToBlobStore(bs blobstore.RandomAccessBlobStore, c Cipher) error {
	raw, err := bs.Open(INodeDBSnapshotBlobpath, blobstore.O_RDWR|blobstore.O_CREATE)
	if err != nil {
		return err
	}
	if err := raw.Truncate(0); err != nil {
		return err
	}

	cio := NewChunkIOWithMetadata(raw, c, ChunkHeader{
		OrigFilename: "*INODEDB_SNAPSHOT*",
		OrigOffset:   0,
	})
	bufio := bufio.NewWriter(&blobstore.OffsetWriter{cio, 0})
	zw := zlib.NewWriter(bufio)
	if err := idb.SerializeSnapshot(zw); err != nil {
		return err
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

func LoadINodeDBFromBlobStore(bs blobstore.RandomAccessBlobStore, c Cipher) (*INodeDB, error) {
	raw, err := bs.Open(INodeDBSnapshotBlobpath, blobstore.O_RDONLY)
	if err != nil {
		return nil, err
	}

	cio := NewChunkIO(raw, c)
	log.Printf("serialized blob size: %d", cio.Size())
	zr, err := zlib.NewReader(&io.LimitedReader{&blobstore.OffsetReader{cio, 0}, cio.Size()})
	if err != nil {
		return nil, err
	}
	log.Printf("LoadINodeDBFromBlobStore: zlib init success!")
	idb, err := DeserializeINodeDBSnapshot(zr)
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

	return idb, nil
}
