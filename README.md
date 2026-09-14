# goforge

An interactive CLI (à la `create-t3-app` / the TanStack CLI) that scaffolds a
new Go HTTP project: pick a framework, database, ORM, Redis, Docker, and an
auth setup, and it generates a real, working project on disk.

## Install / run

```bash
go run .          # from this directory, or
go build -o goforge . && ./goforge
```

It'll walk you through:

1. Project name + module path
2. HTTP framework — Gin, Fiber, Echo, or stdlib `net/http` (Go 1.22 `ServeMux`)
3. Database — Postgres, MySQL, SQLite, MongoDB, or none
4. ORM (if applicable) — Bun, GORM, sqlx, or raw `database/sql`
5. Redis — yes/no
6. Dockerfile + docker-compose — yes/no
7. Auth — yes/no; if yes: **JWT** or **server-side sessions**, plus optional
   OAuth (Google/GitHub/GitLab/Discord via `markbates/goth`)
8. Logger (slog/zerolog/zap), env loading (godotenv/viper), Makefile, CI
   workflow, `git init`

Then it writes out a complete project — `cmd/server`, `internal/config`,
`internal/router`, `internal/db`, `internal/cache`, `internal/auth` — wired
together and ready to `go mod tidy && go run ./cmd/server`.

## Why not a "better-auth for Go"?

You asked for this specifically, so: as of writing there are a few very new
(days-to-months old, effectively unproven — zero-import) ports like
`go-better-auth`/`gobetterauth` chasing the same idea as the TS `better-auth`
library, plus more established but heavier options like `authboss` or
`ory/kratos`. None of them are the obvious, boring, stable choice yet, and a
half-adopted external auth framework is a worse starting point than code you
own outright. So `goforge` generates a small, self-contained auth module
instead — plain JWT or session logic you can actually read in five minutes,
built to be extended (see `internal/auth/core.go`'s `Claims`/`Session` types)
rather than configured. If one of those libraries matures, swapping it in is
a contained job because `internal/auth` is the only thing that talks to it.

## How auth extensibility works

- **JWT**: `Claims` embeds `jwt.RegisteredClaims` plus an `Extra map[string]any`.
  Add typed fields straight to the struct for anything you always want
  (`Role string`, `TenantID string`...), or stash ad-hoc values via
  `claims.Set("scope", "admin")` — both flow through issuing and parsing
  automatically.
- **Session**: `Session.Data` is the same idea — a free-form bag you read/write
  with `Get`/`Set`, backed by an in-memory store by default, or Redis
  automatically if you enabled it.
- **OAuth**: one `internal/auth/oauth.go`, framework-adapted in each
  `internal/auth/<framework>.go`. Adding a provider is: import it, add one
  `goth.UseProviders(...)` line, add its client id/secret to config — same
  mechanism regardless of which framework or auth type you picked.

## Project layout (this tool, not what it generates)

```
main.go                     wires prompt -> generator, prints next steps
internal/config             config.Project — the answers to the wizard
internal/prompt             the huh-based interactive wizard
internal/generator          embeds templates/, renders + writes the project
internal/generator/templates/
  common/                   go.mod, main.go, config, logger, README, CI, Makefile
  router/{gin,fiber,echo,nethttp}.go.tmpl
  db/{postgres,mysql,sqlite}_{bun,gorm,sqlx,database-sql}.go.tmpl, mongodb.go.tmpl
  redis/cache.go.tmpl
  auth/jwt/                 core + per-framework routes/middleware
  auth/session/             core + stores (memory/redis) + per-framework routes/middleware
  auth/oauth/core.go.tmpl   shared goth wiring, used by both auth types
  docker/                   Dockerfile, docker-compose.yml
```

Each template is self-guarded with `{{if ...}}` on the relevant `Project`
field, and `generator.Generate` skips writing anything that renders blank —
so adding a ninth framework or a fifth database is additive, not a rewrite.

## Tests

`go test ./internal/generator/...` renders all ~560 framework × db × orm ×
redis × auth × oauth combinations and gofmt-checks every generated `.go`
file, so a broken template combination fails CI instead of showing up as a
broken `git clone` for whoever runs the CLI.
