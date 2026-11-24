# SSO deployment acceptance and recovery

[中文](ACCEPTANCE_zh.md) · [Startup configuration](README.md) · [Release checklist](../../sso/RELEASE_CHECKLIST.md)

This guide targets this example's center and two applications. The `acceptance` package uses real HTTP addresses of already-running services. It does not start processes, access Redis administration, change configuration or stop services. The guide and fixtures have not been executed; they are not evidence of a passing deployment.

## Automated HTTP acceptance

Start center, alpha and beta using the README. Use a dedicated test account: global logout revokes that account's participating sessions, including other browsers. The fixture expects this example's login form, not arbitrary production login, CAPTCHA or MFA. Adapt the login steps when replacing the authenticator.

In a fourth PowerShell terminal, from the repository root:

```powershell
$env:SSO_ACCEPTANCE_RUN = 'true'
$env:SSO_ACCEPTANCE_CENTER = 'http://localhost:9200'
$env:SSO_ACCEPTANCE_ALPHA = 'http://localhost:9201'
$env:SSO_ACCEPTANCE_BETA = 'http://localhost:9202'
$env:SSO_ACCEPTANCE_USER = 'alice'
$env:SSO_ACCEPTANCE_PASSWORD = 'local-example-password'
try {
    go test ./examples/sso_deployment/acceptance -run '^TestLiveDeployment$' -count=1 -v -timeout=3m
    if ($LASTEXITCODE -ne 0) { throw 'SSO HTTP acceptance failed; inspect the named stage.' }
} finally {
    Remove-Item Env:SSO_ACCEPTANCE_RUN -ErrorAction SilentlyContinue
    Remove-Item Env:SSO_ACCEPTANCE_PASSWORD -ErrorAction SilentlyContinue
}
```

For HTTPS, supply the three public root origins without paths, trailing slashes, queries or credentials. Inject a dedicated password securely in the current terminal instead of using demo credentials. Certificates must be trusted by the system; TLS verification is never skipped. Origins must be distinct; plain HTTP is limited to localhost/loopback. Center and applications must also reach each other's configured public addresses.

Only `SSO_ACCEPTANCE_RUN=true` enables live traffic. Otherwise the live case skips. Ordinary `go test ./examples/sso_deployment/...` still runs offline boundary tests for the acceptance client, logout reports and cleanup requests. Do not persist the opt-in variable in general CI. Release CI does not deploy services or replace this acceptance step.

| Step | Required result |
| --- | --- |
| First Alpha login | Center login preserves the authorization target and PKCE callback creates a local session |
| Beta login | Reuses the center session without submitting credentials again |
| Cookies/responses | Issued session/state cookies are HttpOnly, host-only, SameSite=Lax and Secure on HTTPS; responses are no-store |
| Callback boundaries | Oversized query returns 413, original callback still works, consumed callback returns 400 |
| Forged logout | GET logout returns 405, wrong Origin returns 403, unsigned callback returns 401 |
| Local logout | Old Alpha session fails, Beta remains active, Alpha signs in again through the center |
| Global logout | HTTP 200, code=0, result=ok, complete stage and notifications; each actual application ID appears once with delivered status; both old application sessions fail; the old center cookie leads back to login |

Redirects are never followed implicitly: every hop must match the expected origin/path, and passwords are POSTed only to the configured center `/login`. Cookies, signed URLs, response bodies and passwords remain out of logs/files. Each request has a 15-second timeout and a 1 MiB response limit. Failures report stage/status without credential-bearing network diagnostics. Cleanup attempts global logout with a fresh request and reports unconfirmed cleanup for manual recovery. Unfinished state/nonce records expire under their normal TTL; no Redis flush occurs.

Application IDs are read from each authorization redirect, must be distinct, and must remain stable across repeated login. The logout report must include those IDs; duplicate, missing or unrelated IDs cannot substitute for an expected application. Extra targets are allowed only when every reported target is delivered. Cleanup uses the same business-result checks for known applications: HTTP 401 or HTTP 200 with partial results does not confirm cleanup. Cleanup is skipped only after the main flow has verified the logout report and revocation of both application sessions and the original center session. The offline cases in [logout_boundary_test.go](acceptance/logout_boundary_test.go) cover these checks with injected transports; they have not been executed.

The proxy must allow a request line slightly larger than 8 KiB to exercise the application's 413. The updated nginx template uses `large_client_header_buffers 4 16k`. A 414 or proxy error page from another gateway means rejection happened there and is not proof of library-level rejection. Do not raise SSO limits to hide that distinction.

An HTTP cookie jar does not enforce browser SameSite/CSP policies; this test only inspects cookie attributes. It also does not prove restart continuity, actual replica routing, Redis failure recovery or account isolation. Complete the following acceptance separately.

## Browser and parallel tabs

1. Use a fresh browser profile/private session over HTTPS. Verify successful Alpha login and Secure/HttpOnly/SameSite cookie attributes.
2. Begin login in two tabs, then complete both callbacks. Starting the second flow must not overwrite the first state.
3. Open Beta to verify center reuse; exercise local and global logout. Refresh protected pages and check access rather than only disappearing buttons.
4. Record browser version, origins, proxy version and outcome. Do not export HAR files containing passwords, cookies, tickets, nonces or signed queries; retain only paths, statuses and stages.

## Failure and recovery matrix

An operator controls services/network in an isolated test environment. Fixtures accept no shell commands and stop no processes. Restore health and log in to both apps before each scenario. Session continuity requires Redis and stable namespaces, cookie keys and registration configuration.

| Scenario | Action | During failure | After recovery |
| --- | --- | --- | --- |
| Beta notification failure | Sign in to both apps, stop Beta, globally log out from Alpha | Alpha is locally logged out; center reports notification-stage failure and failed target, retaining center session and bindings | Start Beta, sign in through Alpha and initiate a fresh global logout; do not refresh the old signed URL |
| Application restart | With Redis, stop/restart Beta without changing configuration | Requests fail while stopped | Original unexpired browser session remains usable; memory mode has no continuity guarantee |
| Cross-replica callback | Two app replicas share Redis, keys, ID and public origin; route initiation to A and callback to B | Login completes once | Callback replay fails; retain actual routing evidence rather than assuming load-balancer distribution |
| Redis unreachable | After login, block service access to dedicated test Redis without deleting data | Readiness 503; existing-session access errors instead of memory fallback or successful anonymous access | Restore connectivity and verify readiness/old sessions if still unexpired and data retained; restart failed login flows |
| Center restart | Stop/restart a Redis-backed center with identical registrations and keys | New authorization/exchange is temporarily unavailable | Unexpired center session remains reusable; consumed nonces/tickets remain unusable |
| Shutdown drain | Terminate with requests in flight and observe process/connections/logs | Stops accepting requests, drains then closes connections after grace period, waits for handlers | Storage outlives handlers; supervisor policy handles application hooks that ignore cancellation |

The proxy restricts `/health/ready`; verify it from an allowed operations source. External 403 is not Redis-failure evidence. Browser global logout and administrative `LogoutAccount` have different stage ordering; the outage scenario above concerns browser logout. See [account logout](../../sso/ACCOUNT_LOGOUT.md) for administrative recovery.

## Evidence and completion

Record source version, environment, storage mode, steps, expected/actual statuses, restoration and sanitized diagnostics per scenario. Keep HTTP, browser and Redis/replica/failure results separate; skips are not passes. After restoring all services/network, repeat HTTP acceptance to detect lingering effects.

Only the acceptance entry point, boundary regression sources and instructions are complete. Runtime results remain pending user execution and belong in the [release checklist](../../sso/RELEASE_CHECKLIST.md).
