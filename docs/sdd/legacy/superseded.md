# Superseded intentions

Already executed. Live docs describe how the code works **now**. Do not treat these as open work.

**Still open** (not listed here — one-liner on the live area/step file): SQLite → Postgres (`4.1`), memory Cache → Redis (`4.2`), Entitlement always-allow → `4.5`, one `cmd/bot` → `4.21`.

## Consumer-owned interfaces

Repository and clock interfaces were planned in `ports`, then moved to the **consumer**. Service owns `BankRepository` / `UserRepository` / `OperationRepository` / `Transactor` / `Clock`. memorycache owns `Clock`. Tests mock that package’s interfaces, never a sibling layer’s.

## Cache ownership

Telegram owns `Cache`. `ports.Cache` remains so memorycache does not import the telegram adapter.

## Transactor

`3.6` added service `Transactor` so add/spend/set/delete (then rename and transfer) are atomic with the operations row. This supersedes the earlier “no unit of work in 3.6” cut.

## Stage 3 features that were “later” and have shipped

`/feedback` forward, chat hygiene (edit/delete wizard debris), per-user FIFO pending commands, compact of obvious queued no-ops, sparse outcome emojis, `/rename`, same-currency `/transfer`. Hardened private MVP completed at `3.11`.
