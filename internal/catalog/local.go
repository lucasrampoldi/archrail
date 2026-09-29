package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/archrail/archrail/internal/model"
)

// ErrProfileNotFound is returned by Load when the named profile does not exist.
var ErrProfileNotFound = errors.New("profile not found")

// ErrVersionNotFound is returned by Load when the profile exists but the
// requested version does not.
var ErrVersionNotFound = errors.New("profile version not found")

// Local is a catalog source backed by a local directory. The expected layout
// is:
//
//	<root>/profiles/<name>/<version>/profile.yaml
//	<root>/profiles/<name>/<version>/standards/*.yaml
type Local struct {
	Root string
}

// NewLocal constructs a Local source rooted at path.
func NewLocal(path string) *Local { return &Local{Root: path} }

func (l *Local) profilesDir() string { return filepath.Join(l.Root, "profiles") }

// List enumerates profiles, detecting duplicate name+version declarations.
func (l *Local) List() ([]ProfileID, error) {
	if err := l.assertReadable(); err != nil {
		return nil, err
	}
	var ids []ProfileID
	seen := map[string]string{} // name@version -> source dir
	dir := l.profilesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading catalog %q: %w", dir, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, nameEntry := range entries {
		if !nameEntry.IsDir() {
			continue
		}
		nameDir := filepath.Join(dir, nameEntry.Name())
		versions, err := os.ReadDir(nameDir)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", nameDir, err)
		}
		sort.Slice(versions, func(i, j int) bool { return versions[i].Name() < versions[j].Name() })
		for _, verEntry := range versions {
			if !verEntry.IsDir() {
				continue
			}
			pdir := filepath.Join(nameDir, verEntry.Name())
			p, err := l.loadProfileMeta(pdir)
			if err != nil {
				return nil, err
			}
			key := p.Name + "@" + p.Version
			if prev, dup := seen[key]; dup {
				return nil, fmt.Errorf("duplicate profile %s in catalog: %q and %q", key, prev, pdir)
			}
			seen[key] = pdir
			ids = append(ids, ProfileID{Name: p.Name, Version: p.Version, Description: p.Description})
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].Name != ids[j].Name {
			return ids[i].Name < ids[j].Name
		}
		return ids[i].Version < ids[j].Version
	})
	return ids, nil
}

// Load resolves a single profile with its merged rule set.
func (l *Local) Load(name, version string) (*model.Profile, error) {
	if err := l.assertReadable(); err != nil {
		return nil, err
	}
	pdir := filepath.Join(l.profilesDir(), name, version)
	info, err := os.Stat(pdir)
	if err != nil || !info.IsDir() {
		return nil, l.notFoundErr(name, version)
	}
	p, err := l.loadProfileMeta(pdir)
	if err != nil {
		return nil, err
	}
	rules, err := loadStandards(pdir)
	if err != nil {
		return nil, err
	}
	p.Rules = rules
	p.SourceDir = pdir
	return p, nil
}

func (l *Local) notFoundErr(name, version string) error {
	// Distinguish "no such profile" from "no such version".
	nameDir := filepath.Join(l.profilesDir(), name)
	if info, err := os.Stat(nameDir); err == nil && info.IsDir() {
		versions := l.versionsOf(name)
		return fmt.Errorf("%w: %s@%s (available versions: %s)",
			ErrVersionNotFound, name, version, strings.Join(versions, ", "))
	}
	names := l.names()
	return fmt.Errorf("%w: %s@%s (available profiles: %s)",
		ErrProfileNotFound, name, version, strings.Join(names, ", "))
}

func (l *Local) versionsOf(name string) []string {
	var out []string
	entries, _ := os.ReadDir(filepath.Join(l.profilesDir(), name))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func (l *Local) names() []string {
	var out []string
	entries, _ := os.ReadDir(l.profilesDir())
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func (l *Local) assertReadable() error {
	info, err := os.Stat(l.Root)
	if err != nil {
		return fmt.Errorf("catalog path %q is not accessible: %w", l.Root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("catalog path %q is not a directory", l.Root)
	}
	return nil
}

func (l *Local) loadProfileMeta(pdir string) (*model.Profile, error) {
	file := filepath.Join(pdir, "profile.yaml")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", file, err)
	}
	var p model.Profile
	if err := model.DecodeStrict(file, data, &p); err != nil {
		return nil, err
	}
	if p.Name == "" {
		return nil, fmt.Errorf("%s: profile is missing required field 'name'", file)
	}
	if p.Version == "" {
		return nil, fmt.Errorf("%s: profile is missing required field 'version'", file)
	}
	return &p, nil
}

// loadStandards reads and merges every standards/*.yaml file under pdir.
func loadStandards(pdir string) ([]model.Rule, error) {
	sdir := filepath.Join(pdir, "standards")
	entries, err := os.ReadDir(sdir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %q: %w", sdir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext := strings.ToLower(filepath.Ext(e.Name())); ext == ".yaml" || ext == ".yml" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	var rules []model.Rule
	for _, name := range files {
		path := filepath.Join(sdir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", path, err)
		}
		var sf struct {
			Rules []model.Rule `yaml:"rules"`
		}
		if err := model.Decode(path, data, &sf); err != nil {
			return nil, err
		}
		rel := filepath.Join("standards", name)
		for i := range sf.Rules {
			sf.Rules[i].SourceFile = rel
			rules = append(rules, sf.Rules[i])
		}
	}
	return rules, nil
}
