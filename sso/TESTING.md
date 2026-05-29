# SSO Testing

This document describes recommended verification flows for the standalone SSO module, including in-memory mode, Redis mode, Server/Client integration, and single logout.

## Unit Tests

The root SSO module uses in-memory storage by default:

```powershell
go test ./sso -v
```

Main coverage:

- Ticket issue, validate, consume, revoke, and expiration.
- Shared Token issue, validate, revoke, and expiration.
- Remote Session create, validate, renew, revoke, and expiration.
- OAuth2 Code issue, consume, revoke, and boundary errors.
- HTTP protocol endpoints: `authorize`, `token`, `introspect`, `userinfo`, `revoke`, `logout`.
- ClientApp: authorization URL, ticket exchange, signature verification, single logout callback Handler.
- ClientSession: register client sessions, update, query, and clear.

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

1. The client app is not logged in and redirects to `http://localhost:9100/login`.
2. The login center writes the center Cookie and redirects back to `/sso/authorize`.
3. The login center issues a Ticket and redirects to the client `/sso/callback`.
4. The client app calls `/sso/token` and receives `loginId`.
5. The client app creates local session state and `/protected` returns the login subject.

## Single Logout Verification

After login, open:

```text
http://localhost:9100/sso/logout?loginId=user-1001
```

Expected result:

- SSO Server clears the center Cookie.
- SSO Server pushes `/sso/logout-callback` to registered clients.
- Client deletes local sessions for the received `loginId`.
- Opening `http://localhost:9101/protected` again redirects to the login center.

If signing is enabled, set the same `SecretKey` on both Server and Client and set `CheckSign` to `true`. Client-side `LogoutCallbackHandler` verifies callback signatures automatically.

`LogoutCallbackHandler` also validates callback timestamps. By default, only callbacks within 5 minutes are accepted, which helps prevent replayed old requests.

## Redis Mode Verification

Production deployments should use Redis storage:

```go
import ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithKeyPrefix("dtoken:"),
	sso.WithAuthType("sso:"),
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
```

Recommended Redis key types to observe:

- `sso:client:`: registered client app data.
- `sso:ticket:`: one-time Ticket, deleted after consume.
- `sso:oauth2:code:`: OAuth2 Code, deleted after consume.
- `sso:client-session:`: client session records used by single logout, cleared after center logout succeeds.

Verification checklist:

1. Before login, confirm the client registration key exists.
2. During login, observe the Ticket key appear briefly.
3. After client ticket exchange succeeds, the Ticket key should be deleted.
4. After `/sso/logout`, the matching `sso:client-session:` key should be deleted.

Optional integration test:

```powershell
$env:DTOKEN_SSO_REDIS="redis://:password@127.0.0.1:6379/0"
go test ./sso/storage/redis/... -v
```

When `DTOKEN_SSO_REDIS` is not set, this test is skipped automatically.

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
