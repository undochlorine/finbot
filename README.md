# Finbot

Private Telegram bot for splitting money into named banks (Travelling, Gifts, Live, …), updating balances, and seeing a total of the banks that count.

MVP is Go + SQLite. Architecture is hexagonal: Telegram and SQLite are adapters; use cases live in `internal/service` and depend only on domain types and ports.

The living work plan is [`plan.md`](plan.md).

## Environment

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `BOT_TOKEN` | yes | — | From [@BotFather](https://t.me/BotFather) |
| `SQLITE_PATH` | no | `./data/finbot.db` | File on disk; use a volume in Docker (`/data/finbot.db`) |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error` |
| `TRIAL_DURATION` | no | `168h` (7 days) | Frozen on first signup as `trial_ends_at`. `0` means no trial. Negative values are rejected. Not enforced until stage 2 |

Copy `.env.example` to `.env` for local secrets. `.env` is gitignored. Process environment wins over `.env`.

## Run locally

Full BotFather / command checklist lands in step `3.3`. Until then:

```bash
# values can live in .env instead of exports
go run ./cmd/bot
```

Until `2.1`, the process loads config (including `.env`), sets the log level, and exits. Missing `BOT_TOKEN` or an invalid `LOG_LEVEL` / `TRIAL_DURATION` is a non-zero exit.
