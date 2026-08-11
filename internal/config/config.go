// Package config holds the resolved application configuration. It has no
// knowledge of flags, environment variables, or viper - callers (the cli
// package) resolve values from those sources and build a Config via New.
package config

import (
	"fmt"
	"strings"
)

type Config struct {
	DatabaseURL      string
	ListenAddr       string
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	SessionSecret    string
}

// requiredField pairs a Config value with the flag/env name to report if
// it's missing.
type requiredField struct {
	value string
	name  string
}

// New validates the given values and returns a Config, or an error listing
// every missing required value.
func New(databaseURL, listenAddr, oidcIssuerURL, oidcClientID, oidcClientSecret, oidcRedirectURL, sessionSecret string) (*Config, error) {
	required := []requiredField{
		{databaseURL, "--database-url (WOW1_DATABASE_URL)"},
		{oidcIssuerURL, "--oidc-issuer-url (WOW1_OIDC_ISSUER_URL)"},
		{oidcClientID, "--oidc-client-id (WOW1_OIDC_CLIENT_ID)"},
		{oidcClientSecret, "--oidc-client-secret (WOW1_OIDC_CLIENT_SECRET)"},
		{oidcRedirectURL, "--oidc-redirect-url (WOW1_OIDC_REDIRECT_URL)"},
		{sessionSecret, "--session-secret (WOW1_SESSION_SECRET)"},
	}

	var missing []string
	for _, f := range required {
		if f.value == "" {
			missing = append(missing, f.name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return &Config{
		DatabaseURL:      databaseURL,
		ListenAddr:       listenAddr,
		OIDCIssuerURL:    oidcIssuerURL,
		OIDCClientID:     oidcClientID,
		OIDCClientSecret: oidcClientSecret,
		OIDCRedirectURL:  oidcRedirectURL,
		SessionSecret:    sessionSecret,
	}, nil
}
