package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/archrail/archrail/internal/model"
)

// ---- forbidden-import ----------------------------------------------------

type forbiddenImport struct{}

func (forbiddenImport) Name() string { return "forbidden-import" }

func (forbiddenImport) Params() []ParamDoc {
	return []ParamDoc{
		{"from", true, "source component or layer the constraint applies to"},
		{"to", false, "target component or layer that must not be imported"},
		{"pattern", false, "raw import-specifier substring that must not appear"},
	}
}

func (forbiddenImport) Validate(rule model.Rule) error {
	var from, to, pattern string
	if _, err := rule.DecodeParams("from", &from); err != nil {
		return err
	}
	if _, err := rule.DecodeParams("to", &to); err != nil {
		return err
	}
	if _, err := rule.DecodeParams("pattern", &pattern); err != nil {
		return err
	}
	if from == "" {
		return fmt.Errorf("missing required parameter 'from'")
	}
	if to == "" && pattern == "" {
		return fmt.Errorf("requires either 'to' or 'pattern'")
	}
	return nil
}

func (forbiddenImport) Evaluate(in EvalInput) ([]model.Violation, []model.Note, error) {
	var from, to, pattern string
	in.Rule.DecodeParams("from", &from)
	in.Rule.DecodeParams("to", &to)
	in.Rule.DecodeParams("pattern", &pattern)

	sev := in.EffectiveSeverity()
	var vs []model.Violation
	var notes []model.Note
	for _, ff := range in.Facts.Files {
		if !scopeMatches(in.Topo, ff.Component, from) || !in.InScope(ff.Path) {
			continue
		}
		if !ff.HasExtractor {
			notes = append(notes, model.Note{RuleID: in.Rule.ID, File: ff.Path, Reason: "skipped: no import extractor"})
			continue
		}
		for _, imp := range ff.Imports {
			bad := false
			if pattern != "" && strings.Contains(imp, pattern) {
				bad = true
			}
			if to != "" && scopeMatches(in.Topo, in.Topo.ResolveImportComponent(imp), to) {
				bad = true
			}
			if bad {
				vs = append(vs, model.Violation{
					RuleID:     in.Rule.ID,
					Severity:   sev,
					File:       ff.Path,
					Message:    fmt.Sprintf("%s imports forbidden target %q", from, imp),
					SourceFile: in.Rule.SourceFile,
				})
			}
		}
	}
	return vs, notes, nil
}

// ---- layer-boundary ------------------------------------------------------

type layerBoundary struct{}

func (layerBoundary) Name() string { return "layer-boundary" }

func (layerBoundary) Params() []ParamDoc {
	return []ParamDoc{
		{"allow", true, "list of {from: <layer>, to: [<layers>]} allowed dependency edges"},
	}
}

type layerAllow struct {
	From string   `yaml:"from"`
	To   []string `yaml:"to"`
}

func (layerBoundary) Validate(rule model.Rule) error {
	var allow []layerAllow
	found, err := rule.DecodeParams("allow", &allow)
	if err != nil {
		return err
	}
	if !found || len(allow) == 0 {
		return fmt.Errorf("requires a non-empty 'allow' list")
	}
	return nil
}

func (layerBoundary) Evaluate(in EvalInput) ([]model.Violation, []model.Note, error) {
	var allow []layerAllow
	in.Rule.DecodeParams("allow", &allow)
	allowed := map[string]map[string]bool{}
	for _, a := range allow {
		if allowed[a.From] == nil {
			allowed[a.From] = map[string]bool{}
		}
		for _, t := range a.To {
			allowed[a.From][t] = true
		}
	}

	sev := in.EffectiveSeverity()
	var vs []model.Violation
	var notes []model.Note
	for _, ff := range in.Facts.Files {
		if ff.Component == "" || !in.InScope(ff.Path) {
			continue
		}
		srcLayer := in.Topo.LayerOf(ff.Component)
		if srcLayer == "" {
			continue
		}
		if !ff.HasExtractor {
			notes = append(notes, model.Note{RuleID: in.Rule.ID, File: ff.Path, Reason: "skipped: no import extractor"})
			continue
		}
		for _, imp := range ff.Imports {
			tgtComp := in.Topo.ResolveImportComponent(imp)
			if tgtComp == "" {
				continue
			}
			tgtLayer := in.Topo.LayerOf(tgtComp)
			if tgtLayer == "" || tgtLayer == srcLayer {
				continue
			}
			if !allowed[srcLayer][tgtLayer] {
				vs = append(vs, model.Violation{
					RuleID:     in.Rule.ID,
					Severity:   sev,
					File:       ff.Path,
					Message:    fmt.Sprintf("layer %q must not depend on layer %q (import %q)", srcLayer, tgtLayer, imp),
					SourceFile: in.Rule.SourceFile,
				})
			}
		}
	}
	return vs, notes, nil
}

// ---- forbidden-dependency / required-dependency --------------------------

type forbiddenDependency struct{}

func (forbiddenDependency) Name() string { return "forbidden-dependency" }
func (forbiddenDependency) Params() []ParamDoc {
	return []ParamDoc{
		{"manifest", true, "glob for the dependency manifest (e.g. package.json)"},
		{"dependency", true, "dependency name that must NOT be declared"},
	}
}
func (forbiddenDependency) Validate(rule model.Rule) error { return validateDepParams(rule) }
func (forbiddenDependency) Evaluate(in EvalInput) ([]model.Violation, []model.Note, error) {
	return evalDependency(in, false)
}

type requiredDependency struct{}

func (requiredDependency) Name() string { return "required-dependency" }
func (requiredDependency) Params() []ParamDoc {
	return []ParamDoc{
		{"manifest", true, "glob for the dependency manifest (e.g. package.json)"},
		{"dependency", true, "dependency name that MUST be declared"},
	}
}
func (requiredDependency) Validate(rule model.Rule) error { return validateDepParams(rule) }
func (requiredDependency) Evaluate(in EvalInput) ([]model.Violation, []model.Note, error) {
	return evalDependency(in, true)
}

func validateDepParams(rule model.Rule) error {
	var manifest, dep string
	rule.DecodeParams("manifest", &manifest)
	rule.DecodeParams("dependency", &dep)
	if manifest == "" || dep == "" {
		return fmt.Errorf("requires 'manifest' and 'dependency'")
	}
	return nil
}

func evalDependency(in EvalInput, required bool) ([]model.Violation, []model.Note, error) {
	var manifest, dep string
	in.Rule.DecodeParams("manifest", &manifest)
	in.Rule.DecodeParams("dependency", &dep)

	sev := in.EffectiveSeverity()
	var vs []model.Violation
	needle := fmt.Sprintf("%q", dep)
	for _, ff := range in.Facts.Files {
		if !matchGlob(manifest, ff.Path) && !(!strings.Contains(manifest, "/") && basename(ff.Path) == manifest) {
			continue
		}
		if !in.InScope(ff.Path) {
			continue
		}
		content, err := in.Facts.ReadFile(ff.Path)
		if err != nil {
			return nil, nil, err
		}
		present := strings.Contains(string(content), needle)
		if required && !present {
			vs = append(vs, model.Violation{
				RuleID: in.Rule.ID, Severity: sev, File: ff.Path,
				Message:    fmt.Sprintf("required dependency %q is missing", dep),
				SourceFile: in.Rule.SourceFile,
			})
		}
		if !required && present {
			vs = append(vs, model.Violation{
				RuleID: in.Rule.ID, Severity: sev, File: ff.Path,
				Message:    fmt.Sprintf("forbidden dependency %q is declared", dep),
				SourceFile: in.Rule.SourceFile,
			})
		}
	}
	return vs, nil, nil
}

// ---- required-files ------------------------------------------------------

type requiredFiles struct{}

func (requiredFiles) Name() string { return "required-files" }
func (requiredFiles) Params() []ParamDoc {
	return []ParamDoc{
		{"component", true, "component whose skeleton is checked"},
		{"files", true, "list of file globs that must be present in the component"},
	}
}
func (requiredFiles) Validate(rule model.Rule) error {
	var comp string
	var files []string
	rule.DecodeParams("component", &comp)
	rule.DecodeParams("files", &files)
	if comp == "" || len(files) == 0 {
		return fmt.Errorf("requires 'component' and a non-empty 'files' list")
	}
	return nil
}
func (requiredFiles) Evaluate(in EvalInput) ([]model.Violation, []model.Note, error) {
	var comp string
	var files []string
	in.Rule.DecodeParams("component", &comp)
	in.Rule.DecodeParams("files", &files)

	compFiles := in.Facts.FilesIn(comp)
	// Only evaluate when the component has at least one in-scope file (diff mode).
	inScope := false
	for _, ff := range compFiles {
		if in.InScope(ff.Path) {
			inScope = true
			break
		}
	}
	if !inScope {
		return nil, nil, nil
	}

	sev := in.EffectiveSeverity()
	var vs []model.Violation
	sort.Strings(files)
	for _, want := range files {
		found := false
		for _, ff := range compFiles {
			if matchGlob(want, ff.Path) || basename(ff.Path) == want {
				found = true
				break
			}
		}
		if !found {
			vs = append(vs, model.Violation{
				RuleID: in.Rule.ID, Severity: sev, File: comp,
				Message:    fmt.Sprintf("component %q is missing required file %q", comp, want),
				SourceFile: in.Rule.SourceFile,
			})
		}
	}
	return vs, nil, nil
}

// ---- helpers -------------------------------------------------------------

// scopeMatches reports whether a component belongs to the named scope, where
// scope is either a component name or a layer name.
func scopeMatches(t *Topology, component, scope string) bool {
	if component == "" || scope == "" {
		return false
	}
	if component == scope {
		return true
	}
	return t.LayerOf(component) == scope
}

func basename(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
