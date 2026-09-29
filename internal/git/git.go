// Package git provides the minimal Git integration Archrail needs for
// diff-scoped checking. It shells out to the git binary and never falls back
// silently when git or a ref is unavailable.
package git

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrGitUnavailable indicates the git binary could not be found or run.
var ErrGitUnavailable = errors.New("git is not available")

// available reports whether the git binary is runnable.
func available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// ChangedFiles returns repo-relative, slash-normalized paths changed relative to
// baseRef (including working-tree changes). It errors rather than falling back
// when git or the ref is unavailable.
func ChangedFiles(root, baseRef string) ([]string, error) {
	if !available() {
		return nil, ErrGitUnavailable
	}
	if err := verifyRef(root, baseRef); err != nil {
		return nil, err
	}
	out, err := run(root, "diff", "--name-only", baseRef)
	if err != nil {
		return nil, err
	}
	// Include untracked files too, so newly added files are gated.
	untracked, err := run(root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return normalizeList(out + "\n" + untracked), nil
}

// StagedFiles returns repo-relative, slash-normalized staged file paths.
func StagedFiles(root string) ([]string, error) {
	if !available() {
		return nil, ErrGitUnavailable
	}
	out, err := run(root, "diff", "--name-only", "--cached")
	if err != nil {
		return nil, err
	}
	return normalizeList(out), nil
}

func verifyRef(root, ref string) error {
	if _, err := run(root, "rev-parse", "--verify", ref+"^{commit}"); err != nil {
		return fmt.Errorf("base ref %q is not available in this repository", ref)
	}
	return nil
}

func run(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

func normalizeList(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, filepath.ToSlash(line))
	}
	return out
}
