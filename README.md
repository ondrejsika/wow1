# wow1

A minimal, self-contained per-user TODO list web app in Go, with login via
any OpenID Connect provider. Server-rendered HTML + [htmx](https://htmx.org)
(vendored locally), PostgreSQL via `pgx/v5`, no web framework, no ORM,
no build step for the frontend. Templates, static assets, and database
migrations are compiled into a single binary via `embed.FS`.

## Run with Docker Compose

```sh
cp .env.example .env   # fill in OIDC_* values, see below
docker compose up --build
```

The app will be at http://localhost:8080. Postgres 17 runs alongside it
and migrations are applied automatically on startup.

## Run locally (Go + your own Postgres)

```sh
docker compose up -d db   # or point --database-url at any Postgres 17

go run . server \
  --database-url="postgres://wow1:wow1@localhost:5432/wow1?sslmode=disable" \
  --listen-addr=":8080" \
  --oidc-issuer-url="https://your-issuer.example.com" \
  --oidc-client-id="..." \
  --oidc-client-secret="..." \
  --oidc-redirect-url="http://localhost:8080/auth/callback" \
  --session-secret="some-long-random-string"
```

Every flag can also be set via an environment variable instead (see below);
`wow1 server --help` lists all flags.

## Build a static binary

```sh
CGO_ENABLED=0 go build -ldflags="-s -w" -o wow1 .
./wow1 server --help
./wow1 version
```

The result is a single, statically-linked, self-contained binary (no
templates/static/migrations directories needed alongside it at runtime).

## CLI and configuration

wow1 is a [cobra](https://github.com/spf13/cobra) CLI with two
subcommands:

- `wow1 server` — starts the web app
- `wow1 version` — prints the build version

Configuration for `server` is resolved with
[viper](https://github.com/spf13/viper): every flag can also be set via an
environment variable prefixed `WOW1_` (dashes become underscores), and
precedence is **flag > env var > default**. There are no config files.

| Flag                   | Env var                   | Required | Default   |
|-------------------------|----------------------------|----------|-----------|
| `--database-url`         | `WOW1_DATABASE_URL`         | yes      | —         |
| `--listen-addr`          | `WOW1_LISTEN_ADDR`          | no       | `:8080`   |
| `--oidc-issuer-url`      | `WOW1_OIDC_ISSUER_URL`      | yes      | —         |
| `--oidc-client-id`       | `WOW1_OIDC_CLIENT_ID`       | yes      | —         |
| `--oidc-client-secret`   | `WOW1_OIDC_CLIENT_SECRET`   | yes      | —         |
| `--oidc-redirect-url`    | `WOW1_OIDC_REDIRECT_URL`    | yes      | —         |
| `--session-secret`       | `WOW1_SESSION_SECRET`       | yes      | —         |

`--database-url` is a Postgres connection string, e.g.
`postgres://user:pass@host:5432/db?sslmode=disable`. `--oidc-issuer-url` is
the OIDC issuer base URL (discovery via
`/.well-known/openid-configuration`). `--session-secret` is a random secret
used to sign the session cookie (HMAC).

Missing required values fail fast on startup with a single error listing
everything that's missing, e.g.:

```
Error: missing required configuration: --database-url (WOW1_DATABASE_URL), --oidc-issuer-url (WOW1_OIDC_ISSUER_URL), ...
```

## Testing against an OIDC provider

Any standards-compliant OIDC provider works, as long as it exposes
discovery at `<OIDC_ISSUER_URL>/.well-known/openid-configuration` and the
client is configured with `OIDC_REDIRECT_URL` as an allowed redirect URI.
The app requests the `openid profile email` scopes and reads the `email`
and `name` claims from the ID token.

**Google**
- Create an OAuth 2.0 Client ID (Web application) in the Google Cloud
  Console, add `http://localhost:8080/auth/callback` as an authorized
  redirect URI.
- `OIDC_ISSUER_URL=https://accounts.google.com`

**Keycloak**
- Create a client in your realm with a confidential access type and
  `http://localhost:8080/auth/callback` as a valid redirect URI.
- `OIDC_ISSUER_URL=http://<keycloak-host>/realms/<realm>`

**Zitadel**
- Create a Web application with the authorization code flow and
  `http://localhost:8080/auth/callback` as a redirect URI.
- `OIDC_ISSUER_URL=https://<your-instance>.zitadel.cloud`

## How it works

- `main.go` embeds `templates/`, `static/`, and `migrations/` and hands
  them to `internal/cli`.
- `internal/cli` defines the cobra commands (`server`, `version`) and, for
  `server`, binds flags to viper with the `WOW1_` env prefix, then builds
  a `config.Config`. Nothing outside `internal/cli` touches viper.
- `internal/config` is a plain struct plus validation — it has no
  knowledge of flags or env vars.
- `internal/server` wires up the database pool, migrations, OIDC, and
  routing from a `config.Config`, and runs the HTTP server.
- `internal/store` holds `pgxpool`-backed queries and runs migrations via
  `golang-migrate` (iofs source, reading from the embedded FS).
- `internal/session` implements a signed, stateless session cookie (no
  session store) using HMAC-SHA256 over `SESSION_SECRET`.
- `internal/oidcauth` implements the OIDC authorization code flow with
  state and nonce verification, and upserts the user on login.
- `internal/handlers` implements the task CRUD handlers. Every query is
  scoped to the session's `user_id`; a task that doesn't belong to the
  caller looks like it doesn't exist (404).
- All interactivity in `templates/` is done with htmx attributes
  (`hx-get`/`hx-post`/`hx-put`/`hx-delete` + out-of-band swaps for
  inserting new rows) — there is no custom JavaScript.
