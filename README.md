# Finbot

Private Telegram bot for splitting money into named banks (Travelling, Gifts, Live, …), updating balances, and seeing a total of the banks that count.

MVP is Go 1.27 + SQLite. Architecture is clean/hexagonal: Telegram and SQLite are adapters; use cases live in `internal/service` and depend only on domain types and the interfaces that package owns.

**FSM** (Finite State Machine) is the per-user conversation step stored in Cache: which command is in flight and what the bot is waiting for (bank name, yes/no, amount, …). Agents start at [`AGENTS.md`](AGENTS.md). Product rules: [`docs/sdd/requirements.md`](docs/sdd/requirements.md).

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
| `SQLITE_PATH` | no | `./data/finbot.db` | File on disk; use a volume in Docker (`/data/finbot.db`) |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error` |
| `TRIAL_DURATION` | no | `168h` (7 days) | Frozen on first signup as `trial_ends_at`. `0` means no trial. Negative values are rejected. Not enforced until stage 2 |
| `ADMIN_TELEGRAM_ID` | no | unset | Numeric Telegram user id that receives `/feedback`. Empty or `0` makes `/feedback` reply that it is unavailable. Negative and non-numeric values fail startup. |
| `DEFAULT_CURRENCY` | no | `USD` | Stored on **new** banks. Empty/missing uses `USD`. Any other trimmed value is kept as-is (no ISO check). Existing banks keep the migration default `USD`. Hidden in copy until later. |

Copy `.env.example` to `.env` for local secrets. `.env` is gitignored. Process environment wins over `.env`.

## Run locally

A new machine needs **Go 1.27** ([install](https://go.dev/dl/)), a Telegram account, and a bot token. Clone this repo and work from its root. Talk to the bot in a **private chat** (one user ↔ one bot). Keep a single process per token: two long-polling clients on the same token fight each other.

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

Set `BOT_TOKEN` in `.env` to the token from BotFather. Set `ADMIN_TELEGRAM_ID` to your numeric Telegram user id if you want `/feedback` forwarded to you; leave it empty to keep `/feedback` unavailable. Leave `SQLITE_PATH`, `LOG_LEVEL`, `TRIAL_DURATION`, and `DEFAULT_CURRENCY` at the defaults unless you need to change them. You can export the same variables in the shell instead; real env wins over `.env`.

### 3. Start the process

```bash
go run ./cmd/bot
```

The process loads config, creates the SQLite directory if needed, opens the database, registers the slash command menu with Telegram, and long-polls until Ctrl+C (SIGINT) or SIGTERM. Missing `BOT_TOKEN`, an unusable `SQLITE_PATH`, or an invalid `LOG_LEVEL` / `TRIAL_DURATION` / `ADMIN_TELEGRAM_ID` is a non-zero exit. The host needs outbound HTTPS to `api.telegram.org`.

### 4. Open Telegram

Find the bot by the username you gave BotFather and send `/start`. `/help` lists commands. The `/` hint and Commands button should show the menu after the process has started. If an old Telegram client still shows no Commands hint, close and reopen the chat.

Try `/newbank Travelling`, then `/banks`. To confirm the SQLite file survives a restart, follow [Verify persistence](#verify-persistence).

## Verify persistence

Banks live in the SQLite file at `SQLITE_PATH` (default `./data/finbot.db`). A process restart must keep them.

1. Start the bot with a stable path (`export SQLITE_PATH=./data/finbot.db` or the same value in `.env`).
2. Create a bank (`/newbank Travelling`) and note `/banks`.
3. Stop the process with Ctrl+C (SIGINT) or SIGTERM so the file is closed cleanly. Do not delete `data/finbot.db` or `data/finbot.db-*` (WAL sidecars).
4. Start the same command again with the same `SQLITE_PATH`.
5. `/banks` still lists Travelling with the same balance.

The adapter test `TestReopenKeepsData` is the same open → write → close → reopen path:

```bash
go test -tags=integration -count=1 ./internal/adapter/sqlite/ -run TestReopenKeepsData
```

## Tests and lint

```bash
make test-unit          # domain, service (mocks), config, cache, clock, telegram wiring, cmd/bot
make test-integration   # SQLite adapter against a temp DB file
make test               # both
make lint               # golangci-lint using .golangci.yaml
```

SQLite tests are tagged `//go:build integration` so they are not part of `make test-unit`.

## CI

GitHub Actions workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on pushes to `master`/`main`, pull requests, and manual dispatch.

**Common stage** (parallel):

| Job | Command |
| --- | --- |
| `unit:test` | `make test-unit` |
| `lint` | golangci-lint |
| `integration:test` | `make test-integration` |

Job `common` succeeds only if those three succeeded. Later pipeline stages (none yet) must `needs: common`; if any common job fails they will not run.

No repository secrets are needed for this pipeline.

### One-time GitHub setup

Actions usually work as soon as the workflow file is on the default branch. If a run does not appear:

1. Open the repo on GitHub: [undochlorine/finbot](https://github.com/undochlorine/finbot).
2. **Settings → Actions → General**. Under “Actions permissions”, choose **Allow all actions and reusable workflows**. Save.
3. Push this branch (or merge to `master`). Open the **Actions** tab and confirm a **CI** run with `unit:test`, `lint`, `integration:test`, and `common`.

Optional, after the first green run (status check names only appear then):

1. **Settings → Branches → Add branch ruleset** (or classic **Branch protection rule**) for `master`.
2. Enable **Require status checks to pass before merging**.
3. Search and require **`common`** (that single check already means unit, lint, and integration passed).
4. Save. Do not require GitHub secrets for CI at this stage.
