package otaru

import (
	"fmt"
)

type INodeID uint32

type INodeDB struct {
	nodes  map[INodeID]INode
	lastID INodeID
}

func NewINodeDB() *INodeDB {
	return &INodeDB{
		nodes:  make(map[INodeID]INode),
		lastID: 0,
	}
}

func (idb *INodeDB) Put(n INode) error {
	_, ok := idb.nodes[n.ID()]
	if ok {
		return fmt.Errorf("INodeID collision: %v", n)
	}
	idb.nodes[n.ID()] = n
	return nil
}

func (idb *INodeDB) PutMustSucceed(n INode) {
	if err := idb.Put(n); err != nil {
		panic(fmt.Sprintf("Failed to put node: %v", err))
	}
}

func (idb *INodeDB) Get(id INodeID) INode {
	node := idb.nodes[id]

	if node == nil {
		return nil
	}

	if node.ID() != id {
		panic("INodeDB is corrupt!")
	}
	return node
}

func (idb *INodeDB) GenerateNewID() INodeID {
	id := idb.lastID + 1
	idb.lastID = id
	return id
}
