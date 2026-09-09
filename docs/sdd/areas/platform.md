# Platform (current)

Human run/CI checklist is canonical in [`README.md`](../../../README.md). This file is the short agent summary.

## Env

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `BOT_TOKEN` | yes | — | From BotFather |
| `SQLITE_PATH` | no | `./data/finbot.db` | File on disk; volume in Docker (`/data/finbot.db`) |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error`. Invalid values fail startup |
| `TRIAL_DURATION` | no | `168h` (7 days) | Frozen on first signup as `trial_ends_at`. `0` means no trial. Negative rejected. Not enforced until `4.5` |
| `ADMIN_TELEGRAM_ID` | no | unset | Numeric Telegram user id that receives `/feedback`. Empty or `0` makes `/feedback` unavailable. Negative and non-numeric fail startup |
| `DEFAULT_CURRENCY` | no | `USD` | Stored on **new** banks. Empty/missing uses `USD`. Any other trimmed value is kept as-is (no ISO check) |
| `DATABASE_URL` | no | unset | Parsed into config. **Not** used by `cmd/bot` until `4.1.3` |
| `POSTGRES_TEST_URL` | no | Compose `finbot_test` URL in tests | Adapter integration tests. Unreachable Postgres **fails** (does not skip) |

Copy `.env.example` to `.env` for local secrets. `.env` is gitignored. Process environment wins over `.env`.

## Docker

`Dockerfile` exists (multi-stage, static-ish Go binary, data dir as a volume path). Not deployed until hosting.

Local Postgres: [`docker-compose.yml`](../../../docker-compose.yml) — `postgres:16`, healthcheck `pg_isready`, named volume, user/password `finbot`/`finbot` (local/CI only). Databases `finbot` and `finbot_test`. `docker compose up -d` for adapter tests. CI does **not** run Compose.

## CI

Workflow: [`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml). Runs on push to `master`/`main`, pull requests, and `workflow_dispatch`.

| Job | What |
| --- | --- |
| `unit:test` | `make test-unit` (no Postgres) |
| `lint` | golangci-lint (`v2.13`) with `.golangci.yaml` (includes `integration` build tags) |
| `integration:test` | `make test-integration` — SQLite temp file **and** `./internal/adapter/postgres/...`. Job provides a `postgres:16` **service container** (`POSTGRES_DB=finbot_test`, `POSTGRES_TEST_URL`) |
| `common` | gate: succeeds only if the three jobs succeeded |

Future pipeline stages must `needs: common`. No CI secrets yet (tests do not need `BOT_TOKEN`).

## Later (step files only)

| Topic | Step |
| --- | --- |
| Postgres adapter (`FOR UPDATE`, repos) then cutover | [`4.1.2`](../steps/4.01.2-postgres-adapter.md)–[`4.1.3`](../steps/4.01.3-postgres-cutover.md) |
| Cheap hosting + managed Postgres (vendor chosen at the time) | [`4.03`](../steps/4.03-hosting.md) |
| Rate limit / replicas / partitioning | [`4.09`](../steps/4.09-load.md) |
| Dashboards / metrics backend | [`4.17`](../steps/4.17-dashboards.md) |
| `cmd/bot`, `cmd/worker`, `cmd/web` | [`4.21`](../steps/4.21-modular-binaries.md) |
