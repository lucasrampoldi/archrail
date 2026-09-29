package model

// Component is a named logical module identified by one or more path globs
// (doublestar syntax, e.g. "src/handlers/**").
type Component struct {
	Name  string   `yaml:"name"`
	Paths []string `yaml:"paths"`
}

// Layer groups components into an architectural layer.
type Layer struct {
	Name       string   `yaml:"name"`
	Components []string `yaml:"components"`
}

// ExtractorSelection binds an import extractor to a set of file extensions.
type ExtractorSelection struct {
	Name       string   `yaml:"name"`
	Extensions []string `yaml:"extensions"`
}

// Defaults holds default values applied across a profile/project.
type Defaults struct {
	Severity Severity `yaml:"severity"`
}

// SemanticConfig configures the optional LLM evaluator.
type SemanticConfig struct {
	// Model is the Claude model identifier used via the Vercel AI Gateway.
	// Empty means the documented default is used.
	Model string `yaml:"model"`
	// BaseURL optionally overrides the Vercel AI Gateway base URL.
	BaseURL string `yaml:"baseURL"`
}
