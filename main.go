// Command wow1 is a self-contained per-user TODO list web app with OIDC
// login. Templates and static assets are embedded into the binary; the
// database schema is kept in sync via GORM AutoMigrate.
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

func main() {
	root := cli.NewRootCmd(templatesFS, staticFS)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
