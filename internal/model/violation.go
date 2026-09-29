package model

// Violation is a single architecture finding produced by a rule.
type Violation struct {
	RuleID     string   `json:"ruleId"`
	Severity   Severity `json:"severity"`
	File       string   `json:"file"`
	Line       int      `json:"line,omitempty"`
	Message    string   `json:"message"`
	SourceFile string   `json:"sourceFile,omitempty"`

	// LLMDerived marks findings produced by the semantic (LLM) evaluator.
	LLMDerived bool   `json:"llmDerived,omitempty"`
	Model      string `json:"model,omitempty"`

	// Baselined is set when the finding matches a baseline entry (reported but
	// not gate-failing).
	Baselined bool `json:"baselined,omitempty"`
}

// Note records something that was intentionally not evaluated (e.g. a file with
// no import extractor, or a semantic rule skipped because the LLM was
// unavailable). Notes are reported but are never violations, and never count as
// passing.
type Note struct {
	RuleID string `json:"ruleId,omitempty"`
	File   string `json:"file,omitempty"`
	Reason string `json:"reason"`
}
