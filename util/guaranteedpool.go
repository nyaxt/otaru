package util

import (
	"sync"
)

type GuaranteedPool struct {
	bufC chan interface{}
	pool *sync.Pool
}

func NewGuaranteedPool(new func() interface{}, guaranteed int) *GuaranteedPool {
	return &GuaranteedPool{
		bufC: make(chan interface{}, guaranteed),
		pool: &sync.Pool{New: new},
	}
}

func (p *GuaranteedPool) Get() interface{} {
	select {
	case x := <-p.bufC:
		return x
	default:
		return p.pool.Get()
	}
}

func (p *GuaranteedPool) Put(x interface{}) {
	select {
	case p.bufC <- x:
		return
	default:
		p.pool.Put(x)
	}
}
