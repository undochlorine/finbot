# Stage 4 — Product stage 2 (not MVP)

Architecture is already shaped so these are adapter/job additions, not a rewrite. Leave `todo` until the user starts this stage. High-level tech stays out of these steps except where a step was widened (Postgres `4.1.x`, Redis `4.2.x`, hosting `4.3.x`, load `4.9`). Same bar as `4.6` for steps still stubbed: details when that step starts.

Monetization sequence (do not skip `4.6`):

1. `4.5` trial enforcement + limited free tier
2. `4.6` payment strategy (blocked on a conversation with the user)
3. `4.7` billing against that strategy
4. `4.8` Telegram admin whitelist privileges (`100` / `50` / `30` / custom % off)

Postgres is split so one session cannot swallow the cutover: `4.1.1` foundation → `4.1.2` adapter → `4.1.3` cutover. Design: [`docs/superpowers/specs/2026-09-09-postgresql-migration-design.md`](../../superpowers/specs/2026-09-09-postgresql-migration-design.md). Redis is split the same way: `4.2.1` foundation+adapter → `4.2.2` cutover. Design: [`docs/superpowers/specs/2026-09-09-redis-cache-design.md`](../../superpowers/specs/2026-09-09-redis-cache-design.md). Hosting is split: `4.3.1` packaging → `4.3.2` provision. Design: [`docs/superpowers/specs/2026-09-09-hosting-design.md`](../../superpowers/specs/2026-09-09-hosting-design.md). Product after money: `4.10`–`4.21`.

Pickup: `Follow AGENTS.md. Let's move to step 4.4` — then only the files that row lists.

| Id | Status | File |
| --- | --- | --- |
| 4.1 | `done` | [PostgreSQL umbrella](4.01-postgresql.md) |
| 4.1.1 | `done` | [Postgres foundation](4.01.1-postgres-foundation.md) |
| 4.1.2 | `done` | [Postgres adapter](4.01.2-postgres-adapter.md) |
| 4.1.3 | `done` | [Postgres cutover](4.01.3-postgres-cutover.md) |
| 4.2 | `done` | [Redis umbrella](4.02-redis.md) |
| 4.2.1 | `done` | [Redis foundation + adapter](4.02.1-redis-foundation.md) |
| 4.2.2 | `done` | [Redis cutover](4.02.2-redis-cutover.md) |
| 4.3 | `done` | [Cheap hosting umbrella](4.03-hosting.md) |
| 4.3.1 | `done` | [Hosting packaging](4.03.1-hosting-packaging.md) |
| 4.3.2 | `done` | [Hosting provision](4.03.2-hosting-provision.md) |
| 4.4 | `todo` | [Inactivity notify and delete](4.04-inactivity.md) |
| 4.5 | `todo` | [Trial period (enforce)](4.05-trial.md) |
| 4.6 | `todo` | [Payment strategy](4.06-payment-strategy.md) |
| 4.7 | `todo` | [Billing implementation](4.07-billing.md) |
| 4.8 | `todo` | [Admin and whitelist](4.08-admin-whitelist.md) |
| 4.9 | `todo` | [Load resistance](4.09-load.md) |
| 4.10 | `todo` | [Transaction history](4.10-history.md) |
| 4.11 | `todo` | [Finance tips](4.11-tips.md) |
| 4.12 | `done` | [Language picker](4.12-language.md) |
| 4.13 | `todo` | [Multi-currency banks](4.13-multi-currency.md) |
| 4.14 | `todo` | [Cross-currency transfer](4.14-fx-transfer.md) |
| 4.15 | `todo` | [Referral program](4.15-referral.md) |
| 4.16 | `todo` | [Contributors program](4.16-contributors.md) |
| 4.17 | `todo` | [Dashboards / metrics](4.17-dashboards.md) |
| 4.18 | `todo` | [Landing website](4.18-landing.md) |
| 4.19 | `todo` | [Web admin](4.19-web-admin.md) |
| 4.20 | `todo` | [Telegram channel](4.20-channel.md) |
| 4.21 | `todo` | [Modular binaries](4.21-modular-binaries.md) |
