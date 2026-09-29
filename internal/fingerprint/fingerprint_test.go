package fingerprint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/archrail/archrail/internal/engine"
	"github.com/archrail/archrail/internal/extract"
	"github.com/archrail/archrail/internal/fingerprint"
	"github.com/archrail/archrail/internal/model"
)

func build(t *testing.T) (*engine.Topology, *engine.Facts) {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "src/handlers/user.ts", "import { r } from '../repositories/userRepo';\n")
	write(t, dir, "src/repositories/userRepo.ts", "import express from 'express';\nexport const r = 1;\n")
	// A dependency that must be excluded from the fingerprint.
	write(t, dir, "node_modules/express/index.js", "import secret from './secret';\n")

	profile := &model.Profile{
		Components: []model.Component{
			{Name: "handlers", Paths: []string{"src/handlers/**"}},
			{Name: "repositories", Paths: []string{"src/repositories/**"}},
		},
		Layers: []model.Layer{
			{Name: "presentation", Components: []string{"handlers"}},
			{Name: "data", Components: []string{"repositories"}},
		},
	}
	topo, err := engine.BuildTopology(profile, &model.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sel, err := extract.BuildSelector(extract.DefaultRegistry(), []model.ExtractorSelection{{Name: "js", Extensions: []string{".ts", ".js"}}})
	if err != nil {
		t.Fatal(err)
	}
	facts, err := engine.BuildFacts(dir, topo, sel, engine.NewExcluder(nil))
	if err != nil {
		t.Fatal(err)
	}
	return topo, facts
}

func TestFingerprintDeterministic(t *testing.T) {
	topo, facts := build(t)
	a := fingerprint.Build(topo, facts).Render()
	b := fingerprint.Build(topo, facts).Render()
	if a != b {
		t.Fatalf("fingerprint not byte-stable:\n%s\n---\n%s", a, b)
	}
}

func TestFingerprintExcludesDependencies(t *testing.T) {
	topo, facts := build(t)
	out := fingerprint.Build(topo, facts).Render()
	if strings.Contains(out, "node_modules") {
		t.Errorf("fingerprint leaked node_modules:\n%s", out)
	}
	// The internal edge handlers -> repositories should be present.
	if !strings.Contains(out, "from: handlers") {
		t.Errorf("expected handlers edge in fingerprint:\n%s", out)
	}
	// express should be detected as an external framework dependency.
	if !strings.Contains(out, "express") {
		t.Errorf("expected express in stack/externalDeps:\n%s", out)
	}
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
