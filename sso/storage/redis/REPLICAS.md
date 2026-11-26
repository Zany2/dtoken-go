# Redis replica lifecycle acceptance

[中文](REPLICAS_zh.md) · [Adapter](README.md) · [Release checklist](../../RELEASE_CHECKLIST.md)

These integration tests use live Redis, independent connection pools and separate Server/ClientApp/BrowserClient instances. Actual HTTP handlers run through in-process transport without listening. They verify shared-storage semantics, not separate operating-system processes, a real browser or a Redis cluster.

## Acceptance matrix

| Test | Expected behavior |
| --- | --- |
| `TestRedisReplicasAtomicCredentials` | PKCE-bound Ticket and Code reject incorrect proofs without consumption; exactly one of eight concurrent redemptions through two connections succeeds |
| `TestRedisReplicasReplayAndRotation` | Two centers accept a signed authorization exactly once; reconnecting a center retains nonce history; re-signing with a different key cannot bypass replay, and retired signatures do not consume fresh nonces |
| `TestRedisReplicasBrowserCallback` | All four modes save browser state before the originating connection closes; the failed object returns no identity; replacement replicas recover the flow from Redis and accept exactly one of eight callbacks. Ticket/Code use PKCE |
| `TestRedisReplicasAccountLogoutRecovery` | Two applications, two accounts and four credential modes; Beta returns 503 after center/credential revocation, retaining retry bindings; a replacement center completes delivery and cleanup, Bob remains valid, and Alice can receive new credentials |

Center/application sessions in the logout test are application-owned Redis markers deleted by actual CenterLogout and callback hooks to verify composition order. Full Auth token and browser session behavior remains covered by the [deployment example](../../../examples/sso_deployment/README.md). Failures close client connections or return 503; tests do not stop Redis or cover partitions, persistence loss or failover.

## Running

Prepare Redis 6.2+ with GETDEL support and run from the repository root:

```powershell
$env:DTOKEN_REDIS_URL = 'redis://127.0.0.1:6379/15'
go test ./sso/storage/redis/... -run TestRedisReplicas -count=1 -v -timeout=5m
go test -race ./sso/storage/redis/... -run TestRedisReplicas -count=1 -v -timeout=5m
```

Use a test instance. Missing configuration explicitly skips integration; configured but unreachable Redis fails. Connection errors omit raw Redis URLs to avoid logging credentials. Use the workspace Go version and a supported race toolchain.

The existing CI Redis-service step includes this separate module and now enables `-race` for it. Ordinary steps without a Redis URL still skip these tests; only the configured Redis step exercises the backend. Tests, builds and CI were not run in this batch, and Redis was not contacted.

## Data and cleanup

Each case uses a random `dtoken:test:replicas:<random>:` prefix, with sub-prefixes for protocol state, browser flows, nonces and session markers. After all concurrent requests finish, replicas close and an independent management connection scans only the allocated prefix, checks every returned key and deletes those records. No FLUSHDB, FLUSHALL or unscoped deletion is used.

TTLs follow actual lifecycles: flows, nonces and credentials expire; registration and account revocation markers may persist. Abrupt termination or unavailable Redis can prevent cleanup and leave test records, so use a dedicated test instance. Test cleanup is not a production revocation-marker retention policy.

## Remaining deployment acceptance

Separately verify real multi-process rolling updates, load balancing, HTTPS/SameSite cookies, browser tabs, Redis persistence/failover, sustained load and network timeouts. Record Go/Redis versions, source revision and local changes, commands and results. Authored tests do not constitute a passing execution record.
