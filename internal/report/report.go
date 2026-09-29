// Package report renders conformance results in human-readable and JSON forms
// and computes the gate outcome.
package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/archrail/archrail/internal/model"
)

// Report is the full outcome of a check, ready to render.
type Report struct {
	Profile    string            `json:"profile"`
	Violations []model.Violation `json:"violations"`
	Notes      []model.Note      `json:"notes"`
	Summary    Summary           `json:"summary"`
}

// Summary holds aggregate counts and the gate outcome.
type Summary struct {
	Errors      int  `json:"errors"`
	Warnings    int  `json:"warnings"`
	Baselined   int  `json:"baselined"`
	Skipped     int  `json:"skipped"`
	GateFailing bool `json:"gateFailing"`
}

// New builds a report from raw findings and computes the summary.
func New(profile string, violations []model.Violation, notes []model.Note) *Report {
	r := &Report{Profile: profile, Violations: violations, Notes: notes}
	if r.Violations == nil {
		r.Violations = []model.Violation{}
	}
	if r.Notes == nil {
		r.Notes = []model.Note{}
	}
	for _, v := range r.Violations {
		switch {
		case v.Baselined:
			r.Summary.Baselined++
		case v.Severity == model.SeverityError:
			r.Summary.Errors++
			r.Summary.GateFailing = true
		case v.Severity == model.SeverityWarn:
			r.Summary.Warnings++
		}
	}
	r.Summary.Skipped = len(r.Notes)
	return r
}

// JSON renders the stable machine-readable report.
func (r *Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// Human renders the default human-readable report.
func (r *Report) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Archrail conformance report (profile: %s)\n", r.Profile)
	if len(r.Violations) == 0 && len(r.Notes) == 0 {
		b.WriteString("\n✓ No architecture violations found.\n")
		return b.String()
	}
	if len(r.Violations) > 0 {
		b.WriteString("\nViolations:\n")
		for _, v := range r.Violations {
			b.WriteString("  " + formatViolation(v) + "\n")
		}
	}
	if len(r.Notes) > 0 {
		b.WriteString("\nNotes (not evaluated):\n")
		for _, n := range r.Notes {
			loc := n.File
			if n.RuleID != "" {
				loc = n.RuleID + " " + loc
			}
			fmt.Fprintf(&b, "  - %s: %s\n", strings.TrimSpace(loc), n.Reason)
		}
	}
	fmt.Fprintf(&b, "\nSummary: %d error(s), %d warning(s), %d baselined, %d skipped.\n",
		r.Summary.Errors, r.Summary.Warnings, r.Summary.Baselined, r.Summary.Skipped)
	if r.Summary.GateFailing {
		b.WriteString("Result: FAIL (non-baselined error violations present).\n")
	} else {
		b.WriteString("Result: PASS.\n")
	}
	return b.String()
}

func formatViolation(v model.Violation) string {
	loc := v.File
	if v.Line > 0 {
		loc = fmt.Sprintf("%s:%d", v.File, v.Line)
	}
	tags := ""
	if v.Baselined {
		tags += " [baselined]"
	}
	if v.LLMDerived {
		tags += " [LLM:" + v.Model + "]"
		if v.Severity == model.SeverityWarn {
			tags += " [advisory]"
		}
	}
	src := ""
	if v.SourceFile != "" {
		src = " (" + v.SourceFile + ")"
	}
	return fmt.Sprintf("[%s] %s: %s — %s%s%s", strings.ToUpper(string(v.Severity)), v.RuleID, loc, v.Message, src, tags)
}
