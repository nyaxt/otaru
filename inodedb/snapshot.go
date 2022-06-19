package inodedb

import (
	"encoding/gob"
	"fmt"

	"go.uber.org/zap"
)

func (s *DBState) EncodeToGob(enc *gob.Encoder) error {
	numNodes := uint64(len(s.nodes))
	if err := enc.Encode(numNodes); err != nil {
		return fmt.Errorf("Failed to encode numNodes: %v", err)
	}
	for id, node := range s.nodes {
		if id != node.GetID() {
			zap.S().Panicf("nodes map key (%d) != node.GetID() result (%d)", id, node.GetID())
		}

		if err := node.EncodeToGob(enc); err != nil {
			return fmt.Errorf("Failed to encode node: %v", err)
		}
	}

	if err := enc.Encode(s.lastID); err != nil {
		return fmt.Errorf("Failed to encode lastID: %v", err)
	}
	if err := enc.Encode(s.version); err != nil {
		return fmt.Errorf("Failed to encode version: %v", err)
	}
	return nil
}

func DecodeDBStateFromGob(dec *gob.Decoder) (*DBState, error) {
	s := NewDBState()

	var numNodes uint64
	if err := dec.Decode(&numNodes); err != nil {
		return nil, fmt.Errorf("failed to decode numNodes: %v", err)
	}
	for i := uint64(0); i < numNodes; i++ {
		n, err := DecodeNodeFromGob(dec)
		if err != nil {
			return nil, fmt.Errorf("failed to decode node: %v", err)
		}
		s.nodes[n.GetID()] = n
	}

	if err := dec.Decode(&s.lastID); err != nil {
		return nil, fmt.Errorf("Failed to decode lastID: %v", err)
	}
	if err := dec.Decode(&s.version); err != nil {
		return nil, fmt.Errorf("Failed to decode version: %v", err)
	}

	return s, nil
}

type GobEncodable interface {
	EncodeToGob(enc *gob.Encoder) error
}

func serializeCommon(enc *gob.Encoder, t Type, c INodeCommon) error {
	if err := enc.Encode(t); err != nil {
		return fmt.Errorf("Failed to encode Type: %v", err)
	}

	if err := enc.Encode(c.ID); err != nil {
		return fmt.Errorf("Failed to encode ID: %v", err)
	}

	if err := enc.Encode(c.OrigPath); err != nil {
		return fmt.Errorf("Failed to encode OrigPath: %v", err)
	}

	if err := enc.Encode(&c.Uid); err != nil {
		return fmt.Errorf("Failed to encode Uid: %v", err)
	}

	if err := enc.Encode(&c.Gid); err != nil {
		return fmt.Errorf("Failed to encode Gid: %v", err)
	}

	if err := enc.Encode(&c.PermMode); err != nil {
		return fmt.Errorf("Failed to encode PermMode: %v", err)
	}

	if err := enc.Encode(&c.ModifiedT); err != nil {
		return fmt.Errorf("Failed to encode ModifiedT: %v", err)
	}

	return nil
}

func (dn *DirNode) EncodeToGob(enc *gob.Encoder) error {
	if err := serializeCommon(enc, dn.GetType(), dn.INodeCommon); err != nil {
		return err
	}

	if err := enc.Encode(dn.Entries); err != nil {
		return fmt.Errorf("Failed to encode Entries: %v", err)
	}

	return nil
}

func (fn *FileNode) EncodeToGob(enc *gob.Encoder) error {
	if err := serializeCommon(enc, fn.GetType(), fn.INodeCommon); err != nil {
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

func deserializeCommon(dec *gob.Decoder, t Type, c *INodeCommon) error {
	if err := dec.Decode(&c.ID); err != nil {
		return fmt.Errorf("Failed to decode ID: %v", err)
	}

	if err := dec.Decode(&c.OrigPath); err != nil {
		return fmt.Errorf("Failed to decode OrigPath: %v", err)
	}

	if err := dec.Decode(&c.Uid); err != nil {
		return fmt.Errorf("Failed to decode Uid: %v", err)
	}

	if err := dec.Decode(&c.Gid); err != nil {
		return fmt.Errorf("Failed to decode Gid: %v", err)
	}

	if err := dec.Decode(&c.PermMode); err != nil {
		return fmt.Errorf("Failed to decode PermMode: %v", err)
	}

	if err := dec.Decode(&c.ModifiedT); err != nil {
		return fmt.Errorf("Failed to decode ModifiedT: %v", err)
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

func DecodeNodeFromGob(dec *gob.Decoder) (INode, error) {
	var t Type
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("Failed to decode Type: %v", err)
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
	return nil, fmt.Errorf("Invalid Type: %d", t)
}
