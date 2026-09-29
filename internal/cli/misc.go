package cli

import (
	"flag"
	"fmt"

	"github.com/archrail/archrail/internal/catalog"
	"github.com/archrail/archrail/internal/engine"
	"github.com/archrail/archrail/internal/exitcode"
	"github.com/archrail/archrail/internal/project"
	"github.com/archrail/archrail/internal/semantic"
)

// buildRegistry returns the rule-type registry with the semantic type included.
func buildRegistry() *engine.Registry {
	reg := engine.NewRegistry()
	reg.Register(semantic.New())
	return reg
}

func runRuleTypes(env Env, args []string) exitcode.Code {
	fs := flag.NewFlagSet("rule-types", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitcode.Operational
	}
	fmt.Fprint(env.Stdout, buildRegistry().Describe())
	return exitcode.Success
}

func runProfiles(env Env, args []string) exitcode.Code {
	fs := flag.NewFlagSet("profiles", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	catalogPath := fs.String("catalog", "", "override the catalog path to list")
	if err := fs.Parse(args); err != nil {
		return exitcode.Operational
	}

	path := *catalogPath
	if path == "" {
		p, err := project.Discover(env.Dir)
		if err != nil {
			return fail(env, err)
		}
		path = p.CatalogPath()
	}

	ids, err := catalog.NewLocal(path).List()
	if err != nil {
		return fail(env, err)
	}
	if len(ids) == 0 {
		fmt.Fprintln(env.Stdout, "No profiles found in catalog.")
		return exitcode.Success
	}
	fmt.Fprintf(env.Stdout, "Profiles in %s:\n", path)
	for _, id := range ids {
		fmt.Fprintf(env.Stdout, "  %s@%s — %s\n", id.Name, id.Version, id.Description)
	}
	return exitcode.Success
}
