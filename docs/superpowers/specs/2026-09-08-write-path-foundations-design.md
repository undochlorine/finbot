# 3.6 Write-path foundations

Stage 3 schema and service hooks that Stage 4 will read. Telegram UX is unchanged except `/start` may persist a referral payload. No `/history`, `/language`, currency picker, or paywall.

## Goal

After this step:

1. Add / spend / set / delete each append one `operations` row.
2. First valid `/start <telegram-id>` stores `users.referred_by` once; later values do not overwrite.
3. Existing users get `locale = en`. New banks get `DEFAULT_CURRENCY` (default `USD`). Existing banks get `USD` from the migration default.
4. `Entitlement` always returns full access.
5. User-facing copy is locale-keyed with only `en` loaded.
6. Handler logs include `user_id` and `command`.

## Non-goals

- `/history`, `/language`, currency in copy, paywall, entitlement enforcement in handlers
- Writing `transfer` operations (schema allows the type; 3.7 writes the rows)
- Referral rewards (4.15)
- Extra language catalogs (4.12)

## Architecture

Approach A: the service owns `OperationRepository` and `Transactor`. Mutators update the bank and append in one transaction.

```text
Telegram  →  handlers  →  service mutators  →  Transactor.InTx
                                         ↘  BankRepository
                                         ↘  OperationRepository.Append
              /start payload  →  UserRepository.SetReferredByIfEmpty
```

Service depends on domain + its own interfaces only. It does not import `ports`, SQL, or Telegram.

## Schema

One numbered migration: `internal/adapter/sqlite/migrations/002_write_path_foundations.sql`.

```sql
ALTER TABLE users ADD COLUMN locale TEXT NOT NULL DEFAULT 'en';
ALTER TABLE users ADD COLUMN referred_by INTEGER;

ALTER TABLE banks ADD COLUMN currency TEXT NOT NULL DEFAULT 'USD';

CREATE TABLE operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users (telegram_id) ON DELETE CASCADE,
    bank_id INTEGER REFERENCES banks (id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    balance_after_cents INTEGER,
    meta TEXT,
    created_at TEXT NOT NULL,
    CHECK (type IN ('add', 'spend', 'set', 'delete', 'transfer'))
);
```

- `referred_by` is a Telegram user id, **not** a foreign key. The referrer may not have a `users` row yet.
- Existing banks receive `USD` from the column default, even if `DEFAULT_CURRENCY` is later something else. Only **new** banks use config.
- `operations.user_id` is always the acting user, not the referrer.

## Domain

```go
const LocaleEN = "en"

type User struct {
    // existing fields...
    Locale     string
    ReferredBy *UserID
}

type Bank struct {
    // existing fields...
    Currency string
}

type OperationType string

const (
    OperationAdd      OperationType = "add"
    OperationSpend    OperationType = "spend"
    OperationSet      OperationType = "set"
    OperationDelete   OperationType = "delete"
    OperationTransfer OperationType = "transfer"
)

type Operation struct {
    ID           int64
    UserID       UserID
    BankID       *int64
    Type         OperationType
    Amount       Money
    BalanceAfter *Money
    Meta         string
    CreatedAt    time.Time
}

type Access string

const AccessFull Access = "full"

func Entitlement(_ User, _ time.Time) Access {
    return AccessFull
}
```

`Entitlement` ignores trial, plan, and discount. Tests cover always-full only. Handlers do not call it in 3.6.

## Service

`New` gains `ops OperationRepository`, `tx Transactor`, and `defaultCurrency string`.

```go
type OperationRepository interface {
    Append(ctx context.Context, op domain.Operation) error
}

type Transactor interface {
    InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type UserRepository interface {
    Upsert(ctx context.Context, user domain.User) (domain.User, error)
    TouchActivity(ctx context.Context, userID domain.UserID, at time.Time) error
    SetReferredByIfEmpty(ctx context.Context, userID, referredBy domain.UserID) error
}
```

No `List`/`Get` on operations (4.10).

### Mutators

| Use case | Operation | `amount_cents` | `balance_after_cents` | `meta` |
| --- | --- | --- | --- | --- |
| Add | `add` | the added amount (positive) | new balance | bank name as typed |
| Spend | `spend` | the spent amount (**positive**, not negated) | new balance | bank name as typed |
| Set | `set` | the requested absolute amount | new balance | bank name as typed |
| Delete | `delete` | last balance | null | bank name as typed |
| CreateBank | none | — | — | — |
| Toggle | none | — | — | — |

Add/spend/set/delete all set `Operation.BankID` to the bank’s id. After a successful delete, SQLite `ON DELETE SET NULL` nulls that column on the row. `meta` keeps the typed name so `/history` can still label the row.

Delete order: `GetByID` → `Append` → `Delete`, all inside `Transactor.InTx`. If any step fails, the whole transaction rolls back.

Empty `meta` is stored as SQL `NULL`, not `""`.

`CreateBank` sets `bank.Currency = defaultCurrency`. Totals still `SUM` included banks with no currency grouping. Repository `Create` uses `COALESCE(NULLIF(currency, ''), 'USD')` so an empty bind cannot skip the default.

Failed amount validation (negative add/spend, overflow) must not append.

`created_at` on operations is `clock.Now()`.

### Referral

`UpsertUser` does not take a referrer. On insert it sets `Locale` to `domain.LocaleEN` and leaves `ReferredBy` nil (explicit columns; do not send an empty locale string that would bypass the SQL default). Conflict still must not rewrite `locale`, `referred_by`, `trial_ends_at`, `plan`, or `discount_percent`.

`SetReferredByIfEmpty(userID, referredBy)`:

1. Service: if `referredBy == userID`, return nil without calling the repository.
2. SQLite also no-ops self, then `UPDATE users SET referred_by = ? WHERE telegram_id = ? AND referred_by IS NULL`.
3. If the update affects 0 rows, load the user: missing → `ErrUserNotFound`; present → already set, return nil.

## Config

`DEFAULT_CURRENCY`:

- Missing or empty after trim → `USD`
- Any other non-empty trimmed string is stored as-is (no ISO check, no forced uppercase)
- Invalid values do not exist beyond “empty uses default”

Document in `.env.example` and README. Pass into `service.New`.

## Telegram

### `/start`

Welcome copy is unchanged. After the existing upsert middleware:

1. Parse payload with the same `commandPayload` helper (`/start 123` → `"123"`; `/start@bot 123` still works because the matcher strips `@bot` only for the command name; payload is the remainder after the first space).
2. Valid payload: `strconv.ParseInt`, base 10, value `> 0`, and not equal to the sender’s Telegram id.
3. Invalid (missing, non-numeric, `0`, negative, self): do not call `SetReferredByIfEmpty`.
4. Valid: call `SetReferredByIfEmpty`. On error, log with `user_id` + `command=start`; still send the welcome.

No extra chat copy for success, ignore, or self.

`telegram.Service` adds `SetReferredByIfEmpty`. Middleware `UpsertUser` signature stays `(ctx, userID, username)`.

### Locale at the edge

Middleware stashes `user.Locale` (from upsert return) on `context`. Handlers load copy via `text.For(localeFrom(ctx))`. Unknown/empty locale → `en`. Until 4.12 this is always `en`.

### Unchanged

Slash menu, `/help`, wizards, chat hygiene, empty states, no currency in copy, no entitlement checks.

## Text catalog

`internal/text` exposes a locale-keyed catalog. Only `en` is registered.

```go
func For(locale string) Catalog
```

Unknown locale → `en`. Static strings are fields (`For("en").Start`). Formatters stay functions on `Catalog` (`For("en").Added(...)`). English strings stay identical.

## Logging

Every telegram log that has a sender includes `user_id` (`int64` Telegram id). Logs inside a command handler also include `command` (canonical name: `start`, `help`, `newbank`, `add`, `spend`, `set`, `delete`, `bank`, `toggle`, `banks`, `total`, `all`, `cancel`, `feedback`). Middleware upsert errors use `user_id` (replace today’s `telegram_id`). No metrics backend.

## Tests

Mockery on the consumer’s interfaces. Extend `.mockery.yml` for `OperationRepository` and `Transactor`. Do not hand-write stubs.

**Unit**

- Service add/spend/set/delete: one `Append` with the table above, inside `InTx`. Create and toggle: `Append` not called. Invalid add/spend: `Append` not called.
- Failed append rolls back the balance change (integration).
- `SetReferredByIfEmpty`: first valid id stored; second call no-op; self no-op (service does not call the repository).
- `Entitlement`: always `AccessFull`.
- Config: empty/missing → `USD`; `" eur "` → `"eur"`.
- Telegram: `/start 99` calls `SetReferredByIfEmpty`; self, `0`, and `"nope"` do not; welcome still sent.
- Text: `For("en")` returns English; `For("fr")` falls back to `en`.

**Integration** (temp SQLite file)

- Open applies 002; second open is idempotent (`schema_migrations` has 001 and 002).
- Existing bank after migrate has `currency = USD`.
- Create through the repository/service with config `EUR` stores `EUR`.
- Add inserts one operations row with `meta` = name; delete inserts then leaves `bank_id` null; add row keeps `meta` after delete.
- `SetReferredByIfEmpty` writes once; second call leaves the first value.

## Docs and plan

- README env table + `.env.example`: `DEFAULT_CURRENCY`
- Status for 3.6 was tracked in the former `plan.md` (now `AGENTS.md` + `docs/sdd/`); those decisions live in `docs/sdd/legacy/decisions-log.md`
- No git commit unless asked

## Files (expected)

- Create: `internal/adapter/sqlite/migrations/002_write_path_foundations.sql`; `internal/domain/operation.go`; `internal/domain/entitlement.go` (+ tests); `internal/adapter/sqlite/operation.go` (+ integration test); `internal/adapter/sqlite/tx.go`
- Modify: domain user/bank; service `New`, bank mutators, user repo interface, `Transactor`; sqlite user/bank scan/insert + `tx.go`; config; `internal/text`; telegram middleware/handlers/bot Service interface; `.mockery.yml`; README; `.env.example`; former `plan.md` (now `AGENTS.md` + `docs/sdd/`)
