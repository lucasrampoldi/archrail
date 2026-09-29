package catalog_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/archrail/archrail/internal/catalog"
)

func writeProfile(t *testing.T, root, name, version, body string) {
	t.Helper()
	dir := filepath.Join(root, "profiles", name, version)
	if err := os.MkdirAll(filepath.Join(dir, "standards"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListAndLoad(t *testing.T) {
	root := t.TempDir()
	writeProfile(t, root, "a", "1.0.0", "name: a\nversion: \"1.0.0\"\ndescription: first\n")
	writeProfile(t, root, "b", "2.0.0", "name: b\nversion: \"2.0.0\"\ndescription: second\n")

	src := catalog.NewLocal(root)
	ids, err := src.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0].Name != "a" || ids[1].Name != "b" {
		t.Fatalf("unexpected list: %+v", ids)
	}

	p, err := src.Load("a", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if p.Description != "first" {
		t.Fatalf("wrong profile loaded: %+v", p)
	}
}

func TestProfileNotFound(t *testing.T) {
	root := t.TempDir()
	writeProfile(t, root, "a", "1.0.0", "name: a\nversion: \"1.0.0\"\n")
	_, err := catalog.NewLocal(root).Load("missing", "1.0.0")
	if !errors.Is(err, catalog.ErrProfileNotFound) {
		t.Fatalf("want ErrProfileNotFound, got %v", err)
	}
}

func TestVersionNotFound(t *testing.T) {
	root := t.TempDir()
	writeProfile(t, root, "a", "1.0.0", "name: a\nversion: \"1.0.0\"\n")
	_, err := catalog.NewLocal(root).Load("a", "9.9.9")
	if !errors.Is(err, catalog.ErrVersionNotFound) {
		t.Fatalf("want ErrVersionNotFound, got %v", err)
	}
}

func TestMissingCatalogPath(t *testing.T) {
	_, err := catalog.NewLocal(filepath.Join(t.TempDir(), "does-not-exist")).List()
	if err == nil {
		t.Fatal("expected error for missing catalog path")
	}
}

func TestDuplicateProfile(t *testing.T) {
	root := t.TempDir()
	// Two directories declaring the same name+version.
	writeProfile(t, root, "a", "1.0.0", "name: dup\nversion: \"1.0.0\"\n")
	writeProfile(t, root, "b", "1.0.0", "name: dup\nversion: \"1.0.0\"\n")
	_, err := catalog.NewLocal(root).List()
	if err == nil {
		t.Fatal("expected duplicate-profile error")
	}
}
