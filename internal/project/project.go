// Package project handles the adopting repository: discovering the `.archrail/`
// directory, loading and validating its config, and resolving the adopted
// profile from the catalog into an evaluable topology + rule set.
package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/archrail/archrail/internal/model"
)

// DirName is the project configuration directory.
const DirName = ".archrail"

// ConfigName is the project configuration file within DirName.
const ConfigName = "archrail.yaml"

// Project is a discovered, loaded adopting repository.
type Project struct {
	Root       string // repository root (contains .archrail/)
	Dir        string // absolute path to .archrail/
	ConfigPath string
	Config     *model.Config
}

// Discover walks up from start to find a `.archrail/archrail.yaml`, loads it, and
// returns the project. It errors clearly when none is found.
func Discover(start string) (*Project, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		cfgPath := filepath.Join(dir, DirName, ConfigName)
		if _, err := os.Stat(cfgPath); err == nil {
			return load(dir, cfgPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("no %s/%s found (run `archrail init` to create one)", DirName, ConfigName)
		}
		dir = parent
	}
}

func load(root, cfgPath string) (*Project, error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	var cfg model.Config
	if err := model.DecodeStrict(cfgPath, data, &cfg); err != nil {
		return nil, err
	}
	p := &Project{
		Root:       root,
		Dir:        filepath.Join(root, DirName),
		ConfigPath: cfgPath,
		Config:     &cfg,
	}
	return p, p.validate()
}

func (p *Project) validate() error {
	c := p.Config
	if c.Version == 0 {
		return fmt.Errorf("%s: missing 'version'", p.ConfigPath)
	}
	if c.Version > model.SchemaVersion {
		return fmt.Errorf("%s: config schema version %d is newer than this CLI supports (%d); please upgrade archrail",
			p.ConfigPath, c.Version, model.SchemaVersion)
	}
	if c.Catalog.Source == "" {
		c.Catalog.Source = "local"
	}
	if c.Catalog.Source != "local" {
		return fmt.Errorf("%s: unsupported catalog source %q (v0 supports only \"local\")", p.ConfigPath, c.Catalog.Source)
	}
	if c.Catalog.Path == "" {
		return fmt.Errorf("%s: catalog.path is required for the local catalog source", p.ConfigPath)
	}
	if c.Profile.Name == "" || c.Profile.Version == "" {
		return fmt.Errorf("%s: no profile adopted; set profile.name and profile.version to a profile from the catalog", p.ConfigPath)
	}
	return nil
}

// CatalogPath returns the absolute catalog path (resolving relative paths against
// the repo root).
func (p *Project) CatalogPath() string {
	cp := p.Config.Catalog.Path
	if filepath.IsAbs(cp) {
		return cp
	}
	return filepath.Join(p.Root, cp)
}

// BaselinePath returns the conventional baseline file path.
func (p *Project) BaselinePath() string {
	return filepath.Join(p.Dir, "baseline.yaml")
}
