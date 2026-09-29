// Package baseline records known, pre-existing violations so brownfield repos
// can adopt Archrail without an immediate red build. Only new (non-baselined)
// violations fail the gate.
package baseline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/archrail/archrail/internal/model"
)

// Entry is one baselined violation. The locator is content-derived (a digest of
// the message) rather than a line number, so unrelated edits do not churn the
// baseline.
type Entry struct {
	Rule    string `yaml:"rule"`
	File    string `yaml:"file"`
	Locator string `yaml:"locator"`
}

// File is the on-disk baseline document.
type File struct {
	Version    int     `yaml:"version"`
	Violations []Entry `yaml:"violations"`
}

// Set is a loaded baseline for fast lookup.
type Set struct {
	keys map[string]bool
}

// Key computes the stable identity of a violation.
func Key(v model.Violation) string {
	return v.RuleID + "\x00" + v.File + "\x00" + locator(v.Message)
}

func locator(msg string) string {
	sum := sha256.Sum256([]byte(msg))
	return hex.EncodeToString(sum[:])[:12]
}

// Load reads a baseline file. A missing file yields an empty set (not an error).
func Load(path string) (*Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Set{keys: map[string]bool{}}, nil
		}
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s := &Set{keys: map[string]bool{}}
	for _, e := range f.Violations {
		s.keys[e.Rule+"\x00"+e.File+"\x00"+e.Locator] = true
	}
	return s, nil
}

// Apply marks violations present in the baseline as baselined (in place).
func (s *Set) Apply(vs []model.Violation) {
	for i := range vs {
		if s.keys[Key(vs[i])] {
			vs[i].Baselined = true
		}
	}
}

// Generate writes a deterministic baseline file from the given violations.
func Generate(path string, vs []model.Violation) error {
	f := File{Version: 1}
	for _, v := range vs {
		f.Violations = append(f.Violations, Entry{Rule: v.RuleID, File: v.File, Locator: locator(v.Message)})
	}
	sort.Slice(f.Violations, func(i, j int) bool {
		a, b := f.Violations[i], f.Violations[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.Locator < b.Locator
	})
	out, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
