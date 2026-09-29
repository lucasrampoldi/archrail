package project

import (
	"github.com/archrail/archrail/internal/catalog"
	"github.com/archrail/archrail/internal/engine"
	"github.com/archrail/archrail/internal/extract"
	"github.com/archrail/archrail/internal/model"
)

// Resolved is a fully resolved project: the adopted profile, evaluable topology,
// selector, and effective defaults.
type Resolved struct {
	Project  *Project
	Profile  *model.Profile
	Topology *engine.Topology
	Selector *extract.Selector
	Registry *engine.Registry
	Defaults model.Severity
}

// Resolve loads the adopted profile from the catalog and prepares everything the
// engine needs. reg is the rule-type registry (with any extra types registered).
func (p *Project) Resolve(reg *engine.Registry) (*Resolved, error) {
	src := catalog.NewLocal(p.CatalogPath())
	profile, err := src.Load(p.Config.Profile.Name, p.Config.Profile.Version)
	if err != nil {
		return nil, err
	}

	if err := reg.ValidateRuleSet(profile.Rules); err != nil {
		return nil, err
	}

	topo, err := engine.BuildTopology(profile, p.Config)
	if err != nil {
		return nil, err
	}

	selections := p.Config.Extractors
	if len(selections) == 0 {
		selections = profile.Extractors
	}
	sel, err := extract.BuildSelector(extract.DefaultRegistry(), selections)
	if err != nil {
		return nil, err
	}

	def := p.Config.Defaults.Severity.Resolve(profile.Defaults.Severity)

	return &Resolved{
		Project:  p,
		Profile:  profile,
		Topology: topo,
		Selector: sel,
		Registry: reg,
		Defaults: def,
	}, nil
}

// BuildFacts scans the repository for facts using the resolved selector and
// exclusions.
func (r *Resolved) BuildFacts() (*engine.Facts, error) {
	ex := engine.NewExcluder(r.Project.Config.Exclude)
	return engine.BuildFacts(r.Project.Root, r.Topology, r.Selector, ex)
}
