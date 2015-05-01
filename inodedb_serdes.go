package otaru

import (
	"fmt"
	"io"

	"encoding/gob"
)

const (
	INodeDBSnapshotSignatureMagic1 = 0x05 // o t
	INodeDBSnapshotSignatureMagic2 = 0xdb

	INodeDBSnapshotVersion = 0
)

var (
	INodeDBSnapshotSignatureMagic = []byte{INodeDBSnapshotSignatureMagic1, INodeDBSnapshotSignatureMagic2}
)

func (c *INodeCommon) SerializeCommon(enc gob.Encoder) error {
	if err := enc.Encode(c.INodeType); err != nil {
		return fmt.Errorf("failed to encode INodeType: %v", err)
	}

	if err := enc.Encode(c.INodeID); err != nil {
		return fmt.Errorf("failed to encode INodeID: %v", err)
	}

	if err := enc.Encode(c.OrigPath); err != nil {
		return fmt.Errorf("failed to encode OrigPath: %v", err)
	}

	return nil
}

func (fn *FileNode) SerializeSnapshot(w io.Writer) error {
	enc := gob.NewEncoder(w)

	if err := enc.Encode(fn.INodeCommon); err != nil {
		return err
	}

	if err := enc.Encode(fn.Size); err != nil {
		return fmt.Errorf("failed to encode Size: %v", err)
	}

	if err := enc.Encode(fn.Chunks); err != nil {
		return fmt.Errorf("failed to encode Chunks: %v", err)
	}

	return nil
}

func (dn *DirNode) SerializeSnapshot(w io.Writer) error {
	enc := gob.NewEncoder(w)

	if err := enc.Encode(dn.INodeCommon); err != nil {
		return err
	}

	if err := enc.Encode(dn.Entries); err != nil {
		return fmt.Errorf("failed to encode Entries: %v", err)
	}

	return nil
}

func (idb *INodeDB) SerializeSnapshot(w io.Writer) error {
	if _, err := w.Write(INodeDBSnapshotSignatureMagic); err != nil {
		return fmt.Errorf("failed to write signature magic: %v", err)
	}

	enc := gob.NewEncoder(w)

	if err := enc.Encode(INodeDBSnapshotVersion); err != nil {
		return fmt.Errorf("failed to encode version: %v", err)
	}

	if err := enc.Encode(idb.lastID); err != nil {
		return fmt.Errorf("failed to encode lastID: %v", err)
	}

	count := uint64(len(idb.nodes))
	if err := enc.Encode(count); err != nil {
		return fmt.Errorf("failed to encode node count: %v", err)
	}

	for _, n := range idb.nodes {
		if err := n.SerializeSnapshot(w); err != nil {
			return err
		}
	}

	return nil
}
