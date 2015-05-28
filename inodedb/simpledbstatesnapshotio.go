package inodedb

import (
	"bytes"
	"fmt"

	"encoding/gob"
)

type SimpleDBStateSnapshotIO struct {
	Buf bytes.Buffer
}

var _ = DBStateSnapshotIO(&SimpleDBStateSnapshotIO{})

func NewSimpleDBStateSnapshotIO() *SimpleDBStateSnapshotIO {
	return &SimpleDBStateSnapshotIO{}
}

func (io *SimpleDBStateSnapshotIO) SaveSnapshot(s *DBState) error {
	io.Buf.Reset()

	enc := gob.NewEncoder(&io.Buf)
	if err := s.EncodeToGob(enc); err != nil {
		return fmt.Errorf("Failed to encode DBState: %v", err)
	}

	return nil
}

func (io *SimpleDBStateSnapshotIO) RestoreSnapshot() (*DBState, error) {
	dec := gob.NewDecoder(&io.Buf)
	return DecodeDBStateFromGob(dec)
}
