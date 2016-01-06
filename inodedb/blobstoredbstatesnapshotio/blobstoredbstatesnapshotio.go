package blobstoredbstatesnapshotio

import (
	"encoding/gob"
	"fmt"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/metadata/statesnapshot"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("bsdbssio")

type SSLocator interface {
	Locate(history int) (string, int64, error)
	GenerateBlobpath() string
	Put(blobpath string, txid int64) error
}

type DBStateSnapshotIO struct {
	bs blobstore.BlobStore
	c  btncrypt.Cipher

	loc SSLocator

	snapshotVer inodedb.TxID
}

var _ = inodedb.DBStateSnapshotIO(&DBStateSnapshotIO{})

func New(bs blobstore.BlobStore, c btncrypt.Cipher, loc SSLocator) *DBStateSnapshotIO {
	return &DBStateSnapshotIO{bs: bs, c: c, loc: loc, snapshotVer: -1}
}

func (sio *DBStateSnapshotIO) SaveSnapshot(s *inodedb.DBState) <-chan error {
	errC := make(chan error, 1)

	currVer := s.Version()
	if sio.snapshotVer > currVer {
		logger.Warningf(mylog, "SaveSnapshot: ASSERT fail: snapshot version %d newer than current ver %d", sio.snapshotVer, currVer)
	} else if sio.snapshotVer == currVer {
		logger.Debugf(mylog, "SaveSnapshot: Current ver %d is already snapshotted. No-op.", sio.snapshotVer)
		close(errC)
		return errC
	}

	buf, err := statesnapshot.EncodeBytes(func(enc *gob.Encoder) error { return s.EncodeToGob(enc) })
	if err != nil {
		errC <- err
		close(errC)
		return errC
	}

	go func() {
		defer close(errC)

		ssbp := sio.loc.GenerateBlobpath()
		if err := statesnapshot.SaveBytes(ssbp, sio.c, sio.bs, buf); err != nil {
			errC <- err
			return
		}

		if err := sio.loc.Put(ssbp, int64(s.Version())); err != nil {
			errC <- err
			return
		}

		sio.snapshotVer = s.Version()
	}()
	return errC
}

func (sio *DBStateSnapshotIO) restoreNthSnapshot(i int) (*inodedb.DBState, error) {
	ssbp, txid, err := sio.loc.Locate(i)
	if err != nil {
		return nil, err
	}

	if i == 0 {
		logger.Debugf(mylog, "Attempting to restore latest state snapshot \"%s\"", ssbp)
	} else {
		logger.Infof(mylog, "Retrying state snapshot restore with an older state snapshot \"%s\"", ssbp)
	}

	var state *inodedb.DBState
	err = statesnapshot.Restore(
		ssbp, sio.c, sio.bs,
		func(dec *gob.Decoder) error {
			var err error
			state, err = inodedb.DecodeDBStateFromGob(dec)
			return err
		})
	if err != nil {
		return nil, err
	}

	if inodedb.TxID(txid) != state.Version() {
		logger.Warningf(mylog, "SSLocator TxID mismatch. SSLocator %v != Actual SS %v", inodedb.TxID(txid), state.Version())
	}
	sio.snapshotVer = state.Version()
	return state, nil
}

const maxhist = 3

func (sio *DBStateSnapshotIO) RestoreSnapshot() (*inodedb.DBState, error) {
	for i := 0; i < maxhist; i++ {
		state, err := sio.restoreNthSnapshot(i)
		if err == nil {
			return state, nil
		} else {
			logger.Warningf(mylog, "Failed to recover state snapshot: %v", err)
		}
	}
	return nil, fmt.Errorf("Failed to restore %d snapshots. Aborted.", maxhist)
}

func (sio *DBStateSnapshotIO) FindUnneededTxIDThreshold() (inodedb.TxID, error) {
	state, err := sio.restoreNthSnapshot(maxhist - 1)
	if err != nil {
		return inodedb.AnyVersion, err
	}

	return state.Version(), nil
}

func (sio *DBStateSnapshotIO) String() string {
	return fmt.Sprintf("DBStateSnapshotIO{%v}", util.TryGetImplName(sio.loc))
}
