package engine

import (
	"fmt"
	"sort"

	"github.com/archrail/archrail/internal/model"
)

// Result is the outcome of evaluating a rule set.
type Result struct {
	Violations []model.Violation
	Notes      []model.Note
}

// Evaluate runs every rule against the facts and returns a deterministically
// ordered result. Rules are evaluated in sorted id order; findings are sorted so
// output is byte-stable for deterministic rules.
func Evaluate(rules []model.Rule, reg *Registry, topo *Topology, facts *Facts, def model.Severity, subjects map[string]bool, sem SemanticContext) (*Result, error) {
	ordered := append([]model.Rule{}, rules...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })

	res := &Result{}
	for _, rule := range ordered {
		rt, ok := reg.Get(rule.Type)
		if !ok {
			return nil, fmt.Errorf("rule %q: unknown type %q", rule.ID, rule.Type)
		}
		in := EvalInput{
			Rule:            rule,
			Topo:            topo,
			Facts:           facts,
			DefaultSeverity: def,
			Subjects:        subjects,
			Semantic:        sem,
		}
		vs, notes, err := rt.Evaluate(in)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", rule.ID, err)
		}
		res.Violations = append(res.Violations, vs...)
		res.Notes = append(res.Notes, notes...)
	}

	sortViolations(res.Violations)
	sortNotes(res.Notes)
	return res, nil
}

func sortViolations(vs []model.Violation) {
	sort.SliceStable(vs, func(i, j int) bool {
		a, b := vs[i], vs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		return a.Message < b.Message
	})
}

func sortNotes(ns []model.Note) {
	sort.SliceStable(ns, func(i, j int) bool {
		a, b := ns[i], ns[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		return a.Reason < b.Reason
	})
}
