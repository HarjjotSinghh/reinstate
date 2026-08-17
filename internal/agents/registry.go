package agents

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	catalogMu sync.RWMutex
	catalog   = map[string]Descriptor{}
)

// MustRegister adds a descriptor to the catalog. It panics on an empty key, a
// duplicate key, a T0Reason present at a tier above T0 or absent at T0, or a
// capability constructor above the declared tier.
func MustRegister(d Descriptor) {
	d.Key = strings.TrimSpace(d.Key)
	if d.Key == "" {
		panic("agents: empty catalog key")
	}
	if d.Tier == TierKnown {
		if d.T0Reason == "" {
			panic(fmt.Sprintf("agents: %s: T0Reason required at T0", d.Key))
		}
	} else if d.T0Reason != "" {
		panic(fmt.Sprintf("agents: %s: T0Reason %q is only valid at T0", d.Key, d.T0Reason))
	}
	// A root without a marker resolves on the bare directory, so unrelated
	// tooling that creates ~/.<agent> makes the probe report an agent that is
	// not installed.
	if d.Storage.Roots != nil && strings.TrimSpace(d.Storage.Marker) == "" {
		panic(fmt.Sprintf("agents: %s: Storage.Marker required when Storage.Roots is declared", d.Key))
	}
	if above := d.constructorsAboveTier(); len(above) > 0 {
		panic(fmt.Sprintf(
			"agents: %s: capability constructor %s above declared tier %s",
			d.Key, strings.Join(above, ", "), d.Tier,
		))
	}

	catalogMu.Lock()
	defer catalogMu.Unlock()
	if _, exists := catalog[d.Key]; exists {
		panic(fmt.Sprintf("agents: duplicate catalog key %q", d.Key))
	}
	catalog[d.Key] = d
}

// All returns every registered descriptor, sorted by key.
func All() []Descriptor {
	return snapshot(func(Descriptor) bool { return true })
}

// Get returns the descriptor for key.
func Get(key string) (Descriptor, bool) {
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	d, ok := catalog[key]
	return d, ok
}

// Keys returns every registered key, sorted.
func Keys() []string {
	all := All()
	keys := make([]string, len(all))
	for i, d := range all {
		keys[i] = d.Key
	}
	return keys
}

// AtLeast returns descriptors whose declared tier is at least t, sorted by key.
func AtLeast(t Tier) []Descriptor {
	return snapshot(func(d Descriptor) bool { return d.Tier >= t })
}

// Capable returns descriptors that expose capability c, sorted by key.
func Capable(c Capability) []Descriptor {
	return snapshot(func(d Descriptor) bool { return d.hasCapability(c) })
}

func snapshot(keep func(Descriptor) bool) []Descriptor {
	catalogMu.RLock()
	out := make([]Descriptor, 0, len(catalog))
	for _, d := range catalog {
		if keep(d) {
			out = append(out, d)
		}
	}
	catalogMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func resetForTest() {
	catalogMu.Lock()
	catalog = map[string]Descriptor{}
	catalogMu.Unlock()
}
