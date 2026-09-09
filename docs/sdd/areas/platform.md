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
| `REDIS_URL` | yes | — | Redis URL (`redis://` or `rediss://`, host non-empty). Local Compose: `redis://:finbot@127.0.0.1:6379/0`. Missing or invalid fails load. Unreachable Redis fails startup |
| `REDIS_TEST_URL` | no | Compose Redis URL in tests | `rediscache` integration tests. Unreachable Redis **fails** (does not skip) |

Copy `.env.example` to `.env` for local secrets. `.env` is gitignored. Process environment wins over `.env`.

## Docker

`Dockerfile` exists (multi-stage, static-ish Go binary). It does not store Postgres or Redis on a local volume; pass `DATABASE_URL` and `REDIS_URL` at run time.

Local Compose: [`docker-compose.yml`](../../../docker-compose.yml) — `postgres:16` (healthcheck `pg_isready`, named volume, user/password `finbot`/`finbot`, databases `finbot` and `finbot_test`) and `redis:7-alpine` (port `6379`, `requirepass finbot`, `maxmemory 64mb`, `maxmemory-policy volatile-ttl`, healthcheck `redis-cli -a finbot ping`, no volume). Local/CI credentials only. `docker compose up -d` before `go run ./cmd/bot` or adapter tests. CI does **not** run Compose. Compose is not production.

## Hosting

Production host is Railway Hobby: one **worker** (not a web service), replica **1**, sleep off, no public domain. [`railway.toml`](../../../railway.toml) is the in-repo contract (Docker builder, image `ENTRYPOINT`, restart on failure, stop-then-start). Railway Config as Code may be ignored for new services; set the dashboard equivalents. Hosted `DATABASE_URL` should use TLS (`sslmode=require` or the vendor URL). App uses **private** plugin URLs. Two long-poll processes on the same bot token fight. Railway GitHub auto-deploy must stay **off**; Actions is the only deployer. Distroless image has no shell — use deployment logs, not Railway Console. `4.3.2` go-live is in progress (operator project exists; Actions must be the deployer).

## CI

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml). Runs on push to `master`/`main`, pull requests, and `workflow_dispatch`.

| Job | What |
| --- | --- |
| `unit:test` | `make test-unit` (no Postgres, no Redis) |
| `lint` | golangci-lint (`v2.13`) with `.golangci.yaml` (includes `integration` build tags) |
| `integration:test` | `make test-integration` — `./internal/adapter/postgres/...` and `./internal/adapter/rediscache/...`. Job provides `postgres:16` (`POSTGRES_DB=finbot_test`, `POSTGRES_TEST_URL`) and `redis:7-alpine` (`REDIS_TEST_URL`) **service containers** |
| `common` | gate: succeeds only if the three jobs succeeded |
| `deploy` | `needs: common`. Push to `master`/`main` only. Railway CLI `railway up --ci`. Job-level `if` must not use `secrets` (GitHub rejects the whole file). Empty `RAILWAY_TOKEN` / `RAILWAY_SERVICE_ID` fails the job. Optional `RAILWAY_PROJECT_ID` / `RAILWAY_ENVIRONMENT`. Tests still do not need `BOT_TOKEN` |

Branch protection should require `common`, not `deploy`. Human runbook: [`README.md`](../../../README.md).

## Later (step files only)

| Topic | Step |
| --- | --- |
| Cheap hosting go-live (Railway project, secrets, restore drill) | [`4.03.2`](../steps/4.03.2-hosting-provision.md) |
| Rate limit / replicas / partitioning | [`4.09`](../steps/4.09-load.md) |
| Dashboards / metrics backend | [`4.17`](../steps/4.17-dashboards.md) |
| `cmd/bot`, `cmd/worker`, `cmd/web` | [`4.21`](../steps/4.21-modular-binaries.md) |
