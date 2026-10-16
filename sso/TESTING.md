# SSO Testing

This document describes recommended verification flows for the standalone SSO module, including in-memory mode, Redis mode, Server/Client integration, and single logout.

## Unit Tests

Run from the repository root. The SSO module uses in-memory storage by default; include its subpackages:

```powershell
go test ./sso/... -v
```

Redis and the four examples are separate workspace modules, so the command above does not include them:

```powershell
go test ./sso/storage/redis/... -v
go test ./examples/sso_server/... ./examples/sso_client/... -v
go test ./examples/sso_gin_server/... ./examples/sso_gin_client/... -v
```

Redis constructor tests run without a service; the Redis integration flow is skipped unless `DTOKEN_REDIS_URL` is set. These commands do not launch the demo applications.

Main coverage:

- Ticket issue, validate, consume, revoke, and expiration.
- Shared Token issue, validate, revoke, and expiration.
- Remote Session create, validate, renew, revoke, and expiration.
- OAuth2 Code issue, consume, revoke, and boundary errors.
- HTTP protocol endpoints: `authorize`, `token`, `introspect`, `userinfo`, `revoke`, `logout`.
- ClientApp: authorization URL, ticket exchange, signature verification, single logout callback Handler.
- ClientSession: register client sessions, update, query, and clear.
- Signed ClientApp/HTTPServer flows across all four modes, with custom parameter names, route prefixes, and both atomic and ordinary storage capabilities.
- Rejected client/redirect bindings preserve credentials; consumed or revoked credentials cannot expose user info.
- Authorization redirects and JSON responses prohibit caching; malformed logout forms return HTTP 400 without invoking the local logout action.
- Both client examples check browser state, server-side session expiry, and session replacement after successful exchange.

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

Keys concatenate `keyPrefix + authType + key suffix + identifier`. The suffix constants already begin with `sso:`, so this configuration produces two adjacent `sso:` segments.

Verification checklist:

1. Before login, confirm the client registration key exists.
2. During login, observe the Ticket key appear briefly.
3. After client ticket exchange succeeds, the Ticket key should be deleted.
4. After `/sso/logout` completes under the configured policy, the matching `dtoken:sso:sso:client-session:` key should be deleted.

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
