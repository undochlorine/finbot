# Telegram (current)

Bot surface, conversation FSM, chat hygiene, pending queue, emojis. Product rules: [`../requirements.md`](../requirements.md).

Production `Cache` is in-process memory. Redis is [`../steps/4.02-redis.md`](../steps/4.02-redis.md). `/history` is [`../steps/4.10-history.md`](../steps/4.10-history.md).

**FSM** (Finite State Machine) is the per-user conversation step (which command is in flight, waiting for a name vs a yes/no, pending bank name, …). Stored in `Cache` with TTL **10 minutes**. Telegram owns the `Cache` interface. `ports.Cache` remains so `memorycache` does not import telegram. FSM stores `prompt_id` (the bot message edited in place) and `sweep_ids` (typed answers to delete when the flow ends). **Not** the pending-command FIFO.

## Commands

The process registers these with Telegram `setMyCommands` on startup so clients show the Commands menu and `/` autocomplete. BotFather `/setcommands` is not required.

| Command | Flow |
| --- | --- |
| `/start` | Upsert user, short intro, point to `/help`. First valid `/start <telegram-id>` stores `referred_by` if still null. Later payloads do not overwrite. Self-referral is ignored. No rewards until `4.15`. |
| `/help` | List commands |
| `/newbank` | Ask name → duplicate-name error immediately if taken (stay on name) → else ask “Count in total?” yes/no buttons (or type `yes`/`no`). Shortcut: `/newbank Travelling`. Names may contain spaces, so `/newbank Holiday yes` is a bank named `Holiday yes`, not a name plus include flag. |
| `/add` | Pick bank (buttons or arg) → amount. Adds to balance |
| `/spend` | Same as add, subtracts (negative allowed) |
| `/set` | Pick bank → amount. Sets absolute balance |
| `/delete` | Pick bank → confirm button → delete bank |
| `/bank` | Pick bank or arg → show one bank |
| `/toggle` | Pick bank → flip `include_in_total` → confirm new state |
| `/banks` | List all banks (name, balance, whether in total) |
| `/total` | Sum of included banks only |
| `/all` | Full list + total |
| `/cancel` | Clear the in-flight flow. Idle `/cancel` says nothing is pending. |
| `/feedback` | Ask for text → forward to `ADMIN_TELEGRAM_ID` (user id + username + body) → thank the user. If admin id is unset, say unavailable. No inbox table. |
| `/rename` | Pick bank (buttons or arg) → type the new name. Duplicate-name error stays on the name step. Recasing the same bank is allowed. Shortcut: `/rename Travelling`. If that full name is missing, `/rename Travelling Holiday` renames `Travelling` → `Holiday`. |
| `/transfer` | Pick from-bank → to-bank (other banks) → amount. Same currency only. Reject same bank / invalid amount. |

**Later (`4.x`):** `/history`, `/language`, currency on `/newbank`, finance tips, paid-promo copy on extra-bank commands.

**Empty state:** if the user has no banks, mutating/list commands say so and point to `/newbank`.

**Shortcuts:** put details after the slash command instead of waiting for a prompt. Bank names may contain spaces. Examples: `/newbank Travelling`, `/add Travelling 100`, `/spend Gifts 12.50`, `/set Live 0`, `/bank Travelling`, `/delete Travelling`, `/rename Travelling`, `/rename Travelling Holiday` (only if `Travelling Holiday` is not itself a bank). `/transfer Holiday Gifts 50` (exactly two names and an amount; otherwise the full remainder is the from-bank).

`/newbank` does **not** take a yes/no include flag on the same line. After the name is accepted, the bot asks whether the bank counts in the total.

**Callback data:** versioned and namespaced, e.g. `v1:add:<bankID>`, `v1:rename:<bankID>`, `v1:transfer:from:<bankID>`, `v1:transfer:to:<bankID>`.

## Chat hygiene

The chat should read as a ledger: what the user asked for, and what changed. Wizard steps are not history.

**Keep**

| Kind | Why |
| --- | --- |
| Slash commands (`/add`, `/add Travelling 100`, `/start`, `/help`, `/cancel`, …) | The user’s intent and timestamp. Shortcuts *are* the record. Deleting them feels like the bot eating the chat. |
| Outcomes | Created / added / spent / set / deleted / delete-cancelled / toggled / renamed / bank card / `/banks` / `/total` / `/all` |
| Terminal errors | No banks, unknown bank from a shortcut, something went wrong, feedback unavailable, nothing to cancel, flow expired |
| The user’s `/feedback` body | The thanks line does not repeat it |
| `/cancel` + `Canceled.` | Abort should stay visible |

**Remove** (edit in place or delete)

| Kind | Why |
| --- | --- |
| Bot prompts | Ask name / include / which bank / amount / delete confirm / new name / feedback ask. Same message is edited as the step advances; success edits it into the outcome and strips buttons. |
| Typed answers in a flow | Bank name, amount, yes/no — already restated in the outcome |
| Recovered validation prompts | Invalid amount/name, name taken: gone once the flow succeeds or is cancelled |
| Replaced flow | Starting `/spend` while `/add` is pending deletes the add wizard. No extra `Canceled.` |

**Mechanism:** FSM `prompt_id` is the one bot message to edit. `sweep_ids` are extra messages (typed answers) deleted on success, `/cancel`, or replace. Expired callback: edit that message to expired, strip buttons. Stale callback on a leftover message: delete it. Delete/edit failures are logged; the outcome still goes out.

**Do not delete:** slash commands, outcomes, `/feedback` text, `/start` `/help` `/banks` `/total` `/all` replies.

## Error / edge cases (handlers + service)

- Duplicate bank name (case-insensitive) → clear error, do not overwrite
- Unknown bank → error, offer `/banks`
- Invalid amount (empty, `abc`, `1.234`, overflow, **negative add/spend**) → error, ask again
- Negative add/spend is invalid; negative **balances** are still allowed (`/spend` below zero, `/set` to a negative)
- Delete last bank → allowed
- Concurrent updates from the **same** user → FIFO (not a SQLite race). Two different users may overlap.
- Very large amounts → reject if cents would overflow `int64`
- User with zero banks asking `/total` → `0.00` (empty sum). `/banks`, `/all`, and `/bank` with no banks still use the `/newbank` empty-state hint.
- `/rename`: unknown bank → error, offer `/banks`; name taken by **another** bank → stay on the name step; empty name → ask again; recasing the same bank succeeds. Slash args try the full remainder as the current name first; if missing, first word is the old name and the rest is the new name.
- `/transfer`: reject different currency, same bank, invalid amount (including zero); empty state same as `/add`; one bank → need another `/newbank`. Slash args: exactly `From To Amount` (two words + money) tries both banks; if a bank is missing, or the payload is not that shape, the full remainder is the from-name only.

Starting another flow command replaces the pending FSM. Read-only commands leave it in place. Plain text with no FSM stays silent. Expired callbacks (cache miss) ask to start over; stale callbacks (wrong in-flight flow) are ignored.

## Pending commands

`github.com/go-telegram/bot` runs each handler with `go` unless `WithNotAsyncHandlers` is set. Production `New()` sets that option **and** a per-user FIFO: enqueue stays on the single poll worker so a burst keeps Telegram order; drain still runs handlers so compact can see a waiting list. Middleware on the inner bot (`Start` never calls our `ProcessUpdate` wrapper).

**Keep handling every update**, in the order Telegram sent them (`update_id` / enqueue order), except for the compact rules below.

**Where:** `PendingCommands` lives in the telegram adapter (RAM). One FIFO per Telegram user id. Not FSM `Cache`. Not SQLite. Not Redis. One bot process now; a distributed queue is out of scope (`4.9` if replicas ever exist).

**How:** enqueue the update and return; one drain goroutine per busy user pops FIFO and calls `next`. Other users are not blocked. Cap **32** waiting items per user; if full, drop the **oldest waiting** item and `slog.Warn` (burst bound, not a product rate limiter).

**Same FIFO:** slash commands, typed FSM answers, and callbacks. A button must not race a later `/spend`.

The FIFO only orders. `WithNotAsyncHandlers` alone is not the fix: it serializes the whole process and never sees a burst as a list, so compact cannot run.

**Compact** is a pure function over the waiting slash-command items (not callbacks, not typed answers). Apply when a drain starts and after each processed item if anything is still waiting. Locked rules — do **not** simulate bank state (that would drop `/add Travelling 100` that follows `/newbank Travelling` in the same burst):

| Pattern | Keep | Drop |
| --- | --- | --- |
| Consecutive identical **idempotent** slash text (`/help`, `/start`, `/banks`, `/total`, `/all`, `/bank <name>`) | first | the rest of the run |
| Consecutive identical **empty wizard starts** (`/newbank`, `/add`, `/spend`, `/set`, `/delete`, `/toggle`, `/bank`, `/rename`, `/transfer`, `/feedback`, `/cancel` with no args) | first | the rest of the run |
| Empty wizard start whose **next waiting slash** is also a flow-start (would replace per FSM polish) | the later flow-start | the empty one. **Do not** drop if the next item is `/cancel`, `/help`, `/start`, `/banks`, `/total`, `/all`, or a typed/callback update |
| Consecutive identical `/newbank <same name>` | first | later copies (second would be name-taken) |
| Consecutive identical `/delete <same name>` | first | later copies (second would be unknown bank) |

**Do not collapse:** `/add` `/spend` `/set` shortcuts even when the line is identical (two `/add Travelling 100` are two adds); non-consecutive duplicates (`/help` `/banks` `/help` runs help twice); callbacks; typed FSM answers; `/feedback` message bodies.

Collapsed items are **not** answered with an extra “skipped” line. The user’s slash messages stay in chat (hygiene keeps them).

Empty `/transfer` is an empty-wizard command for compact.

## Emojis

Alive, not noisy. Copy stays in `internal/text`. Prefer **one** leading emoji per user-facing outcome. Never a stack of them.

The table is a **baseline**, not a closed list. An implementer may add other office-ish / finance-ish emojis (e.g. 📊 🏦 💼 💰 📁 🧾 📌) on outcomes, lists, `/start`, or `/transfer`. Same taste: calm, useful, not cute spam.

| Copy | Baseline |
| --- | --- |
| `/start` welcome (first line) | 👋 |
| Bank created | ✅ |
| Added | 💸 |
| Spent | 💸 |
| Set | ✅ |
| Deleted | 🗑️ |
| Toggled | ✅ |
| Feedback thanks | 🙏 |
| Renamed | ✏️ |
| Transferred | 💸 |

**No emoji** on `/help` lines, command-menu descriptions, errors, wizard prompts, `Canceled.`, or `Nothing to cancel.`. Lists (`/banks` `/total` `/all` `/bank`) may get a single restrained marker if it stays scannable.
