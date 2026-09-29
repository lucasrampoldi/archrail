// Package catalog resolves architecture profiles from a catalog source.
//
// The Source abstraction is the seam that lets additional catalog backends
// (e.g. remote/URL-based) be added later without changing the profile format
// or a project's adoption declaration. v0 ships only the local-path source.
package catalog

import "github.com/archrail/archrail/internal/model"

// ProfileID identifies a profile available in a catalog.
type ProfileID struct {
	Name        string
	Version     string
	Description string
}

// Source resolves profiles from some backing store.
type Source interface {
	// List enumerates every profile available in the catalog.
	List() ([]ProfileID, error)
	// Load resolves a single profile by exact name and version, including its
	// merged rule set.
	Load(name, version string) (*model.Profile, error)
}
