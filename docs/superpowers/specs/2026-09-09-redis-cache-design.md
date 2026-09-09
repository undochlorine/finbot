# Redis session store (Stage 4.2)

Replace the in-process memory `Cache` with Redis as the **runtime store for conversation FSM**. Postgres stays the system of record for money. Local and CI use Compose / a GitHub Actions Redis service. Managed Redis is `4.3`. Rate limits, distributed locks, and app replicas are `4.9`.

FSM is expendable: a Redis restart drops in-flight wizards (user sees the existing “start over” path). That is acceptable. Do not treat Redis as a second database.

## Goal

`cmd/bot` talks to Redis for `ports.Cache` (Get / Set / Delete). Telegram FSM behavior is unchanged: one JSON blob per user, **10-minute sliding TTL** on save, miss means idle or expired. Startup fails closed if Redis is unreachable. Handler-time Redis errors stay fail-soft (log, treat as miss) so a blip does not kill the process.

## Non-goals

- Caching bank lists, `/total`, `/all`, or any money snapshot
- Dual `CACHE_DRIVER=memory|redis` after cutover
- Redis Cluster, Sentinel, replicas, or persistence-as-recovery
- Rate limits / token bucket, distributed per-user locks, moving the pending FIFO into Redis
- Widening `ports.Cache` with `Incr` / `SetNX` / Lua
- Changing command UX, FSM JSON, or `fsmTTL`
- Hosting vendor (`4.3`), metrics backend (`4.17`)

## Decision (locked)

**A — Redis as FSM session store only.** New `internal/adapter/rediscache` implements the existing KV `Cache` port. Key prefix + `Open` (pool, timeouts, ping) are the 4.9/4.3 seam. Production `cmd/bot` requires `REDIS_URL` the same way it requires `DATABASE_URL`. Keep `memorycache` for fast unit tests of TTL semantics; it is not a production fallback.

Rejected:

- **B — cache-aside for totals / bank lists.** Per-user rows are tiny, already indexed, and serialized per user by the FIFO. A stale balance is a product bug. Revisit only if `4.9` load evidence says List/Total is hot; then a repository decorator (not telegram) with write invalidation.
- **C — Redis platform kit in 4.2** (rate limit, locks, Lua, Cluster). Right abstractions, wrong step. Building unused ports now is the dual-driver mistake in a different coat.
- **D — optional Redis with in-memory fallback.** Two runtimes, split-brain the moment a replica appears, same class of problem as `sqlite|postgres`.

## Current seam

Hexagonal layout already isolates Cache:

- Telegram owns `Cache` (Get / Set / Delete). `ports.Cache` remains so adapters do not import telegram
- `internal/adapter/memorycache` implements it (mutex map + `Clock`)
- `cmd/bot` wires `memorycache.New(clk)`
- FSM key is `fsm:{telegram_id}`; value is JSON; TTL **10 minutes**, refreshed on `saveFSM` only (sliding on write, not on read)
- Pending-command FIFO stays in the telegram adapter (RAM). **Not Cache. Not Redis.** Locked until `4.9` if replicas exist
- Runtime Get/Set/Delete errors: telegram logs and treats Get as miss; Set/Delete failures are logged. Do not change that policy in 4.2

Do not change product commands. Do not move FSM encoding into the Redis adapter. Do not put Redis types in `service`.

What memory hid and Redis must not:

| MVP | Redis |
| --- | --- |
| Process restart drops every wizard | Same if Redis restarts; **survives bot process restart** |
| One process is the only Cache | Any future replica can share FSM (lock around drain is still `4.9`) |
| No network, no timeout | Explicit dial/read/write timeouts; ping at Open |
| `Clock` for expiry | Redis `SET` with `PX` (millisecond TTL); no Clock on the adapter |
| Unbounded map | `maxmemory` + `volatile-ttl` in Compose; all our keys have TTL |

## Decomposition

Implement in two steps so one agent session cannot swallow Compose/CI/Open **and** production wiring. Pickup: `Follow AGENTS.md. Let's move to step 4.2.1`. Saying `4.2` means start `4.2.1`, not both.

The Redis `Cache` adapter is small (unlike Postgres repos). Foundation and adapter are one step. Cutover is the second, matching `4.1.3`.

| Step | File | Runtime still memorycache? |
| --- | --- | --- |
| **4.2.1** Foundation + adapter | [`docs/sdd/steps/4.02.1-redis-foundation.md`](../../sdd/steps/4.02.1-redis-foundation.md) | yes |
| **4.2.2** Cutover | [`docs/sdd/steps/4.02.2-redis-cutover.md`](../../sdd/steps/4.02.2-redis-cutover.md) | no |

`4.2.1` may parse `REDIS_URL` / `REDIS_TEST_URL` for tests/`Open` without switching the bot.

## Architecture

```text
cmd/bot
  config.Load  →  REDIS_URL
  rediscache.Open (pool, timeouts, ping, key prefix)
  telegram.New(..., cache, ...)

internal/adapter/rediscache/
  open.go     ParseURL, timeouts, pool, Ping, Close
  cache.go    Get / Set / Delete with prefix; redis.Nil → miss
```

Client: `github.com/redis/go-redis/v9`. Vanilla Redis **7** (Compose `redis:7-alpine`). No RedisJSON, no Redis Stack, no extra modules (managed Redis in `4.3` will not all have them).

`Open` is the **only** place that constructs the client. `4.9` Cluster / `4.3` `rediss://` TLS stay a URL + `Open` change, not a rewrite of Get/Set/Delete.

Telegram keeps `fsm:%d`. The adapter prepends the prefix so Redis keys are:

```text
{prefix}fsm:{telegram_id}
```

Default prefix: `finbot:v1:`. Example: `finbot:v1:fsm:123456`. Bump to `finbot:v2:` only if the FSM blob becomes incompatible (old keys die in ≤10 minutes). Tests pass a unique prefix into `Open` so they never `FLUSHALL` / `FLUSHDB` (a local bot may share the Compose instance).

Logical Redis DB is always **0**. Do not use `SELECT` / URL path `/1` for tests — Cluster in `4.9` does not do that.

### User isolation

Many users share **one** Redis and **one** prefix. Isolation is the key, not Redis ACLs or a per-user connection.

Telegram already keys Cache by the sender’s Telegram id (`fsmKey` → `fsm:%d`). That id is globally unique. Redis then stores exact strings:

```text
finbot:v1:fsm:111      # Alice mid-/add
finbot:v1:fsm:222      # Bob mid-/newbank
```

`GET`/`SET`/`DEL` operate on one key. Alice’s save cannot overwrite Bob’s blob; Alice’s get cannot return Bob’s JSON. Decimal ids do not prefix-collide (`fsm:1` ≠ `fsm:10` ≠ `fsm:11`). The adapter prefix is **app** namespacing (`finbot:v1:`), not a user namespace — do not put a user id in the prefix and drop it from `fsmKey`.

This is the same model as today’s shared in-memory map. Redis does not add a `WHERE user_id = $1` the way Postgres does; application code must keep passing `from.ID` into `loadFSM`/`saveFSM`/`clearFSM` (already true). Do not add `SCAN finbot:v1:fsm:*` on a user request path.

Concurrent users are many keys, not one key: the FIFO serializes **one** user; other users proceed in parallel and hit their own keys. Pool size 10 is for that concurrency. Same-user last-write-wins is unchanged.

`4.2.1` must prove key isolation: Set `fsm:111`, Set `fsm:222` with different values, Get each, Delete one, the other remains.

### Caching strategy (what Redis is)

| Data | Strategy | Why |
| --- | --- | --- |
| FSM JSON | **Session store** (Redis is SoR). `SET` with `PX` = 10m on every save. `DEL` on cancel/success. Miss = expired / idle | Ephemeral conversation; must survive process restart; must be shareable across future replicas |
| Banks, totals, operations | **No cache.** Postgres `FOR UPDATE` + indexes | Correctness over a millisecond; write/read ratio is high; FIFO already serializes one user |
| Pending FIFO | **Stay in process RAM** | Locked product rule; distributed queue is `4.9` |
| Rate limit / user lock / payment idempotency | **Reserved prefixes, not implemented** | Same client, new consumer-owned ports in later steps |

FSM write policy is last-write-wins on one key. That is safe today because the per-user FIFO is in-process. Two bot processes in `4.9` can race GET→modify→SET; that step adds a per-user lock (`SET` NX + TTL or equivalent) **around the drain**, not a new FSM encoding.

### TTL and memory policy

| Item | Choice |
| --- | --- |
| FSM TTL | **10 minutes**, owned by telegram (`fsmTTL`). Adapter does not hardcode it |
| Refresh | Sliding on **save** only (already true). Do not `GETEX` on load |
| `ttl <= 0` on Set | Delete the key (match `memorycache`) |
| Precision | Milliseconds (`SET PX`); go-redis already does this from `time.Duration` |
| Compose `maxmemory` | `64mb` (FSM blobs are tiny; this is a safety rail) |
| `maxmemory-policy` | `volatile-ttl` — only keys with TTL; nearest expiry first. Never `allkeys-lru` (could evict an active wizard while keeping junk) |
| Persistence | **Not a product requirement.** No Compose volume for Redis. RDB/AOF default of the image is irrelevant; a flush is the same as “every wizard expired” |
| Lock/rate-limit keys later | Must also carry TTL so `volatile-ttl` can evict them under pressure |

### Fail policy

| Moment | Policy |
| --- | --- |
| Config / Open / Ping | **Fail closed** — do not long-poll Telegram (same as Postgres) |
| Handler Get error (timeout, network) | **Fail soft** — log, treat as miss (existing telegram behavior) |
| Handler Set/Delete error | **Fail soft** — log; user may lose a wizard step |
| `context` already canceled | Return error (match `memorycache`) |
| Miss (`redis.Nil`) | `ok=false`, `err=nil` — never an error |

Timeouts are short so a sick Redis cannot stall the per-user drain behind a 5s Postgres-class wait.

### Pool and timeouts (constants in code, not a pile of env vars)

Mirror the Postgres style (named constants in `open.go`):

| Constant | Value | Why |
| --- | --- | --- |
| DialTimeout | 3s | Slow network vs missing Compose |
| ReadTimeout | 400ms | FSM payload is small; FIFO drain must not block |
| WriteTimeout | 400ms | Same |
| PoolTimeout | 1s | Wait for a pool conn; then error |
| PoolSize | 10 | Same order as Postgres `MaxOpenConns`; concurrent users, not one global lock |
| MinIdleConns | 2 | Avoid cold dial on the first update |
| ConnMaxIdleTime | 5m | Match Postgres idle |
| ConnMaxLifetime | 30m | Match Postgres |
| MaxRetries | 1 | One retry on transient; do not multiply 400ms into seconds |

### Reserved key space (implement later, do not collide)

All keys under `finbot:v1:`. 4.2 only writes `fsm:`.

| Prefix after `finbot:v1:` | Owner step | Role |
| --- | --- | --- |
| `fsm:{user_id}` | 4.2 | Conversation JSON |
| `rl:{user_id}` | 4.9 | Token bucket / rate limit |
| `lock:{user_id}` | 4.9 | Hold while a replica drains that user |
| `idemp:{…}` | 4.7 / 4.9 | Payment idempotency if needed |
| `snap:{user_id}` | 4.9 only if measured | Optional money snapshot; cache-aside + write invalidation in a **repo decorator**, never in telegram |

4.9 implements new ports on the **same** `rediscache` package (or thin wrappers). Do **not** overload `Cache` with `Incr`. Rate limit and lock are different consumers and own their interfaces.

## Config

After `4.2.2`, `REDIS_URL` is required (`redis://` or `rediss://`, host non-empty). `rediss://` is accepted from day one so `4.3` TLS is a URL change.

Until `4.2.2`, `cmd/bot` still uses `memorycache`. `4.2.1` may parse `REDIS_URL` / `REDIS_TEST_URL` for tests/`Open` without switching the bot.

Local default (Compose):  
`redis://:finbot@127.0.0.1:6379/0`

Tests:  
`REDIS_TEST_URL` or the same local URL.

Password `finbot` is **local/CI only** (`requirepass`). No Redis ACL user in 4.2.

No `REDIS_KEY_PREFIX` env. Prefix is the `Open` argument; production passes `""` → `finbot:v1:`.

## Compose and CI

**Local:** add a `redis` service next to Postgres in `docker-compose.yml`: `redis:7-alpine`, port `6379:6379`, `requirepass finbot`, `maxmemory 64mb`, `maxmemory-policy volatile-ttl`, healthcheck `redis-cli -a finbot ping`. **No named volume.**

**CI:** GitHub Actions integration job gains a Redis **service container** (same image/password, port 6379, health). `REDIS_TEST_URL` for tests. Do not require Compose in CI.

`make test-integration` after `4.2.1` also runs `./internal/adapter/rediscache/...` (`//go:build integration`). Unreachable Redis **fails** (does not skip); error text tells the operator to start Compose.

## memorycache after cutover

Unlike SQLite, memorycache is not a second dialect of the system of record. **Keep it** for unit tests of KV/TTL without Redis. `cmd/bot` must not use it after `4.2.2`. Telegram tests stay on mockery.

## Testing

- Unit: `memorycache` table tests stay. Telegram still mocks `telegram.Cache`. Config parse tests for URL accept/reject.
- `4.2.1`: ping; Get/Set/Delete behavioral cases equivalent to memorycache (hit, miss, overwrite, delete, `ttl<=0` deletes, expire, canceled context, key isolation). Unique prefix per test + cleanup. Fail if Redis is unreachable.
- `4.2.2`: `cmd/bot` fails without `REDIS_URL` or when Redis is unreachable (same shape as the Postgres unreachable test). Docker/README/env examples include Redis.

Do not hand-write a fake Redis. Do not add miniredis unless the user later approves that extra double. Real Redis in integration, like Postgres.

## Files (target)

| Path | Role |
| --- | --- |
| `internal/adapter/rediscache/*` | Open + Cache |
| `docker-compose.yml` | Local Redis 7 |
| `internal/config/config.go` | `REDIS_URL` (required in 4.2.2) |
| `cmd/bot/main.go` | Open rediscache (4.2.2) |
| `.github/workflows/ci.yml` | Redis service (from 4.2.1) |
| `Makefile` | integration package path |
| `README.md` / `.env.example` | Human run (4.2.2) |

Do not edit `docs/sdd/legacy/` except the decisions log when this lock is recorded.
