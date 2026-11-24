# SSO Testing

This document describes recommended verification flows for the standalone SSO module, including in-memory mode, Redis mode, Server/Client integration, and single logout.

## Release integration checks

The existing CI now includes an explicit race step for SSO, its Redis adapter and all five SSO examples. The external-consumer fixture in [sso-consumer_test.go](../.github/scripts/testdata/sso-consumer_test.go) checks four modes, checked constructors, the HTTP facade, signing, custom parameter names, replay configuration, PKCE and consumed/revoked credentials using only public APIs. It is copied into a temporary external module by [check-consumer.sh](../.github/scripts/check-consumer.sh); ordinary module test commands do not discover this `testdata` fixture.

Both configuration guides now contain complete marked programs consumed by the existing documentation compiler. These additions are pending execution. Commands, published-version selection and verification boundaries are in the [upgrade guide](MIGRATION.md).

New minimum-Go CI entries and a dedicated four-module consumer cover published Redis construction/ownership and browser-to-account-logout public APIs. The dedicated consumer disables workspace/toolchain upgrades and runs race detection without Redis. The scripts, discovery regressions and CI additions have not run; see [dedicated release checks](MIGRATION.md#dedicated-sso-release-checks).

## Unit Tests

Center error regressions cover six routes, private diagnostics, initialization paths, replay storage recovery, four credential modes and pre/post-consumption exchange failures. Logout regressions also check observer counts for strict/best-effort outcomes. Sources have not run; see [HTTP error verification](HTTP_ERRORS.md#lifecycle-and-verification).

Cookie rotation regression sources cover the cookie trust scope, primary-only issuance, preparation/cutover/retirement, signed expiry and pending PKCE state. Deployment cases exercise old/new replicas against shared memory storage, center-session reuse and revocation after rotation. These cases have not run; configuration, focused commands and rollout boundaries are in [cookie rotation](SIGNING_ROTATION.md#browser-session-and-state-cookie-rotation).

[logout_callback_errors_test.go](logout_callback_errors_test.go) covers callback error redaction for ordinary/atomic replay storage and application hooks, retained direct-API diagnostics, protocol status categories, and fresh retries after hook failure. These regressions have only been statically reviewed, not executed; see [callback behavior](LOGOUT_RELIABILITY.md#retry-policy).

[request_reparse_test.go](request_reparse_test.go) checks repeated validation after body parsing failure and preserves unused nonce/credentials for fresh requests. Deployment HTTP acceptance also requires captured cookies and the expected center authorization redirect after revocation. Static review only; new regressions have not run.

[request_limits_test.go](request_limits_test.go) adds inbound limit coverage for every center entry point, all four credential/browser modes, streaming reads, cached forms, safe parsing errors, and preserved nonce/state/credentials after rejection. Sources have not run; see [HTTP request limits](REQUEST_LIMITS.md) for focused commands and compatibility.

[signing_keys_test.go](signing_keys_test.go) adds six rotation groups covering trust scopes, six-action nonce continuity, callback retirement, four credential modes, retry signing and configuration errors. The deployment module adds differing-key login/logout and configuration checks. These tests have not run; see [rotation verification](SIGNING_ROTATION.md).

Request consistency regressions in [request_validation_test.go](request_validation_test.go) add eight test groups covering configured scalar fields, four credential modes, signed/unsigned requests, query/body duplicates, custom parameter names, rejection before nonce/state/credential consumption, and absence of application/logout effects. Tests have not been executed. See [request validation](REQUEST_VALIDATION.md) for compatibility and focused commands.

The current static regression review adds the following cancellation/timeout matrix: nine test functions plus their parameterized scenarios. Sources are complete; tests have **not been executed**:

| Source | Boundary |
| --- | --- |
| [cancellation_center_review_test.go](cancellation_center_review_test.go) | Cancellation before issuance and after persistence in all four modes; introspect/userinfo hooks; enrichment after Ticket/Code consumption |
| [cancellation_logout_review_test.go](cancellation_logout_review_test.go) | Transports/bodies ignoring attempt deadlines, timeout retries and recovery, parent cancellation classification, center logout stages, client acknowledgements and nonce consumption |
| [cancellation_browser_review_test.go](cancellation_browser_review_test.go) | Browser state persistence, local login and local/global logout hooks, consumed state remaining consumed |

Canceled flows must not return identity, successful redirects or logout acknowledgements, or enter later side-effecting stages. Completed storage writes or application effects are not rolled back; consumed credentials and nonces are not restored. Custom hooks/transports must still honor Context: checks run when calls return and cannot interrupt an implementation that never returns.

Run focused checks first, then the full module and deployment checks below and in the [release checklist](RELEASE_CHECKLIST.md):

```powershell
go test ./sso/... -run TestCancellationReview -count=1
go test -race ./sso/... -run TestCancellationReview -count=1
```

These added cases use memory storage and controlled transports, without real Redis or external services. Formatting and static source inspection are not Go compilation, test or race-detector results.

Account logout regression sources (`account_logout_test.go`, `example_account_logout_test.go`, and the deployment module's `account_logout_test.go`) cover revocation before notification, four credential modes, ordinary/shared storage, stage failures and recovery, cancellation, safe reporting, disabled SLO, and real Auth sessions with signed multi-app callbacks across replicas. These sources have not been executed. Focused commands and lifecycle limits are in [account logout](ACCOUNT_LOGOUT.md).

The current finishing batch adds `origin_review_test.go`, `client_cancellation_test.go`, and `examples/sso_deployment/lifecycle_review_test.go`. They cover origin equivalence versus exact redirect binding, invalid ports/SameSite, late successful responses after cancellation, absolute center deadlines, and waiting for canceled handlers before resource disposal. Invalid `Close` calls on resource-free SSO memory storage were removed from earlier regression sources. These cases have not been executed; see the consolidated [release checklist](RELEASE_CHECKLIST.md).

Run from the repository root. The SSO module uses in-memory storage by default; include its subpackages:

```powershell
go test ./sso/... -v
```

Redis and the example applications are separate workspace modules, so the command above does not include them:

```powershell
go test ./sso/storage/redis/... -v
go test ./examples/sso_server/... ./examples/sso_client/... -v
go test ./examples/sso_gin_server/... ./examples/sso_gin_client/... -v
go test ./examples/sso_deployment/... -v
```

Redis constructor tests run without a service; the Redis integration flow is skipped unless `DTOKEN_REDIS_URL` is set. These commands do not launch the demo applications.

Main coverage:

- The complete deployment example adds real-handler multi-app and replica flows, signed global logout, forged callback rejection, consumed browser state, deployment configuration, login/logout origin validation, readiness failures, cookie policy, and borrowed storage ownership. These example cases have been authored but not executed; follow [deployment acceptance](DEPLOYMENT.md) for Redis and browser verification.

- Configuration/startup regression sources (`config_validation_test.go`): invalid routes and decoded collisions without mux panic, rejection before storage writes, borrowed storage ownership, startup storage failures, registration metadata, signing/cookie/replay combinations, safe diagnostics, Browser/RemoteAuth constructor integration, and checked signed authorization/exchange. These added cases have not been executed in this implementation pass; see [configuration guidance](CONFIGURATION.md).

- Ticket issue, validate, consume, revoke, and expiration.
- Shared Token issue, validate, revoke, and expiration.
- Remote Session create, validate, renew, revoke, and expiration.
- OAuth2 Code issue, consume, revoke, and boundary errors.
- HTTP protocol endpoints: `authorize`, `token`, `introspect`, `userinfo`, `revoke`, `logout`.
- ClientApp: authorization URL, ticket exchange, signature verification, single logout callback Handler.
- Client protocol response limits: default/custom limits, exact-boundary acceptance, unknown or inaccurate Content-Length, success/error bodies, read failures at and before the limit, body closure, and no partial results across exchange, introspection, userinfo, and revocation. Both exchange modes reject timeout values below -1 while preserving valid sentinels and large integer values.
- Exchange-result enrichment preserves credential identity and scopes, carries center-session metadata, and fails without returning partial data or registering new callbacks; hook failures cannot replay consumed credentials.
- Authorization metadata comes from a trusted hook for all four modes; invalid requests are rejected before the hook, and hook failures do not issue credentials or register callbacks.
- Account logout invalidates all four credential modes across clients and shared-storage Server instances, with namespace isolation, ordinary-storage fallback, fresh issuance after logout, and legacy-record compatibility.
- Logout-version storage errors remain visible, failed logout preserves cookies, and atomic Ticket/Code consumption rechecks revocation after taking the credential.
- ClientSession: register client sessions, update, query, and clear.
- Dedicated signing-key precedence, global-key compatibility, key rotation, and rejection of cross-client signature impersonation.
- A single logout fan-out signs for each target client, including a mix of dedicated and fallback keys.
- Logout acknowledgements distinguish HTTP success from business success; malformed, oversized, and unreadable responses retain strict-mode retry state, and all failed client IDs are reported.
- Signed ClientApp/HTTPServer flows across all four modes, with custom parameter names, route prefixes, and both atomic and ordinary storage capabilities.
- Rejected client/redirect bindings preserve credentials; consumed or revoked credentials cannot expose user info.
- Authorization redirects and JSON responses prohibit caching; malformed logout forms return HTTP 400 without invoking the local logout action.
- Both client examples check browser state, server-side session expiry, and session replacement after successful exchange.
- Both client examples cap local lifetime using positive center Token/Session TTLs, count exchange latency, handle unspecified/unlimited and oversized values, and preserve existing sessions on invalid TTLs. Enriched exchange tests verify Cookie and local-store deadlines together.
- Both center examples compose Auth login, trusted session binding, exchange enrichment, and account logout; cases cover copied Cookies, multiple browsers, account isolation, retired and unbound Tickets, token nondisclosure, and strict/best-effort callback behavior.
- Application-state checks during introspection/userinfo cover all four modes: rejected sessions hide metadata, lookup failures remain errors, invalid clients/credentials never invoke the hook, nil hooks retain existing behavior, and revocation remains independent. Both centers reject reads after only the original Auth token is revoked, even while the SSO Ticket and another browser remain active.

## Replay Protection

The replay regression tests cover timestamp boundaries, missing or duplicate metadata, operation binding, invalid-signature isolation, atomic claims across instances, ordinary-storage serialization, storage faults, and nonce retention. HTTP cases exercise all four credential modes with custom paths and parameter names, authorization login round trips, and callback failure/retry behavior.

Run these checks manually from the workspace root:

```powershell
go test ./sso/... -run TestReplay -count=1
go test -race ./sso/... -run TestReplay -count=1
```

These tests use in-process HTTP handlers and memory-backed storage. They do not verify real Redis connectivity or deployment clock synchronization. For deployment, enable protection on both sides, use the same storage/prefix/window across replicas, then verify that replaying the exact captured request fails while a newly generated request succeeds. See [configuration and retry semantics](./REPLAY_PROTECTION.md).

## Browser Integration Regression

`browser_test.go` and `browser_protocol_test.go` cover parallel tabs, browser binding, duplicate callbacks, state expiry, return paths, all credential modes with replay protection, state consumption concurrency/faults, session deadlines, and logout origins. `examples/sso_client/browser_test.go` checks application session replacement and local revocation before global logout. Run manually from the workspace root:

```powershell
go test ./sso/... -run TestBrowser -count=1
go test -race ./sso/... -run TestBrowser -count=1
go test ./examples/sso_client/... -run TestBrowserDemoLifecycle -count=1
```

These regressions do not replace real-browser verification of SameSite, Secure, cookie paths, and concurrent tabs, or shared Redis deployment checks. See the [browser integration guide](./BROWSER_CLIENT.md).

## Single Logout Reliability Regression

`logout_notifications_test.go` and `logout_notification_lifecycle_test.go` cover failed-target retries, terminal acknowledgements, response cleanup, shared concurrency across requests, attempt timeouts, parent cancellation, removed registrations and changed allow-lists, key rotation and fresh retry nonces, strict/best-effort lifecycle results, observer isolation, and safe HTTP output. Run manually from the workspace root:

```powershell
go test ./sso/... -run 'TestLogoutNotification|TestLogoutAcknowledgement|TestLogoutReportsAllFailedClients|TestCenterLogoutLifecycle' -count=1
go test -race ./sso/... -run TestLogoutNotification -count=1
go test ./examples/sso_server/... ./examples/sso_gin_server/... -count=1
```

These cases use memory storage and controlled transports. They do not replace real-network timeout, Redis recovery, or multi-replica capacity verification. See [single logout reliability](./LOGOUT_RELIABILITY.md) for configuration and lifecycle limits.

## Gin Example Integration

Start the login center:

```powershell
go run ./examples/sso_gin_server
```

Start the client app:

```powershell
go run ./examples/sso_gin_client
```

Open the protected resource:

```text
http://localhost:9101/protected
```

Expected flow:

1. The unauthenticated client creates a signed five-minute browser-state Cookie and redirects to `/sso/authorize`; an unauthenticated center then redirects to `http://localhost:9100/login`.
2. The login center writes the center Cookie and redirects back to `/sso/authorize`.
3. The login center issues a Ticket and redirects to the client `/sso/callback`, echoing browser state in `back`.
4. The client checks its signed state Cookie against `back` before calling `/sso/token` for `loginId`. Missing, mismatched, tampered, or expired state must not trigger exchange.
5. The client creates a local session with a two-hour server-side deadline. `/protected` returns the login subject, and a copied Cookie cannot extend the session.

Only one authorization flow is pending per browser. Accepted state is cleared from the browser even if exchange fails; restart from `/protected` instead of refreshing the callback. Restarting either demo process invalidates its process-local state.

## Single Logout Verification

After login, while the center login Cookie is still present, open:

```text
http://localhost:9100/sso/logout
```

The server resolves the logout subject from the trusted login resolver; a `loginId` query parameter cannot select an arbitrary account.

Both centers validate the Cookie against their Auth instance. Successful logout invalidates all center browser sessions for the account; replaying a saved Cookie must redirect authorization to `/login`. Another account should still authorize, and a fresh login should issue usable Tickets. Exchange returns a positive `remainTokenTimeout` and `deviceId` without exposing the center token. Revoke the original Auth token between authorization and exchange to verify that another active browser cannot rescue its Ticket.

When the client callback succeeds:

- SSO Server clears the center Cookie.
- SSO Server pushes `/sso/logout-callback` to registered clients.
- Client deletes local sessions for the received `loginId`.
- Opening `http://localhost:9101/protected` again redirects to the login center.

The standard-library server uses strict single logout: a callback failure retains the center Cookie and client-session index for retry, although callbacks that already succeeded are not rolled back. The Gin server uses best-effort single logout: it clears the center Cookie and index even if a callback fails. An unreachable client can therefore retain its local sessions until local logout or their deadline. A successful center response alone does not prove that every client logged out.

Client `/logout` only removes the current browser's local session. If the center remains logged in, visiting `/protected` can immediately sign in again.

If signing is enabled, set the same `SecretKey` on both Server and Client and set `CheckSign` to `true`. Client-side `LogoutCallbackHandler` verifies callback signatures automatically.

`LogoutCallbackHandler` requires the configured client ID and a timestamp within five minutes of the client's clock. This rejects stale requests but does not deduplicate valid callbacks within that window; the local logout action should be idempotent. Client ID and timestamp alone do not authenticate the sender when signing is disabled.

## Redis Mode Verification

The live Redis [replica lifecycle suite](storage/redis/REPLICAS.md) adds four flows: concurrent PKCE redemption, authorization replay and key rotation, browser-state recovery across connections, and cross-center recovery after partial account logout. Set `DTOKEN_REDIS_URL` explicitly; the CI Redis step enables race detection for the SSO Redis module. The suite starts no example processes and has not run in this batch.

Production deployments should use Redis storage:

```go
import (
	"github.com/Zany2/dtoken-go/sso"
	ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"
)

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithKeyPrefix("dtoken:"),
	sso.WithAuthType("sso:"),
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
defer server.Close()
```

Recommended Redis key prefixes for the configuration above:

- `dtoken:sso:sso:client:`: registered client app data.
- `dtoken:sso:sso:ticket:`: one-time Ticket, deleted after consume.
- `dtoken:sso:sso:oauth2:code:`: OAuth2 Code, deleted after consume.
- `dtoken:sso:sso:client-session:`: client session records used by single logout, cleared when the configured logout policy completes.
- `dtoken:sso:sso:logout-version:`: persistent account revocation markers; do not delete them independently while associated credentials may remain.

Keys concatenate `keyPrefix + authType + key suffix + identifier`. The suffix constants already begin with `sso:`, so this configuration produces two adjacent `sso:` segments.

Verification checklist:

1. Before login, confirm the client registration key exists.
2. During login, observe the Ticket key appear briefly.
3. After client ticket exchange succeeds, the Ticket key should be deleted.
4. After `/sso/logout` completes under the configured policy, the matching `dtoken:sso:sso:client-session:` key should be deleted.
5. The account's `dtoken:sso:sso:logout-version:` marker should remain. Previously issued versioned credentials should become inactive, while newly issued credentials should validate successfully.

Optional integration test:

```powershell
$env:DTOKEN_REDIS_URL="redis://:password@127.0.0.1:6379/0"
go test ./sso/storage/redis/... -v
```

When `DTOKEN_REDIS_URL` is not set, this test is skipped automatically.

Use a test Redis instance with `GETDEL` support (Redis 6.2+). Each integration run uses an isolated key prefix and cleans up its records before closing the connection. The flow covers Ticket/Code consumption, Shared Token revocation, Remote Session renewal and revocation, and client-session cleanup. A skipped integration test does not verify a real Redis deployment.

## Security Boundaries

- Production deployments should enable `CheckSign` and configure `SecretKey`.
- Ticket and OAuth2 Code `redirect` values must exactly match the client's `RedirectURIs`.
- Single logout `callback` values must belong to the current client: exact `RedirectURIs` match, same origin as a registered redirect URI, or explicit `AllowOrigins` match.
- Avoid adding overly broad origins to `AllowOrigins`, otherwise malicious callback URLs can increase SSRF risk.
- Logout callbacks are valid for 5 minutes by default and are rejected by the Client after that window.

## API Naming Stability

Current recommended names:

| Name | Role |
| --- | --- |
| `ClientApp` | Client-side integration helper |
| `ClientSession` | Login-center binding between login subject and client app |
| `LogoutCallback` | Parsed single logout callback data received by the client app |
| `VerifyLogoutCallback` | Client-side manual callback verification and parsing |
| `LogoutCallbackHandler` | Client-side standard callback Handler |
| `LogoutCallbackBestEffort` | Server-side policy for clearing center records when callbacks fail |

These names match the current SSO responsibilities and do not need another split or rename for now.

## PKCE credential-binding regression

Run these manually; they were not executed in this change:

```powershell
go test ./sso/... -run 'TestPKCE|TestBrowserProtocolModes' -count=1
go test -race ./sso/... -run 'TestPKCEConsumptionLifecycle|TestPKCEBrowserFlowIsolation' -count=1
go test ./examples/sso_client/... -run TestBrowserDemo -count=1
```

Coverage includes the RFC 7636 vector, S256 validation, current registration policy, legacy bypass, ordinary/atomic storage, invalid-proof preservation, concurrent single consumption, logout invalidation, HTTP identity bypass, POST proofs, signing, custom parameters, and swapped tab credentials. Real Redis and browser integration still require deployment checks. Only formatting and static review were performed. See the [PKCE guide](PKCE.md) for configuration and rollout.

## Remote authentication regression

Run manually; this command was not executed in this change:

```powershell
go test ./sso/... -run 'TestRemoteAuth|TestRemoteBearerCredential' -count=1
```

Regression sources cover both reusable modes, signing/replay protection, custom protocol names, per-request checks, revocation/account logout, scopes, ambiguous bearer input, identity binding, cancellation/timeouts, business Context, response size limits, and bounded errors. See `example_remote_auth_test.go` for integration code. Only formatting and static checks were performed; no services, builds, tests, or real storage integration were run. See [remote authentication](REMOTE_AUTH.md).

## Dynamic authorization policy regression

Run manually; this command was not executed in this change:

```powershell
go test ./sso/... -run 'TestAuthorization|TestExchangeResultEnrichment|TestPKCE|TestRemoteAuth' -count=1
```

New sources cover issuance/inspection in all four modes, nil versus empty grants, scope expansion rejection, trusted policy input, preserved credentials on ordinary/atomic storage, PKCE ordering, callback registration, exchange without SLO, bounded policy errors, cancellation, signed replay-protected browser login, and remote authentication integration. Only formatting and static checks were performed; tests, builds, examples, and real storage integration were not run. See [authorization policy](AUTHORIZATION_POLICY.md).

## Running-service acceptance

`examples/sso_deployment/acceptance` adds one opt-in live HTTP scenario and two offline client-boundary cases. Ordinary example tests skip live access. Setting `SSO_ACCEPTANCE_RUN=true` logs in and globally logs out the configured account; use a dedicated test account. See the [acceptance guide](../examples/sso_deployment/ACCEPTANCE.md) for variables, expected statuses and failure scenarios. No cases or services were run in this pass.
