# SSO release acceptance checklist

[中文](RELEASE_CHECKLIST_zh.md) · [Testing details](TESTING.md) · [Deployment](DEPLOYMENT.md)

This checklist consolidates acceptance of the SSO implementation and examples. Regression sources have been authored and statically reviewed. Builds, tests, race detection, Redis, browser, and proxy integration have **not** been executed in this implementation pass. Static review is not a passing runtime result.

## Automated checks to run

Run from the repository root with the workspace's Go version and dependencies available:

```powershell
go test ./sso/... -count=1
go test ./examples/sso_server/... ./examples/sso_client/... ./examples/sso_gin_server/... ./examples/sso_gin_client/... ./examples/sso_deployment/... -count=1
go test -race ./sso/... ./sso/storage/redis/... ./examples/sso_server/... ./examples/sso_client/... ./examples/sso_gin_server/... ./examples/sso_gin_client/... ./examples/sso_deployment/... -count=1
go test ./sso/storage/redis/... -count=1
```

The Redis module's integration tests require `DTOKEN_REDIS_URL` pointing to a dedicated test Redis with GETDEL support (6.2+). Without that variable, Redis integration is skipped; constructor checks alone do not validate Redis. Race detection requires a supported Go race toolchain. The deployment shutdown test opens a temporary loopback listener. See [testing details](TESTING.md) for the full commands and storage behavior.

| Area | Required result | Current status |
| --- | --- | --- |
| Minimum Go | SSO and SSO Redis pass with their own declared Go, workspace and automatic toolchain upgrades disabled | CI matrix and -sso discovery added; pending execution |
| JSON response integrity | Encode before commitment; fixed 500 on failure, retained diagnostics and unchanged credential consumption | Five regression groups added; pending execution; see [response encoding](HTTP_ERRORS.md#response-encoding) |
| Center HTTP failures | Six endpoints hide internal diagnostics, retain classification and observe failures once without changing credential lifecycles | Regression sources and public facade example added; pending execution; see [HTTP errors](HTTP_ERRORS.md) |
| SSO release dependency set | Root, SSO, general Redis and SSO Redis remain at exact requested versions without replacements | Dedicated consumer and manual workflow added; pending execution |
| HTTP request limits | Six center endpoints and both callback types reject overflow with 413 before nonce/state/credential consumption | Six regression groups added; pending execution |
| Redis replica lifecycle | Independent connections preserve one-time credential/state/nonce competition, rotation, failed-logout recovery and account isolation | Four integration groups and CI race entry added; pending live Redis execution; see [acceptance](storage/redis/REPLICAS.md) |
| Signing-key rotation | Preparation, switch and retirement preserve client isolation and nonce history; delivery uses only active keys | Six SSO groups and two deployment tests added; pending execution |
| External consumers and documentation | SSO public APIs work with workspace disabled; requested published versions remain selected; marked configuration examples compile | Consumer fixture and CI entries added; pending execution |
| Request parameter consistency | Duplicate protocol fields and mixed credentials rejected before nonce/credential consumption; valid scalar and multi-value extension requests preserved | Eight test groups added; pending execution |
| Cancellation across protocol stages | Late hooks/transports cannot return identity, redirects or acknowledgements; attempt timeouts remain retryable, parent cancellation stops progress | Nine test functions added; pending execution |
| Administrative account logout | Center and old credentials revoked before delivery; failed targets retain retry bindings; recovery across replicas preserves other accounts | Sources added; pending execution |
| Protocol and state | All four modes, PKCE, replay checks, authorization policy, revocation and logout lifecycle regressions pass | Pending execution |
| Origins and configuration | Default-port/case/IP equivalence accepted only for origin comparisons; exact redirect binding preserved; invalid ports, userinfo and SameSite rejected | Sources added; pending execution |
| Cancellation | Pre-canceled requests never reach transport; canceled response bodies are closed; no identity or session accepted; browser state remains consumed | Sources added; pending execution |
| Session lifetime | Center authorization, exchange, and app sessions enforce absolute deadlines even when Auth TTL is longer | Sources added; pending execution |
| Resource ownership | Shutdown waits for handler completion after connection closure; borrowed storage is not prematurely disposed | Sources added; pending execution |
| Independent modules | SSO, Redis adapter and all five example modules checked explicitly | Pending execution |

For isolated-module, documentation and external-consumer commands, see the [upgrade and release guide](MIGRATION.md). The regular consumer check uses checkout replacements; the optional three-version invocation checks published root/Gin/SSO modules without replacements. The dedicated four-version SSO check also covers both Redis adapters; minimum Go has a separate CI matrix. Commands and boundaries are in the same upgrade guide.

## Deployment checks to perform

A [deployment acceptance entry point and recovery matrix](../examples/sso_deployment/ACCEPTANCE.md) now provide an opt-in live HTTP login/logout flow. Browser, replica and Redis failure checks remain separate. Only sources/static checks are complete; runtime results remain pending.

| Scenario | Acceptance evidence to record |
| --- | --- |
| Browser and HTTPS proxy | Alpha and Beta login, parallel tabs, Secure/SameSite cookies, forged Origin rejection, and redirects preserve the configured public origins |
| Shared Redis replicas | Begin login and complete callback on different replicas; session continuity after restart; a consumed callback cannot be replayed |
| Local/global logout | Local logout leaves other apps intact; global logout removes participating account sessions; another account remains active |
| Notification failure | Stop one app, observe strict-mode failure and partial completion, recover it and retry with a fresh request |
| Storage failure | Readiness becomes 503; identity and session checks do not silently fall back to memory; recover and retry through a fresh login when needed |
| Shutdown | Exercise both normal drain and forced connection closure with in-flight operations; hooks observe cancellation before storage disposal |

The 45-second drain interval is not a hard process deadline: shutdown still waits for handler completion after closing connections. Cancellation-ignoring hooks need a separate supervisor termination policy. Record environment, commit/worktree version, command, result and any skipped checks before considering runtime acceptance complete. Replace the deployment example's demo account authenticator before production use.

## Acceptance record template

Fill each row from execution evidence. Keep unexecuted checks pending; skipped Redis cases are not passes.

| Item | Record |
| --- | --- |
| Verification source | Commit SHA, uncommitted changes and selected workflow ref |
| Dependency set | Four requested and selected versions; whether replacements are enabled |
| Environment | OS/architecture, Go, GOWORK, GOTOOLCHAIN and race toolchain |
| Automated checks | Commands or CI links, exit codes, failures/skips; currently pending |
| Deployment | Redis version, replica layout, browser/proxy environment, recovery results; currently pending |
| Release decision | Passed items, blockers, owner and date; currently pending acceptance |
