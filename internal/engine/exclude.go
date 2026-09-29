package engine

import "github.com/bmatcuk/doublestar/v4"

// DefaultExcludes are dependency, vendored, and generated paths that Archrail
// never treats as first-party source (and never sends to the LLM).
var DefaultExcludes = []string{
	"**/.git/**",
	"**/node_modules/**",
	"**/vendor/**",
	"**/dist/**",
	"**/build/**",
	"**/.next/**",
	"**/target/**",
	"**/__pycache__/**",
	"**/*.lock",
	"**/package-lock.json",
	"**/yarn.lock",
	"**/pnpm-lock.yaml",
	"**/go.sum",
}

// Excluder decides whether a repo-relative path is excluded.
type Excluder struct {
	patterns []string
}

// NewExcluder combines the built-in defaults with any extra config patterns.
func NewExcluder(extra []string) *Excluder {
	pats := append([]string{}, DefaultExcludes...)
	pats = append(pats, extra...)
	return &Excluder{patterns: pats}
}

// Excluded reports whether relPath matches any exclusion pattern.
func (e *Excluder) Excluded(relPath string) bool {
	for _, p := range e.patterns {
		if ok, _ := doublestar.Match(p, relPath); ok {
			return true
		}
	}
	return false
}
