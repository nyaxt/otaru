package blobstoredbstatesnapshotio

import (
	"encoding/gob"
	"fmt"
	"log"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/metadata"
	"github.com/nyaxt/otaru/metadata/statesnapshot"
)

type SSLocator interface {
	Locate(history int) (string, error)
	Put(blobpath string, txid int64) error
}

type DBStateSnapshotIO struct {
	bs blobstore.RandomAccessBlobStore
	c  btncrypt.Cipher

	loc SSLocator

	snapshotVer inodedb.TxID
}

var _ = inodedb.DBStateSnapshotIO(&DBStateSnapshotIO{})

func New(bs blobstore.RandomAccessBlobStore, c btncrypt.Cipher, loc SSLocator) *DBStateSnapshotIO {
	return &DBStateSnapshotIO{bs: bs, c: c, loc: loc, snapshotVer: -1}
}

func (sio *DBStateSnapshotIO) SaveSnapshot(s *inodedb.DBState) error {
	currVer := s.Version()
	if sio.snapshotVer > currVer {
		log.Printf("SaveSnapshot: ASSERT fail: snapshot version %d newer than current ver %d", sio.snapshotVer, currVer)
	} else if sio.snapshotVer == currVer {
		log.Printf("SaveSnapshot: Current ver %d is already snapshotted. No-op.", sio.snapshotVer)
		return nil
	}

	ssbp := metadata.GenINodeDBSnapshotBlobpath()
	if err := statesnapshot.Save(
		ssbp, sio.c, sio.bs,
		func(enc *gob.Encoder) error { return s.EncodeToGob(enc) },
	); err != nil {
		return err
	}

	if err := sio.loc.Put(ssbp, int64(s.Version())); err != nil {
		return err
	}

	sio.snapshotVer = s.Version()
	return nil
}

const maxhist = 3

func (sio *DBStateSnapshotIO) RestoreSnapshot() (*inodedb.DBState, error) {
	var state *inodedb.DBState

	for i := 0; i < maxhist; i++ {
		ssbp, err := sio.loc.Locate(i)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			log.Printf("Attempting to restore latest state snapshot \"%s\"", ssbp)
		} else {
			log.Printf("Retrying state snapshot restore with an older state snapshot \"%s\"", ssbp)
		}

		if err := statesnapshot.Restore(
			ssbp, sio.c, sio.bs,
			func(dec *gob.Decoder) error {
				var err error
				state, err = inodedb.DecodeDBStateFromGob(dec)
				return err
			},
		); err != nil {
			log.Printf("Failed to recover state snapshot \"%s\": %v", ssbp, err)
			continue
		}
		sio.snapshotVer = state.Version()
		return state, nil
	}
	return nil, fmt.Errorf("Failed to restore %d snapshots. Aborted.", maxhist)
}
