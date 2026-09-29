package extract

import (
	"sort"
	"strings"

	"github.com/archrail/archrail/internal/model"
)

// Selector chooses an extractor for a file based on its extension, according to
// the configured extractor selections.
type Selector struct {
	byExt map[string]Extractor
}

// BuildSelector resolves the configured extractor selections against the
// registry. An unknown extractor name is an error. When no selections are given,
// a sensible default mapping is used.
func BuildSelector(reg *Registry, selections []model.ExtractorSelection) (*Selector, error) {
	s := &Selector{byExt: map[string]Extractor{}}
	if len(selections) == 0 {
		selections = defaultSelections()
	}
	for _, sel := range selections {
		e, ok := reg.Get(sel.Name)
		if !ok {
			return nil, &UnknownExtractorError{Name: sel.Name, Known: reg.Names()}
		}
		for _, ext := range sel.Extensions {
			s.byExt[normalizeExt(ext)] = e
		}
	}
	return s, nil
}

// For returns the extractor for a repo-relative path, or (nil, false) when the
// path's extension has no configured extractor.
func (s *Selector) For(path string) (Extractor, bool) {
	i := strings.LastIndex(path, ".")
	if i < 0 {
		return nil, false
	}
	e, ok := s.byExt[strings.ToLower(path[i:])]
	return e, ok
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

func defaultSelections() []model.ExtractorSelection {
	return []model.ExtractorSelection{
		{Name: "js", Extensions: []string{".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"}},
		{Name: "go", Extensions: []string{".go"}},
		{Name: "regex", Extensions: []string{".py", ".java", ".rb"}},
	}
}

// UnknownExtractorError is returned when a configured extractor is not
// registered.
type UnknownExtractorError struct {
	Name  string
	Known []string
}

func (e *UnknownExtractorError) Error() string {
	sort.Strings(e.Known)
	return "unknown import extractor " + e.Name + " (known: " + strings.Join(e.Known, ", ") + ")"
}
