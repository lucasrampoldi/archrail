package semantic_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/archrail/archrail/internal/extract"
	"github.com/archrail/archrail/internal/model"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func jsSelector(t *testing.T) *extract.Selector {
	t.Helper()
	sel, err := extract.BuildSelector(extract.DefaultRegistry(), []model.ExtractorSelection{
		{Name: "js", Extensions: []string{".ts", ".js"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sel
}

func mustParams(t *testing.T, m map[string]any) map[string]yaml.Node {
	t.Helper()
	out := map[string]yaml.Node{}
	for k, v := range m {
		b, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var n yaml.Node
		if err := yaml.Unmarshal(b, &n); err != nil {
			t.Fatal(err)
		}
		if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
			out[k] = *n.Content[0]
		} else {
			out[k] = n
		}
	}
	return out
}
