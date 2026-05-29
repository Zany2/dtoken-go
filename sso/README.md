# SSO

SSO lets multiple business applications connect to one centralized login center. SSO lives in the optional module `github.com/Zany2/dtoken-go/sso` and is not coupled to the base authentication and authorization architecture by default; the default constructor includes JSON codec and in-memory storage for local verification and unit tests, while production deployments should use `github.com/Zany2/dtoken-go/sso/storage/redis` or custom persistent storage. It provides primitives for Ticket, shared token, remote session, and OAuth2 authorization-code style modes. After the user signs in at the SSO center, the server can issue the credential configured for the target application, and the application can validate, consume, or remotely check it before creating its own local login state.

The module provides protocol endpoints, parameter names, signing helpers, server options, client URL builders, and basic `net/http` server routes. Gin, Fiber, Echo, and other frameworks can mount the standard Handler directly, while more framework-specific wrappers can be added later.

## Module Boundary

- `Server`: the login-center core for registering apps and issuing, validating, or consuming Ticket, shared token, remote session, and authorization-code credentials.
- `Client`: a business application registered with the login center, such as an admin system, open platform, or order service.
- `ClientApp`: a client-side helper for building authorization, ticket-check, and signout URLs.
- `HTTPServer`: standard-library `net/http` routes for authorization redirect, Ticket exchange, and logout.
- `CookieOptions`: same-site shared-cookie helper configuration for lightweight SSO under one parent domain.
- `Endpoints` / `ParamNames`: centralized SSO HTTP endpoint and parameter names.
- `Signer`: HMAC-SHA256 request parameter signing helper.

## Subpackages

| Subpackage | Description |
| --- | --- |
| `sso` | Core protocol, credential models, server APIs, and backward-compatible entry points |
| `sso/httpserver` | `net/http` integration facade for modular imports |
| `sso/codec/json` | Built-in JSON codec |
| `sso/storage/memory` | Built-in in-memory storage for local verification and tests only |
| `sso/storage/redis` | Production-oriented Redis storage |

The root package keeps `sso.NewServer()`, `sso.NewHTTPServer()`, `sso.JSONCodec`, `sso.MemoryStorage`, and other existing entry points, so current code does not need to change.

## Use Cases

- A centralized login center distributes login state to multiple applications.
- A business application exchanges a one-time Ticket for its own local Token.
- Trusted systems reuse a short-lived shared token.
- Applications check a centralized remote session at the SSO center.
- Standardized callback integrations use an OAuth2 authorization-code style flow.
- Admin systems, user centers, open platforms, and other multi-application login scenarios.
- Systems that need centralized SSO client, callback URL, and scope management.

## Basic Flow

1. Register an SSO client with `clientId`, `clientSecret`, callback URL, and allowed modes.
2. A user visits a business application and is redirected to the SSO center when not logged in.
3. The user signs in at the SSO center.
4. The SSO center calls `GenerateTicket` and redirects back to the application callback URL.
5. The application calls `ConsumeTicket` to validate and consume the Ticket.
6. The application creates its own local login state from the returned `LoginID`.

## Design Direction

| Capability | Role | Description |
| --- | --- | --- |
| `ModeTicket` | Recommended default | The app exchanges a one-time Ticket for user identity |
| `ModeSharedToken` | Trusted internal systems | Trusted systems reuse a short-lived shared token |
| `ModeRemoteSession` | Centralized session | The app checks session state at the login center |
| `ModeOAuth2` | Authorization-code extension | SSO authorization-code primitive, not the full OAuth2 Token Server |

## Example

```go
ctx := context.Background()

// NewServer includes JSON codec and in-memory storage by default.
server := sso.NewServer()

err := server.RegisterClient(&sso.Client{
	ClientID:     "app-a",
	ClientSecret: "secret-a",
	Name:         "App A",
	RedirectURIs: []string{
		"https://app.example.com/sso/callback",
	},
	Modes:  []sso.Mode{sso.ModeTicket},
	Scopes: []string{"profile", "email"},
})
if err != nil {
	return err
}

ticket, err := server.GenerateTicket(
	ctx,
	"app-a",
	"user-1001",
	"https://app.example.com/sso/callback",
	[]string{"profile"},
	map[string]any{"scene": "web"},
)
if err != nil {
	return err
}

info, err := server.ConsumeTicket(
	ctx,
	ticket.Ticket,
	"app-a",
	"secret-a",
	"https://app.example.com/sso/callback",
)
if err != nil {
	return err
}

fmt.Println(info.LoginID)
```

For production, use the Redis default component:

```go
import ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
```

You can also inject an existing storage explicitly:

```go
server := sso.NewServer(
	sso.WithStorage(storage),
)
```

## Client Helper

```go
app := sso.NewClientApp(sso.ClientConfig{
	ClientID:  "app-a",
	ServerURL: "https://sso.example.com",
	SecretKey: "sign-secret",
	CheckSign: true,
	Endpoints: sso.DefaultEndpoints(),
	Params:    sso.DefaultParamNames(),
})

callbackURL := "https://app.example.com/sso/callback"
authURL, err := app.AuthURL(callbackURL, nil)
if err != nil {
	return err
}

fmt.Println(authURL)

exchangeURL, err := app.ExchangeTicketURLWithRedirect("ticket-value", callbackURL, nil)
if err != nil {
	return err
}

fmt.Println(exchangeURL)

result, err := app.ExchangeTicket(ctx, "ticket-value", callbackURL)
if err != nil {
	return err
}

fmt.Println(result.LoginID)
```

If the application needs unified logout notifications, enable callback registration:

```go
app := sso.NewClientApp(sso.ClientConfig{
	ClientID:           "app-a",
	ClientSecret:       "secret-a",
	ServerURL:          "https://sso.example.com",
	SecretKey:          "sign-secret",
	CheckSign:          true,
	RegisterCallback:  true,
	LogoutCallbackURL: "https://app.example.com/sso/logout-callback",
})
```

`ClientApp` will include the `callback` parameter when it builds the authorization URL. After successful authorization, the SSO Server records the login subject and client callback URL. When the center handles `/sso/logout`, it pushes logout callbacks to the registered client applications.

Client applications can use the built-in Handler to process callbacks. It verifies POST, form values, and signatures:

```go
mux.HandleFunc("/sso/logout-callback", app.LogoutCallbackHandler(func(r *http.Request, callback sso.LogoutCallback) error {
	// Delete local Session, Cookie, or Token here.
	deleteLocalSessionsByLoginID(callback.LoginID)
	return nil
}))
```

`LogoutCallbackMaxAge` defaults to 5 minutes and rejects stale or clearly future logout callback timestamps to reduce replay risk.

`ClientApp` can also call SSO Server HTTP APIs directly:

| API | Description |
| --- | --- |
| `ExchangeTicket` | Exchange a Ticket for login subject information |
| `ExchangeOAuth2Code` | Exchange an OAuth2 Code for login subject information |
| `Introspect` | Check whether a Ticket, shared token, remote session, or OAuth2 Code is active |
| `UserInfo` | Read login subject information from a valid credential |
| `Revoke` | Revoke a Ticket, shared token, remote session, or OAuth2 Code |

## HTTP Redirect Integration

`HTTPServer` exposes standard-library Handlers. You can mount them on `http.ServeMux` directly or bridge them into Gin, Echo, Fiber, and similar frameworks.

```go
server := sso.NewServer()
server.RegisterClient(&sso.Client{
	ClientID:     "app-a",
	ClientSecret: "secret-a",
	RedirectURIs: []string{
		"https://app.example.com/sso/callback",
	},
	Modes: []sso.Mode{sso.ModeTicket},
})

httpSSO := sso.NewHTTPServer(server, sso.HTTPOptions{
	ServerOptions: sso.ServerOptions{
		EnableSLO: true,
		LogoutCallbackTimeout:    3 * time.Second,
		LogoutCallbackBestEffort: false,
		CheckSign: false,
		Endpoints: sso.DefaultEndpoints(),
		Params:    sso.DefaultParamNames(),
	},
	LoginIDResolver: func(r *http.Request) (string, bool) {
		// Connect your login state here, such as a center cookie or existing auth module.
		return "user-1001", true
	},
})

mux := http.NewServeMux()
httpSSO.Register(mux)
```

Default routes:

| Route | Description |
| --- | --- |
| `GET /sso/authorize` | Checks center login state, issues a Ticket, and redirects back to the client app |
| `GET/POST /sso/token` | Exchanges a Ticket or OAuth2 Code for login subject information |
| `GET/POST /sso/introspect` | Checks whether a Ticket, shared token, remote session, or OAuth2 Code is active |
| `GET/POST /sso/userinfo` | Reads login subject, client, and scope information from a valid credential |
| `GET/POST /sso/revoke` | Revokes a Ticket, shared token, remote session, or OAuth2 Code |
| `GET/POST /sso/logout` | Clears shared cookie state, pushes registered client logout callbacks, and returns logout result |

## Single Logout

`EnableSLO` is enabled by default. A client application can send its logout callback URL during the login redirect, and the login center will record a client session binding:

```text
/sso/authorize?client=app-a&redirect=https://app.example.com/sso/callback&callback=https://app.example.com/sso/logout-callback
```

When the user logs out from the login center, `HTTPServer` sends concurrent `POST` callbacks to the registered applications for that login subject. The form body contains:

| Parameter | Description |
| --- | --- |
| `loginId` | Login subject ID |
| `client` | Client application ID |
| `timestamp` | Callback timestamp |
| `sign` | Signature when signing is enabled |

After receiving the callback, the client application should delete its local Session, Cookie, or Token. After all callbacks succeed, the login center clears the stored client session records for that login subject.

`LogoutCallbackBestEffort` controls failure behavior. When it is `false`, any failed client callback makes the center logout return an error. When it is `true`, the server still attempts all callbacks and clears center-side client session records. `LogoutHTTPClient` can inject a custom HTTP client for proxy, TLS, or stricter timeout control.

To reduce SSRF risk from malicious `callback` values, the login center only records logout callback URLs that belong to the current client: exact `RedirectURIs` matches, same-origin URLs as a registered redirect URI, or origins explicitly configured in `AllowOrigins`.

## Redis Storage

Production deployments should use Redis storage for the SSO Server. One-time credentials such as Ticket and OAuth2 Code require atomic get-and-delete behavior, which the Redis component provides. ClientSession records used by single logout can also be shared across multiple SSO Server instances.

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

For Redis verification, focus on four flows: Ticket issue and consume, OAuth2 Code issue and consume, `RegisterClientSession` storing client sessions, and `/sso/logout` pushing callbacks before clearing client sessions.

You can also run the optional Redis integration test with an environment variable:

```powershell
$env:DTOKEN_SSO_REDIS="redis://:password@127.0.0.1:6379/0"
go test ./sso/storage/redis/... -v
```

## Shared Cookie

For applications under the same parent domain, shared cookies can be used as the SSO-center session source. This fits deployments such as `sso.example.com`, `app-a.example.com`, and `app-b.example.com`.

```go
cookie := sso.CookieOptions{
	Name:     "dtoken_sso",
	Domain:   ".example.com",
	Path:     "/",
	MaxAge:   2 * time.Hour,
	HTTPOnly: true,
	Secure:   true,
	SameSite: http.SameSiteLaxMode,
}

// Write the shared cookie after the user signs in at the login center.
sso.SetLoginIDCookie(w, cookie, "user-1001")

// HTTPServer can resolve the current user from the shared cookie.
httpSSO := sso.NewHTTPServer(server, sso.HTTPOptions{
	Cookie:          cookie,
	LoginIDResolver: sso.LoginIDFromCookie(cookie),
})
```

## Signing

```go
values := url.Values{}
values.Set("client", "app-a")
values.Set("ticket", "ticket-value")

signer := sso.NewSigner("sign-secret")
signedValues := signer.AttachSign(values)

if !signer.Verify(signedValues) {
	return errors.New("invalid sign")
}
```

## Core API

| API | Description |
| --- | --- |
| `RegisterClient` | Register an SSO client |
| `UnregisterClient` | Unregister an SSO client |
| `GetClient` | Query SSO client configuration |
| `GenerateTicket` | Generate a one-time Ticket with the default TTL |
| `GenerateTicketWithTimeout` | Generate a one-time Ticket with a custom TTL |
| `ValidateTicket` | Validate a Ticket without consuming it |
| `ConsumeTicket` | Validate and consume a Ticket |
| `RevokeTicket` | Revoke a Ticket |
| `GetTicketTTL` | Query remaining Ticket TTL |
| `GenerateSharedToken` / `ValidateSharedToken` | Generate and validate a reusable SSO shared token |
| `RevokeSharedToken` / `GetSharedTokenTTL` | Revoke a shared token and query its remaining TTL |
| `CreateRemoteSession` / `ValidateRemoteSession` | Create and validate a centralized remote session |
| `RenewRemoteSession` / `RevokeRemoteSession` | Renew or revoke a remote session |
| `GenerateOAuth2Code` / `ConsumeOAuth2Code` | Generate and consume an SSO OAuth2 authorization code |
| `RevokeOAuth2Code` / `GetOAuth2CodeTTL` | Revoke an authorization code and query its remaining TTL |
| `RegisterClientSession` | Record a client login binding for single logout callbacks |
| `GetClientSessions` | Query registered client sessions for a login subject |
| `ClearClientSessions` | Clear registered client sessions for a login subject |

## Notes

- A Ticket is a one-time credential and is deleted from storage after successful consumption.
- `ConsumeTicket` validates the client secret, target client, callback URL, expiration state, and allowed SSO mode.
- The built-in `MemoryStorage` used by `sso.NewServer()` is intended for local debugging and tests only. Data is lost after process restart and it is not suitable for multi-instance deployments.
- Production deployments should use `sso/storage/redis`. Redis storage implements atomic get-and-delete, which is required by one-time Ticket and OAuth2 Code consumption.
- Production deployments should enable `CheckSign` and configure `SecretKey` so Server and Client traffic, including logout callbacks, is protected against tampering.
- Logout callback URLs are checked against client registration data. Avoid adding overly broad origins to `AllowOrigins`.
- Custom storage must implement `adapter.AtomicStorage` so Ticket and SSO OAuth2 code consumption can read and delete atomically.
- `ModeSharedToken` is for trusted internal systems that reuse a short-lived credential and is client-scoped by default.
- `ModeRemoteSession` is for applications that remotely check login state at the SSO center.
- `ModeOAuth2` is an SSO authorization-code primitive, not the full OAuth2 Token Server.
- `Signer` ignores the `sign` field itself and signs sorted parameter names and values, making it suitable for tamper protection between Server and Client.
- Current HTTP routes cover `ModeTicket` redirect exchange, OAuth2 Code exchange, credential introspection, credential revocation, userinfo, and unified logout callback pushing.

## Testing And Examples

- [SSO testing guide](./TESTING.md)
- [Gin SSO Server example](../examples/sso_gin_server/README.md)
- [Gin SSO Client example](../examples/sso_gin_client/README.md)
- [net/http SSO Server example](../examples/sso_server/README.md)
- [net/http SSO Client example](../examples/sso_client/README.md)
