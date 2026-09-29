package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/archrail/archrail/internal/model"
)

// ParamDoc documents a rule-type parameter.
type ParamDoc struct {
	Name        string
	Required    bool
	Description string
}

// EvalInput is passed to a rule type's Evaluate.
type EvalInput struct {
	Rule            model.Rule
	Topo            *Topology
	Facts           *Facts
	DefaultSeverity model.Severity

	// Subjects restricts which files are treated as rule subjects (diff-scoping).
	// A nil map means "all files are in scope".
	Subjects map[string]bool

	// Semantic carries the LLM evaluation context; only the semantic rule type
	// consults it.
	Semantic SemanticContext
}

// InScope reports whether a repo-relative path is a rule subject.
func (in EvalInput) InScope(path string) bool {
	if in.Subjects == nil {
		return true
	}
	return in.Subjects[path]
}

// EffectiveSeverity resolves the rule's severity against the default.
func (in EvalInput) EffectiveSeverity() model.Severity {
	return in.Rule.Severity.Resolve(in.DefaultSeverity)
}

// RuleType is a registered, evaluable architecture rule kind.
type RuleType interface {
	Name() string
	Params() []ParamDoc
	Validate(rule model.Rule) error
	Evaluate(in EvalInput) ([]model.Violation, []model.Note, error)
}

// Registry maps rule-type names to implementations.
type Registry struct {
	byName map[string]RuleType
}

// NewRegistry returns a registry with the deterministic built-in rule types.
func NewRegistry() *Registry {
	r := &Registry{byName: map[string]RuleType{}}
	r.Register(forbiddenImport{})
	r.Register(layerBoundary{})
	r.Register(forbiddenDependency{})
	r.Register(requiredDependency{})
	r.Register(requiredFiles{})
	return r
}

// Register adds a rule type (used to add the semantic type at wiring time).
func (r *Registry) Register(rt RuleType) { r.byName[rt.Name()] = rt }

// Get returns the named rule type.
func (r *Registry) Get(name string) (RuleType, bool) {
	rt, ok := r.byName[name]
	return rt, ok
}

// Names returns registered rule-type names, sorted.
func (r *Registry) Names() []string {
	var out []string
	for n := range r.byName {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Describe returns a human-readable listing of rule types and their parameters.
func (r *Registry) Describe() string {
	var b strings.Builder
	for _, n := range r.Names() {
		rt := r.byName[n]
		fmt.Fprintf(&b, "%s\n", n)
		for _, p := range rt.Params() {
			req := "optional"
			if p.Required {
				req = "required"
			}
			fmt.Fprintf(&b, "  - %s (%s): %s\n", p.Name, req, p.Description)
		}
	}
	return b.String()
}

// ValidateRuleSet checks a full rule set: unique ids, known types, valid
// severity, and per-type parameter validation.
func (r *Registry) ValidateRuleSet(rules []model.Rule) error {
	seen := map[string]string{}
	for _, rule := range rules {
		if rule.ID == "" {
			return fmt.Errorf("%s: a rule is missing its 'id'", rule.SourceFile)
		}
		if prev, dup := seen[rule.ID]; dup {
			return fmt.Errorf("duplicate rule id %q in %q and %q", rule.ID, prev, rule.SourceFile)
		}
		seen[rule.ID] = rule.SourceFile
		if err := rule.Severity.Validate(); err != nil {
			return fmt.Errorf("rule %q (%s): %w", rule.ID, rule.SourceFile, err)
		}
		rt, ok := r.Get(rule.Type)
		if !ok {
			return fmt.Errorf("rule %q (%s): unknown type %q (known types: %s)",
				rule.ID, rule.SourceFile, rule.Type, strings.Join(r.Names(), ", "))
		}
		if err := rt.Validate(rule); err != nil {
			return fmt.Errorf("rule %q (%s): %w", rule.ID, rule.SourceFile, err)
		}
	}
	return nil
}
