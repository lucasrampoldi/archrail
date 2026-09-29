package semantic_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/archrail/archrail/internal/engine"
	"github.com/archrail/archrail/internal/model"
	"github.com/archrail/archrail/internal/semantic"
)

// fakeBackend is a controllable SemanticBackend for tests.
type fakeBackend struct {
	verdict engine.SemanticVerdict
	err     error
	lastReq engine.SemanticRequest
}

func (f *fakeBackend) Evaluate(req engine.SemanticRequest) (engine.SemanticVerdict, error) {
	f.lastReq = req
	return f.verdict, f.err
}

func baseInput(t *testing.T, sem engine.SemanticContext, severity model.Severity) engine.EvalInput {
	t.Helper()
	topo, err := engine.BuildTopology(&model.Profile{
		Components: []model.Component{{Name: "handlers", Paths: []string{"src/handlers/**"}}},
	}, &model.Config{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFile(t, dir, "src/handlers/user.ts", "export const x = 1;")
	facts, err := engine.BuildFacts(dir, topo, jsSelector(t), engine.NewExcluder(nil))
	if err != nil {
		t.Fatal(err)
	}
	rule := model.Rule{
		ID: "obs", Type: "semantic", Severity: severity, SourceFile: "standards/s.yaml",
		Params: mustParams(t, map[string]any{
			"assertion": "handlers use the approved observability abstraction",
			"scope":     "handlers",
		}),
	}
	return engine.EvalInput{
		Rule: rule, Topo: topo, Facts: facts,
		DefaultSeverity: model.SeverityError, Semantic: sem,
	}
}

func TestSemanticAdvisoryByDefault(t *testing.T) {
	be := &fakeBackend{verdict: engine.SemanticVerdict{Conforms: false, Findings: []engine.SemanticFinding{{File: "src/handlers/user.ts", Message: "no tracer"}}}}
	in := baseInput(t, engine.SemanticContext{Available: true, Backend: be, Model: "m"}, model.SeverityUnset)
	vs, notes, err := semantic.New().Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("unexpected notes: %v", notes)
	}
	if len(vs) != 1 || vs[0].Severity != model.SeverityWarn {
		t.Fatalf("want 1 warn violation, got %+v", vs)
	}
	if !vs[0].LLMDerived || vs[0].Model != "m" {
		t.Fatalf("finding not labeled LLM-derived: %+v", vs[0])
	}
}

func TestSemanticConfigurableError(t *testing.T) {
	be := &fakeBackend{verdict: engine.SemanticVerdict{Conforms: false, Findings: []engine.SemanticFinding{{Message: "bad"}}}}
	in := baseInput(t, engine.SemanticContext{Available: true, Backend: be, Model: "m"}, model.SeverityError)
	vs, _, err := semantic.New().Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].Severity != model.SeverityError {
		t.Fatalf("want 1 error violation, got %+v", vs)
	}
}

func TestSemanticGracefulDegradation(t *testing.T) {
	in := baseInput(t, engine.SemanticContext{Available: false}, model.SeverityUnset)
	vs, notes, err := semantic.New().Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Fatalf("expected no violations when unavailable, got %+v", vs)
	}
	if len(notes) != 1 || !strings.Contains(notes[0].Reason, "LLM unavailable") {
		t.Fatalf("want skip note, got %+v", notes)
	}
}

func TestSemanticRequireErrorsWhenUnavailable(t *testing.T) {
	in := baseInput(t, engine.SemanticContext{Available: false, Require: true}, model.SeverityUnset)
	_, _, err := semantic.New().Evaluate(in)
	if err == nil {
		t.Fatal("expected error when semantic required but unavailable")
	}
}

func TestSemanticBackendErrorDegrades(t *testing.T) {
	be := &fakeBackend{err: errors.New("timeout")}
	in := baseInput(t, engine.SemanticContext{Available: true, Backend: be}, model.SeverityUnset)
	vs, notes, err := semantic.New().Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 || len(notes) != 1 {
		t.Fatalf("want skip note on backend error, got vs=%v notes=%v", vs, notes)
	}
}

func TestSemanticConforms(t *testing.T) {
	be := &fakeBackend{verdict: engine.SemanticVerdict{Conforms: true}}
	in := baseInput(t, engine.SemanticContext{Available: true, Backend: be}, model.SeverityUnset)
	vs, notes, err := semantic.New().Evaluate(in)
	if err != nil || len(vs) != 0 || len(notes) != 0 {
		t.Fatalf("want clean pass, got vs=%v notes=%v err=%v", vs, notes, err)
	}
}
