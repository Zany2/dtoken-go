# Browser client integration

The same browser handlers support HTTP and shared-store exchange. Set `ClientApp`'s `SharedStoreServer` for local exchange, or `UseHTTPCheck=true` to force HTTP. Both paths share state, PKCE, local-session hooks, and logout. See [deployment patterns](DEPLOYMENT_MODES.md).

`BrowserClient` composes login initiation, browser binding, callback validation, credential exchange/lookup, session deadlines, and logout handlers on top of `ClientApp`. Local sessions remain application-owned: use root `dtoken.Auth` or an existing session system without global registration. Existing low-level APIs remain unchanged.

## Composition

Logout Origin checks accept equivalent host case, IP spelling, and default ports, but reject paths (even `/`), query strings, fragments, userinfo, multiple origins, and `null`. This does not relax exact authorization callback registration. Unknown state-cookie `SameSite` values and invalid callback ports are rejected at construction.

Client protocol requests check cancellation before transport and after reading the bounded response body. A custom transport returning success after cancellation cannot establish a browser session. Already consumed login state stays consumed; begin a new login instead of replaying that callback.

Browser handlers also check cancellation before state creation, after persistence and after application login/logout hooks. Canceled state persistence does not produce a new state cookie or authorization redirect. Canceled hooks do not produce the framework's success redirect; hooks may already have completed their own effects or written cookies, which are not rolled back. Persisted state expires normally, and consumed state remains consumed.

Here `stateStore` is application-initialized `contract.Storage`; `establishSession` and `clearSession` implement the application hooks below. Configure `clientApp` with the client identity, keys, center URL, and logout callback registration. Production deployments should also configure signing and [replay protection](./REPLAY_PROTECTION.md).

```go
browser, err := sso.NewBrowserClient(clientApp, sso.BrowserClientOptions{
    Storage: stateStore,
    CallbackURL: "https://app.example/sso/callback",
    StateCookie: sso.CookieOptions{
        Name: "app_login_state",
        SecretKey: stateCookieSecret,
        MaxAge: 5 * time.Minute,
    },
    SessionMaxAge: 2 * time.Hour,
})
if err != nil {
    return err
}

mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.Header().Set("Allow", "GET")
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if err := browser.BeginLogin(w, r, "/orders", nil); err != nil {
        http.Error(w, "cannot start login", http.StatusInternalServerError)
    }
})
mux.HandleFunc("/sso/callback", browser.CallbackHandler(establishSession))
mux.HandleFunc("/logout", browser.LogoutHandler(clearSession, false, "/"))
mux.HandleFunc("/logout-all", browser.LogoutHandler(clearSession, true, "/"))
mux.HandleFunc("/sso/logout-callback", clientApp.LogoutCallbackHandler(
    func(r *http.Request, callback sso.LogoutCallback) error {
        return revokeLocalSessionsByLoginID(r.Context(), callback.LoginID)
    },
))
```

Protected pages check the application's local session first, calling `BeginLogin` only when unauthenticated. Pass an application-local destination as `returnTo` (empty means `/`), plus optional authorization `extra url.Values`. The component replaces `Params.Back` with random state and retains the business destination exclusively in server-side storage.

Gin can use `gin.WrapF(browser.CallbackHandler(...))` and `gin.WrapF(browser.LogoutHandler(...))`, or pass `c.Writer` and `c.Request` through an adapter. Frameworks do not need their own copies of state verification and exchange logic.

## Application session hooks

```go
func establishSession(w http.ResponseWriter, r *http.Request, result *sso.BrowserLoginResult) error
func clearSession(w http.ResponseWriter, r *http.Request) error
```

- Establish a session using `result.Identity.LoginID`, scopes, and metadata. Both the server-side session deadline and cookie deadline must be bounded by `result.ExpiresAt`; cookies alone do not enforce server-side expiration. Subsequent automatic renewal must not exceed that deadline.
- Create the new session before retiring this browser's previous session. On creation failure, preserve the previous session and return an error. Hooks may set cookies, but must not write response bodies or redirect.
- `clearSession` revokes the current browser's server-side session and clears its cookie, tolerating repeated calls. Center callbacks revoke local sessions by `LoginID` through the existing signed `ClientApp.LogoutCallbackHandler`.
- Successful callbacks redirect with 303 to the saved destination. A failed session hook returns 500 without a success redirect.
- Ticket and OAuth2 Code use exchange; shared tokens and remote sessions use `UserInfo`, validating active status, mode, client binding, and identity before local login. The component does not store the shared credential as a browser session.
- Local lifetime is bounded by `SessionMaxAge` and the center's positive lifetime limits, including request latency. For exchange results, `0` means unspecified and `-1` means unlimited; both still respect the local cap. Shared-token/remote-session lookup with zero remaining seconds is rejected.

For custom error pages or diagnostics, call `CompleteLogin` directly, then establish the session and redirect yourself. Built-in handlers return fixed public errors without exposing upstream bodies, credentials, or internal storage details. Do not show detailed returned errors directly to users.

## State, concurrency, and deployment

Each flow has an independent random state in its authorization URL and a separate binding secret in a signed HttpOnly cookie. Storage retains only the binding hash, destination, and deadline. Callbacks require both a matching state record and browser cookie.

Each flow uses its own cookie name, so completing one tab clears only that flow. Concurrent flows still share the application's final local-session cookie; the application decides which account wins if different accounts log in concurrently. This is not per-tab account isolation.

Parameters and browser binding are checked before consumption. `contract.AtomicStorage.GetAndDelete` allows only one callback across replicas to proceed. Ordinary storage uses an instance-local mutex around `Get` / `Delete`; share one `BrowserClient` instance, and do not assume cross-instance concurrency guarantees.

After consumption, exchange failure, lost responses, or local-session failure require starting login again. Refreshing the original callback cannot establish another session. Invalid browser binding and duplicate parameters do not consume valid state. Storage failures never bypass checks.

Storage is borrowed and never closed by the component. The default prefix derives from client ID and callback URL; explicitly isolate environments with `KeyPrefix`. Replicas must share storage, prefix, cookie signing key, and configuration. Records are JSON strings compatible with existing Redis storage. Memory storage loses pending flows on restart. Both record deadlines and storage TTL enforce expiration without background cleanup. The host application owns login-entry rate limiting.

Cookies default to host-only, callback path, SameSite=Lax, and HttpOnly=true; HTTPS callbacks automatically enable Secure. Cookie paths must cover the callback. Domain cookies and SameSite=Strict are rejected to preserve cross-site return behavior. Use distinct cookie name prefixes for applications on the same host and configure a separate stable `StateCookie.SecretKey`.

## Navigation and logout

Destinations must be application-local paths such as `/orders?id=1`. Absolute URLs, `//host`, backslashes, and control characters, including encoded equivalents, are rejected. Extra callback parameters cannot replace the saved destination.

`LogoutHandler` requires POST with exactly one `Origin` matching the configured callback origin. Missing or mismatched origins return 403. Invoke it through same-origin browser forms or requests, and configure the external callback URL behind reverse proxies. Replace GET logout links with POST buttons. Direct API callers need application-specific authentication and CSRF composition.

- Local logout invokes the application revocation hook, then redirects with 303 to the supplied local `returnTo`.
- Global logout builds the center URL, revokes the local session, then redirects to the center. The center resolves its own trusted subject through its login cookie and reports the final outcome. Automatic navigation back to the application after center logout is not part of the current protocol.
- Local success does not prove logout across every application. Center failure does not restore the local session. Logout does not cancel previously pending authorization flows in other tabs.

## Example and verification

The standard-library client includes an optional full integration, preserving its original default entry point. Run manually from the workspace root:

```powershell
go run ./examples/sso_client -browser-flow
```

Use it with the existing center example and open `/protected`; the page provides local/global POST logout buttons. It reuses the example's in-memory sessions and development settings. Production applications need their own persistent sessions and signing configuration. See [browser.go](../examples/sso_client/browser.go).

Regression sources cover all modes, signing/replay composition, custom parameters, parallel tabs, browser binding, repeated callbacks, atomic and ordinary storage concurrency, storage faults, session deadlines, and logout origin checks. See the [testing guide](./TESTING.md).

## Automatic PKCE integration

Set `BrowserClientOptions.PKCE: true` for Ticket or OAuth2 code mode. Each flow gets a private server-stored verifier; authorization sends an S256 challenge, and completion retrieves the proof from bound flow state and submits it through POST. The center can require this with `Client.RequirePKCE: true`. Both options default to false. Shared Token and Remote Session modes do not support this option. Pending flows must restart when the option changes. See the [PKCE guide](PKCE.md).
