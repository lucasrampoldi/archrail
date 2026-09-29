package model

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Rule is a single architecture constraint declared in a profile's standards
// file. Common fields are typed; type-specific parameters are captured raw in
// Params and decoded by the rule-type implementation.
type Rule struct {
	ID          string   `yaml:"id"`
	Type        string   `yaml:"type"`
	Description string   `yaml:"description"`
	Severity    Severity `yaml:"severity"`

	// Params holds every key other than the common fields above. Rule-type
	// implementations decode the parameters they need from here.
	Params map[string]yaml.Node `yaml:",inline"`

	// SourceFile records the standards file this rule was loaded from, for
	// reporting. It is not part of the YAML.
	SourceFile string `yaml:"-"`
}

// DecodeParams decodes the named parameter into out. It returns false (no
// error) when the parameter is absent.
func (r Rule) DecodeParams(key string, out any) (bool, error) {
	node, ok := r.Params[key]
	if !ok {
		return false, nil
	}
	if err := node.Decode(out); err != nil {
		return true, fmt.Errorf("rule %q: parameter %q: %w", r.ID, key, err)
	}
	return true, nil
}

// standardsFile is the on-disk shape of a standards/*.yaml file.
type standardsFile struct {
	Rules []Rule `yaml:"rules"`
}
