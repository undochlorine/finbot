# SDD framework restructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `plan.md` with a routed SDD tree (`AGENTS.md` + `docs/sdd/`) so a wake-up reads only the files needed for the current step.

**Architecture:** Copy content out of `plan.md` along the mapping in `docs/superpowers/specs/2026-09-09-sdd-framework-design.md`. Live docs describe now; `docs/sdd/steps/` holds 4.x stubs unchanged; `docs/sdd/legacy/` holds finished stages and superseded intent behind a skip banner. Delete `plan.md`.

**Tech Stack:** Markdown only. No Go, no new runtime deps.

## Global Constraints

- Do not change application code or product behavior.
- Do not widen Stage 4 stubs; copy Goal/Notes/DoD/Depends as they are in `plan.md` today.
- Do not invent Stage 4 technical choices.
- `AGENTS.md` is the only always-read file.
- Do not tell agents to read `docs/sdd/legacy/` unless the user asks, a live doc points at a superseded decision, or history of a superseded decision is required.
- Future rewrites still open (SQLite→Postgres, memory→Redis, Entitlement always-allow→4.5, one `cmd/bot`→4.21) stay as one-liners in live area/step files, not in `legacy/superseded.md`.
- Already-executed rewrites (ports→consumer interfaces, telegram owns Cache, Transactor replacing “no unit of work”) go only in `legacy/superseded.md`.
- README remains canonical for human run/CI; `areas/platform.md` is a short agent summary that links to README.
- No empty area files for i18n/growth/web.
- Branch is not `master`/`main`. Name: `feature/asicaci/sdd-framework`, created from latest `origin/master`.
- Filenames: `docs/sdd/steps/4.01-postgresql.md` … `4.21-modular-binaries.md` (zero-padded), plus `docs/sdd/steps/README.md`.
- Status values: `todo` | `to review` | `done`. Current step `4.1`, last done `3.11`.
- After the split, no live pointer to `plan.md` in `README.md`, `AGENTS.md`, `docs/sdd/**` except `legacy/`, or `docs/superpowers/specs/`. Mentions inside the SDD design spec and this plan that describe the deletion are allowed. Completed historical specs must not send an agent to a missing `plan.md` file.
- Copy 4.6’s “written strategy in this plan.md” to “written strategy in `docs/sdd/legacy/decisions-log.md` plus a short subsection on this step file” so the stub does not point at a deleted file.
- Do not commit on `master`.

---

### Task 1: Branch off latest origin/master

**Files:** none yet (git only). The untracked spec `docs/superpowers/specs/2026-09-09-sdd-framework-design.md` must come along.

**Interfaces:**
- Consumes: `origin/master`
- Produces: local branch `feature/asicaci/sdd-framework` checked out, not `master`

- [ ] **Step 1: Fetch and create the branch**

```bash
git fetch origin
git checkout -b feature/asicaci/sdd-framework origin/master
git branch --show-current   # expect feature/asicaci/sdd-framework
```

If the spec file is missing after checkout, copy it from the previous working tree.

- [ ] **Step 2: Confirm HEAD is not master**

```bash
test "$(git branch --show-current)" != master
test "$(git branch --show-current)" != main
```

Expected: both tests exit 0.

---

### Task 2: `AGENTS.md` (always-read entry)

**Files:**
- Create: `AGENTS.md`
- Source: `plan.md` lines 1–33 and 995–997 (status, protocol, pickup)

**Interfaces:**
- Consumes: status fields from `plan.md` header
- Produces: routing table every later task must match (step → areas → step file)

- [ ] **Step 1: Write `AGENTS.md`**

Keep it short. Must include: skip-legacy rule in the first screen; status table (Current step `4.1`, Last done `3.11` same-currency `/transfer`, MVP target private-use Telegram finance bot in Go + SQLite, GitHub `undochlorine/finbot`, Go module `finbot`, Path `/Users/a.sicaci/Projects/finbot`); protocol (one step, mark `to review`, update current/last, `make lint` when Go changes, no commit/push/remote unless asked, mockery + table tests + temp SQLite integration); routing table below; pointers to `docs/sdd/requirements.md`, `README.md`, `docs/superpowers/specs/`, `docs/sdd/steps/README.md`.

Wake-up line: `Follow AGENTS.md. Let's move to step 4.1`

Routing (always also read `docs/sdd/requirements.md` + the step file):

| Step | Extra areas | Step file |
| --- | --- | --- |
| 4.1 | persistence | `docs/sdd/steps/4.01-postgresql.md` |
| 4.2 | telegram, platform | `docs/sdd/steps/4.02-redis.md` |
| 4.3 | platform | `docs/sdd/steps/4.03-hosting.md` |
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

Bugfix (not a numbered step): `AGENTS.md` + `requirements.md` + owning area (usually telegram and/or service). Skip `legacy/` and skip 4.x unless the bug is a Stage 4 hook.

Area paths are `docs/sdd/areas/<name>.md`.

- [ ] **Step 2: Confirm 4.1 routing does not list telegram, billing, or legacy**

```bash
python3 - <<'PY'
from pathlib import Path
t = Path("AGENTS.md").read_text()
assert "docs/sdd/legacy" in t
assert "Do not read `docs/sdd/legacy/`" in t or "do not read `docs/sdd/legacy/`" in t.lower() or "Do not read" in t
assert "4.1" in t and "persistence" in t
print("AGENTS.md present and mentions skip + 4.1")
PY
```

---

### Task 3: `docs/sdd/requirements.md`

**Files:**
- Create: `docs/sdd/requirements.md`
- Source: `plan.md` Vision (35–106), Locked product decisions (109–139), Entitlement order (344–356)

**Interfaces:**
- Consumes: product rules from `plan.md`
- Produces: the business-rules file every 4.x step also reads

- [ ] **Step 1: Write requirements.md**

Include: short vision (banks, include-in-total, Telegram); stages 0–3 done / stage 4 public product one-liners; Stage 3 vs Stage 4 placement table (copy from `plan.md`); mermaid stage flowchart is optional (keep if it stays short); locked product decisions that are still true; entitlement **order** (trial → 100% whitelist → paid → 1–99% → limited free tier).

Must not include: package tree, Go struct sketches, env/CI, step DoDs, “until telegram exists”, “no Transactor in 3.6”.

Chat hygiene: one line + link to `docs/sdd/areas/telegram.md`. Pending queue: one line + same telegram file. Emojis: one line + telegram file. Money/currency: Stage 3 UX is one currency; `4.13`/`4.14` for picker/FX. Entitlement **enforcement** is `4.5`; point at `docs/sdd/areas/billing.md` for the always-allow implementation status.

- [ ] **Step 2: Grep that it has no package tree or `cmd/bot/main.go`**

```bash
! grep -n "cmd/bot/main.go" docs/sdd/requirements.md
! grep -n "type Bank struct" docs/sdd/requirements.md
```

---

### Task 4: Area files (current technical state)

**Files:**
- Create: `docs/sdd/areas/telegram.md`
- Create: `docs/sdd/areas/persistence.md`
- Create: `docs/sdd/areas/service.md`
- Create: `docs/sdd/areas/billing.md`
- Create: `docs/sdd/areas/platform.md`
- Source: `plan.md` architecture, data model, telegram surface, tech choices, layer rules, local run, CI, stage-4 hooks

**Interfaces:**
- Consumes: current-state sections of `plan.md`
- Produces: five area files; future work is one-liner + step link only

- [ ] **Step 1: `telegram.md`**

Copy: command table, shortcuts, callback versioning, empty state, chat hygiene (keep/remove/mechanism), error/edge cases, pending FIFO + compact table, emojis table, FSM/Cache as they work now (telegram owns Cache; `ports.Cache` remains so memorycache does not import telegram; TTL 10 minutes; not the pending FIFO).

One-liner: production Cache is in-process memory; Redis is `docs/sdd/steps/4.02-redis.md`. Do not paste Redis how-to or `/history` UX.

- [ ] **Step 2: `persistence.md`**

Copy: `users` / `banks` / `operations` columns (locale, currency, plan, trial_ends_at, discount_percent, referred_by as **schema only**), Total SQL, SQLite driver `modernc.org/sqlite`, WAL, numbered SQL migrations at startup.

One-liner: Postgres adapter is `docs/sdd/steps/4.01-postgresql.md`. Do not describe trial/paywall semantics (that is billing).

- [ ] **Step 3: `service.md`**

Copy: package tree, hexagonal / interface-per-consumer, driven dependencies (who owns which interface **as code is now**), Transactor + operations write path, domain sketches matching code now, layer rules, mockery rules.

Trim: “Cache and Notifier stay in ports until telegram exists” — replace with current fact (telegram owns Cache/HTTPClient; ports still has Cache so memorycache does not import telegram; Notifier still in ports). Do not add payment ports.

- [ ] **Step 4: `billing.md`**

Semantics only: Plan stored, trial frozen at signup, DiscountPercent nil/0–100, Entitlement helper always returns full access, referred_by set-if-null from `/start`, TRIAL_DURATION loaded but not enforced. Point enforcement/checkout/admin at 4.5–4.8 step files. No column DDL lists (persistence owns those).

- [ ] **Step 5: `platform.md`**

Short env table (BOT_TOKEN, SQLITE_PATH, LOG_LEVEL, TRIAL_DURATION, ADMIN_TELEGRAM_ID, DEFAULT_CURRENCY), Dockerfile exists, CI jobs `unit:test` / `lint` / `integration:test` / gate `common`. Link to README as canonical run checklist. One-liners: hosting `4.03`, load/replicas `4.09`, extra `cmd/` `4.21`. Do not choose Fly/Railway/Hetzner.

- [ ] **Step 6: Confirm no empty extra areas**

```bash
ls docs/sdd/areas
# expect exactly: billing.md persistence.md platform.md service.md telegram.md
```

---

### Task 5: Stage 4 step files (verbatim stubs)

**Files:**
- Create: `docs/sdd/steps/README.md`
- Create: `docs/sdd/steps/4.01-postgresql.md` through `4.21-modular-binaries.md`
- Source: `plan.md` 850–991

**Interfaces:**
- Consumes: Stage 4 intro + each `#### 4.x` block
- Produces: 21 step files + index; 4.01 is PostgreSQL only

- [ ] **Step 1: Write `docs/sdd/steps/README.md`**

Include: architecture already shaped so these are adapter/job additions; leave `todo` until the user starts the stage; high-level tech stays out (same bar as 4.6); monetization sequence 4.5 → 4.6 → 4.7 → 4.8; keep 4.1–4.3, 4.6, 4.7, 4.9 as written; product after money 4.10–4.21; a table of 4.1–4.21 with status `todo` and links to the files.

- [ ] **Step 2: Split each `#### 4.x` block into its file**

Preserve Status/Goal/Notes/DoD/Depends text. In `4.06-payment-strategy.md` only, replace the live `plan.md` pointer as specified in Global Constraints.

Filenames:

```
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
```

Each file starts with `# 4.x Title` matching the heading (e.g. `# 4.1 PostgreSQL adapter`).

- [ ] **Step 3: Confirm 21 files and 4.1 has no monetization sequence dump**

```bash
ls docs/sdd/steps/4.*.md | wc -l   # 21
! grep -n "Monetization sequence" docs/sdd/steps/4.01-postgresql.md
grep -n "implement the same repository ports on Postgres" docs/sdd/steps/4.01-postgresql.md
```

---

### Task 6: Legacy archive

**Files:**
- Create: `docs/sdd/legacy/README.md`
- Create: `docs/sdd/legacy/decisions-log.md`
- Create: `docs/sdd/legacy/stages-0-3.md`
- Create: `docs/sdd/legacy/superseded.md`
- Source: `plan.md` 550–580 (decisions), 584–847 (steps 0.1–3.11)

**Interfaces:**
- Consumes: finished steps + dated log + superseded intent
- Produces: skippable archive; live docs must not be the only place for facts needed to implement 4.x

- [ ] **Step 1: `legacy/README.md`**

First paragraph must state: historical only; do not read for implementation, current behavior, or the next 4.x step; read only if the user asks, a live doc points at a superseded decision, or you must explain why old code looks like X. Then list the three files.

- [ ] **Step 2: `decisions-log.md`**

Copy the dated table from `plan.md` as-is (including the header row).

- [ ] **Step 3: `stages-0-3.md`**

Copy Stage 0–3 step blocks (0.1–3.11) including Status/Goal/Files/DoD and the closer “Hardened private MVP is complete when 0.1–3.11 are `done`.” Prefix the file with one line: archived completed steps; current work is in `AGENTS.md` and `docs/sdd/steps/`.

- [ ] **Step 4: `superseded.md`**

Only already-executed later-rewrites:

1. Repository/clock interfaces moved from `ports` to the consumer; tests mock the consumer’s interfaces.
2. Telegram owns `Cache`; `ports.Cache` remains so memorycache does not import telegram.
3. `Transactor` lands in 3.6 (supersedes the earlier “no unit of work in 3.6” cut).
4. Stage 3 items that were “later” and have shipped: `/feedback`, chat hygiene, pending FIFO, compact, emojis, `/rename`, same-currency `/transfer`.

Do **not** list SQLite→Postgres, memory→Redis, always-allow→4.5, one binary→4.21.

---

### Task 7: README, historical spec pointers, delete `plan.md`

**Files:**
- Modify: `README.md` (plan.md links)
- Modify: `docs/superpowers/specs/2026-09-08-write-path-foundations-design.md` (plan.md as a live path)
- Modify: `docs/superpowers/specs/2026-09-08-same-currency-transfer-design.md` if it points at `plan.md`
- Delete: `plan.md`

**Interfaces:**
- Consumes: new paths from Tasks 2–6
- Produces: no live `plan.md` file; README points at `AGENTS.md` + requirements + telegram hygiene

- [ ] **Step 1: Update README**

Replace “Details and product decisions live in `plan.md`” with `AGENTS.md` (agents) and `docs/sdd/requirements.md` (product rules).

Replace chat-hygiene link `plan.md#chat-hygiene-35` with `docs/sdd/areas/telegram.md`.

- [ ] **Step 2: Update completed specs so they do not send an agent to a missing file**

In `2026-09-08-write-path-foundations-design.md`, change remaining `plan.md` path mentions to “former `plan.md`, now `AGENTS.md` + `docs/sdd/` (status for 3.6 is historical; decisions live in `docs/sdd/legacy/decisions-log.md`)”. Do not rewrite the whole spec.

Same-currency spec: grep; only change if it links to `plan.md` as a current path.

Leave `2026-09-09-sdd-framework-design.md` as the design of this split (it may mention `plan.md` as the file being deleted).

- [ ] **Step 3: Delete `plan.md`**

```bash
rm plan.md
test ! -f plan.md
```

---

### Task 8: Verification

**Files:** none new

- [ ] **Step 1: Mapping and routing**

```bash
test -f AGENTS.md
test ! -f plan.md
test -f docs/sdd/requirements.md
test -f docs/sdd/steps/README.md
test -f docs/sdd/legacy/README.md
ls docs/sdd/steps/4.*.md | wc -l   # 21
ls docs/sdd/areas | wc -l          # 5
```

- [ ] **Step 2: No live plan.md pointers (except design spec, this plan, and legacy)**

```bash
rg -n 'plan\.md' --glob '!docs/superpowers/specs/2026-09-09-sdd-framework-design.md' --glob '!docs/superpowers/plans/**' --glob '!docs/sdd/legacy/**'
```

Expected: no matches. If the write-path spec still mentions the phrase “former plan.md”, that is OK only if it does not present `plan.md` as a file to open. Tighten until agents cannot follow a link to `plan.md`.

- [ ] **Step 3: Skip banner and 4.1 isolation**

```bash
head -n 5 docs/sdd/legacy/README.md
# first paragraph includes do not read / historical
grep -n "4.1" AGENTS.md
# 4.1 row lists persistence and 4.01-postgresql, not telegram, not billing, not stages-0-3
```

- [ ] **Step 4: 4.x stubs still have Goal text**

```bash
grep -l "**Goal:**" docs/sdd/steps/4.*.md | wc -l   # 21
```

---

## Self-review vs spec

- Wake-up, tree, AGENTS.md contents, requirements vs areas vs steps vs legacy, routing table, README canonical, no empty i18n/growth/web areas, delete plan.md, 4.06 pointer fix, verification grep: each has a task.
- No TBD/TODO placeholders.
- Step filenames match the spec tree.
