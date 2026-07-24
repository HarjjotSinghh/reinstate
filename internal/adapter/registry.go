package adapter

import (
	"fmt"
	"sync"
)

// Registry maps adapter names to implementations.
type Registry struct {
	mu     sync.RWMutex
	byName map[string]Adapter
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byName: map[string]Adapter{}}
}

// Register adds an adapter.
func (r *Registry) Register(a Adapter) error {
	if a == nil {
		return fmt.Errorf("nil adapter")
	}
	name := a.Name()
	if name == "" {
		return fmt.Errorf("empty adapter name")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byName[name]; ok {
		return fmt.Errorf("adapter %q already registered", name)
	}
	r.byName[name] = a
	return nil
}

// Get returns an adapter by name.
func (r *Registry) Get(name string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.byName[name]
	return a, ok
}

// Names returns registered adapter names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byName))
	for n := range r.byName {
		out = append(out, n)
	}
	return out
}

// All returns all adapters.
func (r *Registry) All() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Adapter, 0, len(r.byName))
	for _, a := range r.byName {
		out = append(out, a)
	}
	return out
}
