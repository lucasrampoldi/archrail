// Package exitcode defines Archrail's documented process exit codes.
//
// The exit-code contract is part of Archrail's public integration surface:
// pre-commit hooks and CI gates depend only on these values (and the JSON
// report). Keep them stable.
package exitcode

// Code is a documented Archrail exit code.
type Code int

const (
	// Success indicates a clean run: no gate-failing violations.
	Success Code = 0
	// Violations indicates at least one non-baselined error violation was found.
	Violations Code = 1
	// Operational indicates an operational error: invalid standards, missing
	// Git/base ref, unreadable catalog, bad usage, etc. Never used to signal
	// "code did not conform".
	Operational Code = 2
)

// Int returns the integer value of the exit code.
func (c Code) Int() int { return int(c) }
