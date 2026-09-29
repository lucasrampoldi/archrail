package baseline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/archrail/archrail/internal/baseline"
	"github.com/archrail/archrail/internal/model"
)

func TestGenerateApplyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.yaml")

	v := model.Violation{RuleID: "r1", File: "a.ts", Message: "bad thing", Severity: model.SeverityError}
	other := model.Violation{RuleID: "r2", File: "b.ts", Message: "new thing", Severity: model.SeverityError}

	if err := baseline.Generate(path, []model.Violation{v}); err != nil {
		t.Fatal(err)
	}

	set, err := baseline.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	vs := []model.Violation{v, other}
	set.Apply(vs)

	if !vs[0].Baselined {
		t.Errorf("expected v to be baselined")
	}
	if vs[1].Baselined {
		t.Errorf("expected new violation NOT to be baselined")
	}
}

func TestMissingBaselineIsEmpty(t *testing.T) {
	set, err := baseline.Load(filepath.Join(t.TempDir(), "none.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	vs := []model.Violation{{RuleID: "r", File: "f", Message: "m"}}
	set.Apply(vs)
	if vs[0].Baselined {
		t.Error("nothing should be baselined for an absent baseline file")
	}
}

func TestDeterministicOutput(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "b1.yaml")
	p2 := filepath.Join(dir, "b2.yaml")
	vs := []model.Violation{
		{RuleID: "z", File: "z.ts", Message: "m1"},
		{RuleID: "a", File: "a.ts", Message: "m2"},
	}
	if err := baseline.Generate(p1, vs); err != nil {
		t.Fatal(err)
	}
	// Reverse order should produce identical file (sorted).
	if err := baseline.Generate(p2, []model.Violation{vs[1], vs[0]}); err != nil {
		t.Fatal(err)
	}
	b1 := readFile(t, p1)
	b2 := readFile(t, p2)
	if b1 != b2 {
		t.Errorf("baseline output not deterministic:\n%s\n---\n%s", b1, b2)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
