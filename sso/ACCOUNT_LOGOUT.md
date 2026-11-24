# SSO account logout

[中文](ACCOUNT_LOGOUT_zh.md) · [Logout delivery](LOGOUT_RELIABILITY.md) · [Deployment](DEPLOYMENT.md)

`HTTPServer.LogoutAccount(r, loginID)` composes center-session revocation, SSO account credential invalidation, application notifications and binding cleanup for trusted administrative workflows. Use it for forced logout, session cleanup after account suspension, and authorized recovery after the original center cookie has become invalid. It targets all clients for an account within the current SSO storage namespace; it has no device or individual-login selector.

This method adds no HTTP route, does not resolve cookies, form identities or `LoginIDResolver`, and does not authenticate administrators. The caller must authorize the operator's access to the selected target. The request carries context and information needed by application hooks; never pass an arbitrary public request parameter directly as the target.

## Integration

Reuse the HTTPServer configured with registrations, signing, replay protection and notification delivery. An explicit center-session revoker is required:

```go
options := sso.DefaultHTTPOptions()
// Configure registrations, signing, replay protection and LoginIDResolver.
options.CenterLogout = func(r *http.Request, loginID string) error {
    return centerAuth.LogoutByLoginID(r.Context(), loginID)
}
handler, err := sso.NewHTTPServerChecked(server, options)
if err != nil {
    return err
}

// Authenticate and authorize the operator first; suspension also blocks new login/issuance.
result, err := handler.LogoutAccount(r, authorizedLoginID)
// Audit result.Stage, progress flags and target outcomes without exposing raw errors.
if err != nil {
    return err
}
_ = result.Complete
```

See [example_account_logout_test.go](example_account_logout_test.go) for a compilable composition example. The `sso/httpserver.Server` alias exposes the same method and exports `AccountLogoutResult`. The deployment example retains `application.centerHTTP` for trusted application integration; it exposes no management route, and its demo authenticator does not implement account suspension.

`CenterLogout` must revoke the supplied account's center sessions and tolerate repeated calls. It must not infer the target from the administrator's cookie. Missing hooks return `ErrAccountLogoutConfig` before any mutation or notification; existing constructors and browser logout retain their optional-hook semantics.

## Ordering and partial completion

The method returns `(*AccountLogoutResult, error)`, including progress on failure:

| Stage | Operation | Success indicator |
| --- | --- | --- |
| `validation` | Check initialization, target, request, hook and cancellation | None |
| `center_logout` | Invoke application-owned `CenterLogout` | `CenterRevoked` |
| `credentials` | Replace the account credential generation using `RevokeLoginCredentials` | `CredentialsRevoked` |
| `notifications` | Deliver callbacks when SLO is enabled | Per-target `Notifications` |
| `cleanup` | Clear bindings after successful notification delivery | `BindingsCleared` |
| `complete` | All enabled stages succeeded | `Complete` |

Center sessions and old credentials are revoked before delivery. An unreachable application therefore cannot preserve otherwise revocable center state. Notification failures always return errors and retain every binding, including when `LogoutCallbackBestEffort` is enabled. Completed effects are not rolled back: successful targets may already be logged out, while unavailable targets can retain local sessions.

With `EnableSLO=false`, center sessions and credentials are revoked without notification or binding cleanup. `Notifications` is nil and `BindingsCleared` is false; `Complete` can still be true. Complete describes the enabled lifecycle, not proof that every application is offline. Removed clients and bindings without callbacks keep the notification module's existing skipped semantics.

Flags record successful returns, not independent verification of application hooks. Errors can still follow partial effects. Cancellation is checked around stages; custom hooks/storage returning success after cancellation cannot make the overall operation succeed, while acknowledged effects remain recorded.

`Cause` retains the returned error for `errors.Is`. Overall `LoginID`, overall and per-target `Cause` are excluded from JSON. Reports still contain client IDs and should be restricted to authorized operators. This method writes no response, clears no administrator cookie, and does not invoke browser-specific `OnLogoutResult`; audit and metrics use the returned report.

## Retry and suspension

Delivery reuses current registrations, allow-lists, signing keys, fresh replay nonces, concurrency limits, timeouts, acknowledgements and bounded retries. After failure, call `LogoutAccount` again with a fresh context: center revocation, generation replacement and all current target notifications run again, including previously successful targets. Both center and application logout hooks must be idempotent. Recovery needs no valid center browser session or replay of an old signed request.

There is no durable job queue, saved target-success progress or automatic background retry. Retrying can affect sessions established in between calls. For suspension, first disable login in the identity system and block issuance/exchange/inspection through [AuthorizationPolicy](AUTHORIZATION_POLICY.md), then perform logout and recovery. Finish or cancel outstanding administrative operations before restoring the account. A forced logout alone allows the account to authenticate again if login remains enabled.

Account generations cover versioned Tickets, authorization codes, shared tokens and remote sessions. Legacy records without a version retain existing compatibility behavior and require separate expiry/migration planning. Trusted low-level issuance APIs do not invoke HTTP policy; account status must also be enforced at those entry points.

Instances sharing storage and prefixes see the same revocation generation and binding index. Operations remain sequential with ordinary storage, without transaction rollback or cross-instance locks. Coordinate strict exclusion against new login through application-owned account state. Logout does not retract business requests already running.

## Verification status

Added regression sources cover all four modes, ordinary/atomic storage, shared instances, account isolation, revocation before delivery failure, binding retention and recovery, storage-stage failures, cancellation at each stage, missing configuration, disabled SLO, safe serialization and real Auth sessions with signed/replay-protected multi-app deployment callbacks.

Only formatting and static inspection were performed; tests, builds and services were not executed. Run manually from the repository root:

```powershell
go test ./sso/... -run 'TestAccountLogout|ExampleHTTPServer_LogoutAccount' -count=1
go test ./examples/sso_deployment/... -run TestDeploymentAccountLogout -count=1
go test -race ./sso/... ./examples/sso_deployment/... -run 'TestAccountLogout|TestDeploymentAccountLogout' -count=1
```

Continue full regression and Redis/browser acceptance using the [release checklist](RELEASE_CHECKLIST.md).
