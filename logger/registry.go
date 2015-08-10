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
		l = &CategoryLogger{
			BE:       &r.mux,
			Category: c,
			Level:    Debug,
		}
		r.catmap[c] = l
	}
	return l
}

type CategoryEntry struct {
	Category string `json:"category"`
	Level    `json:"level"`
}

func (cl *CategoryLogger) View() CategoryEntry {
	return CategoryEntry{Category: cl.Category, Level: cl.Level}
}

func (r *registry) Categories() []CategoryEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	ret := make([]CategoryEntry, 0, len(r.catmap))
	for _, cl := range r.catmap {
		ret = append(ret, cl.View())
	}
	return ret
}
