package engine

// SemanticContext carries everything the semantic rule type needs to call an
// LLM, without engine depending on the gateway or fingerprint packages. It is
// wired up by the CLI when semantic evaluation is enabled.
type SemanticContext struct {
	// Available reports whether an evaluator backend and credentials are present.
	Available bool
	// Require, when true, turns unavailability into an operational error instead
	// of a skip.
	Require bool
	// Model is the evaluating model identifier (for labeling findings).
	Model string
	// Fingerprint is the pre-rendered, deterministic architecture fingerprint
	// sent to the LLM as structural context.
	Fingerprint string
	// Backend performs the actual evaluation. Nil when unavailable.
	Backend SemanticBackend
}

// SemanticRequest is one assertion evaluated against curated code context.
type SemanticRequest struct {
	Assertion   string
	Fingerprint string
	// Files maps repo-relative paths to their contents (curated, first-party
	// only; never dependencies/vendored).
	Files map[string]string
}

// SemanticFinding is a single violation the LLM reports for an assertion.
type SemanticFinding struct {
	File    string
	Message string
}

// SemanticVerdict is the backend's result for one assertion.
type SemanticVerdict struct {
	// Conforms is true when the code satisfies the assertion.
	Conforms bool
	Findings []SemanticFinding
}

// SemanticBackend evaluates a natural-language assertion against code context.
type SemanticBackend interface {
	Evaluate(req SemanticRequest) (SemanticVerdict, error)
}
