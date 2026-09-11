# Finbot

Private Telegram bot for splitting money into named banks (Travelling, Gifts, Live, …), updating balances, and seeing a total of the banks that count.

Runtime is Go 1.27 + Postgres + Redis. Architecture is clean/hexagonal: Telegram, Postgres, and Redis are adapters; use cases live in `internal/service` and depend only on domain types and the interfaces that package owns.

**FSM** (Finite State Machine) is the per-user conversation step stored in Redis: which command is in flight and what the bot is waiting for (bank name, yes/no, amount, …). Agents start at [`AGENTS.md`](AGENTS.md). Product rules: [`docs/sdd/requirements.md`](docs/sdd/requirements.md).

### Command shortcuts

Users can type details on the same line as a slash command instead of waiting for a prompt. Bank names may contain spaces.

- `/newbank Travelling` — skip the “what should this bank be called?” prompt
- `/newbank Holiday fund` — the whole remainder is the name; `yes`/`no` on that line is part of the name, not the include-in-total flag
- Later: `/add Travelling 100`, `/bank Travelling`, and similar

`/start` and `/help` in the bot mention this. Include-in-total is chosen only after a unique name is accepted.

### Chat history

The bot keeps slash commands and results (`Added 100 to "Travelling"…`, `/banks`, `/help`, …). It edits or deletes wizard prompts (which bank, how much, yes/no) and deletes the short answers you type during a flow (the amount `100`, a typed bank name). Your `/feedback` text stays. Policy: [`docs/sdd/areas/telegram.md`](docs/sdd/areas/telegram.md).

## Environment

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `BOT_TOKEN` | yes | — | From [@BotFather](https://t.me/BotFather) |
| `DATABASE_URL` | yes | — | Postgres URL. Local Compose: `postgres://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable` |
| `REDIS_URL` | yes | — | Redis URL (`redis://` or `rediss://`, host non-empty). Local Compose: `redis://:finbot@127.0.0.1:6379/0` |
| `BOT_TTL` | no | `1h` | Sliding conversation FSM TTL (Go duration). Must be `> 0` |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error` |
| `TRIAL_DURATION` | no | `168h` (7 days) | Frozen on first signup as `trial_ends_at`. `0` means no trial. Negative values are rejected. Not enforced until stage 2 |
| `ADMIN_TELEGRAM_ID` | no | unset | Numeric Telegram user id that receives `/feedback`. Empty or `0` makes `/feedback` reply that it is unavailable. Negative and non-numeric values fail startup. |
| `DEFAULT_CURRENCY` | no | `USD` | Stored on **new** banks. Empty/missing uses `USD`. Any other trimmed value is kept as-is (no ISO check). Existing banks keep the migration default `USD`. Hidden in copy until later. |
| `ENVFILE` | no | unset | If set, load that dotenv file first. Process environment still wins. Railway does not set this. |

Copy `.env.example` to `.env` for local secrets. `.env` is gitignored. Process environment wins over `ENVFILE`. Compose reads `.env` for `${BOT_TOKEN}` interpolation; the bot container gets in-network `DATABASE_URL` / `REDIS_URL` from `docker-compose.yml`, not the host URLs in `.env`.

## Run locally

A new machine needs **Go 1.27** ([install](https://go.dev/dl/)), **Docker**, a Telegram account, and a bot token. Clone this repo and work from its root. Talk to the bot in a **private chat** (one user ↔ one bot). Keep a single process per token: two long-polling clients on the same token fight each other. Use a **dev BotFather token** locally; leave the production token on Railway.

```bash
git clone https://github.com/undochlorine/finbot.git
cd finbot
```

### 1. Create a bot and token

1. Open [@BotFather](https://t.me/BotFather) in Telegram.
2. Send `/newbot`. Choose a display name, then a username that ends in `bot`.
3. Copy the HTTP API token BotFather prints.
4. Do **not** run BotFather `/setcommands`. On startup this process registers the Commands menu itself (`setMyCommands`).

### 2. Configure env

From the repo root:

```bash
cp .env.example .env
```

Set `BOT_TOKEN` in `.env` to the **dev** token from BotFather. Set `ADMIN_TELEGRAM_ID` to your numeric Telegram user id if you want `/feedback` forwarded to you; leave it empty to keep `/feedback` unavailable. Optional: `LOG_LEVEL`, `TRIAL_DURATION`, `DEFAULT_CURRENCY`, `BOT_TTL`. Host URLs in `.env` (`DATABASE_URL`, `REDIS_URL`) are for `ENVFILE=.env go run ./cmd/bot` against published Compose ports. `make up` always points the bot container at the `postgres` and `redis` services.

### 3. Start Postgres, Redis, and the bot

```bash
make up
```

That is `docker compose up --build -d`: Postgres, Redis, and the bot image. Wait until `postgres` and `redis` are healthy; the bot starts after that. Follow bot logs with `docker compose logs -f bot`. The bot database is `finbot`; integration tests use `finbot_test`. Redis holds conversation FSM (default TTL `1h`, `BOT_TTL`). There is no Redis volume; a Redis restart drops in-flight wizards.

Stop with `make down`.

To run the bot on the host instead of in Compose (same `.env`, published ports):

```bash
docker compose up -d postgres redis
ENVFILE=.env go run ./cmd/bot
```

Do not combine `make up` and `go run` with the same `BOT_TOKEN`.

### 4. Open Telegram

Find the bot by the username you gave BotFather and send `/start`. `/help` lists commands. The `/` hint and Commands button should show the menu after the process has started. If an old Telegram client still shows no Commands hint, close and reopen the chat.

Try `/newbank Travelling`, then `/banks`. To confirm Postgres keeps rows across a restart, follow [Verify persistence](#verify-persistence). To confirm Redis keeps an in-flight wizard across a process restart, follow [Verify session cache](#verify-session-cache).

## Verify persistence

Banks live in Postgres at `DATABASE_URL`. A process restart must keep them.

1. Start Compose (`make up`, or `docker compose up -d postgres redis` plus `ENVFILE=.env go run ./cmd/bot`) with stable `DATABASE_URL` and `REDIS_URL`.
2. Create a bank (`/newbank Travelling`) and note `/banks`.
3. Stop the process with Ctrl+C (SIGINT) or SIGTERM. Do not wipe the Compose volume.
4. Start the same command again with the same `DATABASE_URL`.
5. `/banks` still lists Travelling with the same balance.

The adapter test `TestReopenKeepsData` is the same open → write → close → reopen path:

```bash
go test -tags=integration -count=1 ./internal/adapter/postgres/ -run TestReopenKeepsData
```

## Verify session cache

Conversation FSM lives in Redis at `REDIS_URL` (JSON blob, sliding TTL from `BOT_TTL`, default `1h`). A **process** restart must keep an in-flight wizard if Redis still holds the key. A **Redis** restart drops wizards (same as expiry — start over).

1. Start Compose (`make up`, or `docker compose up -d postgres redis` plus `ENVFILE=.env go run ./cmd/bot`) with stable `DATABASE_URL` and `REDIS_URL`.
2. Start a wizard and stop before finishing (for example `/newbank` and do not send the name).
3. Stop the bot process with Ctrl+C. Leave Redis running.
4. Start the same command again. The wizard is still in flight (typed name continues the same `/newbank`).
5. Restart Redis (`docker compose restart redis`). The next message in that chat is treated as idle / expired; start the command again.

`memorycache` is tests-only. Production `cmd/bot` does not fall back to it.

## Tests and lint

```bash
make test-unit          # domain, service (mocks), config, cache, clock, telegram wiring, cmd/bot
make test-integration   # Postgres + Redis adapters (needs Compose)
make test               # both
make lint               # golangci-lint using .golangci.yaml
```

Postgres and Redis adapter tests are tagged `//go:build integration` so they are not part of `make test-unit`. Unit tests do not need Postgres or Redis.

Local Postgres and Redis for adapter tests (bot container not required):

```bash
docker compose up -d postgres redis
```

`make test-integration` connects to `POSTGRES_TEST_URL` and `REDIS_TEST_URL`, or defaults to `postgres://finbot:finbot@127.0.0.1:5432/finbot_test?sslmode=disable` and `redis://:finbot@127.0.0.1:6379/0`. If Postgres or Redis is unreachable the tests **fail** (they do not skip) and tell you to start Compose. CI uses GitHub Actions service containers, not Compose.

## CI

GitHub Actions workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on pushes to `master`/`main`, pull requests, and manual dispatch.

**Common stage** (parallel):

| Job | Command |
| --- | --- |
| `unit:test` | `make test-unit` |
| `lint` | golangci-lint |
| `integration:test` | `make test-integration` — Postgres and Redis adapters; job provides `postgres:16` (`POSTGRES_TEST_URL`) and `redis:7-alpine` (`REDIS_TEST_URL`) **service containers**. |

Job `common` succeeds only if those three succeeded.

**Deploy** (`needs: common`) is **manual**. It runs only on **Actions → CI → Run workflow** with **Deploy to Railway** checked, on `master` / `main`. It does not run on push, pull requests, or a dispatch with the checkbox off. It deploys the existing `Dockerfile` with the Railway CLI. If deploy fails, the workflow fails. Missing `RAILWAY_TOKEN` or `RAILWAY_SERVICE_ID` fails **deploy**. GitHub does not allow `secrets` in a job-level `if`; that pattern invalidates the whole workflow file, so `unit:test` / `lint` / `common` never start. Use a Railway **project token** as `RAILWAY_TOKEN`; leave `RAILWAY_PROJECT_ID` / `RAILWAY_ENVIRONMENT` unset unless the CLI needs them.

A green GitHub check named like **kind-compassion - bot** is Railway’s GitHub app, not this workflow. If that is the only check, Actions never parsed `ci.yml` and Railway is deploying from git. Turn that off.

Secrets:

| Secret | Required to deploy | Notes |
| --- | --- | --- |
| `RAILWAY_TOKEN` | yes | Railway **project token** for the production environment (not an account API token) |
| `RAILWAY_SERVICE_ID` | yes | Bot worker service id |
| `RAILWAY_PROJECT_ID` | if the CLI asks | Project id |
| `RAILWAY_ENVIRONMENT` | if the CLI asks | Environment name or id |

Do not put production URLs or tokens in git. Branch protection should require **`common`**, not `deploy`, so empty secrets do not block merges.

Turn **off** Railway’s GitHub auto-deploy on the worker (service settings → Disable automatic deployments). Actions is the only deployer so a push cannot race tests. Do not enable Railway “Wait for CI” as a second deployer.

### One-time GitHub setup

Actions usually work as soon as the workflow file is on the default branch. If a run does not appear:

1. Open the repo on GitHub: [undochlorine/finbot](https://github.com/undochlorine/finbot).
2. **Settings → Actions → General**. Under “Actions permissions”, choose **Allow all actions and reusable workflows**. Save.
3. Push this branch (or merge to `master`). Open the **Actions** tab and confirm a **CI** run with `unit:test`, `lint`, `integration:test`, and `common`. Push does **not** deploy. To ship: **Actions → CI → Run workflow**, branch `master` (or `main`), check **Deploy to Railway after common succeeds**.

Optional, after the first green run (status check names only appear then):

1. **Settings → Branches → Add branch ruleset** (or classic **Branch protection rule**) for `master`.
2. Enable **Require status checks to pass before merging**.
3. Search and require **`common`** (that single check already means unit, lint, and integration passed). Do not require `deploy`.
4. Save.

## Production (Railway)

Host is **Railway Hobby**: one always-on **worker** (not a web service) plus managed Postgres and Redis. There is no public HTTP URL for the bot. Sleep / scale-to-zero must stay **off**. Replica count is **1** until step `4.9`; two long-poll processes on the same bot token fight each other. Deploys stop the old container then start the new one (a short gap with no Telegram consumer is expected). Config in git: [`railway.toml`](railway.toml).

### Env

Set these on the Railway worker. The app uses **private** plugin URLs, not the public TCP proxies.

| Variable | Required | Notes |
| --- | --- | --- |
| `BOT_TOKEN` | yes | From [@BotFather](https://t.me/BotFather) |
| `DATABASE_URL` | yes | Private Postgres URL. Use TLS (`sslmode=require` or the vendor URL as issued) |
| `REDIS_URL` | yes | Private Redis URL (`redis://` or `rediss://`, or the vendor URL as issued) |
| `BOT_TTL` | no | Default `1h`. Omit to keep the default |
| `LOG_LEVEL` | no | `info` in production |
| `ADMIN_TELEGRAM_ID` | no | Numeric Telegram user id for `/feedback` |
| `TRIAL_DURATION` | no | Default `168h`. Not enforced until later |
| `DEFAULT_CURRENCY` | no | Default `USD` |

Do not commit production URLs.

### Deploy

Push to `master`/`main` runs tests only. To deploy: **Actions → CI → Run workflow**, select `master`/`main`, check **Deploy to Railway after common succeeds**. That run still requires a green `common`, then `railway up --ci` of the repo `Dockerfile`. GitHub Actions is the only deployer. Railway GitHub auto-deploy must stay off.

### Logs

Railway dashboard: open the **worker** → **Deployments** → logs (not the worker **Console**). The image is distroless (`gcr.io/distroless/static-debian12:nonroot`): it has no `sh`/`bash`, so Console cannot start. That is expected. Hobby log retention is about 7 days. From a machine with the CLI linked: `railway logs`. `stdout`/`stderr` `slog` is the log API; there is no vendor metrics SDK yet.

The project **Logs** sidebar mixes every service. `checkpoint starting` / `checkpoint complete` lines are **Postgres**, not bot errors.

`Conflict: terminated by other getUpdates request` means two long-poll clients used the same `BOT_TOKEN`. A short burst while Railway starts a new container before the old one has exited is the overlap the design accepted. If it continues after the deploy is `ACTIVE` with replica count 1, another poller is still up: Railway GitHub auto-deploy, a second replica, or a laptop `go run`.

### Postgres and Redis access

Banks, users, and operations live in Postgres until you delete them (or until step `4.4` inactivity). There is **no row TTL** and no Redis cache of balances. Redis only holds the conversation FSM (sliding TTL from `BOT_TTL`, default `1h`).

1. **Browser:** Railway Postgres data tab / query UI and Redis UI.
2. **Local client:** enable the plugin TCP proxy (public URL + TLS + password). Connect with TablePlus / `psql` / Redis Insight / `redis-cli`. Do not open Postgres or Redis to `0.0.0.0` without TLS and a password. Do not tunnel through the bot process.

Postgres backups: Railway Postgres → **Backups**. Restore into a scratch database (or Railway’s restore flow) and confirm rows. Redis is not a recovery target.
