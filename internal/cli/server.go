package cli

import (
	"embed"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"wow1/internal/config"
	"wow1/internal/server"
)

// flagNames lists every server flag, in the order they should be reported
// when missing.
var flagNames = []string{
	"database-url",
	"listen-addr",
	"oidc-issuer-url",
	"oidc-client-id",
	"oidc-client-secret",
	"oidc-redirect-url",
	"session-secret",
}

func newServerCmd(templatesFS, staticFS embed.FS) *cobra.Command {
	v := viper.New()
	v.SetEnvPrefix("WOW1")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the wow1 web server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.New(
				v.GetString("database-url"),
				v.GetString("listen-addr"),
				v.GetString("oidc-issuer-url"),
				v.GetString("oidc-client-id"),
				v.GetString("oidc-client-secret"),
				v.GetString("oidc-redirect-url"),
				v.GetString("session-secret"),
			)
			if err != nil {
				return err
			}
			return server.Run(cfg, templatesFS, staticFS)
		},
	}

	flags := cmd.Flags()
	flags.String("database-url", "", "PostgreSQL connection string (required, env WOW1_DATABASE_URL)")
	flags.String("listen-addr", ":8080", "Address to listen on (env WOW1_LISTEN_ADDR)")
	flags.String("oidc-issuer-url", "", "OIDC issuer URL (required, env WOW1_OIDC_ISSUER_URL)")
	flags.String("oidc-client-id", "", "OIDC client ID (required, env WOW1_OIDC_CLIENT_ID)")
	flags.String("oidc-client-secret", "", "OIDC client secret (required, env WOW1_OIDC_CLIENT_SECRET)")
	flags.String("oidc-redirect-url", "", "OIDC redirect URL (required, env WOW1_OIDC_REDIRECT_URL)")
	flags.String("session-secret", "", "Secret used to sign session cookies (required, env WOW1_SESSION_SECRET)")

	for _, name := range flagNames {
		_ = v.BindPFlag(name, flags.Lookup(name))
	}

	return cmd
}
