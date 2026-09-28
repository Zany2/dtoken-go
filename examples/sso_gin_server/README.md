# Gin SSO Server Example

This example shows a centralized login center built with Gin and the standard `sso.HTTPServer` protocol routes.

- `/login`: mock centralized login page.
- `/sso/authorize`: checks center login state, issues a Ticket, and redirects back to the client app.
- `/sso/token`: exchanges a Ticket for login subject information.
- `/sso/logout`: clears the center Cookie and pushes unified logout callbacks to registered client apps.

This is a local mock login center: `/login` accepts any login ID without a password, defaulting to `user-1001` when empty. It listens only on `localhost:9100`. Login submissions use browser origin checks and explicitly reject malformed URL-encoded or multipart forms before issuing a Cookie. Protocol POSTs remain available to the client application. Request signing is disabled to match `examples/sso_gin_client`; Ticket exchange still verifies the client secret and redirect URI.

The center uses in-memory storage and a new random Cookie signing key per process. Restarting clears session records and invalidates old center Cookies. Cookies expire after two hours. Clearing the browser Cookie does not revoke a previously copied Cookie before its signed expiry; immediate revocation requires a server-side session resolver and corresponding session deletion.

This Gin example sets `LogoutCallbackBestEffort: true`: it attempts the registered callbacks, then clears center Cookie and callback records even if a client is offline. An unavailable client may keep its local sessions; a successful center response does not prove every client logged out. Use `false` when callback failures should fail center logout and preserve its state for retry.

## Run

```powershell
go run ./examples/sso_gin_server
```

Run from the repository root with Go 1.25 or later. Deployment also requires real authentication, managed secrets and request signing on both sides, and HTTPS with Secure Cookies.

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

Use this creation block inside `run`, so errors return through deferred cleanup. Redis provides shared protocol storage; replacing storage alone does not replace mock authentication or the process-local Cookie key. The Redis adapter requires Redis 6.2 or later for atomic Ticket consumption.
