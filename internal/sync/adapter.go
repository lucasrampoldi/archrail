// Package sync renders the adopted architecture profile into agent-consumable
// context through a pluggable adapter. v0 ships the canonical ARCHRAIL.md
// adapter.
package sync

import (
	"sort"
	"strings"

	"github.com/archrail/archrail/internal/model"
)

// RenderInput is the data an adapter renders.
type RenderInput struct {
	ProfileName        string
	ProfileVersion     string
	ProfileDescription string
	Rules              []model.Rule
}

// Adapter renders profile context to a named on-disk target.
type Adapter interface {
	Name() string
	// FileName is the output file, relative to the repository root.
	FileName() string
	// Render produces deterministic, byte-stable output for identical input.
	Render(in RenderInput) string
}

// Registry maps adapter names to implementations.
type Registry struct {
	byName map[string]Adapter
}

// DefaultAdapter is the adapter used when none is specified.
const DefaultAdapter = "archrail-md"

// NewRegistry returns a registry with the built-in adapters.
func NewRegistry() *Registry {
	r := &Registry{byName: map[string]Adapter{}}
	r.register(ArchrailMD{})
	return r
}

func (r *Registry) register(a Adapter) { r.byName[a.Name()] = a }

// Get returns the named adapter.
func (r *Registry) Get(name string) (Adapter, bool) {
	a, ok := r.byName[name]
	return a, ok
}

// Names returns registered adapter names, sorted.
func (r *Registry) Names() []string {
	var out []string
	for n := range r.byName {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func sortedRules(rules []model.Rule) []model.Rule {
	out := append([]model.Rule{}, rules...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func decodeString(r model.Rule, key string) string {
	var s string
	r.DecodeParams(key, &s)
	return strings.TrimSpace(s)
}
