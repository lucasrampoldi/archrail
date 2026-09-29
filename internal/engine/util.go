package engine

import "github.com/bmatcuk/doublestar/v4"

// globMatch reports whether path matches the doublestar glob pattern.
func globMatch(pattern, path string) bool {
	ok, err := doublestar.Match(pattern, path)
	return err == nil && ok
}
