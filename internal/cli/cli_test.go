package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/archrail/archrail/internal/cli"
	"github.com/archrail/archrail/internal/exitcode"
)

// fixture builds a catalog + adopting repo and returns the repo path.
type fixture struct {
	repo    string
	catalog string
}

func newFixture(t *testing.T, rules string) *fixture {
	t.Helper()
	base := t.TempDir()
	catalog := filepath.Join(base, "catalog")
	repo := filepath.Join(base, "app")

	prof := filepath.Join(catalog, "profiles", "svc", "1.0.0")
	writeFile(t, filepath.Join(prof, "profile.yaml"), `name: svc
version: "1.0.0"
description: "test profile"
components:
  - name: handlers
    paths: ["src/handlers/**"]
  - name: services
    paths: ["src/services/**"]
  - name: repositories
    paths: ["src/repositories/**"]
layers:
  - name: presentation
    components: [handlers]
  - name: domain
    components: [services]
  - name: data
    components: [repositories]
extractors:
  - name: js
    extensions: [".ts"]
`)
	writeFile(t, filepath.Join(prof, "standards", "rules.yaml"), rules)

	writeFile(t, filepath.Join(repo, ".archrail", "archrail.yaml"), `version: 1
catalog:
  source: local
  path: `+catalog+`
profile:
  name: svc
  version: "1.0.0"
`)
	return &fixture{repo: repo, catalog: catalog}
}

func run(t *testing.T, dir string, args ...string) (exitcode.Code, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := cli.Run(cli.Env{Args: args, Stdout: &out, Stderr: &errb, Dir: dir})
	return code, out.String(), errb.String()
}

const forbidRule = `rules:
  - id: no-handler-db
    type: forbidden-import
    description: "handlers must not import repositories"
    from: handlers
    to: repositories
`

func TestCheckClean(t *testing.T) {
	f := newFixture(t, forbidRule)
	writeFile(t, filepath.Join(f.repo, "src/handlers/user.ts"), "export const x = 1;\n")
	code, out, _ := run(t, f.repo, "check")
	if code != exitcode.Success {
		t.Fatalf("want success, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "No architecture violations") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCheckErrorFailsGate(t *testing.T) {
	f := newFixture(t, forbidRule)
	writeFile(t, filepath.Join(f.repo, "src/handlers/user.ts"), "import {r} from '../repositories/x';\n")
	writeFile(t, filepath.Join(f.repo, "src/repositories/x.ts"), "export const r=1;\n")
	code, out, _ := run(t, f.repo, "check")
	if code != exitcode.Violations {
		t.Fatalf("want violations exit (1), got %d\n%s", code, out)
	}
}

func TestCheckWarnPassesGate(t *testing.T) {
	rules := `rules:
  - id: no-handler-db
    type: forbidden-import
    severity: warn
    description: "advisory"
    from: handlers
    to: repositories
`
	f := newFixture(t, rules)
	writeFile(t, filepath.Join(f.repo, "src/handlers/user.ts"), "import {r} from '../repositories/x';\n")
	writeFile(t, filepath.Join(f.repo, "src/repositories/x.ts"), "export const r=1;\n")
	code, out, _ := run(t, f.repo, "check")
	if code != exitcode.Success {
		t.Fatalf("warn-only should pass gate, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "warning(s)") {
		t.Fatalf("expected a warning in output: %s", out)
	}
}

func TestBaselineSuppresses(t *testing.T) {
	f := newFixture(t, forbidRule)
	writeFile(t, filepath.Join(f.repo, "src/handlers/user.ts"), "import {r} from '../repositories/x';\n")
	writeFile(t, filepath.Join(f.repo, "src/repositories/x.ts"), "export const r=1;\n")

	// Fails first.
	if code, _, _ := run(t, f.repo, "check"); code != exitcode.Violations {
		t.Fatal("expected initial failure")
	}
	// Record baseline.
	if code, _, _ := run(t, f.repo, "check", "--update-baseline"); code != exitcode.Success {
		t.Fatal("update-baseline should succeed")
	}
	// Now passes (suppressed).
	code, out, _ := run(t, f.repo, "check")
	if code != exitcode.Success {
		t.Fatalf("baselined violation should not fail gate, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "baselined") {
		t.Fatalf("expected baselined marker: %s", out)
	}
}

func TestSyncAndDriftCheck(t *testing.T) {
	f := newFixture(t, forbidRule)
	writeFile(t, filepath.Join(f.repo, "src/handlers/user.ts"), "export const x=1;\n")

	// Drift check before generating: stale (missing) => operational error.
	if code, _, _ := run(t, f.repo, "sync", "--check"); code != exitcode.Operational {
		t.Fatal("expected stale sync to be operational error")
	}
	if code, _, _ := run(t, f.repo, "sync"); code != exitcode.Success {
		t.Fatal("sync should succeed")
	}
	if code, _, _ := run(t, f.repo, "sync", "--check"); code != exitcode.Success {
		t.Fatal("sync --check should pass after generating")
	}
	// Deterministic: regenerating yields identical bytes.
	first, _ := os.ReadFile(filepath.Join(f.repo, "ARCHRAIL.md"))
	run(t, f.repo, "sync")
	second, _ := os.ReadFile(filepath.Join(f.repo, "ARCHRAIL.md"))
	if !bytes.Equal(first, second) {
		t.Fatal("sync output is not deterministic")
	}
}

func TestSemanticSkippedWhenNoKey(t *testing.T) {
	rules := `rules:
  - id: obs
    type: semantic
    description: "uses observability"
    assertion: "handlers use the approved observability abstraction"
    scope: handlers
`
	f := newFixture(t, rules)
	writeFile(t, filepath.Join(f.repo, "src/handlers/user.ts"), "export const x=1;\n")
	t.Setenv("AI_GATEWAY_API_KEY", "")
	code, out, _ := run(t, f.repo, "check")
	if code != exitcode.Success {
		t.Fatalf("semantic-only with no key should pass (skip), got %d\n%s", code, out)
	}
	if !strings.Contains(out, "LLM unavailable") {
		t.Fatalf("expected skip note: %s", out)
	}
}

func TestRequireSemanticErrorsWithoutKey(t *testing.T) {
	rules := `rules:
  - id: obs
    type: semantic
    description: "uses observability"
    assertion: "handlers use observability"
    scope: handlers
`
	f := newFixture(t, rules)
	writeFile(t, filepath.Join(f.repo, "src/handlers/user.ts"), "export const x=1;\n")
	t.Setenv("AI_GATEWAY_API_KEY", "")
	code, _, _ := run(t, f.repo, "check", "--require-semantic")
	if code != exitcode.Operational {
		t.Fatalf("--require-semantic without key should be operational error, got %d", code)
	}
}

func TestDiffScoping(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	f := newFixture(t, forbidRule)
	// Pre-existing violation in a committed file.
	writeFile(t, filepath.Join(f.repo, "src/handlers/old.ts"), "import {r} from '../repositories/x';\n")
	writeFile(t, filepath.Join(f.repo, "src/repositories/x.ts"), "export const r=1;\n")
	gitInit(t, f.repo)

	// Full check: fails (old violation present).
	if code, _, _ := run(t, f.repo, "check"); code != exitcode.Violations {
		t.Fatal("full check should fail on pre-existing violation")
	}
	// Diff check vs HEAD with a clean new file: passes (old file not a subject).
	writeFile(t, filepath.Join(f.repo, "src/handlers/new.ts"), "export const y=1;\n")
	code, out, errs := run(t, f.repo, "check", "--base", "HEAD")
	if code != exitcode.Success {
		t.Fatalf("diff check should pass (only new clean file in scope), got %d\n%s\n%s", code, out, errs)
	}
}

func TestDiffScopingMissingRefErrors(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	f := newFixture(t, forbidRule)
	writeFile(t, filepath.Join(f.repo, "src/handlers/user.ts"), "export const x=1;\n")
	gitInit(t, f.repo)
	code, _, _ := run(t, f.repo, "check", "--base", "no-such-ref")
	if code != exitcode.Operational {
		t.Fatalf("missing base ref must be operational error (no silent fallback), got %d", code)
	}
}

func gitInit(t *testing.T, repo string) {
	t.Helper()
	cmds := [][]string{
		{"init"},
		{"config", "user.email", "t@t.dev"},
		{"config", "user.name", "t"},
		{"add", "-A"},
		{"commit", "-m", "init"},
	}
	for _, c := range cmds {
		cmd := exec.Command("git", c...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", c, err, out)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
