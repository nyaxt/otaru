package logger

import (
	"sync"
)

type registry struct {
	mu sync.Mutex

	mux    Mux
	catmap map[string]*CategoryLogger
}

var registryInstance *registry
var muRegistryInstance sync.Mutex

func Registry() *registry {
	muRegistryInstance.Lock()
	defer muRegistryInstance.Unlock()

	if registryInstance == nil {
		registryInstance = &registry{
			catmap: make(map[string]*CategoryLogger),
		}
	}

	return registryInstance
}

func (r *registry) AddOutput(l Logger) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.mux.Ls = append(r.mux.Ls, l)
}

func (r *registry) Category(c string) *CategoryLogger {
	r.mu.Lock()
	defer r.mu.Unlock()

	l, ok := r.catmap[c]
	if !ok {
		l = &CategoryLogger{&r.mux, c}
		r.catmap[c] = l
	}
	return l
}
