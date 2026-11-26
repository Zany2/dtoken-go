# Gin SSO Server Example

For signing, replay protection, PKCE, multiple applications, and Redis replicas together, see the [complete integration example](../sso_deployment/README.md) and [deployment guide](../../sso/DEPLOYMENT.md).

The examples validate SSO configuration before listening: centers use `RegisterClientChecked` / `NewHTTPServerChecked`, and clients check `ClientApp.Validate()`. Invalid configuration becomes a startup error; see [configuration and startup validation](../../sso/CONFIGURATION.md).

This example shows a centralized login center built with Gin and the standard `sso.HTTPServer` protocol routes.

- `/login`: mock centralized login page.
- `/sso/authorize`: checks center login state, issues a Ticket, and redirects back to the client app.
- `/sso/token`: exchanges a Ticket for login subject information.
- `/sso/logout`: clears the center Cookie and pushes unified logout callbacks to registered client apps.

This is a local mock login center: `/login` accepts any login ID without a password, defaulting to `user-1001` when empty. It listens only on `localhost:9100`. Login submissions use browser origin checks and explicitly reject malformed URL-encoded or multipart forms before issuing a Cookie. Protocol POSTs remain available to the client application. Request signing is disabled to match `examples/sso_gin_client`; Ticket exchange still verifies the client secret and redirect URI.

The center builds an instance with `dtoken.NewBuilder().Build()` and uses it for login, identity resolution, and account logout. The signed Cookie holds an opaque Auth token checked against server-side state. Browser sessions have independent tokens, a fixed two-hour lifetime, and no automatic renewal. A new login replaces this browser's previous center session. Successful center logout revokes all center sessions for that account, so copied Cookies cannot request new Tickets; other accounts remain unaffected. Auth and SSO use separate in-memory stores, owned and closed by `run`. Restarting clears their state and changes the Cookie signing key.

`AuthorizationExtra` binds Tickets to a hash reference of the authenticated token. `EnrichExchangeResult` resolves the reference through the account's token index, validates the original session, and returns `deviceId` and `remainTokenTimeout`. It never exposes the center token or trusts a substitute browser Cookie during exchange. Missing, expired, revoked, or unbound sessions fail exchange; failed enrichment does not restore the consumed Ticket. The paired client uses the returned lifetime for its local deadline. No account-session TTL is fabricated; existing local sessions still require callbacks for immediate logout.

This Gin example sets `LogoutCallbackBestEffort: true`: it attempts the registered callbacks, then revokes the account's Auth sessions and SSO credentials before clearing Cookie and callback records, even if a client is offline. An unavailable client may keep its local sessions; a successful center response does not prove every client logged out. Use `false` when callback failures should fail center logout and preserve its state for retry. Administrative logout outside this route should also revoke SSO credentials and notify clients as appropriate.

The center also sets `CheckCredentialActive` for introspection and userinfo, reusing the same Auth session lookup as exchange without renewing the token. Retired center sessions return inactive introspection or HTTP 401 userinfo, even when their SSO credential has not expired. Lookup failures remain errors. This does not remove sessions already created in client applications.

## Run

This example caps logout deliveries at four concurrent attempts and defaults to one attempt per target. Configure `LogoutCallbackMaxAttempts` and `LogoutCallbackRetryInterval` for in-request retries. Even when best-effort center logout succeeds, `data.notifications` and `observeLogoutResult` logs retain unconfirmed client results; `code=0` does not prove logout in every application. See [single logout reliability](../../sso/LOGOUT_RELIABILITY.md).

```powershell
go run ./examples/sso_gin_server
```

Run from the repository root with Go 1.27 or later, as required by `go.work`. The standalone example module declares a minimum of Go 1.25. Deployment also requires real authentication, managed secrets and request signing on both sides, and HTTPS with Secure Cookies.

Default address:

```text
http://localhost:9100
```

Start `examples/sso_gin_client` at the same time, then open:

```text
http://localhost:9101/protected
```

## Redis Storage

The example uses in-memory storage by default so it can run locally without Redis. For production, replace `sso.NewServer()` with the Redis constructor:

```go
import ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
defer server.Close()
```

Use this creation block inside `run`, so errors return through deferred cleanup. This replacement shares only SSO protocol storage. Multiple center instances also need shared Auth storage and a managed common Cookie signing key; the mock login remains unchanged. The Redis adapter requires Redis 6.2 or later for atomic Ticket consumption.
