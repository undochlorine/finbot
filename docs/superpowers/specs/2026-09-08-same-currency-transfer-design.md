# Same-currency `/transfer` (3.11)

## Goal

Move money between one user’s banks when both banks share a currency. Hardened private MVP completes with this step.

## Locked product rules

- **One operations row** on the from-bank. Two balance updates in one `Transactor` transaction.
- **Shortcut:** exactly three tokens after `/transfer`, last token parses as money → try from / to / amount. If a bank is missing, or the payload is not `Word Word Number`, do **not** peel an amount; the full remainder is BankFrom. Missing from-name → unknown-bank error.
- **0 banks:** same `NoBanks` / `/newbank` as `/add`.
- **1 bank:** do not start the wizard; need-two-banks copy + `/newbank`.
- **To-picker** omits the from-bank. Typing the from-bank as to stays on the to-step.
- **Zero and negative amounts** are invalid. Negative **balances** stay allowed (overdraft ok).
- Same bank and different currency are rejected. Overdraft is allowed.

Wizard typed answers never use the three-token shortcut: full text is the from-name or to-name.

## Service

```go
Transfer(ctx, userID, fromID, toID int64, amount Money) (from, to Bank, error)
```

Reject before or inside the tx: `ErrInvalidAmount` (`amount <= 0` or overflow), `ErrSameBank` (same id), `ErrCurrencyMismatch`, `ErrBankNotFound`.

Operation: type `transfer`, `bank_id` = from, `amount_cents` = positive amount moved, `balance_after_cents` = from’s new balance, `meta` = to-bank id as a decimal string.

No new migration: `transfer` is already in `operations.type` CHECK.

## Telegram

Flow: pick from → pick to (other banks) → amount. Hygiene matches `/add`. Callbacks `v1:transfer:from:<id>` and `v1:transfer:to:<id>`. Slash menu, `/help`, `/start`. Empty `/transfer` is an empty-wizard command for `3.8` compact. Outcome uses a single 💸.

## Out of scope

Cross-currency transfer (`4.14`), `/history` (`4.10`), paywall (`4.5`).
