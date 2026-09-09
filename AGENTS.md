# Finbot — agent entry

This is the only file to always read. Then open **only** the files the routing table lists.

Do not read `docs/sdd/legacy/` unless the user asks, a live doc points at a superseded decision, or you must explain why old code looks like X.

## Status

| Field | Value |
| --- | --- |
| **Current step** | `4.3.2` |
| **Last done** | `4.3.1` Railway packaging (CI deploy after `common`; no live bot yet) |
| **MVP target** | private-use Telegram finance bot in Go + Postgres |
| **GitHub** | `undochlorine/finbot` exists; do not push unless asked |
| **Go module** | `finbot` until a remote exists |
| **Path** | `/Users/a.sicaci/Projects/finbot` |

**Status values:** `todo` | `to review` | `done`

## Protocol

1. User says `let's move to step x.y` (or `Follow AGENTS.md. Let's move to step x.y`).
2. Read this file, then **only** that row’s files. Implement **only** that step.
3. Mark the step `to review` in its step file. Update **Current step** / **Last done** here.
4. User accepts → mark `done`. Fold new “how it works now” into the relevant `docs/sdd/areas/` file. Next `todo` becomes the pointer.
5. Do not start later steps unless asked. Do not skip DoD.
6. After a logical Go code scope: `make lint`.
7. Do not create a GitHub remote or commit unless the user asks.
8. Tests: table-driven unit tests for business logic with **mockery on the consumer’s interfaces**. Integration: Postgres (and Redis after `4.2.1`) via Compose locally and GitHub Actions service containers in CI. **Do not hand-write stubs/fakes/spies** for an interface mockery can generate. A custom test double needs a written reason **and explicit user approval**. Plain cases always use mockery. CI: `make test-unit`, `make lint`, `make test-integration`.

## Routing

For `let's move to step X.Y` also read `docs/sdd/requirements.md` and the step file. Extra areas are `docs/sdd/areas/<name>.md`. Do not open other areas or `docs/sdd/legacy/`.

| Step | Extra areas | Step file |
| --- | --- | --- |
| 4.1 | persistence | `docs/sdd/steps/4.01-postgresql.md` (umbrella; do not implement — start `4.1.1`) |
| 4.1.1 | persistence, platform | `docs/sdd/steps/4.01.1-postgres-foundation.md` |
| 4.1.2 | persistence, service | `docs/sdd/steps/4.01.2-postgres-adapter.md` |
| 4.1.3 | persistence, platform | `docs/sdd/steps/4.01.3-postgres-cutover.md` |
| 4.2 | telegram, platform | `docs/sdd/steps/4.02-redis.md` (umbrella; do not implement — start `4.2.1`) |
| 4.2.1 | telegram, platform | `docs/sdd/steps/4.02.1-redis-foundation.md` |
| 4.2.2 | telegram, platform | `docs/sdd/steps/4.02.2-redis-cutover.md` |
| 4.3 | platform | `docs/sdd/steps/4.03-hosting.md` (umbrella; do not implement — start `4.3.1`) |
| 4.3.1 | platform | `docs/sdd/steps/4.03.1-hosting-packaging.md` |
| 4.3.2 | platform | `docs/sdd/steps/4.03.2-hosting-provision.md` |
| 4.4 | billing, platform, telegram | `docs/sdd/steps/4.04-inactivity.md` |
| 4.5 | billing, telegram, service | `docs/sdd/steps/4.05-trial.md` |
| 4.6 | billing | `docs/sdd/steps/4.06-payment-strategy.md` |
| 4.7 | billing, service | `docs/sdd/steps/4.07-billing.md` |
| 4.8 | billing, telegram | `docs/sdd/steps/4.08-admin-whitelist.md` |
| 4.9 | platform, telegram, persistence | `docs/sdd/steps/4.09-load.md` |
| 4.10 | persistence, telegram, service | `docs/sdd/steps/4.10-history.md` |
| 4.11 | telegram, persistence | `docs/sdd/steps/4.11-tips.md` |
| 4.12 | telegram, persistence | `docs/sdd/steps/4.12-language.md` |
| 4.13 | persistence, telegram, service | `docs/sdd/steps/4.13-multi-currency.md` |
| 4.14 | telegram, persistence, service | `docs/sdd/steps/4.14-fx-transfer.md` |
| 4.15 | billing, telegram | `docs/sdd/steps/4.15-referral.md` |
| 4.16 | billing | `docs/sdd/steps/4.16-contributors.md` |
| 4.17 | platform | `docs/sdd/steps/4.17-dashboards.md` |
| 4.18 | platform | `docs/sdd/steps/4.18-landing.md` |
| 4.19 | billing, platform | `docs/sdd/steps/4.19-web-admin.md` |
| 4.20 | telegram, platform | `docs/sdd/steps/4.20-channel.md` |
| 4.21 | platform, service | `docs/sdd/steps/4.21-modular-binaries.md` |

Bugfix (not a numbered step): this file + `docs/sdd/requirements.md` + the area that owns the package (usually `telegram` and/or `service`). Skip `legacy/` and skip 4.x files unless the bug is a Stage 4 hook.

## Wake-up

`Follow AGENTS.md. Let's move to step 4.3.2`

Cheap hosting go-live (Railway worker + managed Postgres/Redis). Design: [`docs/superpowers/specs/2026-09-09-hosting-design.md`](docs/superpowers/specs/2026-09-09-hosting-design.md). Stage 4 index: [`docs/sdd/steps/README.md`](docs/sdd/steps/README.md). Product rules: [`docs/sdd/requirements.md`](docs/sdd/requirements.md). Human run/CI: [`README.md`](README.md).
