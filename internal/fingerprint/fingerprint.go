// Package fingerprint builds a deterministic, structural summary of a repository
// (components, import graph, detected stack) that is sent to the LLM as verified
// context instead of raw code. It reuses the engine's topology and facts, so it
// costs little and stays consistent with deterministic evaluation.
package fingerprint

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/archrail/archrail/internal/engine"
)

// Component summarizes one component in the fingerprint.
type Component struct {
	Name      string   `yaml:"name"`
	Paths     []string `yaml:"paths"`
	FileCount int      `yaml:"fileCount"`
}

// Layer summarizes a layer.
type Layer struct {
	Name       string   `yaml:"name"`
	Components []string `yaml:"components"`
}

// Edge is a summarized component-to-component dependency.
type Edge struct {
	From  string `yaml:"from"`
	To    string `yaml:"to"`
	Count int    `yaml:"count"`
}

// ExternalDep is a first-party component's dependency on an external package.
type ExternalDep struct {
	Component string `yaml:"component"`
	Package   string `yaml:"package"`
}

// Stack is the detected technology stack.
type Stack struct {
	Languages       []string `yaml:"languages"`
	Frameworks      []string `yaml:"frameworks"`
	DockerBaseImage string   `yaml:"dockerBaseImage,omitempty"`
}

// Fingerprint is the deterministic architecture summary.
type Fingerprint struct {
	Components   []Component   `yaml:"components"`
	Layers       []Layer       `yaml:"layers"`
	Edges        []Edge        `yaml:"edges"`
	ExternalDeps []ExternalDep `yaml:"externalDeps"`
	Stack        Stack         `yaml:"stack"`
	Unresolved   []string      `yaml:"unresolved,omitempty"`
}

var knownFrameworks = []string{
	"react", "next", "express", "koa", "fastify", "nestjs", "vue", "angular",
	"svelte", "django", "flask", "fastapi", "spring", "gin", "echo", "fiber",
}

// Build computes the fingerprint from resolved topology and facts. It is
// deterministic and excludes dependency/vendored paths (already filtered out of
// facts).
func Build(topo *engine.Topology, facts *engine.Facts) *Fingerprint {
	fp := &Fingerprint{}

	// Components with file counts.
	counts := map[string]int{}
	langs := map[string]bool{}
	for _, ff := range facts.Files {
		if ff.Component != "" {
			counts[ff.Component]++
		}
		if l := langOf(ff.Path); l != "" {
			langs[l] = true
		}
	}
	for _, c := range topo.Components {
		fp.Components = append(fp.Components, Component{Name: c.Name, Paths: c.Paths, FileCount: counts[c.Name]})
	}
	sort.Slice(fp.Components, func(i, j int) bool { return fp.Components[i].Name < fp.Components[j].Name })

	for _, l := range topo.Layers {
		comps := append([]string{}, l.Components...)
		sort.Strings(comps)
		fp.Layers = append(fp.Layers, Layer{Name: l.Name, Components: comps})
	}
	sort.Slice(fp.Layers, func(i, j int) bool { return fp.Layers[i].Name < fp.Layers[j].Name })

	// Edges + external deps + unresolved.
	edgeCount := map[string]int{}
	extSet := map[string]bool{}
	unresolvedSet := map[string]bool{}
	fwSet := map[string]bool{}
	for _, ff := range facts.Files {
		if ff.Component == "" || !ff.HasExtractor {
			continue
		}
		for _, imp := range ff.Imports {
			tgt := topo.ResolveImportComponent(imp)
			if tgt != "" {
				if tgt != ff.Component {
					edgeCount[ff.Component+"\x00"+tgt]++
				}
				continue
			}
			if strings.HasPrefix(imp, ".") {
				unresolvedSet[imp] = true
				continue
			}
			pkg := externalPackage(imp)
			extSet[ff.Component+"\x00"+pkg] = true
			if fw := frameworkOf(pkg); fw != "" {
				fwSet[fw] = true
			}
		}
	}
	for k, n := range edgeCount {
		parts := strings.SplitN(k, "\x00", 2)
		fp.Edges = append(fp.Edges, Edge{From: parts[0], To: parts[1], Count: n})
	}
	sort.Slice(fp.Edges, func(i, j int) bool {
		if fp.Edges[i].From != fp.Edges[j].From {
			return fp.Edges[i].From < fp.Edges[j].From
		}
		return fp.Edges[i].To < fp.Edges[j].To
	})
	for k := range extSet {
		parts := strings.SplitN(k, "\x00", 2)
		fp.ExternalDeps = append(fp.ExternalDeps, ExternalDep{Component: parts[0], Package: parts[1]})
	}
	sort.Slice(fp.ExternalDeps, func(i, j int) bool {
		if fp.ExternalDeps[i].Component != fp.ExternalDeps[j].Component {
			return fp.ExternalDeps[i].Component < fp.ExternalDeps[j].Component
		}
		return fp.ExternalDeps[i].Package < fp.ExternalDeps[j].Package
	})
	fp.Unresolved = sortedKeys(unresolvedSet)

	fp.Stack.Languages = sortedKeys(langs)
	fp.Stack.Frameworks = sortedKeys(fwSet)
	fp.Stack.DockerBaseImage = detectDockerBaseImage(facts)

	return fp
}

// Render serializes the fingerprint to canonical, byte-stable YAML.
func (fp *Fingerprint) Render() string {
	out, err := yaml.Marshal(fp)
	if err != nil {
		return ""
	}
	return string(out)
}

func externalPackage(spec string) string {
	if strings.HasPrefix(spec, "@") { // scoped npm package
		parts := strings.SplitN(spec, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return spec
	}
	return strings.SplitN(spec, "/", 2)[0]
}

func frameworkOf(pkg string) string {
	p := strings.ToLower(pkg)
	for _, fw := range knownFrameworks {
		if p == fw || strings.HasPrefix(p, fw) || strings.Contains(p, "/"+fw) {
			return fw
		}
	}
	return ""
}

func langOf(path string) string {
	switch {
	case strings.HasSuffix(path, ".go"):
		return "go"
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"):
		return "typescript"
	case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"):
		return "javascript"
	case strings.HasSuffix(path, ".py"):
		return "python"
	case strings.HasSuffix(path, ".java"):
		return "java"
	case strings.HasSuffix(path, ".rb"):
		return "ruby"
	}
	return ""
}

func detectDockerBaseImage(facts *engine.Facts) string {
	for _, ff := range facts.Files {
		if base := ff.Path; strings.HasSuffix(base, "Dockerfile") || strings.Contains(base, "Dockerfile") {
			content, err := facts.ReadFile(ff.Path)
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(content), "\n") {
				l := strings.TrimSpace(line)
				if strings.HasPrefix(strings.ToUpper(l), "FROM ") {
					return strings.TrimSpace(l[5:])
				}
			}
		}
	}
	return ""
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
