// Package semantic implements the non-deterministic LLM evaluator: the
// `semantic` rule type and the Vercel AI Gateway (Claude) backend. It is opt-in,
// advisory by default, and degrades gracefully when the LLM is unavailable.
package semantic

import (
	"fmt"
	"sort"

	"github.com/archrail/archrail/internal/engine"
	"github.com/archrail/archrail/internal/model"
)

// RuleType is the `semantic` rule kind. It implements engine.RuleType.
type RuleType struct{}

// New returns the semantic rule type for registration.
func New() engine.RuleType { return RuleType{} }

func (RuleType) Name() string { return "semantic" }

func (RuleType) Params() []engine.ParamDoc {
	return []engine.ParamDoc{
		{Name: "assertion", Required: true, Description: "natural-language property the code must satisfy"},
		{Name: "scope", Required: false, Description: "component or layer the assertion applies to"},
		{Name: "paths", Required: false, Description: "explicit path globs the assertion applies to"},
	}
}

func (RuleType) Validate(rule model.Rule) error {
	var assertion, scope string
	var paths []string
	rule.DecodeParams("assertion", &assertion)
	rule.DecodeParams("scope", &scope)
	rule.DecodeParams("paths", &paths)
	if assertion == "" {
		return fmt.Errorf("requires an 'assertion'")
	}
	if scope == "" && len(paths) == 0 {
		return fmt.Errorf("requires a target 'scope' or 'paths'")
	}
	return nil
}

func (RuleType) Evaluate(in engine.EvalInput) ([]model.Violation, []model.Note, error) {
	var assertion, scope string
	var paths []string
	in.Rule.DecodeParams("assertion", &assertion)
	in.Rule.DecodeParams("scope", &scope)
	in.Rule.DecodeParams("paths", &paths)

	// Advisory by default: semantic findings default to warn regardless of the
	// global default severity.
	sev := in.Rule.Severity.Resolve(model.SeverityWarn)

	// Curate the target files (first-party, in scope). facts already excludes
	// dependency/vendored/generated paths.
	files := map[string]string{}
	for _, ff := range in.Facts.Files {
		if !in.InScope(ff.Path) {
			continue
		}
		if !targetMatches(in.Topo, ff, scope, paths) {
			continue
		}
		content, err := in.Facts.ReadFile(ff.Path)
		if err != nil {
			return nil, nil, err
		}
		files[ff.Path] = string(content)
	}
	if len(files) == 0 {
		// Nothing in scope to evaluate (e.g. diff-scoping excluded everything).
		return nil, nil, nil
	}

	if !in.Semantic.Available || in.Semantic.Backend == nil {
		if in.Semantic.Require {
			return nil, nil, fmt.Errorf("semantic evaluation required but the LLM is unavailable (missing key or backend)")
		}
		return nil, []model.Note{{RuleID: in.Rule.ID, Reason: "skipped: LLM unavailable"}}, nil
	}

	verdict, err := in.Semantic.Backend.Evaluate(engine.SemanticRequest{
		Assertion:   assertion,
		Fingerprint: in.Semantic.Fingerprint,
		Files:       files,
	})
	if err != nil {
		if in.Semantic.Require {
			return nil, nil, fmt.Errorf("semantic evaluation failed: %w", err)
		}
		return nil, []model.Note{{RuleID: in.Rule.ID, Reason: "skipped: LLM unavailable"}}, nil
	}
	if verdict.Conforms && len(verdict.Findings) == 0 {
		return nil, nil, nil
	}

	var vs []model.Violation
	for _, f := range verdict.Findings {
		file := f.File
		if file == "" {
			file = firstKey(files)
		}
		vs = append(vs, model.Violation{
			RuleID:     in.Rule.ID,
			Severity:   sev,
			File:       file,
			Message:    f.Message,
			SourceFile: in.Rule.SourceFile,
			LLMDerived: true,
			Model:      in.Semantic.Model,
		})
	}
	if len(vs) == 0 && !verdict.Conforms {
		vs = append(vs, model.Violation{
			RuleID: in.Rule.ID, Severity: sev, File: firstKey(files),
			Message:    "assertion not satisfied: " + assertion,
			SourceFile: in.Rule.SourceFile, LLMDerived: true, Model: in.Semantic.Model,
		})
	}
	return vs, nil, nil
}

func targetMatches(t *engine.Topology, ff engine.FileFact, scope string, paths []string) bool {
	if scope != "" {
		if ff.Component == scope || t.LayerOf(ff.Component) == scope {
			return true
		}
	}
	for _, g := range paths {
		if ok, _ := matchPath(g, ff.Path); ok {
			return true
		}
	}
	return false
}

func firstKey(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return keys[0]
	}
	return ""
}
