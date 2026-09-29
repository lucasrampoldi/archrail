// Package cli implements the archrail command-line interface.
package cli

import (
	"fmt"
	"io"

	"github.com/archrail/archrail/internal/exitcode"
)

// Env carries process I/O and working directory so commands are testable.
type Env struct {
	Args    []string
	Stdout  io.Writer
	Stderr  io.Writer
	Dir     string
	Version string
}

const usage = `archrail — architecture conformance for AI-assisted development

Usage:
  archrail <command> [flags]

Commands:
  init         Scaffold a .archrail/ directory that adopts a profile
  sync         Render the adopted profile into ARCHRAIL.md (agent context)
  check        Validate the repository against the adopted profile
  profiles     List profiles available in the resolved catalog
  rule-types   List available rule types and their parameters

Run "archrail <command> --help" for command-specific flags.

Exit codes:
  0  success (no gate-failing violations)
  1  violations found (non-baselined error)
  2  operational error (invalid standards, missing git/base ref, bad usage)
`

// Run dispatches a command and returns the process exit code.
func Run(env Env) exitcode.Code {
	if len(env.Args) == 0 {
		fmt.Fprint(env.Stdout, usage)
		return exitcode.Success
	}
	cmd, rest := env.Args[0], env.Args[1:]
	switch cmd {
	case "-h", "--help", "help":
		fmt.Fprint(env.Stdout, usage)
		return exitcode.Success
	case "-v", "--version", "version":
		v := env.Version
		if v == "" {
			v = "dev"
		}
		fmt.Fprintf(env.Stdout, "archrail %s\n", v)
		return exitcode.Success
	case "init":
		return runInit(env, rest)
	case "sync":
		return runSync(env, rest)
	case "check":
		return runCheck(env, rest)
	case "profiles":
		return runProfiles(env, rest)
	case "rule-types":
		return runRuleTypes(env, rest)
	default:
		fmt.Fprintf(env.Stderr, "unknown command %q\n\n%s", cmd, usage)
		return exitcode.Operational
	}
}

// fail prints an operational error and returns the operational exit code.
func fail(env Env, err error) exitcode.Code {
	fmt.Fprintf(env.Stderr, "archrail: %v\n", err)
	return exitcode.Operational
}
