package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/archrail/archrail/internal/baseline"
	"github.com/archrail/archrail/internal/engine"
	"github.com/archrail/archrail/internal/exitcode"
	"github.com/archrail/archrail/internal/fingerprint"
	"github.com/archrail/archrail/internal/git"
	"github.com/archrail/archrail/internal/project"
	"github.com/archrail/archrail/internal/report"
	"github.com/archrail/archrail/internal/semantic"
)

func runCheck(env Env, args []string) exitcode.Code {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	base := fs.String("base", "", "restrict evaluation to files changed vs this git ref")
	format := fs.String("format", "human", "output format: human|json")
	staged := fs.Bool("staged", false, "restrict evaluation to staged files (for pre-commit)")
	requireSemantic := fs.Bool("require-semantic", false, "fail if semantic evaluation cannot run")
	noSemantic := fs.Bool("no-semantic", false, "skip semantic (LLM) rules entirely")
	updateBaseline := fs.Bool("update-baseline", false, "write current violations to the baseline and exit")
	if err := fs.Parse(args); err != nil {
		return exitcode.Operational
	}
	if *format != "human" && *format != "json" {
		return fail(env, fmt.Errorf("invalid --format %q (want human|json)", *format))
	}

	p, err := project.Discover(env.Dir)
	if err != nil {
		return fail(env, err)
	}
	resolved, err := p.Resolve(buildRegistry())
	if err != nil {
		return fail(env, err)
	}
	facts, err := resolved.BuildFacts()
	if err != nil {
		return fail(env, err)
	}

	// Diff-scoping: no silent fallback when git/ref is unavailable.
	var subjects map[string]bool
	switch {
	case *staged:
		changed, err := git.StagedFiles(p.Root)
		if err != nil {
			return fail(env, err)
		}
		subjects = toSet(changed)
	case *base != "":
		changed, err := git.ChangedFiles(p.Root, *base)
		if err != nil {
			return fail(env, err)
		}
		subjects = toSet(changed)
	}

	sem := buildSemanticContext(resolved, facts, *noSemantic, *requireSemantic)

	res, err := engine.Evaluate(resolved.Profile.Rules, resolved.Registry, resolved.Topology, facts, resolved.Defaults, subjects, sem)
	if err != nil {
		return fail(env, err)
	}

	if *updateBaseline {
		if err := baseline.Generate(p.BaselinePath(), res.Violations); err != nil {
			return fail(env, err)
		}
		fmt.Fprintf(env.Stdout, "Wrote baseline with %d violation(s) to %s\n", len(res.Violations), p.BaselinePath())
		return exitcode.Success
	}

	bl, err := baseline.Load(p.BaselinePath())
	if err != nil {
		return fail(env, err)
	}
	bl.Apply(res.Violations)

	rep := report.New(resolved.Profile.Name+"@"+resolved.Profile.Version, res.Violations, res.Notes)
	if *format == "json" {
		out, err := rep.JSON()
		if err != nil {
			return fail(env, err)
		}
		fmt.Fprintln(env.Stdout, string(out))
	} else {
		fmt.Fprint(env.Stdout, rep.Human())
	}

	if rep.Summary.GateFailing {
		return exitcode.Violations
	}
	return exitcode.Success
}

// buildSemanticContext wires the LLM evaluator based on config, env, and flags.
func buildSemanticContext(resolved *project.Resolved, facts *engine.Facts, noSemantic, require bool) engine.SemanticContext {
	if noSemantic || !hasSemanticRules(resolved) {
		return engine.SemanticContext{}
	}
	fp := fingerprint.Build(resolved.Topology, facts).Render()
	sc := engine.SemanticContext{Require: require, Fingerprint: fp}

	key := os.Getenv(semantic.EnvAPIKey)
	if key == "" {
		// Unavailable: rules skip (or, with --require-semantic, error at eval).
		return sc
	}
	gw := semantic.NewGateway(key, resolved.Project.Config.Semantic.Model, resolved.Project.Config.Semantic.BaseURL)
	sc.Available = true
	sc.Backend = gw
	sc.Model = gw.Model()
	return sc
}

func hasSemanticRules(resolved *project.Resolved) bool {
	for _, r := range resolved.Profile.Rules {
		if r.Type == "semantic" {
			return true
		}
	}
	return false
}

func toSet(paths []string) map[string]bool {
	m := map[string]bool{}
	for _, p := range paths {
		m[p] = true
	}
	return m
}
