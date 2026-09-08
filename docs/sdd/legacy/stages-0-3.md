Archived completed steps (0.1–3.11). Current work is in `AGENTS.md` and `docs/sdd/steps/`.
Do not use this file to decide what to implement next.

## Steps

Implement one id at a time. Update the status field in place.

---

### Stage 0 — Setup

#### 0.1 Go module and directory skeleton

- **Status:** `done`
- **Goal:** `go mod init finbot` and empty package dirs so later steps have a home.
- **Files:** `go.mod`; `cmd/bot/`; `internal/{domain,ports,service,config,text,adapter/telegram,adapter/sqlite,adapter/memorycache,adapter/clock}/`
- **DoD:** `go mod init` succeeded; dirs exist; no application logic yet.

#### 0.2 Env config and gitignore

- **Status:** `done`
- **Goal:** 12-factor config from env; secrets never committed.
- **Files:** `internal/config/config.go`; `.env.example`; `.gitignore` (binaries, `.env`, `data/*.db`, IDE)
- **DoD:** config loads `BOT_TOKEN` (required; from process env or `.env`), `SQLITE_PATH` (default `./data/finbot.db`), `LOG_LEVEL` (`debug`/`info`/`warn`/`error`, default `info`), `TRIAL_DURATION` (default `168h`; `0` = no trial; negative rejected). Missing token is a clear startup error. `.env` is gitignored. Trial duration is stored for later; **do not enforce a paywall**.

#### 0.3 Dockerfile

- **Status:** `done`
- **Goal:** multi-stage image for later cheap hosting; not deployed in MVP.
- **Files:** `Dockerfile`; `.dockerignore`
- **DoD:** image builds a static-ish Go binary; SQLite path and token via env; data dir is a volume path.

#### 0.4 README skeleton

- **Status:** `done`
- **Goal:** enough for a human (or future agent) to know what this is and how to run it later.
- **Files:** `README.md`
- **DoD:** purpose, architecture one-liner, pointer to `plan.md`, env vars listed (including `TRIAL_DURATION`). Full run checklist can wait for `3.3`.

#### 0.5 GitHub Actions CI (common stage)

- **Status:** `done`
- **Goal:** a pipeline that always runs unit tests, lint, and integration tests before any later stage (build/deploy not added yet).
- **Files:** `.github/workflows/ci.yml`; `Makefile` (`test-unit` / `test-integration`); `internal/adapter/sqlite/*_test.go` (`//go:build integration`); README + this plan
- **DoD:** workflow has jobs `unit:test`, `lint`, `integration:test` under a `common` gate. Any failed common job makes `common` fail, so future jobs with `needs: common` do not run. No GitHub secrets required for this stage.

---

### Stage 1 — Domain and persistence

#### 1.1 Domain entities and errors

- **Status:** `done`
- **Goal:** `User` (including `TrialEndsAt`, `DiscountPercent`), `Bank`, `Money` helpers, domain errors (`ErrBankNotFound`, `ErrBankNameTaken`, `ErrInvalidAmount`, …).
- **Files:** `internal/domain/*.go` (+ tests for money parse/format)
- **DoD:** parse/format cents covered by table tests; no IO.

#### 1.2 Ports

- **Status:** `done`
- **Goal:** interfaces only (later split per consumer; see architecture + 1.8).
- **Files:** `internal/ports/*.go`
- **DoD:** `Cache`, `Notifier` remain here until telegram exists. Repository and clock interfaces live on their consumers. IO methods take `context.Context`.

#### 1.3 SQLite migrations

- **Status:** `done`
- **Goal:** `users` + `banks` tables; apply on open.
- **Files:** `internal/adapter/sqlite/migrations/*.sql`; migrate runner
- **DoD:** opening a new DB creates schema (`users` includes `trial_ends_at` and `discount_percent`; `banks` as specified); re-open is idempotent; unique per-user bank name enforced.

#### 1.4 SQLite repositories

- **Status:** `done`
- **Goal:** concrete `BankRepository` and `UserRepository`.
- **Files:** `internal/adapter/sqlite/*.go`
- **DoD:** all queries include `user_id` where data is per-user; WAL enabled.

#### 1.5 Memory cache adapter

- **Status:** `done`
- **Goal:** in-process `Cache` with TTL for FSM keys.
- **Files:** `internal/adapter/memorycache/*.go` (+ tests)
- **DoD:** set/get/delete; expired keys treated as miss.

#### 1.6 Clock adapter

- **Status:** `done`
- **Goal:** real `time.Now` behind a `Clock` interface owned by each consumer; tests can fake it later.
- **Files:** `internal/adapter/clock/*.go`

#### 1.7 Service use cases

- **Status:** `done`
- **Goal:** create/add/spend/set/delete/get/list/total/all/toggle; upsert user (set `TrialEndsAt` only on insert); touch activity.
- **Files:** `internal/service/*.go`
- **DoD:** no Telegram/SQL/`ports` imports. Duplicate name and not-found map to domain errors. Total sums only `IncludeInTotal` banks.

#### 1.8 Service unit tests

- **Status:** `done`
- **Goal:** table-driven tests of business rules with mockery-generated **service** mocks.
- **Files:** `internal/service/*_test.go`; `.mockery.yml`; `internal/service/mocks/`
- **DoD:** covers create, duplicate name, add/spend/set (including negative add/spend invalid), delete, toggle, total vs excluded bank, empty list. Tests depend on `internal/service/mocks`, not `ports/mocks`. No hand-written repository/clock stubs. Tests pass (`go test ./internal/service/...`).

#### 1.9 SQLite integration tests

- **Status:** `done`
- **Goal:** real temp DB file (or `:memory:` if WAL/constraints still apply — prefer temp file to match persistence).
- **Files:** `internal/adapter/sqlite/*_test.go` (`//go:build integration`; run with `make test-integration`)
- **DoD:** two users cannot see each other’s banks; unique name; restart-open of the same file still has data. Tests pass (`make test-integration`). No docker-compose.

---

### Stage 2 — Telegram bot (MVP product)

Numbering is `2.x` for the Telegram stage (not “stage 2” of the product roadmap; that is `4.x`).

#### 2.1 Bot wiring

- **Status:** `done`
- **Goal:** `main` loads config, opens SQLite, builds services, starts long polling, graceful shutdown.
- **Files:** `cmd/bot/main.go`; `internal/adapter/telegram/` bootstrap
- **DoD:** process starts with a token; exits non-zero if token/DB missing. No feature complete yet except process health.

#### 2.2 Activity + user upsert middleware

- **Status:** `done`
- **Goal:** every inbound update upserts the user and sets `LastActivityAt`. First insert sets `TrialEndsAt = now + TRIAL_DURATION` and must not overwrite it later.
- **Files:** telegram middleware / handler wrapper
- **DoD:** first `/start` creates a `users` row with `trial_ends_at` populated. Second message does not extend the trial.

#### 2.3 `/start` and `/help`

- **Status:** `done`
- **Goal:** English copy from `internal/text`.
- **Files:** `internal/text`, telegram handlers
- **DoD:** both commands reply; help lists all MVP commands.

#### 2.4 `/newbank` flow

- **Status:** `done`
- **Goal:** FSM: name (reject duplicates immediately) → include-in-total buttons; name-only shortcut if args present (spaces kept; no yes/no on the command line).
- **DoD:** bank persisted; duplicate name errors in chat as soon as the name is entered; `/start` and `/help` mention command+name shortcuts.

#### 2.5 `/add`, `/spend`, `/set`

- **Status:** `done`
- **Goal:** bank picker buttons or args; then amount; cache FSM.
- **DoD:** balances change correctly; invalid amount re-prompted; empty banks → hint `/newbank`.

#### 2.6 `/delete`

- **Status:** `done`
- **Goal:** pick bank, confirm, delete entirely.
- **DoD:** bank gone from list; confirm step avoids accidental taps.

#### 2.7 `/bank`, `/banks`, `/total`, `/all`

- **Status:** `done`
- **Goal:** read-only views; `/all` = list + total of included banks.
- **DoD:** excluded banks show in list but not in total.

#### 2.8 `/toggle`

- **Status:** `done`
- **Goal:** flip `include_in_total`; reply with new state.
- **DoD:** `/total` changes after toggle without changing balances.

#### 2.9 Conversation FSM polish

- **Status:** `done`
- **Goal:** TTL expiry, cancel (`/cancel`), ignore stale callback data, don’t mix two in-flight flows.
- **DoD:** starting `/add` then `/spend` replaces the pending flow; expired state asks the user to start over.

---

### Stage 3 — MVP harden

#### 3.1 Lint

- **Status:** `done`
- **Goal:** `make lint` → 0 issues. Config is `.golangci.yaml` in this repo.
- **DoD:** command run on the module; issues fixed.

#### 3.2 Persistence check

- **Status:** `done`
- **Goal:** kill and restart the process against the same `SQLITE_PATH`; banks remain.
- **DoD:** documented in README how to verify; agent or user actually ran it once.

#### 3.3 Local run docs

- **Status:** `done`
- **Goal:** BotFather create-bot + token, env, `go run ./cmd/bot`. The slash command menu is registered by the process (`setMyCommands`), not by BotFather.
- **Files:** `README.md`
- **DoD:** a new machine can run MVP from README + a token.

#### 3.4 Feedback (forward)

- **Status:** `done`
- **Goal:** `/feedback` asks for text, then forwards user id + username + body to `ADMIN_TELEGRAM_ID` via `Notifier`, then thanks the user.
- **Files:** `internal/config`; `internal/text`; `internal/adapter/telegram/`; `internal/ports` (`Notifier` already exists)
- **DoD:** if `ADMIN_TELEGRAM_ID` is unset, `/feedback` says it is unavailable. No `feedback` table. Command is registered in the slash menu. Unit tests cover the forward payload and the unset-admin path.

#### 3.5 Chat hygiene

- **Status:** `done`
- **Goal:** lower chat flooding so history is commands + results, not wizard debris. Policy: [Chat hygiene](#chat-hygiene-35).
- **Notes:** FSM stores `prompt_id` and `sweep_ids`. Success edits the prompt into the confirmation (no keyboard) and deletes typed answers. `/cancel` and replaced flows delete the wizard. Expired callbacks edit that message to expired.
- **DoD:** completing `/add` (picker → amount) does not leave a live keyboard on the old prompt. `/cancel` clears the prompt. Typed amount is deleted; `/add` is kept. Tests cover `prompt_id` on FSM and edit/delete on completion.

#### 3.6 Write-path foundations

- **Status:** `done`
- **Goal:** one SQLite migration + domain/service hooks that Stage 4 will read. Telegram UX unchanged except `/start` may persist a referral payload.
- **Schema:** `operations` (append-only: `add`/`spend`/`set`/`delete`/`transfer`); `users.locale` default `en`; `users.referred_by` nullable; `banks.currency` NOT NULL default from `DEFAULT_CURRENCY`.
- **Also:** locale-keyed `internal/text` with **only English**; always-allow `Entitlement` helper; structured `slog` fields (`user_id`, command); `Transactor` for atomic mutators; `meta` snapshots the typed bank name.
- **DoD:** add/spend/set/delete write an operation row **in the same transaction** as the bank change. First valid `/start <payload>` stores `referred_by` once (set-if-null) and does not overwrite later. Existing users get locale `en` and default currency on new banks. Entitlement tests: always full access. **No** `/history`, `/language`, currency picker, or paywall.

#### 3.7 Per-user FIFO pending commands

- **Status:** `done`
- **Goal:** a burst of updates for one user (process down, lagging, or a fat `getUpdates` batch) runs in send order. Still handle **all** of them. Do not collapse yet (`3.8`).
- **Why:** `github.com/go-telegram/bot` v1.25 does `go handler` per update. Middleware inside that goroutine races enqueue. `WithNotAsyncHandlers` keeps enqueue on the poll worker; the FIFO still drains async per user.
- **Notes:** Implement `PendingCommands` in `internal/adapter/telegram` (new file). Middleware on the inner bot — `Start` never hits our `ProcessUpdate` wrapper. One FIFO + one drain goroutine per busy user; other users are not blocked. Enqueue slash commands, typed FSM answers, and callbacks. Cap 32 waiting items; drop oldest waiting + `slog.Warn`. Not Cache, not SQLite. Policy: [Pending commands](#pending-commands-37--38).
- **Files:** `internal/adapter/telegram/` (`pending.go` + tests); wire middleware in `bot.go` so production is covered. Tests may keep `WithNotAsyncHandlers`; the queue must still be correct if both are on.
- **DoD:** two money shortcuts for the same user, started as concurrent library handlers would, apply in enqueue order (`/add Travelling 100` then `/add Travelling 50` → +150). A `/help` then `/banks` burst replies help then banks, never the reverse. Two different users may proceed without waiting on each other. Test the queue type **directly** (ordered drain of three jobs) as well as through handlers. Collapse is **not** implemented. `make lint` and unit tests pass.

#### 3.8 Collapse redundant queued commands

- **Status:** `done`
- **Goal:** still handle the burst, but drop only the obvious redundant/invalid **waiting** slash commands. Do not invent extra heuristics.
- **Notes:** Pure `compact` on the waiting list. Rules are locked in [Pending commands](#pending-commands-37--38). No bank-state simulation. No extra “skipped” chat line. Depends on `3.7`.
- **Files:** `internal/adapter/telegram/` (`compact` next to pending; table tests)
- **DoD:** table tests for every locked rule, plus negatives (`/add Travelling 100` twice both kept; `/help` `/banks` `/help` keeps both helps; `/newbank` then `/cancel` keeps both). A burst `/help` `/help` `/help` produces one help reply. `make lint` and unit tests pass.

#### 3.9 Outcome emojis

- **Status:** `done`
- **Goal:** a few office/finance emojis so chat feels less sterile. Not an emoji on every line.
- **Notes:** Baseline in [Emojis](#emojis-39). An implementer **may add more** of the same family (📊 🏦 💼 💰 📁 🧾 📌, …) — do not treat the table as a closed list. All strings stay in `internal/text`. Do not change handler logic except copy. `/rename` copy can wait for `3.10` if that string does not exist yet.
- **Files:** `internal/text`; tests that pin the strings that land
- **DoD:** baseline outcomes from the table are present (`/start` 👋, add/spend 💸, create/set/toggle ✅, delete 🗑️, feedback thanks 🙏). Extra office/finance emojis are allowed if they stay sparse. `/help` lines, errors, and prompts have no emoji. Existing telegram tests that match full copy are updated. `make lint` and unit tests pass.

#### 3.10 `/rename`

- **Status:** `done`
- **Goal:** rename one of the user’s banks without deleting it.
- **Flow:** pick bank (buttons or `/rename Travelling`) → type the new name. Duplicate of **another** bank errors immediately and stays on the name step (same as `/newbank`). Recasing the same bank (`Holiday` → `holiday`) succeeds and stores the new spelling. Empty name re-prompts. No banks → empty-state hint `/newbank`. Slash args try the full remainder as the current name first; if that bank is missing, `/rename Travelling Holiday` renames `Travelling` → `Holiday`. Chat hygiene: one edited prompt; typed new name is swept; outcome kept.
- **Service:** `Rename(ctx, userID, bankID, newName) (Bank, error)` uses existing `BankRepository.Update`, `Transactor`, and an `operations` row: type `rename`, `amount_cents` 0, `balance_after_cents` current balance, `meta` `Old -> New` (old stored name, new name as typed). SQLite `CHECK` on `operations.type` must allow `rename` (rebuild the table in a new numbered migration; SQLite cannot ALTER CHECK).
- **Also:** `domain.CommandRename`, slash menu, `/help` `/start` mention, callback `v1:rename:<bankID>`. Outcome may use ✏️ or another office/finance emoji from `3.9`. Entitlement later (`4.5`) treats `/rename` as an extra-bank command.
- **Files:** domain, service (+ mockery if the consumer interface grows), sqlite migration + repo tests, `internal/text`, telegram handlers/tests, `commands.go`
- **DoD:** rename persists and is unique per user (case-insensitive). Recase of the same bank works. Name taken by another bank does not overwrite. Operation row is written in the same transaction as the update. Command is in the slash menu and `/help`. Unit tests cover service rules; telegram tests cover picker, shortcut, duplicate name, empty name, empty-bank state. `make lint`, `make test-unit`, `make test-integration` pass.

#### 3.11 Same-currency `/transfer`

- **Status:** `done`
- **Goal:** move money between the user’s own banks in the same currency.
- **Flow:** pick from-bank → to-bank (other banks only) → amount. Empty `/transfer` is an empty-wizard start (`3.8`). 0 banks → `NoBanks` / `/newbank`. 1 bank → need another `/newbank`. Shortcut: exactly `From To Amount` (two words + money) runs immediately if both banks exist; if a bank is missing, or the payload is not that shape, the full remainder is the from-name (no trailing amount peel). Typed wizard answers never use the triple shortcut. Chat hygiene: one edited prompt; typed to-name/amount swept; outcome kept.
- **Service:** `Transfer(ctx, userID, fromID, toID, amount) (from, to Bank, error)` uses existing `Transactor`. Debit from, credit to, one `operations` row: type `transfer`, `amount_cents` positive amount moved, `balance_after_cents` from’s new balance, `meta` counterpart (to) bank id. Reject same bank, different currency, invalid amount (negative, zero, overflow). Negative balances allowed. SQLite CHECK already allows `transfer`.
- **Also:** `domain.CommandTransfer`, slash menu, `/help` `/start` mention, callbacks `v1:transfer:from:<bankID>` and `v1:transfer:to:<bankID>`. Outcome uses 💸. Entitlement later (`4.5`) treats `/transfer` as an extra-bank command.
- **Files:** domain, service (+ mockery on telegram `Service`), sqlite integration tests, `internal/text`, telegram handlers/tests, `commands.go`, `compact.go`
- **DoD:** atomic in service via `Transactor` from `3.6` (two balance updates + one operation row). Reject different currency, same bank, invalid amount. Unit tests cover those cases. Command is in the slash menu and `/help`.

**Hardened private MVP is complete when 0.1–3.11 are `done`.** Through `3.3` the bot is already usable privately; `3.4`–`3.11` are still Stage 3 (no paywall, full banks for everyone).
