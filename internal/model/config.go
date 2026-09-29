package model

// SchemaVersion is the highest config/profile schema version this build
// understands.
const SchemaVersion = 1

// CatalogRef tells Archrail where to resolve profiles from.
type CatalogRef struct {
	// Source names the catalog source implementation. v0 supports "local".
	Source string `yaml:"source"`
	// Path is the local filesystem path to the catalog (for source "local").
	Path string `yaml:"path"`
}

// ProfileRef is a project's pinned adoption of a catalog profile.
type ProfileRef struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// Config is the project-level `.archrail/archrail.yaml`. It adopts a profile
// and may bind/override topology to this repository's paths.
type Config struct {
	Version int        `yaml:"version"`
	Catalog CatalogRef `yaml:"catalog"`
	Profile ProfileRef `yaml:"profile"`

	// Local topology binding/overrides. When set, these merge over the
	// profile's own topology (matched by component/layer name).
	Components []Component          `yaml:"components"`
	Layers     []Layer              `yaml:"layers"`
	Extractors []ExtractorSelection `yaml:"extractors"`
	Defaults   Defaults             `yaml:"defaults"`
	Semantic   SemanticConfig       `yaml:"semantic"`

	// Exclude adds path globs to the built-in dependency/vendored exclusions.
	Exclude []string `yaml:"exclude"`
}
