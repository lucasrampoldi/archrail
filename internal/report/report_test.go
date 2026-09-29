package report_test

import (
	"encoding/json"
	"testing"

	"github.com/archrail/archrail/internal/model"
	"github.com/archrail/archrail/internal/report"
)

func TestJSONSchemaStable(t *testing.T) {
	r := report.New("svc@1.0.0",
		[]model.Violation{
			{RuleID: "no-db", Severity: model.SeverityError, File: "a.ts", Message: "bad", SourceFile: "standards/s.yaml"},
			{RuleID: "obs", Severity: model.SeverityWarn, File: "b.ts", Message: "advisory", LLMDerived: true, Model: "m"},
		},
		[]model.Note{{RuleID: "obs", Reason: "skipped: LLM unavailable"}},
	)
	out, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Profile    string `json:"profile"`
		Violations []struct {
			RuleID     string `json:"ruleId"`
			Severity   string `json:"severity"`
			File       string `json:"file"`
			Message    string `json:"message"`
			SourceFile string `json:"sourceFile"`
			LLMDerived bool   `json:"llmDerived"`
			Model      string `json:"model"`
		} `json:"violations"`
		Notes []struct {
			Reason string `json:"reason"`
		} `json:"notes"`
		Summary struct {
			Errors      int  `json:"errors"`
			Warnings    int  `json:"warnings"`
			Skipped     int  `json:"skipped"`
			GateFailing bool `json:"gateFailing"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("json not parseable to documented schema: %v", err)
	}
	if parsed.Profile != "svc@1.0.0" || len(parsed.Violations) != 2 || len(parsed.Notes) != 1 {
		t.Fatalf("unexpected structure: %+v", parsed)
	}
	if parsed.Summary.Errors != 1 || parsed.Summary.Warnings != 1 || !parsed.Summary.GateFailing {
		t.Fatalf("unexpected summary: %+v", parsed.Summary)
	}
	if !parsed.Violations[1].LLMDerived || parsed.Violations[1].Model != "m" {
		t.Fatalf("semantic finding not labeled in JSON: %+v", parsed.Violations[1])
	}
}

func TestGatePassesWithWarnOnly(t *testing.T) {
	r := report.New("p", []model.Violation{{RuleID: "x", Severity: model.SeverityWarn, File: "f", Message: "m"}}, nil)
	if r.Summary.GateFailing {
		t.Fatal("warn-only should not fail the gate")
	}
}
