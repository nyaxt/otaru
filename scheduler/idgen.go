package scheduler

import (
	"sync/atomic"
)

type ID uint32

const AbortAll ID = 0

type idGen struct {
	lastID ID
}

func (g *idGen) genID() ID {
	return ID(atomic.AddUint32((*uint32)(&g.lastID), 1))
}
