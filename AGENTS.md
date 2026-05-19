# FlowGate — Agent Guide

## Commands

```sh
make test                     # go test -race -count=1 ./...
make build                    # go build -o main.exe cmd/api/main.go
make watch                    # air for live reload
go test -v -run TestHandler ./internal/server/   # single test

# lint (uses golangci-lint v2 — local v1.x won't read config)
golangci-lint run ./...

# sqlc codegen (run after editing .sql query files)
cd internal/repository && sqlc generate && cd ../..

# migrations
goose -s -dir migrations postgres "$DATABASE_URL" up
goose -s -dir migrations create add_some_table sql
```

## Architecture

- **Single module**: `flowgate` (go 1.26.1)
- **Entrypoint**: `cmd/api/main.go` — wires pool in main(), passes to `server.NewServer(pool)`
- **HTTP router**: `github.com/julienschmidt/httprouter`
- **DB driver**: `pgx/v5` via `pgxpool`
- **Migrations**: goose, `migrations/` directory (11 files, numbered)
- **sqlc**: sources in `internal/repository/query/` (10 files), generated Go in `internal/repository/db/` (checked in)
- **Integration tests**: build tag `integration`, run with `go test -tags=integration ./integration/...`
- **Env loading**: `github.com/joho/godotenv/autoload` (blank import in `internal/server/server.go`)
- **Dashboard**: Next.js 15 in `dashboard/` (separate npm workflow)

## CI Pipelines

| Workflow | Trigger | What it runs |
|---|---|---|
| `lint.yml` | push/PR main | golangci-lint v2.12 matrix |
| `unit-testing.yml` | push/PR | `go test -race -count=1 ./...` |
| `integration-testing.yml` | nightly + manual | `go test -tags=integration ./integration/...` (Postgres service) |
| `codecov.yml` | push/PR | tests + coverage upload (60% threshold, 5% patch tolerance) |
| `security.yml` | push/PR | govulncheck + gosec |
| `trivy.yml` | push/PR | Docker build + Trivy HIGH/CRITICAL |
| `release.yml` | v* tags | goreleaser (6 platform cross-compile) |

## Linting Quirks

- **gofumpt v0.9.2** bundled with golangci-lint v2.12. Config uses `module-path: flowgate` (single-level module name — gofumpt inside golangci-lint won't auto-detect it).
- **sloglint**: `attr-only: true` — use `slog.Any("key", val)`, not raw `"key", val` pairs. Context must be scoped, messages lowercased.
- **revive**: all rules enabled. Config excludes magic-number and line-length.
- **gosec**: `cmd/api/main.go` excluded for hardcoded credentials (DSN default).
- **mnd (magic numbers)**: excluded for `cmd/api/main.go` and `internal/server/server.go`.
- **errcheck**: `encoding/json.Marshal` excluded.
- **Formatters**: only `gofumpt` enabled. Do NOT add `goimports` — it conflicts with gofumpt on import grouping.
- **Import groups**: stdlib / project imports / 3rd-party, each separated by blank line.

## Generated Code

- `internal/repository/db/*.go` is sqlc-generated. Do not edit by hand — edit `.sql` files in `internal/repository/query/`, then `sqlc generate`.
- sqlc config in `sqlc.yaml`: pgx/v5, emit_interface, emit_json_tags.
- `migrations/` files are sequential SQL. Add new numbered files, don't alter existing ones.

## Key Dirs

| Path | Purpose |
|---|---|
| `cmd/api/` | Gateway entrypoint |
| `cmd/worker/` | Worker entrypoint |
| `internal/server/` | HTTP server + routes |
| `internal/repository/db/` | sqlc-generated Go (checked in) |
| `internal/repository/query/` | sqlc SQL sources |
| `migrations/` | goose migration SQL |
| `dashboard/` | Next.js 15 frontend (separate project) |
| `deployments/` | Docker Compose |
| `docs/` | PRDs, schema docs, design analysis |
