# Billing hooks (current)

Columns live in [`persistence.md`](persistence.md). Product order lives in [`../requirements.md`](../requirements.md#entitlement). This file is **semantics of what already exists**. Do not implement trial enforcement, checkout, or admin CRUD here — those are `4.5`–`4.8`.

## What the code does now

- `User.LastActivityAt` updated on every successful interaction
- `User.Plan` stored (e.g. `trial` / `free` now; paid plans in `4.7`)
- `User.TrialEndsAt` set **once** on first upsert (`now + TRIAL_DURATION`). Never refresh it on later messages
- `User.DiscountPercent` optional (`*int`): `nil` = not whitelisted; `100` = free; `50` / `30` / custom = that % off after trial. Unused until `4.5`+
- Config `TRIAL_DURATION` (default `168h` / 7 days) is loaded. Trial is **not** enforced
- `users.referred_by` nullable Telegram id; set **once** from `/start` payload (set-if-null). Self-referral ignored. No rewards until `4.15`
- `Entitlement` helper **always returns full access**. Tests: always-allow. Paywall enforcement is [`../steps/4.05-trial.md`](../steps/4.05-trial.md)
- Do **not** add a Billing port until `4.6`–`4.7`

## Later (step files only)

| Topic | Step |
| --- | --- |
| Inactivity warn/delete; paid/100% kept longer | [`4.04`](../steps/4.04-inactivity.md) |
| Enforce trial + limited free tier | [`4.05`](../steps/4.05-trial.md) |
| Payment strategy (provider TBD with the user) | [`4.06`](../steps/4.06-payment-strategy.md) |
| Wire `Plan` / paid-through to that strategy | [`4.07`](../steps/4.07-billing.md) |
| Telegram admin whitelist CRUD | [`4.08`](../steps/4.08-admin-whitelist.md) |
| Referral rewards | [`4.15`](../steps/4.15-referral.md) |
| Contributors promotions | [`4.16`](../steps/4.16-contributors.md) |
