# SSO

Inbound query and form limits cover all six center endpoints, logout callbacks and browser callbacks. See [HTTP request limits](REQUEST_LIMITS.md) for configuration, 413 responses and middleware compatibility.

The [complete integration example](../examples/sso_deployment/README.md) combines a center and multiple applications with signing, replay protection, PKCE, shared Auth sessions, and logout. See [deployment and operations](DEPLOYMENT.md) for Redis, HTTPS proxies, replicas, and failure handling.

For application startup, use `NewHTTPServerChecked`, `NewClientAppChecked`, and `RegisterClientChecked`. See [configuration and startup validation](CONFIGURATION.md) for defaults, compatibility, and error handling.

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
defer server.Close() // The URL/config constructors create the Redis client and transfer ownership to Server.
```

You can also inject an existing storage explicitly:

```go
server := sso.NewServer(
	sso.WithStorage(storage),
)
```

`server.Close()` does not close injected storage by default; close that storage from the owner that created it.

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

// Confidential clients exchange identity over POST so the client secret never enters a URL.
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

`ClientApp` includes the `callback` parameter in the authorization URL and in `ExchangeTicket` / `ExchangeOAuth2Code` requests. With `EnableSLO` enabled, the SSO Server records the login subject and client callback URL after successful authorization or credential consumption. This also supports credentials issued directly through the Server API. Repeated registration updates the existing subject/client binding. When the center handles `/sso/logout`, it pushes logout callbacks to the registered client applications.

Exchange checks callback ownership and `MaxRegisteredClient` before consuming the credential. Invalid callbacks, invalid client credentials, mismatched redirects, and a full registration limit leave the credential available. Registration uses the identity from the consumed credential, never a caller-supplied `loginId`. A storage failure after consumption can still fail registration; obtain a new credential instead of retrying the consumed one. Direct calls to `Server.ConsumeTicket` / `Server.ConsumeOAuth2Code` remain credential primitives and do not register callbacks.

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

`ServerURL` must be an absolute HTTP(S) URL without user credentials, query parameters, or a fragment. A gateway path prefix is supported and retained; endpoints must be paths without query parameters or fragments.

HTTP calls default to a 10-second total timeout and do not follow redirects, including 307/308 redirects that could forward credential forms. An injected `HTTPClient` retains its timeout (including zero) and any explicit `CheckRedirect` policy; a nil policy still rejects redirects. The helper does not modify the injected client. Request context cancellation and earlier deadlines remain effective.

`ClientConfig.MaxResponseBytes` bounds each protocol response body, including non-2xx responses and revocation acknowledgements. Zero or negative values use `DefaultClientMaxResponseBytes` (1 MiB); set a positive byte count for applications returning larger metadata, for example `MaxResponseBytes: 4 << 20`. The limit applies to actual bytes read from `Response.Body` (after decompression when the transport performs it), regardless of `Content-Length`. Responses exactly at the limit are accepted if otherwise valid; detecting overflow reads at most one additional byte. Oversized or unreadable bodies fail without returning partial results, and response bodies are closed on every completed request path. This does not change the server's separate 64 KiB logout-callback acknowledgement limit.

Responses must contain a numeric `code`; identity and introspection calls also require non-null `data`. Exchange and UserInfo responses require a nonempty `loginId`. Numbers in `Extra` are decoded as `json.Number` without losing integer precision. `UserInfo` includes the credential's `active`, `mode`, and `expiresIn` fields.

Both Ticket and OAuth2 Code exchange also reject `remainTokenTimeout` or `remainSessionTimeout` below -1 at the client helper boundary. Zero remains unspecified, -1 means unlimited, and positive values retain their exact integer seconds. These checks validate the response; applications still apply their own local session lifetime policy. A rejected exchange response does not restore a one-time credential already consumed by the center.

### Enriching Exchange Results

Standard HTTP authorization can bind trusted center-session metadata to credentials in all four modes through `AuthorizationExtra`:

```go
options.AuthorizationExtra = func(r *http.Request, clientID, loginID string) (map[string]any, error) {
	// Application function: resolve the authenticated browser session and verify its subject.
	session, err := resolveAuthenticatedCenterSession(r, loginID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"centerSessionID": session.ID}, nil
}
```

`resolveAuthenticatedCenterSession` is application code using the same login source as `LoginIDResolver`. The hook runs after client, redirect, mode, and scope checks; errors stop credential issuance and callback registration. Do not copy an unverified browser-supplied session ID. `Extra` is exposed to authenticated clients through exchange or userinfo responses; include only metadata that the client may read, not usable center login credentials such as the center cookie.

`HTTPOptions.EnrichExchangeResult` can load application-owned center-session information after a Ticket or OAuth2 Code has been authenticated and consumed. The hook receives the authenticated client ID and the credential's result, including its stored `Extra` data. It runs before exchange-time logout callback registration and does not require SLO to be enabled.

```go
options.EnrichExchangeResult = func(r *http.Request, clientID string, result *sso.TicketExchangeResult) error {
	// Application function: load and validate the session bound to this issued credential.
	session, err := loadCenterSession(r.Context(), clientID, result.LoginID, result.Extra["centerSessionID"])
	if err != nil {
		return err
	}
	result.DeviceID = session.DeviceID
	result.RemainTokenTimeout = session.RemainTokenTimeout
	result.RemainSessionTimeout = session.RemainSessionTimeout
	if result.Extra == nil {
		result.Extra = make(map[string]any)
	}
	result.Extra["displayName"] = session.DisplayName
	return nil
}
```

`loadCenterSession` is application code. If sessions are device-specific, bind their reference in the `Extra` supplied when issuing the credential and validate its ownership in the hook. An exchange is a server-to-server request: it normally has no center browser cookie. Do not choose a session using unverified request `loginId`, `deviceId`, or timeout parameters. The hook may also explicitly set `TokenValue` when the client is intended to receive a center token. It must preserve `LoginID`, `CenterID`, and `Scopes`; changing them fails the exchange.

Timeout fields describe the center session, not the short-lived Ticket or authorization code: positive values are remaining seconds, zero is unspecified, and `-1` means no expiration. Values below `-1` are rejected. Return an error for an expired or revoked session; zero must not be used to signal expiry. Client applications should bound their local session lifetime by applicable positive center timeouts and their own policy; the SSO helper does not create or renew local sessions automatically.

The standard-library and Gin client examples apply the smallest positive center timeout with a local two-hour cap, starting the deadline before exchange to include network time conservatively. Invalid or exhausted lifetimes cannot replace an existing local session. Both demo centers use an application-owned Auth instance: authorization binds a hash reference of its token, exchange validates the original session and returns its token TTL and device ID, and center logout revokes the account's Auth sessions. The center token is never returned to clients. This login-time snapshot does not replace logout callbacks.

A hook error or invalid result fails the exchange without returning partial data or adding an exchange-time callback binding. The credential remains consumed and cannot be replayed; obtain a new credential to retry. Any binding created earlier by browser authorization is retained. With no hook, the existing result is unchanged. Direct `Server.ConsumeTicket`, `Server.ConsumeOAuth2Code`, `TicketResult`, and `OAuth2CodeResult` calls do not execute this HTTP hook.

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
| `GET /sso/authorize` | Checks center login state, issues a credential for the requested Mode, and redirects back to the client app |
| `POST /sso/token` | Exchanges a Ticket or OAuth2 Code for login subject information |
| `POST /sso/introspect` | Checks whether a Ticket, shared token, remote session, or OAuth2 Code is active |
| `POST /sso/userinfo` | Reads login subject, client, and scope information from a valid credential |
| `POST /sso/revoke` | Revokes a Ticket, shared token, remote session, or OAuth2 Code |
| `GET/POST /sso/logout` | Clears shared cookie state, pushes registered client logout callbacks, and returns logout result |

## Single Logout

`DefaultServerOptions()` and `DefaultHTTPOptions()` enable `EnableSLO`; a zero-valued options struct does not. A client application can send its logout callback URL during the login redirect, and the login center will record a client session binding:

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

After receiving the callback, the client application should delete its local Session, Cookie, or Token. After notifications succeed (or best-effort handling permits continuation), the login center runs the optional `CenterLogout` hook, then clears the stored client session records and browser cookie.

`LogoutCallbackBestEffort` controls failure behavior. When it is `false`, any failed client callback makes the center logout return an error. When it is `true`, the server still attempts all callbacks and clears center-side client session records. The default callback client does not follow redirects; `LogoutHTTPClient` can inject a custom HTTP client for proxy, TLS, or stricter timeout control.

To reduce SSRF risk from malicious `callback` values, the login center only records logout callback URLs that belong to the current client: exact `RedirectURIs` matches, same-origin URLs as a registered redirect URI, or origins explicitly configured in `AllowOrigins`.

### Logout Acknowledgements

A logout callback must return a 2xx HTTP status and acknowledge success. The built-in `LogoutCallbackHandler` returns JSON with `code: 0`. Nonempty JSON responses must contain an integer `code` equal to zero; business errors such as HTTP 200 with `{"code":1}` fail the callback. Missing/null codes, malformed JSON, HTML pages, response read failures, and bodies larger than 64 KiB also fail. For compatibility, an empty or whitespace-only body and plain text `ok` are still accepted with a 2xx status. Custom handlers should return a success acknowledgement only after local logout completes; other nonempty response formats must migrate to `{"code":0}`.

Strict mode reports every failed client ID and retains the cookie and client bindings without calling `CenterLogout`, so the operation can be retried. Notifications are still attempted for all registered targets. Best-effort mode continues to `CenterLogout` despite acknowledgement failures, as it does for network failures. Successful notifications are not rolled back and may be repeated on retry.

### Center Session Revocation

When `LoginIDResolver` reads a server-side session, configure `HTTPOptions.CenterLogout` to invalidate that session through the same session store or auth instance:

```go
options := sso.DefaultHTTPOptions()
options.SecretKey = "replace-with-a-strong-signing-secret"
options.LoginIDResolver = resolveCenterSession
options.CenterLogout = func(r *http.Request, loginID string) error {
	return revokeCenterSession(r.Context(), loginID)
}
httpSSO := sso.NewHTTPServer(server, options)
```

`resolveCenterSession` and `revokeCenterSession` are application functions using the same source of login state. The hook receives only the authenticated identity resolved by `LoginIDResolver`; a mismatched request `loginId` is rejected. Strict client notification failures stop before the hook. Hook errors fail logout even in best-effort mode and retain callback records and the cookie for retry; hooks and client callbacks must tolerate repeated calls. If session revocation succeeds but later credential revocation or storage cleanup fails, the center session is not restored. With `EnableSLO` false, center logout and account credential revocation still run, but client notifications and binding cleanup are skipped. A nil hook leaves application-owned center sessions untouched; the framework still revokes account SSO credentials and clears the browser cookie.

### Revoking Associated Credentials

New Tickets, shared tokens, remote sessions, and OAuth2 Codes record the account's `LogoutVersion`. After notifications and `CenterLogout` succeed, HTTP logout calls `Server.RevokeLoginCredentials` to advance that version before clearing callback bindings and the cookie. Old versions cannot be validated, exchanged, used for userinfo, or renewed; HTTP introspection returns `active: false`. Revocation covers the account across all clients and devices, while other accounts and auth namespaces remain unaffected.

Applications revoking center sessions through other entry points should also call:

```go
if err := server.RevokeLoginCredentials(ctx, loginID); err != nil {
	return err
}
```

This method only revokes SSO credentials; it does not notify clients or delete application sessions or cookies. Use it from administrative logout or application-owned session revocation flows; natural expiry does not call it automatically. Subsequent legitimate logins can issue new-version credentials. Applications must actually revoke center login state, since copied stateless cookies could otherwise request new credentials. Local sessions already created by a successful exchange remain subject to logout callbacks and local expiry.

Markers use ordinary storage `Get` / `Set`, without requiring CAS, transactions, or credential scans. Server instances sharing storage, prefix, and auth namespace observe the same revocation state. Lookup failures return storage errors instead of treating a credential as valid or normally inactive. Markers persist and must not be deleted independently while associated credentials may remain. Credential payloads expire through their own TTL; `Get*TTL` reports storage lifetime, not validity. Use Validate or Introspect to check validity.

Legacy records without `LogoutVersion` retain their original lifetime and are not retroactively revoked. Full coverage requires waiting for their expiry or an application-controlled migration. Requests that already passed validation are not rolled back, and consumed one-time credentials are not restored when a later step fails.

### Checking Center Sessions During Reads

`HTTPOptions.CheckCredentialActive` adds application-state validation to `/introspect` and `/userinfo` for all four credential modes. The client must authenticate and the credential must pass its normal ownership, lifetime, and logout-version checks before the hook runs. It receives a `CredentialInfo` value with the stored login ID, client ID, mode, scopes, and `Extra`; treat the metadata as read-only.

```go
options.CheckCredentialActive = func(r *http.Request, credential sso.CredentialInfo) (bool, error) {
	return isCenterSessionActive(r.Context(), credential.LoginID, credential.Extra["centerSessionID"])
}
```

`isCenterSessionActive` is an application function using the same session source as login and exchange. Return `false, nil` for an absent, expired, or revoked session: introspection returns only `active: false`, and userinfo returns HTTP 401 without identity data. Return an error for storage or lookup failures; they remain errors rather than inactive responses. A hook cannot reactivate an invalid credential. Reads do not consume or revoke credentials, and `/revoke` does not depend on this hook.

A nil hook preserves the existing SSO-storage-only checks. This option does not change direct Server validation, issuance, exchange, or local client sessions; exchange uses `EnrichExchangeResult` separately. `expiresIn` remains the SSO credential's lifetime. Both center examples share one non-renewing Auth session check across exchange and these read endpoints, so a retired center token cannot continue exposing user information through a still-stored Ticket.

## Redis Storage

Production deployments should use Redis storage for the SSO Server. One-time credentials such as Ticket and OAuth2 Code require atomic get-and-delete behavior, which the Redis component provides. ClientSession records used by single logout can also be shared across multiple SSO Server instances. Registrations are serialized within each Server instance; deployments that concurrently register the same login subject across instances should add a storage-level atomic merge operation before relying on that write pattern.

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
$env:DTOKEN_REDIS_URL="redis://:password@127.0.0.1:6379/0"
go test ./sso/storage/redis/... -v
```

## Shared Cookie

For applications under the same parent domain, shared cookies can be used as the SSO-center session source. This fits deployments such as `sso.example.com`, `app-a.example.com`, and `app-b.example.com`. A strong random `SecretKey` is required; without one, the cookie is not accepted as a trusted login identity.

The signed cookie includes a precise expiration time that the server checks independently of browser `Max-Age`. Older cookies without a signed deadline are rejected after upgrading, so users must sign in again. Clearing a cookie removes the browser copy; it does not revoke other copies of this stateless credential before expiry. For immediate revocation, use a `LoginIDResolver` backed by server-side sessions and invalidate those sessions in the application's logout flow.

```go
cookie := sso.CookieOptions{
	Name:     "dtoken_sso",
	Domain:   ".example.com",
	Path:     "/",
	MaxAge:   2 * time.Hour,
	HTTPOnly: true,
	Secure:   true,
	SameSite: http.SameSiteLaxMode,
	SecretKey: "replace-with-a-strong-cookie-signing-secret",
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

### Per-Client Signing Keys

Configure `Client.SecretKey` at the center and the matching `ClientConfig.SecretKey` in the application to isolate each client's signatures:

```go
err := server.RegisterClient(&sso.Client{
	ClientID:     "app-a",
	ClientSecret: "app-a-authentication-secret",
	SecretKey:    "app-a-signing-secret",
	RedirectURIs: []string{"https://app.example.com/sso/callback"},
	Modes:        []sso.Mode{sso.ModeTicket},
})
if err != nil {
	return err
}

options := sso.DefaultHTTPOptions() // CheckSign is enabled.
options.SecretKey = "legacy-client-signing-secret" // Optional fallback.
options.Cookie.SecretKey = "separate-center-cookie-secret"
httpSSO := sso.NewHTTPServer(server, options)

app := sso.NewClientApp(sso.ClientConfig{
	ClientID:     "app-a",
	ClientSecret: "app-a-authentication-secret",
	ServerURL:    "https://sso.example.com",
	CheckSign:    true,
	SecretKey:    "app-a-signing-secret",
})
```

Signing key selection is `Client.SecretKey` first, then `ServerOptions.SecretKey` when the client key is empty. This applies to authorization, exchange, introspection, userinfo, revocation, logout, and outgoing logout callbacks. Verification may also accept VerificationKey within the selected group; a mismatch never tries fallback keys from another group. `ClientSecret` remains the client authentication credential and is not used as an implicit signing key. `CheckSign: false` retains its existing behavior.

Signed requests with a client ID load that client's current registration; unknown clients and storage failures are rejected. Repeated client parameters are rejected as ambiguous. With no client ID, the server uses the global key, or the anonymous client's key selection when `AllowAnonymousClient` is enabled. Logout still requires a trusted center login identity. Deployments using only dedicated keys may omit the global key, but must configure `Cookie.SecretKey` or a custom `LoginIDResolver` for center login state; client signing keys are never used to sign center cookies.

Updating a client's registration with `RegisterClient` changes the key used by subsequent requests and callbacks. Coordinate the change with that application's `ClientConfig.SecretKey`; previously signed requests stop working after rotation. Clients without `SecretKey` retain the existing global-key behavior. Set different keys for different applications to obtain signature isolation.

## Single Logout Reliability

For per-client logout results, instance-wide delivery limits, retries, and outcome observation, see [single logout reliability](./LOGOUT_RELIABILITY.md). The default remains one attempt; additional attempts retry only eligible failed targets within the same invocation.

## Replay Protection

For opt-in timestamp checks, nonce deduplication, and operation-bound signatures across requests and logout callbacks, see [signed communication and replay protection](./REPLAY_PROTECTION.md). Enable it on both the center and applications with shared storage for replicas.

## Browser Client Integration

`BrowserClient` composes login initiation, independent browser state, callback validation for all modes, local-session deadlines, and local/global logout handlers. Application hooks retain ownership of sessions. See the [browser integration guide](./BROWSER_CLIENT.md). The standard-library client selects the complete example with `-browser-flow`, preserving the original default entry point.

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
| `RevokeLoginCredentials` | Invalidate an account's previously issued versioned credentials across all clients |
| `RegisterClientSession` | Record a client login binding for single logout callbacks |
| `GetClientSessions` | Query registered client sessions for a login subject |
| `ClearClientSessions` | Clear registered client sessions for a login subject |

## Notes

- A Ticket is a one-time credential and is deleted from storage after successful consumption.
- `ConsumeTicket` validates the client secret, target client, callback URL, expiration state, and allowed SSO mode.
- The built-in `MemoryStorage` used by `sso.NewServer()` is intended for local debugging and tests only. Data is lost after process restart and it is not suitable for multi-instance deployments.
- Production deployments should use `sso/storage/redis`. Redis storage implements atomic get-and-delete for cross-instance one-time Ticket and OAuth2 Code consumption.
- Production deployments should enable `CheckSign` and configure `SecretKey` so Server and Client traffic, including logout callbacks, is protected against tampering.
- Logout callback URLs must be absolute HTTP(S) URLs without user information or fragments. Registration and sending both check current client allow-lists; unregistered clients are skipped. Avoid adding overly broad origins to `AllowOrigins`.
- The framework prefers `adapter.AtomicStorage` for atomic Ticket and SSO OAuth2 Code consumption. Plain `Storage` uses a single-server serialized fallback and is not atomic across server instances.
- `ModeSharedToken` is for trusted internal systems that reuse a short-lived credential and is client-scoped by default.
- `ModeRemoteSession` is for applications that remotely check login state at the SSO center.
- `ModeOAuth2` is an SSO authorization-code primitive, not the full OAuth2 Token Server.
- `Signer` ignores the signature field itself, sorts parameter names, and preserves the order of repeated values to match first-value lookup semantics. Verification rejects empty secrets and multiple signature values. Single-value signatures are unchanged; regenerate signatures for repeated parameters when upgrading.
- The HTTP authorization route can issue Ticket, shared-token, remote-session, or OAuth2-Code credentials by Mode, with exchange, introspection, revocation, userinfo, and unified logout callback endpoints.

## Testing And Examples

- [SSO testing guide](./TESTING.md)
- [Gin SSO Server example](../examples/sso_gin_server/README.md)
- [Gin SSO Client example](../examples/sso_gin_client/README.md)
- [net/http SSO Server example](../examples/sso_server/README.md)
- [net/http SSO Client example](../examples/sso_client/README.md)

## Credential binding with PKCE

Ticket and OAuth2 code modes support PKCE S256. Set `Client.RequirePKCE` at the center and `BrowserClientOptions.PKCE` for automatic browser flow management, or integrate `AuthURLWithPKCE` and `CredentialRequest.CodeVerifier` manually. Issuance, redemption, and HTTP identity lookups enforce the binding; invalid proofs preserve credentials during normal consumption. Both switches default to false and existing APIs remain available. See the [PKCE guide](PKCE.md) for the full flow, rollout order, and boundaries.

## Per-request remote authentication

`NewRemoteAuth` authenticates reusable remote sessions and shared tokens. Use `Middleware` for standard HTTP handlers or `Authenticate` for framework adapters. Each request checks the center, client binding, mode, lifetime, and all required scopes; failure blocks access, and successful middleware exposes identity through request context. Bearer input, custom session resolution, an overall validation timeout, and error callbacks are supported without creating local sessions or caching success. See [remote authentication](REMOTE_AUTH.md) for integration and state boundaries.

## Administrative account logout

For trusted administrative workflows, see [account logout](ACCOUNT_LOGOUT.md): `HTTPServer.LogoutAccount(r, loginID)` composes center-session revocation, old SSO credential invalidation, application notifications and recoverable binding cleanup with stage progress. It adds no management HTTP route; operator authorization and account suspension remain application-owned.

## Signing-key rotation

Signing-key rollover is available through optional `VerificationKey` on client registrations, server fallback settings and ClientApp. Outbound traffic still uses only `SecretKey`. See [key rotation](SIGNING_ROTATION.md) for the three-stage rollout and trust boundaries.

## Request parameter consistency

Protocol fields must be single-valued and credential fields must match the selected mode. See [request parameter consistency](REQUEST_VALIDATION.md) for duplicate query/body handling, rejection ordering and migration from ambiguous inputs.

For coordinated upgrades, module verification and behavior changes, see the [upgrade and release guide](MIGRATION.md). Compilation, runtime regression and deployment acceptance are separate checks.

## Dynamic authorization policy

`HTTPOptions.AuthorizationPolicy` evaluates current user access at issuance, exchange, and identity inspection. It can deny access or reduce scopes, never enlarge the requested or original stored grant. Denied exchanges preserve credentials; denied inspections return inactive, allowing `RemoteAuth` to reflect changes on subsequent requests. Nil preserves existing behavior, and trusted low-level Server APIs do not automatically execute HTTP policy. See [authorization policy](AUTHORIZATION_POLICY.md) for integration, error classification, and lifecycle semantics.
