package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/archrail/archrail/internal/engine"
	"github.com/archrail/archrail/internal/extract"
	"github.com/archrail/archrail/internal/model"
)

func hexProfile() *model.Profile {
	return &model.Profile{
		Components: []model.Component{
			{Name: "handlers", Paths: []string{"src/handlers/**"}},
			{Name: "services", Paths: []string{"src/services/**"}},
			{Name: "repositories", Paths: []string{"src/repositories/**"}},
		},
		Layers: []model.Layer{
			{Name: "presentation", Components: []string{"handlers"}},
			{Name: "domain", Components: []string{"services"}},
			{Name: "data", Components: []string{"repositories"}},
		},
	}
}

func setup(t *testing.T, files map[string]string) (*engine.Topology, *engine.Facts) {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	topo, err := engine.BuildTopology(hexProfile(), &model.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sel, err := extract.BuildSelector(extract.DefaultRegistry(), []model.ExtractorSelection{
		{Name: "js", Extensions: []string{".ts", ".js"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	facts, err := engine.BuildFacts(dir, topo, sel, engine.NewExcluder(nil))
	if err != nil {
		t.Fatal(err)
	}
	return topo, facts
}

func rule(t *testing.T, id, typ string, params map[string]any) model.Rule {
	t.Helper()
	r := model.Rule{ID: id, Type: typ, SourceFile: "standards/s.yaml", Params: map[string]yaml.Node{}}
	for k, v := range params {
		b, _ := yaml.Marshal(v)
		var n yaml.Node
		_ = yaml.Unmarshal(b, &n)
		if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
			r.Params[k] = *n.Content[0]
		}
	}
	return r
}

func eval(t *testing.T, rules []model.Rule, topo *engine.Topology, facts *engine.Facts) *engine.Result {
	t.Helper()
	res, err := engine.Evaluate(rules, engine.NewRegistry(), topo, facts, model.SeverityError, nil, engine.SemanticContext{})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestForbiddenImport(t *testing.T) {
	topo, facts := setup(t, map[string]string{
		"src/handlers/user.ts":         "import {r} from '../repositories/userRepo';\n",
		"src/repositories/userRepo.ts": "export const r=1;\n",
	})
	r := rule(t, "no-db", "forbidden-import", map[string]any{"from": "handlers", "to": "repositories"})
	res := eval(t, []model.Rule{r}, topo, facts)
	if len(res.Violations) != 1 {
		t.Fatalf("want 1 violation, got %+v", res.Violations)
	}
}

func TestLayerBoundary(t *testing.T) {
	topo, facts := setup(t, map[string]string{
		"src/handlers/user.ts":         "import {r} from '../repositories/userRepo';\n",
		"src/repositories/userRepo.ts": "export const r=1;\n",
	})
	r := rule(t, "layers", "layer-boundary", map[string]any{
		"allow": []map[string]any{
			{"from": "presentation", "to": []string{"domain"}},
			{"from": "domain", "to": []string{"data"}},
		},
	})
	res := eval(t, []model.Rule{r}, topo, facts)
	if len(res.Violations) != 1 {
		t.Fatalf("want 1 layer violation, got %+v", res.Violations)
	}
}

func TestRequiredAndForbiddenDependency(t *testing.T) {
	topo, facts := setup(t, map[string]string{
		"package.json": `{"dependencies":{"express":"^4","lodash":"^4"}}`,
	})
	req := rule(t, "need-obs", "required-dependency", map[string]any{"manifest": "package.json", "dependency": "@company/obs"})
	forb := rule(t, "no-lodash", "forbidden-dependency", map[string]any{"manifest": "package.json", "dependency": "lodash"})
	res := eval(t, []model.Rule{req, forb}, topo, facts)
	if len(res.Violations) != 2 {
		t.Fatalf("want 2 dependency violations, got %+v", res.Violations)
	}
}

func TestRequiredFiles(t *testing.T) {
	topo, facts := setup(t, map[string]string{
		"src/handlers/user.ts": "export const x=1;\n",
	})
	r := rule(t, "skeleton", "required-files", map[string]any{"component": "handlers", "files": []string{"src/handlers/index.ts"}})
	res := eval(t, []model.Rule{r}, topo, facts)
	if len(res.Violations) != 1 {
		t.Fatalf("want 1 missing-file violation, got %+v", res.Violations)
	}
}

func TestDeterministicByteStability(t *testing.T) {
	topo, facts := setup(t, map[string]string{
		"src/handlers/a.ts":     "import {r} from '../repositories/x';\n",
		"src/handlers/b.ts":     "import {r} from '../repositories/y';\n",
		"src/repositories/x.ts": "export const r=1;\n",
		"src/repositories/y.ts": "export const r=1;\n",
	})
	r := rule(t, "no-db", "forbidden-import", map[string]any{"from": "handlers", "to": "repositories"})
	a, _ := yaml.Marshal(eval(t, []model.Rule{r}, topo, facts).Violations)
	b, _ := yaml.Marshal(eval(t, []model.Rule{r}, topo, facts).Violations)
	if string(a) != string(b) {
		t.Fatalf("evaluation not byte-stable:\n%s\n---\n%s", a, b)
	}
}

func TestSkipNoExtractor(t *testing.T) {
	// A .py file with no configured extractor => skip note, not a violation.
	topo, facts := setup(t, map[string]string{
		"src/handlers/user.py": "from repositories import x\n",
	})
	r := rule(t, "no-db", "forbidden-import", map[string]any{"from": "handlers", "to": "repositories"})
	res := eval(t, []model.Rule{r}, topo, facts)
	if len(res.Violations) != 0 {
		t.Fatalf("expected no violations for unextractable file, got %+v", res.Violations)
	}
	if len(res.Notes) != 1 {
		t.Fatalf("expected a skip note, got %+v", res.Notes)
	}
}

func TestExtractorImports(t *testing.T) {
	js := extract.JS{}
	got := js.Imports([]byte("import a from 'x';\nconst b = require('y');\n"))
	if len(got) != 2 {
		t.Fatalf("want 2 imports, got %v", got)
	}
}
