package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the wow1 build version. Overridden at build time via
// -ldflags "-X wow1/internal/cli.Version=...".
var Version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the wow1 version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), Version)
			return nil
		},
	}
}
