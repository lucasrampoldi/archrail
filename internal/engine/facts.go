package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/archrail/archrail/internal/extract"
)

// FileFact holds the deterministic facts computed for one first-party file.
type FileFact struct {
	Path         string // repo-relative, slash-normalized
	Component    string // "" when unassigned
	Imports      []string
	HasExtractor bool
}

// Facts is the computed knowledge about a repository used by rule evaluation and
// by the architecture fingerprint.
type Facts struct {
	Root        string
	Files       []FileFact
	byComponent map[string][]int
}

// BuildFacts walks the repository from root, skipping excluded paths, assigns
// each file to a component, and extracts imports where an extractor applies. It
// does not execute any repository code. Output ordering is deterministic.
func BuildFacts(root string, topo *Topology, sel *extract.Selector, ex *Excluder) (*Facts, error) {
	f := &Facts{Root: root, byComponent: map[string][]int{}}
	var paths []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if ex.Excluded(rel + "/") {
				return fs.SkipDir
			}
			if base := d.Name(); base == ".git" || base == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if ex.Excluded(rel) {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning repository: %w", err)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		fact := FileFact{Path: rel, Component: topo.AssignComponent(rel)}
		if e, ok := sel.For(rel); ok {
			fact.HasExtractor = true
			content, rerr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if rerr != nil {
				return nil, fmt.Errorf("reading %q: %w", rel, rerr)
			}
			fact.Imports = e.Imports(content)
		}
		idx := len(f.Files)
		f.Files = append(f.Files, fact)
		if fact.Component != "" {
			f.byComponent[fact.Component] = append(f.byComponent[fact.Component], idx)
		}
	}
	return f, nil
}

// FilesIn returns the file facts assigned to a component.
func (f *Facts) FilesIn(component string) []FileFact {
	var out []FileFact
	for _, i := range f.byComponent[component] {
		out = append(out, f.Files[i])
	}
	return out
}

// HasPathMatching reports whether any file path matches the doublestar glob.
func (f *Facts) HasPathMatching(glob string) bool {
	for _, ff := range f.Files {
		if ok := matchGlob(glob, ff.Path); ok {
			return true
		}
	}
	return false
}

// ReadFile returns the contents of a repo-relative path.
func (f *Facts) ReadFile(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(f.Root, filepath.FromSlash(rel)))
}

func matchGlob(glob, path string) bool {
	// Support a bare filename glob (e.g. "package.json") matching at any depth.
	if !strings.Contains(glob, "/") {
		return filepath.Base(path) == glob || globMatch(glob, filepath.Base(path))
	}
	return globMatch(glob, path)
}
