# Platform (current)

Human run/CI checklist is canonical in [`README.md`](../../../README.md). This file is the short agent summary.

## Env

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `BOT_TOKEN` | yes | — | From BotFather |
| `DATABASE_URL` | yes | — | Postgres URL. Local Compose: `postgres://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable`. Missing or non-postgres URL fails startup |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error`. Invalid values fail startup |
| `TRIAL_DURATION` | no | `168h` (7 days) | Frozen on first signup as `trial_ends_at`. `0` means no trial. Negative rejected. Not enforced until `4.5` |
| `ADMIN_TELEGRAM_ID` | no | unset | Numeric Telegram user id that receives `/feedback`. Empty or `0` makes `/feedback` unavailable. Negative and non-numeric fail startup |
| `DEFAULT_CURRENCY` | no | `USD` | Stored on **new** banks. Empty/missing uses `USD`. Any other trimmed value is kept as-is (no ISO check) |
| `POSTGRES_TEST_URL` | no | Compose `finbot_test` URL in tests | Adapter integration tests. Unreachable Postgres **fails** (does not skip) |

Copy `.env.example` to `.env` for local secrets. `.env` is gitignored. Process environment wins over `.env`.

## Docker

`Dockerfile` exists (multi-stage, static-ish Go binary). It does not store the database on a local volume; pass `DATABASE_URL` at run time. Not deployed until hosting.

Local Postgres: [`docker-compose.yml`](../../../docker-compose.yml) — `postgres:16`, healthcheck `pg_isready`, named volume, user/password `finbot`/`finbot` (local/CI only). Databases `finbot` (bot) and `finbot_test` (integration). `docker compose up -d` before `go run ./cmd/bot` or adapter tests. CI does **not** run Compose.

## CI

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml). Runs on push to `master`/`main`, pull requests, and `workflow_dispatch`.

| Job | What |
| --- | --- |
| `unit:test` | `make test-unit` (no Postgres) |
| `lint` | golangci-lint (`v2.13`) with `.golangci.yaml` (includes `integration` build tags) |
| `integration:test` | `make test-integration` — `./internal/adapter/postgres/...` only. Job provides a `postgres:16` **service container** (`POSTGRES_DB=finbot_test`, `POSTGRES_TEST_URL`) |
| `common` | gate: succeeds only if the three jobs succeeded |

Future pipeline stages must `needs: common`. No CI secrets yet (tests do not need `BOT_TOKEN`).

## Later (step files only)

| Topic | Step |
| --- | --- |
| Cheap hosting + managed Postgres (vendor chosen at the time) | [`4.03`](../steps/4.03-hosting.md) |
| Rate limit / replicas / partitioning | [`4.09`](../steps/4.09-load.md) |
| Dashboards / metrics backend | [`4.17`](../steps/4.17-dashboards.md) |
| `cmd/bot`, `cmd/worker`, `cmd/web` | [`4.21`](../steps/4.21-modular-binaries.md) |
