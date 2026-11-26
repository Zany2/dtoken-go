# Gin Core Flow Tests

This directory contains HTTP flow tests for `tests/gin_core_app` and serial and parallel benchmarks from the public Auth API to storage.

The HTTP tests start a Gin app inside the test process with `httptest.NewServer`, then call its endpoints through HTTP.

## Auth benchmarks

`auth_benchmark_test.go` calls public Auth APIs using the bundled Memory or Redis adapter, without an HTTP round trip. Memory is the default. Redis requires explicit selection; a missing URL or failed connection fails the benchmark instead of falling back or skipping.

All 120 Auth scenarios completed three timed Memory samples on 2026-11-26, with additional parallelism measurements; see the [full workload report (Chinese)](../../docs/design/benchmarks/2026-11-26-memory-workloads_zh.md). Earlier local comparisons and real-pool concurrency checks remain in the [measurement record (Chinese)](../../docs/design/benchmarks/2026-11-26-memory-report_zh.md). Redis has not been measured. General commands from the repository root:

```powershell
$env:DTOKEN_BENCH_STORAGE='memory'
go test ./tests/gin_core_flow -run '^$' -bench 'BenchmarkAuth(Validation|Writes)' -benchmem -count=3

$env:DTOKEN_BENCH_STORAGE='redis'
$env:DTOKEN_REDIS_URL='redis://localhost:6379/0'
go test ./tests/gin_core_flow -run '^$' -bench 'BenchmarkAuth(Validation|Writes)' -benchmem -count=3
```

The full matrix can take time. To start with basic validation only:

```powershell
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthValidation$/^redis$/^terminals=1$/^extra=0$/^(is-login|check-login)$' -benchmem -count=3
```

| Scenario | Measurement |
| --- | --- |
| `is-login` / `check-login` | Basic validity checks with activity timeout and auto renewal disabled |
| `details` | Full token details, including Extra retrieval and decoding when present |
| `active-foreground` | Activity validation and task submission; draining after return is excluded from timing |
| `active-complete` | Validation through activity task completion, including the drain wait |
| `renew-active-complete` | Validation through auto renewal and activity maintenance; every iteration is eligible, and successful renewal events are counted |
| `login-logout` | One complete login and logout cycle, restoring the original terminal count |
| `manual-renew` | Renew an existing token for one hour |

Each group uses 1, 100, or 1000 terminals and an Extra string of 0, 1 KiB, or 64 KiB. Only the measured target carries Extra; the transient token in login/logout also carries the selected size. Sizes exclude encoding overhead. Sharing is disabled, concurrent login is enabled, and terminal count is unlimited. Token TTL and enabled activity timeout are one hour; fixture preparation and measurement must finish within that lifetime.

Construction, seeding, result checks, and cleanup are excluded from timing. Each fixture uses a random key prefix; cleanup closes Auth, drains tasks, and deletes only that prefix before closing Redis. Use a dedicated test Redis database. Login/logout keeps terminal list length stable, while the historical terminal counter still increases.

Activity scenarios drain every iteration so task coalescing cannot hide background work. `tasks/op` must be 1 for activity scenarios and 0 otherwise. Activity scenarios enable an error counter that produces no log output; auto renewal also uses a synchronous event counter. Their instrumentation and synchronization costs are included. Basic validation and write scenarios disable logging and event observation.

Workers can run concurrently with the foreground in `active-foreground`, so its `B/op` and `allocs/op` are **not isolated foreground allocations**; background execution before return can also affect timing. Complete scenarios include drain synchronization. These serial averages do not measure concurrent throughput, pool saturation, or p95/p99. Basic parallel validation and permanent TTL comparisons are described below. Preserve the toolchain, Redis version and location, configuration, dataset sizes, and raw output when comparing versions with equivalent semantics.

### Terminal remnants and Provider benchmarks

`auth_query_benchmark_test.go` uses the same storage selection and cleanup. Run from the repository root:

```powershell
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuth(TerminalQueries|PermissionProvider)$' -benchmem -count=3
```

- `BenchmarkAuthTerminalQueries`: 10, 100, or 1000 terminals with 0%, 50%, or 100% missing credentials. `raw` returns every indexed entry; `alive` filters all live entries; `device-alive` queries only the last logged-in device. Each iteration checks the count; final checks verify token values, order, and an unchanged stored index.
- `BenchmarkAuthPermissionProvider`: 1, 100, or 1000 terminals, comparing `hit` (Provider supplies permission), `fallback` (returns nil), and `empty-deny` (returns a non-nil empty list). All three Sessions contain the same permission: the first two must succeed, while the empty list must deny access. Every operation must call the Provider once, reported as `provider-calls/op`.

Before timing, the terminal benchmark deletes the primary records for the first half or all of the index, leaving the Session intact. Extra, activity, and refresh records are not created. This models the missing state after natural expiry on real storage; it **does not measure TTL expiration, Redis background cleanup, or index pruning triggered by login**. Setup uses the current namespace helper and must be adapted when ported to historical key layouts. The Session retains its one-hour TTL even when all credentials are missing.

Both groups disable activity maintenance, auto renewal, and logging; `tasks/op` must be 0. The Provider is a local callback returning a fixed slice, with its counter overhead included; these results do not represent an application database or remote permission service. The timed loops retain error and count assertions. Setup, final full-result checks, and cleanup are excluded from timing.

### Permanent TTL, parallel validation, and login pruning

`auth_lifecycle_benchmark_test.go` uses the same backend selection and random prefix isolation:

```powershell
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthPermanentValidation$' -benchmem -count=3
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthParallelValidation$' -benchmem '-cpu=1,4,8' -count=3
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthParallelWrites$' -benchmem '-cpu=1,4,8' -count=3
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthLoginPruning$' -benchtime=20x -benchmem -count=3
```

- `PermanentValidation`: compares tokens issued with a one-hour TTL and permanent configuration. It times `CheckLogin` and checks storage TTL afterward. It does not measure conversion to permanent storage or persistence of Extra or activity records.
- `ParallelValidation`: prepares 64 credentials in one Auth. `same-account` has 64 terminals for one account; `many-accounts` has one terminal for each of 64 accounts. Each `RunParallel` worker cycles through the credentials from a different starting offset. `-cpu` controls GOMAXPROCS for each run and therefore the default worker count. Failures are aggregated and checked by the main goroutine. `ns/op` is total wall time divided by total operations, reflecting throughput, **not per-request latency or tail latency**. Scheduling and credential rotation overhead are included.
- `LoginPruning`: uses 10, 100, or 1000 terminals with 50% or 100% missing primary records. Each login sets the account limit to the original terminal count, forcing slot validation and pruning. Every iteration restores the same serialized Session snapshot outside timing, times the complete login, then stops timing to verify the persisted index and new token and log out the transient terminal. Final checks preserve the original live credentials. The 50% case retains live entries; the 100% case covers login after all old entries are removed. There are no event observers.

All three disable activity timeout, auto renewal, and logging; `tasks/op` must be 0. Except for permanent credentials, data lasts one hour and each sample must finish within that lifetime. Pruning fixture restoration makes actual runtime substantially longer than reported timed work, so a fixed `-benchtime=20x` is recommended. Repeated restoration affects cache warmth and does not simulate actual TTL expiration. This scenario excludes shared login, device scope, and overflow eviction of live terminals.

`ParallelWrites` adds concurrent login/logout cycles for one account or multiple accounts. Each worker uses a distinct device ID; workers contend on one account lock in the shared-account case and own distinct accounts in the multi-account case. One operation includes both login and logout. Final checks reject leftover terminals, and workers aggregate failures when finished. Account creation costs are included, so this is not equivalent to a workload with all accounts created beforehand. Sharing, terminal limits, activity maintenance, and auto renewal are disabled; `tasks/op` must be 0.

These scenarios do not complete performance acceptance: maintenance coalescing and pool overload, tail latency, and migration rehearsal still require deployment-specific validation. Select all Auth benchmarks with `-bench '^BenchmarkAuth'`, but prefer grouped runs and retain raw output because the full matrix is lengthy. Module regression and release status are tracked in the [non-SSO acceptance record (Chinese)](../../docs/design/performance-acceptance_zh.md).

### Maintenance concurrency with a real pool

`auth_maintenance_load_test.go` always uses bundled Memory and a single-worker Ants pool, without Redis or HTTP. Channels control task and listener execution to check:

- `TestAuthRealPoolCoalescingAndClose`: 64 concurrent checks share one pending token task; an asynchronous listener falls back inline when the shared pool rejects it; shutdown drains both work and listeners and rejects new checks.
- `TestAuthRealPoolSaturationAndRecovery`: 32 distinct tokens across same-account or multiple-account workloads, with activity only, renewal only, or both. Saturation preserves activity without extending TTL, skips optional renewal, and allows retries once capacity returns.

```powershell
go test ./tests/gin_core_flow -run '^TestAuthRealPool' -count=20 '-cpu=1,4,8' -timeout=2m
```

On 2026-11-26, all seven scenarios passed 420 executions in total. Twelve activity/renewal benchmark configurations also passed three samples each; see the [measurement record (Chinese)](../../docs/design/benchmarks/2026-11-26-memory-report_zh.md). Controlled interleavings and normal maintenance costs do not replace race detection, sustained overload tests, or p95/p99 measurements.

## Test List

- `TestAuthFlow`: tests authentication.
  - No token requests `/api/me`, expect unauthorized.
  - Login through `/login`, expect token.
  - Use token to request `/api/me`, expect current user.
  - Logout through `/api/logout`, expect old token rejected.

- `TestTokenMetadataAndStatusFlow`: tests token metadata and status APIs.
  - Check `IsLogin` behavior with and without token.
  - Read token info, device, device ID, create time, and timeout.
  - Login with a custom timeout and verify TTL.
  - Call LoginByToken with an existing token.

- `TestPermissionFlow`: tests permission checks.
  - Login without `article:read`.
  - Request `/api/articles`, expect forbidden.
  - Grant permission through `/api/permissions`.
  - Request `/api/articles` again, expect success.

- `TestPermissionMutationAndLogicFlow`: tests permission mutation and logic checks.
  - Remove granted permissions and verify access is revoked.
  - Verify AND permission checks require all values.
  - Verify OR permission checks and wildcard permissions.

- `TestAccessStatusFlow`: tests boolean permission and role checks.
  - Verify HasPermission and HasRole by login ID.
  - Verify HasPermission and HasRole by token.

- `TestAccessListFlow`: tests permission and role list APIs.
  - Verify GetPermissions/GetRoles by login ID.
  - Verify GetPermissionsByToken/GetRolesByToken.

- `TestAccessProviderFlow`: tests external access provider behavior.
  - Verify provider permissions and roles override session-stored values.
  - Verify terminal-aware provider data can differ by device.

- `TestRoleFlow`: tests role checks.
  - Login without `admin`.
  - Request `/api/admin`, expect forbidden.
  - Grant role through `/api/roles`.
  - Request `/api/admin` again, expect success.

- `TestRoleMutationAndLogicFlow`: tests role mutation and logic checks.
  - Remove granted roles and verify access is revoked.
  - Verify AND role checks require all values.
  - Verify OR role checks require any value.

- `TestRenewFlow`: tests token renewal.
  - Login with short token timeout.
  - Read initial TTL through `/api/token/ttl`.
  - Wait until TTL decreases.
  - Renew through `/api/token/renew`, expect TTL extended.

- `TestAutoRenewFlow`: tests automatic renewal.
  - Enable AutoRenew with a refresh threshold and renew interval.
  - Verify the initial renew interval blocks an otherwise eligible request.
  - After the interval expires, access a protected API and wait for actual TTL growth and a renewal event.
  - Immediately access again and verify the interval prevents an additional renewal event.
  - Observe TTL through the read-only Manager API so measurements do not trigger renewal.

- `TestRenewConfigurationMatrixFlow`: tests disabled renewal, refresh thresholds, and repeated renewal without an interval.
  - Use bounded polling of stored TTL instead of fixed delays for asynchronous renewal.

- `TestRenewBoundaryFlow`: tests manual renewal boundaries.
  - Reject zero and negative renewal values at the HTTP layer.
  - Accept a valid renewal and verify the new TTL.

- `TestTokenExpiredFlow`: tests token expiration.
  - Login with one-second timeout.
  - Wait until timeout passes.
  - Request protected API, expect unauthorized.

- `TestActiveTimeoutFlow`: tests inactive timeout.
  - Login with long absolute TTL and short active timeout.
  - Wait until inactive timeout passes.
  - Request protected API, expect active-timeout code.

- `TestKickoutAndReplaceFlow`: tests token state changes.
  - Kick out current token, expect old token rejected.
  - Replace current token, expect old token rejected.

- `TestSessionFlow`: tests session lookup.
  - Login.
  - Request `/api/session`.
  - Assert login ID and terminal count.

- `TestMultiTerminalSessionFlow`: tests multiple terminals.
  - Login same account from web and mobile.
  - Request `/api/session` with either token.
  - Assert terminal count is two.

- `TestTerminalInspectionFlow`: tests terminal metadata queries.
  - Login from web and mobile.
  - Request `/api/terminal`.
  - Assert terminal device info and online counts.

- `TestSessionQueryFlow`: tests session query APIs.
  - Query token lists by login ID, device, and concrete device ID.
  - Query terminal lists and traverse terminals.
  - Search token values and session IDs.

- `TestSessionAliveFilterFlow`: tests alive token filtering.
  - Mark one terminal offline.
  - Verify token list queries return only live session tokens.
  - Verify the offline token keeps its exact failure cause.

- `TestSessionExpiredTokenFilterFlow`: tests natural expiration filtering.
  - Keep a mobile login alive while a short-lived web token expires.
  - Verify `alive=false` retains both terminal entries and `alive=true` returns only the mobile token.

- `TestTerminalOperationFlow`: tests terminal-scoped operations.
  - Logout one concrete device and keep another terminal online.
  - Kick out all terminals for a device type.
  - Replace all terminals for an account.

- `TestTerminalOperationMatrixFlow`: tests account, device, and concrete-device operations.
  - Logout all terminals for an account.
  - Logout all terminals for a device type.
  - Kick out account and concrete device terminals.
  - Replace device type and concrete device terminals.

- `TestConcurrencyPolicyFlow`: tests login concurrency policies.
  - Shared token reuse for the same device.
  - New token creation for a different concrete device ID.
  - Account-level token reuse when no device dimension is supplied.
  - Account-level max login count overflow.
  - Overflow modes: logout, kickout, and replaced.
  - Device-level max login count overflow.
  - Account and device concurrency scopes.
  - Non-concurrent replacement and new-device rejection for account/device scopes.

- `TestDisableFlow`: tests account and service disable.
  - Account disable rejects old token and new login.
  - Account disable info and TTL can be queried.
  - Service disable rejects only `/api/payment`.
  - Service disable info, level, and TTL can be queried.

- `TestServiceDisableLevelFlow`: tests service disable levels.
  - Disable a service at level 3.
  - Verify lower/equal levels are blocked and higher level is allowed.
  - Untie the service and verify the level check is clear.

- `TestUntieFlow`: tests removing disable states.
  - Untie account disable and login again.
  - Untie service disable and access service again.
  - Untie device disable and login from that device again.

- `TestDeviceDisableFlow`: tests device disable.
  - Disable `web` device for current account.
  - Device disable info and TTL can be queried.
  - Verify web login is rejected.
  - Verify mobile login still succeeds.

- `TestConcreteDeviceDisableFlow`: tests concrete device ID disable.
  - Disable only `web/browser-1`.
  - Concrete device disable info and TTL can be queried.
  - Verify `web/browser-1` is rejected.
  - Verify `web/browser-2` is still accepted.

- `TestNonceFlow`: tests nonce.
  - Generate nonce through `/nonce`.
  - Check nonce validity and TTL without consuming it.
  - Verify once through `/nonce/verify`, expect success.
  - Verify the same nonce again, expect failure.

- `TestNonceTimeoutFlow`: tests nonce custom TTL.
  - Generate a nonce with a one-second TTL.
  - Verify it is initially valid.
  - Wait for expiration and verify it cannot be consumed.

- `TestOAuth2AuthorizationCodeFlow`: tests OAuth2 authorization code flow.
  - Generate authorization code.
  - Exchange code for access and refresh tokens.
  - Verify authorization code is single-use.
  - Introspect access token.
  - Refresh token, verify both old credentials are unusable, and introspect the new access token before revoking it.
  - Revoke refreshed token and verify it is invalid.

- `TestOAuth2AuthorizationCodeBindingFlow`: tests authorization-code client and redirect binding.
  - Reject another registered client and a mismatched redirect URI.
  - Verify the original client can still exchange the same code after those rejected requests.

- `TestOAuth2PasswordAndClientCredentialsFlow`: tests additional OAuth2 grants.
  - Password grant returns user token.
  - Client credentials grant returns client token.
  - Wrong client secret is rejected.

- `TestOAuth2ClientManagementFlow`: tests OAuth2 client management.
  - Register and query a client.
  - Use the registered client for client credentials.
  - Reject disallowed scopes.
  - Unregister the client and verify it can no longer be used.

- `TestMultiAuthIsolationFlow`: tests multiple auth systems.
  - Login the same ID into user-auth and admin-auth.
  - Verify tokens cannot cross auth systems.
  - Read both identities and compare permission and role lists to verify AuthType isolation.

- `TestDecodeFlowData`: rejects missing, null, malformed, and incorrectly typed data; clears stale fields when a destination is reused.
- `TestFlowClientDefaultsToMemory`: runs a real login and protected HTTP request with the Redis environment variable unset.
- `TestFlowStorageCleanup`: removes actual fixture keys, checks auth and nonce invalidation, and preserves an unrelated key in the same backend.

## Run

The automated flow tests run with in-memory storage by default. Set `DTOKEN_REDIS_URL` to run flow scenarios against Redis and exercise its storage and scanning behavior. The default command executes the flows without requiring an external service.

Each fixture receives a random `dt-gcf-<hex>:` key prefix, including across independent test processes. Cleanup scans and deletes only that fixture prefix. Use a dedicated test Redis database.

Example Redis URLs:

```text
redis://localhost:6379/0
redis://:password@localhost:6379/0
```

Without `DTOKEN_REDIS_URL`, flows run on memory storage. If Redis is configured but cannot be initialized, or any application setup fails, the test fails instead of silently skipping.

From this directory:

```powershell
go test ./...
```

To enable the Redis-backed flows, set the endpoint first:

```powershell
$env:DTOKEN_REDIS_URL='redis://:password@localhost:6379/0'
go test ./...
```

If your local Go environment has `GOOS=linux` on Windows, run with Windows target:

```powershell
$env:GOOS='windows'
$env:GOARCH='amd64'
go test ./...
```
