package model

import "fmt"

// Severity is the gating weight of a rule's findings.
type Severity string

const (
	// SeverityUnset means the rule did not declare a severity; the configured
	// default applies.
	SeverityUnset Severity = ""
	// SeverityError makes a finding gate-failing.
	SeverityError Severity = "error"
	// SeverityWarn makes a finding advisory (reported, not gate-failing).
	SeverityWarn Severity = "warn"
)

// Validate ensures the severity is a known value (or unset).
func (s Severity) Validate() error {
	switch s {
	case SeverityUnset, SeverityError, SeverityWarn:
		return nil
	default:
		return fmt.Errorf("invalid severity %q (want %q or %q)", s, SeverityError, SeverityWarn)
	}
}

// Resolve returns the effective severity, applying def when this severity is
// unset. If both are unset, SeverityError is assumed.
func (s Severity) Resolve(def Severity) Severity {
	if s != SeverityUnset {
		return s
	}
	if def != SeverityUnset {
		return def
	}
	return SeverityError
}
