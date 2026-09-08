# SDD framework restructure

Token-efficient Spec-Driven Development for Finbot. Replaces the single `plan.md` source of truth with a routed doc tree so agents do not ingest finished stages, superseded intent, or unrelated modules.

## Goal

An agent that lost chat context can continue from a short wake-up phrase. It reads `AGENTS.md`, then only the files that row lists. It does not open `docs/sdd/legacy/` unless there is a hard reason.

## Non-goals

- Widening Stage 4 step text (keep today’s stubs; the user will extend later)
- Changing product behavior or code
- Inventing new Stage 4 technical choices
- Replacing `docs/superpowers/specs/` (per-feature designs stay there)
- Making `docs/sdd/areas/*.md` a second README for humans (README stays the human run/CI doc)

## Wake-up

Primary: `Follow AGENTS.md. Let's move to step 4.1`

Still valid: `let's move to step 4.1` — same protocol; `AGENTS.md` is what to open first.

Do not tell agents to read `plan.md` (that file is deleted).

## Tree

```text
AGENTS.md                          # always-read: status, protocol, routing, skip-legacy
README.md                          # human ops; points at AGENTS.md + requirements.md
docs/sdd/
  requirements.md                  # locked product rules + vision + stage map
  areas/
    telegram.md                    # current bot surface / FSM / hygiene / queue / emojis
    persistence.md                 # current schema + SQLite
    service.md                     # hexagonal rules, domain as code is now, tests/mockery
    billing.md                     # columns/config that already exist (not 4.5–4.8 how-to)
    platform.md                    # env/Docker/CI summary; README is canonical for run
  steps/
    README.md                      # 4.x index + monetization sequence; not a dump
    4.01-postgresql.md
    4.02-redis.md
    4.03-hosting.md
    4.04-inactivity.md
    4.05-trial.md
    4.06-payment-strategy.md
    4.07-billing.md
    4.08-admin-whitelist.md
    4.09-load.md
    4.10-history.md
    4.11-tips.md
    4.12-language.md
    4.13-multi-currency.md
    4.14-fx-transfer.md
    4.15-referral.md
    4.16-contributors.md
    4.17-dashboards.md
    4.18-landing.md
    4.19-web-admin.md
    4.20-channel.md
    4.21-modular-binaries.md
  legacy/                          # DO NOT READ by default
    README.md
    decisions-log.md
    stages-0-3.md
    superseded.md
docs/superpowers/specs/            # unchanged
```

Delete `plan.md`. No stub left at the old path.

## Components

### `AGENTS.md` (always-read)

Keep this short. It is the only file every session must open.

Must contain:

1. **Status header** — current step, last done, MVP target, GitHub, Go module, path (same fields as today’s plan header).
2. **Agent protocol** — one step at a time; mark `to review`; update current/last; lint/tests; no commit/push/remote unless asked; mockery + table tests + temp SQLite integration (same rules as today).
3. **Routing table** — working-on → files to read. Default for `let's move to step X.Y`: `AGENTS.md` + `docs/sdd/requirements.md` + listed area file(s) + `docs/sdd/steps/<file>`.
4. **Skip rule** — do not read `docs/sdd/legacy/` unless the user asks, a live doc points at a superseded decision, or you must explain why old code looks like X.
5. **Pointers** — `docs/sdd/requirements.md`, `README.md` for human run, `docs/superpowers/specs/` for feature designs.

Status values stay `todo` | `to review` | `done`. After a step is accepted, update the header in `AGENTS.md` and the status line in that step file.

### `docs/sdd/requirements.md`

Business rules that still apply. No package tree, no Go sketches, no env/CI, no step DoDs.

- Short vision (banks, include-in-total, Telegram)
- Stage 3 vs Stage 4 placement table (current vs later product)
- Locked product decisions that are still true (isolation, money as cents, negative balances, English through Stage 3, chat hygiene policy pointer, entitlement **order**, trial freeze, whitelist as percent, modular monolith until 4.21, mockery, …)
- Entitlement order lives here (product rule). Implementation status (always-allow) lives in `areas/billing.md`

### Area files (current technical state only)

Each file describes **how the code works now**. Future work is a one-liner plus a step-file link. Do not copy 4.x Goal/DoD into area files.

| File | Contains | Does not contain |
| --- | --- | --- |
| `telegram.md` | Command table, shortcuts, callback versioning, chat hygiene, pending FIFO + compact rules, emojis, FSM/Cache as they work now | Redis how-to (`4.2`), `/history` UX (`4.10`) |
| `persistence.md` | `users` / `banks` / `operations` columns (including locale, currency, plan fields as schema only), Total query, SQLite WAL/migrations/driver | Postgres adapter design (`4.1`); trial/paywall semantics |
| `service.md` | Package tree, hexagonal / interface-per-consumer, layer rules, domain sketches matching code, Transactor + operations write path, test/mockery rules | Payment ports (`4.6`–`4.7`) |
| `billing.md` | Semantics of existing hooks: Plan, TrialEndsAt, DiscountPercent, Entitlement always-full, referred_by, TRIAL_DURATION not enforced | Column DDL (persistence); trial enforcement, checkout, admin CRUD (`4.5`–`4.8`) |
| `platform.md` | Env var list, Docker, CI jobs, log level; link to README as canonical run checklist | Hosting vendor choice (`4.3`), replicas (`4.9`), extra `cmd/` (`4.21`) |

Areas that are still mostly future (i18n catalogs, growth, web) do **not** get empty files. Current hooks stay in telegram/billing; product work stays in the 4.x step file.

### Step files (`docs/sdd/steps/`)

One file per `4.1`–`4.21`. Content is today’s stub: Status, Goal, Notes/DoD/Depends as written now. No new technical invention.

Filename uses zero-padded `4.xx` plus a short slug so directory order matches step order.

Stage 4 intro (monetization sequence, “architecture already shaped”, 4.1–4.21 status list) lives only in `docs/sdd/steps/README.md`. File `4.01-postgresql.md` is PostgreSQL only.

### `docs/sdd/legacy/`

Historical only. Banner in `legacy/README.md` restates the skip rule.

| File | Source in today’s `plan.md` |
| --- | --- |
| `stages-0-3.md` | Stage 0–3 step blocks (0.1–3.11) including Goal/Files/DoD and the “hardened MVP complete” closer |
| `decisions-log.md` | Dated decisions table as-is |
| `superseded.md` | Intentions that already happened: ports → consumer interfaces; telegram owns Cache; Transactor replacing “no unit of work in 3.6”; Stage 3 features that shipped (feedback, hygiene, queue, emojis, rename, transfer) framed as “was later, now done” |

**Not superseded** (still future — keep a one-liner in the live area/step, not in `superseded.md`): SQLite → Postgres, memory Cache → Redis, Entitlement always-allow → `4.5`, one `cmd/bot` → `4.21`.

### README

Replace every `plan.md` link with `AGENTS.md` and/or `docs/sdd/requirements.md`. Chat-hygiene link moves to `docs/sdd/areas/telegram.md`. Keep run/env/CI as the human canonical doc. `platform.md` is a shorter agent summary that points at README.

## Routing (agent load)

Always: `AGENTS.md`.

Then, for `let's move to step X.Y`, also read `requirements.md` + the step file + areas below. Do not open other areas or `legacy/`.

| Step | Extra areas |
| --- | --- |
| 4.1 | persistence |
| 4.2 | telegram, platform |
| 4.3 | platform |
| 4.4 | billing, platform, telegram (Notifier) |
| 4.5 | billing, telegram, service |
| 4.6 | billing |
| 4.7 | billing, service |
| 4.8 | billing, telegram |
| 4.9 | platform, telegram, persistence |
| 4.10 | persistence, telegram, service |
| 4.11 | telegram, persistence |
| 4.12 | telegram, persistence |
| 4.13 | persistence, telegram, service |
| 4.14 | telegram, persistence, service |
| 4.15 | billing, telegram |
| 4.16 | billing |
| 4.17 | platform |
| 4.18 | platform |
| 4.19 | billing, platform |
| 4.20 | telegram, platform |
| 4.21 | platform, service |

For work that is not a numbered step (bugfix on current bot): `AGENTS.md` + `requirements.md` + the area that owns the package (usually `telegram` and/or `service`). Still skip `legacy/` and skip 4.x files unless the bug is a Stage 4 hook.

## Data flow (doc ownership)

```text
wake-up → AGENTS.md
            ├─ status / protocol
            ├─ routing row → requirements.md
            │                 area file(s)
            │                 steps/4.xx-*.md
            └─ skip → legacy/
```

Live docs describe **now**. Step files describe **next for that id**. Legacy describes **how we got here**. If a fact is needed to implement the current step, it must live in requirements, an area, or that step file — not only in legacy.

When a 4.x step is `done`, fold the new “how it works now” into the relevant area file, mark the step `done`, and do not leave the only copy of current behavior in the step file.

## Error handling / drift

- Two sources of truth for the same live rule is a bug. Prefer area/requirements; step files hold upcoming work.
- `AGENTS.md` current step must match the step file whose status is the pointer (`todo` or `to review`).
- Links to `plan.md` after deletion are broken — grep and fix (`README.md`, specs that point at plan).
- Existing specs under `docs/superpowers/specs/` may still mention `plan.md`; update those pointers to the new paths.

## Testing / verification

No application tests. After the split:

1. Every heading/section in today’s `plan.md` maps to exactly one live file, one legacy file, or “dropped as duplicate of README”.
2. No live pointer to `plan.md` (`README.md`, `AGENTS.md`, `docs/sdd/**` except `legacy/`, `docs/superpowers/specs/`). `legacy/` may mention the old filename.
3. Each of 4.1–4.21 exists as its own file with the same Goal/DoD text as today.
4. `AGENTS.md` routing lists every 4.x step.
5. `legacy/README.md` states the skip rule in the first paragraph.
6. Wake-up for 4.1 does not require opening telegram, billing, or stages-0-3.

## `plan.md` section map

| Today’s `plan.md` section | Destination |
| --- | --- |
| Status header, agent protocol, how to pick up work, suggested next message | `AGENTS.md` |
| Vision, Stage 3 vs 4 placement, locked product decisions, entitlement order | `requirements.md` |
| Architecture, driven deps, layer rules, domain sketches, Transactor/write-path | `areas/service.md` |
| Data model tables, Total query | `areas/persistence.md` |
| Telegram surface, hygiene, pending, emojis | `areas/telegram.md` |
| Stage-4 hooks that exist (plan/trial/discount/entitlement/referral) | `areas/billing.md` |
| Tech choices (SQLite/driver), what not to do in Stage 3 (trim to still-true) | `areas/persistence.md` + `areas/platform.md` + step links |
| Local run, CI | README (canonical) + `areas/platform.md` (short) |
| Decisions log | `legacy/decisions-log.md` |
| Steps 0.1–3.11 | `legacy/stages-0-3.md` |
| Stage 4 intro | `steps/README.md` |
| Steps 4.1–4.21 | `steps/4.xx-*.md` |
| Already-executed “later rewrite” notes | `legacy/superseded.md` |

## Implementation notes

- Copy text; do not rewrite Stage 4 stubs while moving them.
- Trim live docs: remove “until telegram exists”, “no Transactor in 3.6”, and other already-executed later-rewrites from requirements/areas.
- Keep future rewrites as one line + step link (`Cache` is memory; Redis is `4.2`).
- Do not commit unless the user asks (repo convention).
