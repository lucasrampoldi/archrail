package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/archrail/archrail/internal/model"
)

// Topology is the resolved set of components and layers used for evaluation.
type Topology struct {
	Components []model.Component
	Layers     []model.Layer

	layerByComponent map[string]string
	componentsByName map[string]bool
	layersByName     map[string]bool
}

// BuildTopology merges profile topology with project-level overrides (matched by
// name) and indexes it. Overrides replace a same-named component/layer; new
// names are appended.
func BuildTopology(profile *model.Profile, cfg *model.Config) (*Topology, error) {
	comps := mergeComponents(profile.Components, cfg.Components)
	layers := mergeLayers(profile.Layers, cfg.Layers)

	t := &Topology{
		Components:       comps,
		Layers:           layers,
		layerByComponent: map[string]string{},
		componentsByName: map[string]bool{},
		layersByName:     map[string]bool{},
	}
	for _, c := range comps {
		t.componentsByName[c.Name] = true
	}
	for _, l := range layers {
		t.layersByName[l.Name] = true
		for _, cn := range l.Components {
			if !t.componentsByName[cn] {
				return nil, fmt.Errorf("layer %q references unknown component %q", l.Name, cn)
			}
			t.layerByComponent[cn] = l.Name
		}
	}
	return t, nil
}

func mergeComponents(base, over []model.Component) []model.Component {
	idx := map[string]int{}
	out := append([]model.Component{}, base...)
	for i, c := range out {
		idx[c.Name] = i
	}
	for _, c := range over {
		if i, ok := idx[c.Name]; ok {
			out[i] = c
		} else {
			idx[c.Name] = len(out)
			out = append(out, c)
		}
	}
	return out
}

func mergeLayers(base, over []model.Layer) []model.Layer {
	idx := map[string]int{}
	out := append([]model.Layer{}, base...)
	for i, l := range out {
		idx[l.Name] = i
	}
	for _, l := range over {
		if i, ok := idx[l.Name]; ok {
			out[i] = l
		} else {
			idx[l.Name] = len(out)
			out = append(out, l)
		}
	}
	return out
}

// HasComponent reports whether a component with the given name exists.
func (t *Topology) HasComponent(name string) bool { return t.componentsByName[name] }

// HasLayer reports whether a layer with the given name exists.
func (t *Topology) HasLayer(name string) bool { return t.layersByName[name] }

// LayerOf returns the layer a component belongs to (or "").
func (t *Topology) LayerOf(component string) string { return t.layerByComponent[component] }

// AssignComponent returns the component owning a repo-relative path, applying
// most-specific-glob-wins precedence (ties broken by declaration order). Returns
// "" when no component matches.
func (t *Topology) AssignComponent(relPath string) string {
	best := ""
	bestScore := -1
	for _, c := range t.Components {
		for _, pat := range c.Paths {
			ok, err := doublestar.Match(pat, relPath)
			if err != nil || !ok {
				continue
			}
			if s := literalScore(pat); s > bestScore {
				bestScore = s
				best = c.Name
			}
		}
	}
	return best
}

// literalScore measures glob specificity as the number of literal (non-wildcard)
// characters. More literal characters => more specific.
func literalScore(pattern string) int {
	n := 0
	for _, r := range pattern {
		switch r {
		case '*', '?', '[', ']', '{', '}':
			// wildcard/meta: not literal
		default:
			n++
		}
	}
	return n
}

// componentPatterns returns all path globs for a component (empty if unknown).
func (t *Topology) componentPatterns(name string) []string {
	for _, c := range t.Components {
		if c.Name == name {
			return c.Paths
		}
	}
	return nil
}

// ComponentsInLayer returns the component names in a layer, sorted.
func (t *Topology) ComponentsInLayer(layer string) []string {
	var out []string
	for _, l := range t.Layers {
		if l.Name == layer {
			out = append(out, l.Components...)
		}
	}
	sort.Strings(out)
	return out
}

// ResolveImportComponent maps a raw import specifier to a component name using
// deterministic heuristics: it matches the specifier (as a path) against
// component globs and against a "/<name>/" segment. Returns "" when the
// specifier resolves to no known component (e.g. an external package).
func (t *Topology) ResolveImportComponent(spec string) string {
	norm := normalizeSpec(spec)
	best := ""
	bestScore := -1
	for _, c := range t.Components {
		matched := false
		for _, pat := range c.Paths {
			if ok, _ := doublestar.Match(pat, norm); ok {
				matched = true
				break
			}
			// Also match when the specifier is a suffix of a glob's literal dir.
			if strings.Contains("/"+norm+"/", "/"+c.Name+"/") {
				matched = true
				break
			}
		}
		if !matched {
			// Fall back to a component-name segment match.
			if strings.Contains("/"+norm+"/", "/"+c.Name+"/") {
				matched = true
			}
		}
		if matched {
			if s := literalScore(strings.Join(c.Paths, "")); s > bestScore {
				bestScore = s
				best = c.Name
			}
		}
	}
	return best
}

func normalizeSpec(spec string) string {
	s := strings.TrimSpace(spec)
	s = strings.TrimPrefix(s, "./")
	for strings.HasPrefix(s, "../") {
		s = strings.TrimPrefix(s, "../")
	}
	return strings.Trim(s, "/")
}
