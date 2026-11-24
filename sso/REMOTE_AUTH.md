# SSO remote session authentication

`RemoteAuth` validates each protected application request against the center using `ClientApp.Introspect`. It supports `ModeRemoteSession` and `ModeSharedToken`, with no success cache, local session creation, automatic renewal, or retries. Ticket and OAuth2 code flows remain one-time exchanges and cannot be used here.

## Protecting an API

```go
app := sso.NewClientApp(sso.ClientConfig{
    Mode: sso.ModeRemoteSession,
    ClientID: "orders",
    ClientSecret: "replace-client-secret",
    ServerURL: "https://sso.example.com",
    CheckSign: true,
    SecretKey: "replace-signing-secret",
})
guard, err := sso.NewRemoteAuth(app, sso.RemoteAuthOptions{
    RequiredScopes: []string{"orders:read"},
    Timeout: 2 * time.Second,
})
if err != nil {
    return err
}
mux.Handle("/orders", guard.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    identity, _ := sso.RemoteIdentityFromContext(r.Context())
    // Resource ownership checks still belong to the application.
    _, _ = fmt.Fprintf(w, "Orders for %s", identity.LoginID)
})))
```

The default resolver accepts exactly one `Authorization: Bearer <credential>` header. Use a remote session ID or shared token according to the configured mode. Incoming application requests cannot switch modes. Duplicate or combined Authorization headers, controls, empty values, and wrong schemes are rejected. URL parameters and cookies are never fallback sources. A center login cookie, center Auth token, or one-time ticket is not a reusable SSO credential.

Construction validates configuration without network access: mode, center introspection URL, client ID, signing, replay protection, parameter names, timeout, and scopes. Existing HTTP client settings, response size limits, endpoints, custom parameters, signatures, and replay protection remain effective. Redirects are not followed by default; an explicitly supplied redirect policy remains the caller's responsibility.

Successful middleware calls inject identity into the request context, without adding the raw credential to it. Treat identity, Scopes, and Extra as read-only request data, not durable session state. `ExpiresIn` remains the center's reported snapshot; the guard conservatively accounts for request latency when deciding whether it is still usable. Validation occurs before entering the business handler and does not interrupt a long-running handler or WebSocket after entry.

## Center issuance and state

Register matching modes and scopes. Issue credentials only after trusted application authentication:

```go
err := server.RegisterClient(&sso.Client{
    ClientID: "orders",
    ClientSecret: "replace-client-secret",
    SecretKey: "replace-signing-secret",
    Modes: []sso.Mode{sso.ModeRemoteSession},
    Scopes: []string{"orders:read"},
})
if err != nil {
    return err
}
session, err := server.CreateRemoteSession(ctx, "orders", authenticatedLoginID,
    []string{"orders:read"}, nil)
```

`authenticatedLoginID` must come from a trusted authentication result. Deliver and store `session.SessionID` through an application-controlled secure flow; this is not an anonymous issuance endpoint. HTTP authorization can also issue reusable credentials, still requiring registered callbacks and browser state verification. `BrowserClient` login results do not include the reusable credential, so existing local browser sessions do not automatically become remote-authenticated sessions.

The center's `HTTPOptions.CheckCredentialActive` runs during every introspection. Connect it to application-owned state when account disablement or center session expiry should affect subsequent requests. Revoking a center Auth session alone does not revoke independently issued SSO credentials without this hook or explicit SSO revocation.

`HTTPOptions.AuthorizationPolicy` can additionally evaluate current membership and reduce scopes after the application-state check. Explicit denial produces a remote-authentication 401; allowing access without RequiredScopes produces 403; policy failures produce 503. The policy does not delete stored credentials, so restoring permissions can allow an otherwise active credential again. See [authorization policy](AUTHORIZATION_POLICY.md).

Subsequent checks reflect:

- Individual `RevokeRemoteSession`, `RevokeSharedToken`, or `ClientApp.Revoke` operations.
- Account `RevokeLoginCredentials` or center HTTP logout that reaches credential revocation.
- `CheckCredentialActive` returning false (inactive) or an error (validation unavailable).
- Expired credentials, removed clients, and disallowed modes. Client configuration failures can produce 503 rather than user-level 401.

Existing strict/best-effort logout semantics remain. A strict logout stopped during notification has not yet revoked account credentials. Logout after authentication has completed does not undo an already-entered business request.

## Application-owned credential resolution

Keep reusable credentials in application session storage and let the browser hold only the application's session identifier:

```go
guard, err := sso.NewRemoteAuth(app, sso.RemoteAuthOptions{
    ResolveCredential: func(r *http.Request) (string, error) {
        // Application function: validate local session ownership and expiry.
        return resolveStoredRemoteSession(r.Context(), r)
    },
    RequiredScopes: []string{"orders:read"},
})
```

The resolver returns `sso.ErrRemoteCredential` for invalid or missing input and real storage errors for outages. It receives a request carrying the authentication deadline and must honor its Context. Cookie CSRF protection, session fixation prevention, and local logout remain application responsibilities. Remote authentication does not set or delete cookies.

## Scopes, timeouts, and errors

All `RequiredScopes` must match exactly and case-sensitively. `*` has no wildcard meaning. An empty list requires identity only; empty scopes and scopes containing whitespace or controls are invalid configuration. The constructor copies the scope slice.

Zero `Timeout` defaults to three seconds; negative values are invalid. It covers both credential resolution and the center request. Earlier caller deadlines and the HTTP client's timeout remain effective. Custom resolvers and transports must cooperate with Context: the guard does not launch goroutines to forcibly interrupt them. A late success after cancellation is rejected. The validation timeout is not attached to the downstream business request.

| Condition | Error | Default HTTP status |
| --- | --- | --- |
| Missing/invalid bearer, center reports inactive, exhausted lifetime | `ErrRemoteCredential` | 401 |
| Valid identity lacks any required scope | `ErrRemoteScope` | 403 |
| Network, resolver storage, center error, malformed response, wrong identity binding, timeout/cancellation | `ErrRemoteUnavailable` | 503 |
| Uninitialized guard or missing downstream handler | `ErrRemoteAuthConfig` | 500 |

A center protocol 401 can mean the application's client secret or signature is misconfigured, so it is not blindly mapped to an end-user 401. Default JSON errors contain fixed messages, not upstream bodies, credentials, or raw causes. Middleware sets `Cache-Control: no-store`, `Pragma: no-cache`, and `Referrer-Policy: no-referrer`. No failure branch calls the business handler.

`OnError` replaces the default response and supports application error observation. Classify with `errors.Is`; do not expose `err.Error()` directly. Diagnostic errors preserve upstream causes and cancellation, and may contain sensitive upstream response text. Redact logs as appropriate.

## Framework adapters and direct calls

`guard.Authenticate(r)` returns a validated `*sso.CredentialInfo` or error without writing a response or injecting context. Framework adapters can use it directly and own error handling, cache headers, and identity propagation. Gin can mount a standard handler using `gin.WrapH(guard.Middleware(handler))`. No SSO state is injected into the core authentication engine or shared HTTP capability bindings.

See [example_remote_auth_test.go](example_remote_auth_test.go). The client has no success cache; consistency of any center storage or application-hook caches remains their responsibility. Each protected request adds one center request, so account for its latency and load.

## Verification

Regression sources cover both reusable modes, center signatures and replay protection, custom parameters/endpoints, repeated validation, application state changes, individual revocation, account logout/isolation, input and response checks, scopes, cancellation, timeouts, bounded errors, and transport response limits. Formatting and static checks were performed; tests, builds, services, and examples were not run. See [testing](TESTING.md) for manual commands.
