# SSO

SSO lets multiple business applications connect to one centralized login center. The first version provides Ticket mode: after the user signs in at the SSO center, the server generates a one-time SSO Ticket for the target application. The target application receives the Ticket on its callback URL, validates and consumes it, and then creates its own local login state.

Future SSO modes will be built on the same model, including shared token, remote session, OAuth2-based SSO, and unified logout callbacks. The client model already reserves fields such as `Modes`, `Scopes`, `AllowOrigins`, and `Extra` for these extensions.

## Use Cases

- A centralized login center distributes login state to multiple applications.
- A business application exchanges a one-time Ticket for its own local Token.
- Admin systems, user centers, open platforms, and other multi-application login scenarios.
- Systems that need centralized SSO client, callback URL, and scope management.

## Basic Flow

1. Register an SSO client with `clientId`, `clientSecret`, callback URL, and allowed modes.
2. A user visits a business application and is redirected to the SSO center when not logged in.
3. The user signs in at the SSO center.
4. The SSO center calls `GenerateSSOTicket` and redirects back to the application callback URL.
5. The application calls `ConsumeSSOTicket` to validate and consume the Ticket.
6. The application creates its own local login state from the returned `LoginID`.

## Example

```go
ctx := context.Background()

err := dtoken.RegisterSSOClient(&sso.Client{
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

ticket, err := dtoken.GenerateSSOTicket(
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

info, err := dtoken.ConsumeSSOTicket(
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

## Core API

| API | Description |
| --- | --- |
| `RegisterSSOClient` | Register an SSO client |
| `UnregisterSSOClient` | Unregister an SSO client |
| `GetSSOClient` | Query SSO client configuration |
| `GenerateSSOTicket` | Generate a one-time Ticket with the default TTL |
| `GenerateSSOTicketWithTimeout` | Generate a one-time Ticket with a custom TTL |
| `ValidateSSOTicket` | Validate a Ticket without consuming it |
| `ConsumeSSOTicket` | Validate and consume a Ticket |
| `RevokeSSOTicket` | Revoke a Ticket |
| `GetSSOTicketTTL` | Query remaining Ticket TTL |

## Notes

- A Ticket is a one-time credential and is deleted from storage after successful consumption.
- `ConsumeSSOTicket` validates the client secret, target client, callback URL, expiration state, and allowed SSO mode.
- Custom storage must implement `adapter.AtomicStorage` so Ticket consumption can read and delete atomically.
- The first version implements `sso.ModeTicket`; `ModeSharedToken`, `ModeRemoteSession`, and `ModeOAuth2` are reserved for future SSO modes.
