# SSO

SSO lets multiple business applications connect to one centralized login center. SSO lives in the optional module `github.com/Zany2/dtoken-go/sso` and is not coupled to the base authentication and authorization architecture by default; the default constructor includes JSON codec and in-memory storage, while options can replace them with Redis, database-backed, or custom components. It provides primitives for Ticket, shared token, remote session, and OAuth2 authorization-code style modes. After the user signs in at the SSO center, the server can issue the credential configured for the target application, and the application can validate, consume, or remotely check it before creating its own local login state.

The module provides protocol endpoints, parameter names, signing helpers, server options, client URL builders, and basic `net/http` server routes. Gin, Fiber, Echo, and other frameworks can mount the standard Handler directly, while more framework-specific wrappers can be added later.

## Module Boundary

- `Server`: the login-center core for registering apps and issuing, validating, or consuming Ticket, shared token, remote session, and authorization-code credentials.
- `Client`: a business application registered with the login center, such as an admin system, open platform, or order service.
- `ClientApp`: a client-side helper for building authorization, ticket-check, and signout URLs.
- `HTTPServer`: standard-library `net/http` routes for authorization redirect, Ticket exchange, and logout.
- `CookieOptions`: same-site shared-cookie helper configuration for lightweight SSO under one parent domain.
- `Endpoints` / `ParamNames`: centralized SSO HTTP endpoint and parameter names.
- `Signer`: HMAC-SHA256 request parameter signing helper.

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

In production, replace the default storage or codec with your own components:

```go
server := sso.NewServer(
	sso.WithStorage(redisStorage),
	sso.WithCodec(codec),
	sso.WithConfig(sso.DefaultConfig()),
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
```

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
| `GET/POST /sso/token` | Exchanges a Ticket for login subject information |
| `GET/POST /sso/logout` | Clears shared cookie state and returns logout result |

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

## Notes

- A Ticket is a one-time credential and is deleted from storage after successful consumption.
- `ConsumeTicket` validates the client secret, target client, callback URL, expiration state, and allowed SSO mode.
- Custom storage must implement `adapter.AtomicStorage` so Ticket and SSO OAuth2 code consumption can read and delete atomically.
- `ModeSharedToken` is for trusted internal systems that reuse a short-lived credential and is client-scoped by default.
- `ModeRemoteSession` is for applications that remotely check login state at the SSO center.
- `ModeOAuth2` is an SSO authorization-code primitive, not the full OAuth2 Token Server.
- `Signer` ignores the `sign` field itself and signs sorted parameter names and values, making it suitable for tamper protection between Server and Client.
- Current HTTP routes focus on the `ModeTicket` redirect-and-exchange flow. Full HTTP wrappers for shared token, remote session, and OAuth2 can be extended later.
