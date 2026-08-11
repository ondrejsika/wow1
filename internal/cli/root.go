// Package cli implements the wow1 command-line interface: flag/env
// parsing via cobra and viper, and dispatch into internal/server.
package cli

import (
	"embed"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the "wow1" root command with its subcommands. The
// embedded filesystems are threaded through from main.go, since go:embed
// directives must live alongside the embedded directories.
func NewRootCmd(templatesFS, staticFS embed.FS) *cobra.Command {
	root := &cobra.Command{
		Use:           "wow1",
		Short:         "wow1 is a per-user TODO list app with OIDC login",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newServerCmd(templatesFS, staticFS))
	root.AddCommand(newVersionCmd())

	return root
}
