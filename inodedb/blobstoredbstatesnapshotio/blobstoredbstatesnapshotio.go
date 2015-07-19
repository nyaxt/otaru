package blobstoredbstatesnapshotio

import (
	"encoding/gob"
	"log"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/metadata"
	"github.com/nyaxt/otaru/metadata/statesnapshot"
)

type DBStateSnapshotIO struct {
	bs blobstore.RandomAccessBlobStore
	c  btncrypt.Cipher

	snapshotVer inodedb.TxID
}

var _ = inodedb.DBStateSnapshotIO(&DBStateSnapshotIO{})

func New(bs blobstore.RandomAccessBlobStore, c btncrypt.Cipher) *DBStateSnapshotIO {
	return &DBStateSnapshotIO{bs: bs, c: c, snapshotVer: -1}
}

func (sio *DBStateSnapshotIO) SaveSnapshot(s *inodedb.DBState) error {
	currVer := s.Version()
	if sio.snapshotVer > currVer {
		log.Printf("SaveSnapshot: ASSERT fail: snapshot version %d newer than current ver %d", sio.snapshotVer, currVer)
	} else if sio.snapshotVer == currVer {
		log.Printf("SaveSnapshot: Current ver %d is already snapshotted. No-op.", sio.snapshotVer)
		return nil
	}

	if err := statesnapshot.Save(
		metadata.INodeDBSnapshotBlobpath, sio.c, sio.bs,
		func(enc *gob.Encoder) error { return s.EncodeToGob(enc) },
	); err != nil {
		return err
	}

	sio.snapshotVer = s.Version()
	return nil
}

func (sio *DBStateSnapshotIO) RestoreSnapshot() (*inodedb.DBState, error) {
	var state *inodedb.DBState

	if err := statesnapshot.Restore(
		metadata.INodeDBSnapshotBlobpath, sio.c, sio.bs,
		func(dec *gob.Decoder) error {
			var err error
			state, err = inodedb.DecodeDBStateFromGob(dec)
			return err
		},
	); err != nil {
		return nil, err
	}
	sio.snapshotVer = state.Version()
	return state, nil
}
