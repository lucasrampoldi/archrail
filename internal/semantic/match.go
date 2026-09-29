package semantic

import "github.com/bmatcuk/doublestar/v4"

func matchPath(pattern, path string) (bool, error) {
	return doublestar.Match(pattern, path)
}
