package cli

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/archrail/archrail/internal/exitcode"
	"github.com/archrail/archrail/internal/project"
	"github.com/archrail/archrail/internal/sync"
)

func runSync(env Env, args []string) exitcode.Code {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	checkMode := fs.Bool("check", false, "verify the generated file is up to date without writing")
	adapterName := fs.String("adapter", sync.DefaultAdapter, "sync adapter to use")
	if err := fs.Parse(args); err != nil {
		return exitcode.Operational
	}

	p, err := project.Discover(env.Dir)
	if err != nil {
		return fail(env, err)
	}
	// Resolve gates on profile resolution + validation; refuse to render if invalid.
	resolved, err := p.Resolve(buildRegistry())
	if err != nil {
		return fail(env, err)
	}

	reg := sync.NewRegistry()
	adapter, ok := reg.Get(*adapterName)
	if !ok {
		return fail(env, fmt.Errorf("unknown adapter %q (available: %v)", *adapterName, reg.Names()))
	}

	rendered := adapter.Render(sync.RenderInput{
		ProfileName:        resolved.Profile.Name,
		ProfileVersion:     resolved.Profile.Version,
		ProfileDescription: resolved.Profile.Description,
		Rules:              resolved.Profile.Rules,
	})
	outPath := filepath.Join(p.Root, adapter.FileName())

	if *checkMode {
		existing, err := os.ReadFile(outPath)
		if err != nil || !bytes.Equal(existing, []byte(rendered)) {
			fmt.Fprintf(env.Stderr, "%s is stale; run `archrail sync` to regenerate.\n", adapter.FileName())
			return exitcode.Operational
		}
		fmt.Fprintf(env.Stdout, "%s is up to date.\n", adapter.FileName())
		return exitcode.Success
	}

	if err := os.WriteFile(outPath, []byte(rendered), 0o644); err != nil {
		return fail(env, err)
	}
	fmt.Fprintf(env.Stdout, "Wrote %s (profile %s@%s)\n", adapter.FileName(), resolved.Profile.Name, resolved.Profile.Version)
	return exitcode.Success
}
