# SSO Server Example

For signing, replay protection, PKCE, multiple applications, and Redis replicas together, see the [complete integration example](../sso_deployment/README.md) and [deployment guide](../../sso/DEPLOYMENT.md).

The examples validate SSO configuration before listening: centers use `RegisterClientChecked` / `NewHTTPServerChecked`, and clients check `ClientApp.Validate()`. Invalid configuration becomes a startup error; see [configuration and startup validation](../../sso/CONFIGURATION.md).

This example demonstrates a minimal centralized login center:

- `/login`: mock SSO login page.
- `/sso/authorize`: issues a Ticket and redirects back to the client app.
- `/sso/token`: exchanges a Ticket for login subject information.
- `/sso/logout`: clears the SSO-center cookie and pushes client logout callbacks.

The client app sends a `callback` parameter during login redirect. After successful authorization, the SSO Server records that callback URL. When the user logs out from the login center, the Server notifies registered client apps so they can clear local login state.

This is a local flow demonstration: `/login` accepts an arbitrary login ID without checking a password, and an empty ID defaults to `user-1001`. The server listens only on `localhost:9000`. Login submissions read the POST form body and use Go's cross-origin request protection. The protocol routes remain available for client-to-server requests; request signing is disabled in this paired demo, while Ticket exchange still verifies the registered client secret and redirect URI.

The center builds an instance with `dtoken.NewBuilder().Build()` and uses it for login, identity resolution, and account logout. The signed Cookie carries an opaque Auth token; every authorization checks its server-side state. Browser sessions have independent tokens, a fixed two-hour lifetime, and no automatic renewal. A new login replaces this browser's previous center session. Successful center logout revokes all center sessions for that account, so copied Cookies cannot request new Tickets. Other accounts are unaffected. Auth and SSO use separate in-memory stores, owned and closed by `run`; restarting clears their state and changes the Cookie signing key.

`AuthorizationExtra` binds each Ticket to a hash reference of the authenticated token. `EnrichExchangeResult` resolves that reference through the account's token index, validates the original session, and returns `deviceId` and `remainTokenTimeout`. It does not return the center token, accept a substitute browser Cookie at exchange, or invent an account-session TTL. Missing, expired, revoked, or unbound sessions cannot complete exchange; a failed enrichment still consumes the one-time Ticket. The paired client uses the returned lifetime to bound its local session. Existing client sessions still need logout callbacks for immediate removal.

Logout uses strict callback handling: if a registered client is offline or rejects the callback, center logout returns an error and retains its Auth sessions, Cookie, and callback records for retry. Success revokes account Auth sessions and SSO credentials before clearing Cookie and callback records. Keep the client running when trying `/sso/logout`. Administrative logout outside this route should also revoke SSO credentials and notify clients as appropriate.

The center also sets `CheckCredentialActive` for introspection and userinfo, reusing the same Auth session lookup as exchange without renewing the token. Retired center sessions return inactive introspection or HTTP 401 userinfo, even when their SSO credential has not expired. Lookup failures remain errors. This does not remove sessions already created in client applications.

## Run

This example caps logout deliveries at four concurrent attempts and defaults to one attempt per target. `observeLogoutResult` logs safe summaries of stopped stages and failed targets. Adjust `LogoutCallbackMaxAttempts` and `LogoutCallbackRetryInterval` to enable in-request retries. Logout responses add `data.stage` and `data.notifications`, retaining `data.result="ok"` on success. See [single logout reliability](../../sso/LOGOUT_RELIABILITY.md).

```powershell
go run ./examples/sso_server
```

Run this command from the repository root with Go 1.27 or later, as required by `go.work`. The standalone example module declares a minimum of Go 1.25. For deployment, replace the mock login with actual authentication, configure shared storage and managed secrets, and use HTTPS with Secure Cookies.

Default address:

```text
http://localhost:9000
```

Start `examples/sso_client` at the same time, then open:

```text
http://localhost:9001/protected
```
