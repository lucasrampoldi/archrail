package model

// Profile is a named, versioned, approved architecture: a self-contained
// bundle of topology + rules (+ optional approved stack). Profiles live in a
// catalog and are adopted by projects.
type Profile struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`

	Components []Component          `yaml:"components"`
	Layers     []Layer              `yaml:"layers"`
	Extractors []ExtractorSelection `yaml:"extractors"`
	Defaults   Defaults             `yaml:"defaults"`

	// Stack optionally documents the approved technology stack (e.g.
	// frontend: [react], backend: [express]). It is advisory metadata surfaced
	// to agents; hard enforcement is expressed via dependency rules.
	Stack map[string][]string `yaml:"stack"`

	// Rules is populated by merging all standards/*.yaml files. It is not part
	// of profile.yaml itself.
	Rules []Rule `yaml:"-"`

	// SourceDir records where the profile was loaded from, for diagnostics.
	SourceDir string `yaml:"-"`
}
