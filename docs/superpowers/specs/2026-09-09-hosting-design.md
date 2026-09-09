# Cheap production hosting (Stage 4.3)

Run the **stateless** `cmd/bot` plus **managed PostgreSQL and Redis** on a low bill. Local Compose and CI service containers stay for dev/CI. They are not production.

The bot **long-polls** Telegram. It has no local disk requirement. `DATABASE_URL` and `REDIS_URL` already exist (`4.1.3`, `4.2.2`). Dockerfile already exists.

## Goal

A single always-on bot process in production, pointed at hosted Postgres and Redis, that:

- deploys automatically when CI is green on `master` / `main`
- keeps money in Postgres across deploys and restarts
- treats Redis FSM as expendable
- exposes logs and host CPU/memory without a new metrics stack
- lets the operator inspect Postgres and Redis from a browser **or** a local client
- can grow later by **adding RAM/CPU first**, then replicas / partitioning in `4.9`

## Non-goals

- App replicas, webhooks, load balancer, PgBouncer, partitioning (`4.9`)
- Product dashboards (usage, payments, tariff growth) (`4.17`)
- Landing page / public marketing URL (`4.18`)
- Re-implementing the Postgres or Redis adapters
- Running git `docker-compose.yml` as production
- Zero-downtime overlapping processes (wrong for long polling)

## Decision (locked)

**A — Railway Hobby: one worker + Railway Postgres + Railway Redis.**

Region: **EU** (Amsterdam if offered, else the closest EU region) unless the operator picks otherwise at provision time.

Rejected:

| Option | Why |
| --- | --- |
| Fly Managed Postgres | Basic plan **$38/mo** before the app. Too expensive for low RPS. |
| Neon Free / scale-to-zero | Persistent `pgx` pool either keeps compute awake (burns free CU-hours) or pays cold starts. Always-on Neon Launch is often ~$15+ for compute alone. |
| Fly Machine + unmanaged Postgres + Upstash | Fine technically, cheaper compute, but **you** own Postgres backups. Upstash console is good; Fly unmanaged PG is the weak story. Keep as fallback if Railway’s bill is disliked. |
| Hetzner VPS + Coolify | Cheapest (~€5–8) and scalable by resizing the box, but Postgres/Redis are **not** managed. More SSH, backups, and patching. Keep as the “minimum bill, maximum ops” escape hatch. |
| Render worker + PG + Redis | Per-service floors usually beat Railway for this three-piece stack. |
| Kubernetes / multi-region | No traffic that needs it. |

## Constraints from the running app

- **One long-poll process per bot token.** README already says two clients fight. Replica count in production is **1** until `4.9` (webhooks or a leader lock).
- **Deploy overlap is harmful**, not just “nice if we can skip it.” Stop the old Machine/container, then start the new one. Accept **15–60s** with no Telegram consumer. That is cheaper *and* correct. Webhooks for overlap wait for `4.9`.
- **No public HTTP** is required for the bot to work. Do not add a webhook or a landing server in 4.3. Railway worker = no public domain.
- **Sleep/scale-to-zero is forbidden** for the worker. A sleeping bot misses Telegram updates.
- **Migrations run at `postgres.Open`.** No separate migrate Job.
- **SIGTERM** is already handled in `cmd/bot`. Host stop signal must be SIGTERM with a kill grace of at least ~30s.
- **stdout/stderr `slog`** is the log API. Do not add a vendor SDK.
- **Redis persistence is not a recovery requirement.** FSM TTL is 10 minutes. A Redis restart drops in-flight wizards (same as today).
- **Postgres is the recovery requirement.** Automated backups + one restore drill before calling 4.3 done.
- CI already has a `common` gate. Production deploys **must** `needs: common`. Do not let the host auto-deploy on git push in parallel with tests.

## Target monthly cost (quiet private/public MVP)

Indicative, September 2026 list rates, always-on, low RPS:

| Piece | Typical |
| --- | --- |
| Railway Hobby subscription (includes $5 usage) | $5 |
| Worker (small Go process, well under 512MB) | ~$1–4 usage |
| Postgres + volume | ~$4–8 |
| Redis + volume | ~$1–3 |
| **Expected** | **~$10–18** |

Set a Railway spend alert. Vertical scale is “give the worker or Postgres more RAM,” not a new architecture.

## Architecture

```text
GitHub  master/main
  → Actions: unit, lint, integration → job `common`
  → Actions: deploy (Railway CLI / API)   # only if common succeeded

Railway project (EU)
  bot worker     Dockerfile → /app/bot     replica = 1, no public URL
       │  DATABASE_URL (private, sslmode=require or vendor equivalent)
       │  REDIS_URL    (private redis:// or rediss://)
       ├─ Railway Postgres   backups on; public proxy for local psql/TablePlus
       └─ Railway Redis      public proxy for local redis-cli / Redis Insight

Operator
  Railway dashboard: logs, CPU, memory, data UI
  Local: TablePlus / psql / Redis Insight via the public proxy + TLS
```

Compose remains **dev/CI**. Production secrets live in Railway and GitHub Actions (`RAILWAY_TOKEN`, never in git).

## Access to Postgres and Redis

Must work both ways:

1. **Browser:** Railway Postgres data tab / query UI and Redis UI (or the vendor’s equivalent console).
2. **Local client:** enable Railway TCP proxy (or equivalent public URL), connect with TablePlus, `psql`, Redis Insight, `redis-cli`. App keeps using the **private** URL.

Do not open Postgres/Redis to `0.0.0.0` without TLS and a password. Do not tunnel through the bot process.

## Logs and dashboards

| Want | 4.3 | Later |
| --- | --- | --- |
| App logs | Railway log stream (Hobby: 7 days). `LOG_LEVEL=info` in prod | Optional drain in `4.17` if 7 days is not enough |
| CPU / memory | Railway service metrics | — |
| Tariff / payments / Bot API charts | **No** | `4.17` |
| Uptime ping | Optional: Telegram `/start` after deploy is the smoke test. No HTTP health unless a host requires it (Railway worker does not). | |

## Scale path (do not build now)

1. **Now:** 1 worker. If CPU/RAM hurt, raise the Railway limit / Postgres size.
2. **`4.9`:** per-user rate limit; replica-safe FIFO/lock; only then replica count > 1 (almost certainly **webhooks**); partitioning / PgBouncer if `operations` or connection count hurts.
3. **`4.18`:** landing is a **second** Railway service with a public URL, same project, not a change to the worker.
4. **`4.21`:** extra `cmd/worker` / `cmd/web` binaries become extra Railway services sharing the same Postgres/Redis.

## Decomposition

Agent-doable packaging is real (Actions deploy job, Railway config, production README) but not huge. Provisioning accounts, secrets, the first live bot, and the restore drill are mostly the operator. Split in **two** steps, not three: logs/metrics are host-native.

Pickup: `Follow AGENTS.md. Let's move to step 4.3.1`. Saying `4.3` means start `4.3.1`, not both.

| Step | File | Production bot live? |
| --- | --- | --- |
| **4.3.1** Packaging | [`docs/sdd/steps/4.03.1-hosting-packaging.md`](../../sdd/steps/4.03.1-hosting-packaging.md) | no (repo only) |
| **4.3.2** Provision + go-live | [`docs/sdd/steps/4.03.2-hosting-provision.md`](../../sdd/steps/4.03.2-hosting-provision.md) | yes |

## 4.3.1 — what the agent changes

- GitHub Actions **deploy** job: `needs: common`, `if` push to `master`/`main` only (not pull requests). Uses Railway token from GitHub secrets.
- Disable / document “don’t auto-deploy from Railway GitHub app on push” so CI cannot lose the race.
- `railway.toml` (or equivalent) in repo: Dockerfile builder, restart policy, no public networking, replica 1.
- README: production env, replica rule, how to tail logs, how to connect to PG/Redis, how deploy works.
- `.env.example`: comments that production URLs use TLS; still no real secrets.
- Do **not** add webhook, health HTTP, metrics SDK, or a second binary.

Tests: none required beyond existing CI still passing. This step is pipeline + docs. Do not invent a fake deploy unit test.

## 4.3.2 — what the operator does (agent writes the checklist and stays in the loop)

1. Railway account, Hobby plan, spend alert, EU project.
2. Add Postgres and Redis. Confirm automated backups for Postgres.
3. Create the bot service from the Dockerfile. Set `BOT_TOKEN`, `DATABASE_URL`, `REDIS_URL`, `ADMIN_TELEGRAM_ID`, `LOG_LEVEL=info`. Private URLs for the app. `sslmode=require` (or vendor TLS equivalent) on any public Postgres URL.
4. Put `RAILWAY_TOKEN` (and service/project ids if required) in GitHub Actions secrets. Merge/deploy once via the new job.
5. Smoke: `/start` on Telegram. Restart the worker; banks still there (Postgres). Optional: start a wizard, restart Redis, wizard is gone (expected).
6. **Restore drill:** create a throwaway bank (or note a row count), restore a backup into a scratch DB or use Railway’s restore flow, prove rows come back. Write the exact clicks/commands into README so it is repeatable.
7. Open Postgres from the Railway UI **and** from TablePlus/`psql` via proxy. Same for Redis UI and `redis-cli`.
8. Confirm replica count is 1 and app sleep is off.

## Out of scope reminders

`4.4` inactivity jobs still run **in-process** until `4.21`. Hosting does not add a cron service.

A “product + bot link” page is **not** required: Telegram does not need a public URL. Landing waits for `4.18` (and prices for `4.6`).
