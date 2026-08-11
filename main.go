// Command wow1 is a self-contained per-user TODO list web app with OIDC
// login. Templates, static assets, and migrations are embedded into the
// binary.
package main

import (
	"embed"
	"fmt"
	"os"

	"wow1/internal/cli"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

//go:embed migrations
var migrationsFS embed.FS

func main() {
	root := cli.NewRootCmd(templatesFS, staticFS, migrationsFS)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
