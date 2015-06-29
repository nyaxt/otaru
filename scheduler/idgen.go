package scheduler

import (
	"sync/atomic"
)

type ID uint32

type idGen struct {
	nextJobID ID
}

func (g *idGen) genID() ID {
	return ID(atomic.AddUint32((*uint32)(&g.nextJobID), 1))
}
