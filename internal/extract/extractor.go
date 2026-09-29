// Package extract derives import specifiers from source files without executing
// them. Extraction is pluggable per language and deterministic.
package extract

import (
	"sort"
	"strings"
)

// Extractor derives raw import specifiers from a file's contents. It MUST NOT
// execute the file.
type Extractor interface {
	Name() string
	Imports(content []byte) []string
}

// Registry maps extractor names to implementations.
type Registry struct {
	byName map[string]Extractor
}

// DefaultRegistry returns a registry with all built-in extractors.
func DefaultRegistry() *Registry {
	r := &Registry{byName: map[string]Extractor{}}
	r.register(Generic{})
	r.register(JS{})
	r.register(Go{})
	return r
}

func (r *Registry) register(e Extractor) { r.byName[e.Name()] = e }

// Get returns the named extractor and whether it exists.
func (r *Registry) Get(name string) (Extractor, bool) {
	e, ok := r.byName[name]
	return e, ok
}

// Names returns the registered extractor names, sorted.
func (r *Registry) Names() []string {
	var out []string
	for n := range r.byName {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// dedupeSorted returns unique, sorted, non-empty entries for determinism.
func dedupeSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
