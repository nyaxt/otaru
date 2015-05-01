package otaru

import (
	"bytes"
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

func serializeCommon(enc *gob.Encoder, c INodeCommon) error {
	if err := enc.Encode(c.INodeType); err != nil {
		return fmt.Errorf("Failed to encode INodeType: %v", err)
	}

	if err := enc.Encode(c.INodeID); err != nil {
		return fmt.Errorf("Failed to encode INodeID: %v", err)
	}

	if err := enc.Encode(c.OrigPath); err != nil {
		return fmt.Errorf("Failed to encode OrigPath: %v", err)
	}

	return nil
}

func deserializeCommon(dec *gob.Decoder, t INodeType, c *INodeCommon) error {
	c.INodeType = t

	if err := dec.Decode(&c.INodeID); err != nil {
		return fmt.Errorf("Failed to decode INodeID: %v", err)
	}

	if err := dec.Decode(&c.OrigPath); err != nil {
		return fmt.Errorf("Failed to decode OrigPath: %v", err)
	}

	return nil
}

func (fn *FileNode) SerializeSnapshot(w io.Writer) error {
	enc := gob.NewEncoder(w)

	if err := serializeCommon(enc, fn.INodeCommon); err != nil {
		return err
	}

	if err := enc.Encode(fn.Size); err != nil {
		return fmt.Errorf("Failed to encode Size: %v", err)
	}

	if err := enc.Encode(fn.Chunks); err != nil {
		return fmt.Errorf("Failed to encode Chunks: %v", err)
	}

	return nil
}

func deserializeFileNodeSnapshot(dec *gob.Decoder) (*FileNode, error) {
	fn := &FileNode{}
	if err := deserializeCommon(dec, FileNodeT, &fn.INodeCommon); err != nil {
		return nil, err
	}

	if err := dec.Decode(&fn.Size); err != nil {
		return nil, fmt.Errorf("Failed to decode Size: %v", err)
	}

	if err := dec.Decode(&fn.Chunks); err != nil {
		return nil, fmt.Errorf("Failed to decode Chunks: %v", err)
	}

	return fn, nil
}

func (dn *DirNode) SerializeSnapshot(w io.Writer) error {
	enc := gob.NewEncoder(w)

	if err := serializeCommon(enc, dn.INodeCommon); err != nil {
		return err
	}

	if err := enc.Encode(dn.Entries); err != nil {
		return fmt.Errorf("Failed to encode Entries: %v", err)
	}

	return nil
}

func deserializeDirNodeSnapshot(dec *gob.Decoder) (*DirNode, error) {
	dn := &DirNode{}
	if err := deserializeCommon(dec, DirNodeT, &dn.INodeCommon); err != nil {
		return nil, err
	}

	if err := dec.Decode(&dn.Entries); err != nil {
		return nil, fmt.Errorf("Failed to decode Entries: %v", err)
	}

	return dn, nil
}

func (idb *INodeDB) SerializeSnapshot(w io.Writer) error {
	if _, err := w.Write(INodeDBSnapshotSignatureMagic); err != nil {
		return fmt.Errorf("Failed to write signature magic: %v", err)
	}

	enc := gob.NewEncoder(w)

	if err := enc.Encode(INodeDBSnapshotVersion); err != nil {
		return fmt.Errorf("Failed to encode version: %v", err)
	}

	if err := enc.Encode(idb.lastID); err != nil {
		return fmt.Errorf("Failed to encode lastID: %v", err)
	}

	count := len(idb.nodes)
	if err := enc.Encode(count); err != nil {
		return fmt.Errorf("Failed to encode node count: %v", err)
	}

	for _, n := range idb.nodes {
		if err := n.SerializeSnapshot(w); err != nil {
			return err
		}
	}

	return nil
}

func DeserializeINodeSnapshot(r io.Reader) (INode, error) {
	dec := gob.NewDecoder(r)

	var t INodeType
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("Failed to decode INodeType: %v", err)
	}

	switch t {
	case FileNodeT:
		fn, err := deserializeFileNodeSnapshot(dec)
		return fn, err

	case DirNodeT:
		dn, err := deserializeDirNodeSnapshot(dec)
		return dn, err

	default:
	}
	return nil, fmt.Errorf("Invalid INodeType: %d", t)
}

func DeserializeINodeDBSnapshot(r io.Reader) (*INodeDB, error) {
	magic := make([]byte, 2)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, fmt.Errorf("Failed to read magic: %v", err)
	}
	if !bytes.Equal(magic, INodeDBSnapshotSignatureMagic) {
		return nil, fmt.Errorf("Magic mismatch: %v != %v", magic, INodeDBSnapshotSignatureMagic)
	}

	dec := gob.NewDecoder(r)

	var version int
	if err := dec.Decode(&version); err != nil {
		return nil, fmt.Errorf("Failed to decode version: %v", err)
	}
	if version != INodeDBSnapshotVersion {
		return nil, fmt.Errorf("Version mismatch! Expected: %d, Read: %d", version, INodeDBSnapshotVersion)
	}

	var lastID INodeID
	if err := dec.Decode(&lastID); err != nil {
		return nil, fmt.Errorf("Failed to decode node lastID: %v", err)
	}

	var count int
	if err := dec.Decode(&count); err != nil {
		return nil, fmt.Errorf("Failed to decode node count: %v", err)
	}

	nodes := make(map[INodeID]INode)
	for i := 0; i < count; i++ {
		n, err := DeserializeINodeSnapshot(r)
		if err != nil {
			return nil, err
		}

		id := n.ID()
		if _, ok := nodes[id]; ok {
			return nil, fmt.Errorf("NodeID collision. id: %d", id)
		}
		nodes[id] = n
	}

	idb := &INodeDB{
		nodes:  nodes,
		lastID: lastID,
	}

	return idb, nil
}
