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

func (io *SimpleDBStateSnapshotIO) SaveSnapshot(s *DBState) <-chan error {
	errC := make(chan error, 1)

	io.Buf.Reset()

	enc := gob.NewEncoder(&io.Buf)
	if err := s.EncodeToGob(enc); err != nil {
		errC <- fmt.Errorf("Failed to encode DBState: %v", err)
	}

	close(errC)
	return errC
}

func (io *SimpleDBStateSnapshotIO) RestoreSnapshot() (*DBState, error) {
	dec := gob.NewDecoder(&io.Buf)
	return DecodeDBStateFromGob(dec)
}

func (*SimpleDBStateSnapshotIO) ImplName() string { return "SimpleDBStateSnapshotIO" }
