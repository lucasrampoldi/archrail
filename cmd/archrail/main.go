// Command archrail is the architecture conformance CLI.
package main

import (
	"os"

	"github.com/archrail/archrail/internal/cli"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	code := cli.Run(cli.Env{
		Args:    os.Args[1:],
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Dir:     dir,
		Version: version,
	})
	os.Exit(code.Int())
}
