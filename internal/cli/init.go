package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/archrail/archrail/internal/exitcode"
	"github.com/archrail/archrail/internal/project"
)

func runInit(env Env, args []string) exitcode.Code {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	catalogPath := fs.String("catalog", "", "path to the shared profile catalog (clone/submodule/vendored)")
	profileName := fs.String("profile", "", "name of the profile to adopt")
	profileVer := fs.String("profile-version", "", "version of the profile to adopt")
	force := fs.Bool("force", false, "overwrite an existing .archrail/ directory")
	if err := fs.Parse(args); err != nil {
		return exitcode.Operational
	}

	dir := filepath.Join(env.Dir, project.DirName)
	if _, err := os.Stat(dir); err == nil && !*force {
		return fail(env, fmt.Errorf("%s already exists; pass --force to overwrite", project.DirName))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail(env, err)
	}

	cat := *catalogPath
	todo := ""
	if cat == "" {
		cat = "../archrail-catalog"
		todo = "# TODO: set catalog.path, profile.name and profile.version below.\n"
	}
	name := *profileName
	if name == "" {
		name = "REPLACE_ME"
	}
	ver := *profileVer
	if ver == "" {
		ver = "0.0.0"
	}

	content := fmt.Sprintf(`version: 1
%s
# Where approved architecture profiles are resolved from.
catalog:
  source: local
  path: %s

# The approved architecture this project adopts (pinned by name + version).
profile:
  name: %s
  version: "%s"

# Optional: bind the profile's abstract components to this repo's paths,
# or override topology locally.
# components:
#   - name: handlers
#     paths: ["src/handlers/**"]

# Optional: extra paths to exclude from scanning and from any LLM context.
# exclude:
#   - "generated/**"

# Optional: semantic (LLM) evaluation settings. The API key is read only from
# the %s environment variable and is never stored here.
# semantic:
#   model: anthropic/claude-sonnet-4-6
`, todo, cat, name, ver, "AI_GATEWAY_API_KEY")

	cfgPath := filepath.Join(dir, project.ConfigName)
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		return fail(env, err)
	}
	fmt.Fprintf(env.Stdout, "Created %s\n", cfgPath)
	fmt.Fprintln(env.Stdout, "Next: set the catalog path and adopt a profile, then run `archrail sync` and `archrail check`.")
	return exitcode.Success
}
