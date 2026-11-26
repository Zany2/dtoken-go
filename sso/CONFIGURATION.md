# SSO configuration and startup validation

[中文](CONFIGURATION_zh.md)

`ClientConfig.SharedStoreServer` explicitly enables shared-store Ticket/code exchange; `UseHTTPCheck=true` forces HTTP. Without a local service, historical HTTP behavior is preserved. Validation checks local initialization, credential mode, and parameter-name equality without probing Redis. See [deployment patterns](DEPLOYMENT_MODES.md).

Center HTTP failures use fixed public messages and the optional `OnHTTPError` observer for private diagnostics. See [HTTP errors and diagnostics](HTTP_ERRORS.md) for all six endpoints, status compatibility and callback rules.

Use checked constructors when assembling an application so configuration failures are returned before HTTP listening starts. These checks are local: they do not contact the center, probe storage availability, or verify that client and server secrets match.

## Entry points

Configured protocol names also define scalar request fields. Duplicate values, including query/body duplicates, are rejected before nonce consumption. Credential operations accept only fields matching the selected mode. See [request consistency](REQUEST_VALIDATION.md) for custom names and compatibility.

HTTP(S) URL validation rejects empty, zero, or out-of-range explicit ports. Origin comparisons normalize host case, IP spelling, and effective default ports; authorization redirect URIs remain exact string matches. Browser state cookies also reject unknown `SameSite` values. See [browser integration](BROWSER_CLIENT.md) for Origin header constraints.

| API | Behavior |
| --- | --- |
| `Endpoints.Validate()` / `ParamNames.Validate()` | Validate after filling omitted fields with defaults; do not mutate the input |
| `(*Client).Validate()` | Check registration metadata without persistence |
| `(*Server).RegisterClientChecked(client)` | Validate metadata, then use the existing registration lifecycle |
| `HTTPOptions.Validate()` | Validate routes, parameters, cookies, signing/replay wiring, and inline clients without storage access |
| `NewHTTPServerChecked(server, options)` | Validate all local options, register inline clients, and return any initialization failure |
| `httpserver.NewChecked(server, options)` | The same checked constructor through the optional facade |
| `ClientConfig.Validate()` / `(*ClientApp).Validate()` | Check normalized client settings without HTTP requests |
| `NewClientAppChecked(config)` | Validate and create a client helper |
| `(*HTTPServer).InitializationError()` | Read a retained initialization error; this is not a health check |

`NewBrowserClient` and `NewRemoteAuth` also validate their complete `ClientApp` configuration, including endpoints that the current flow does not use. Their own mode, state-cookie, timeout, and scope restrictions still apply.

`HTTPOptions.RequestLimits` and `ClientConfig.RequestLimits` default to 8 KiB queries and 64 KiB form bodies. Each nonpositive field independently uses its default; it does not disable limits. These inbound limits are independent of response `MaxResponseBytes`. See [HTTP request limits](REQUEST_LIMITS.md) for 413 responses, cached forms and proxy configuration.

## Startup example

<!-- dtoken-compile: sso-configuration -->
```go
package main

import (
	"net/http"

	"github.com/Zany2/dtoken-go/sso"
)

// assembleSSO checks configuration and transfers the server lifetime to the caller on success. assembleSSO 检查配置，成功后由调用方管理 Server 生命周期。
func assembleSSO(resolve sso.LoginIDResolver, exchangeKey, signingKey string) (*sso.Server, http.Handler, *sso.ClientApp, error) {
	server := sso.NewServer()
	options := sso.DefaultHTTPOptions()
	options.RequestLimits = sso.RequestLimits{MaxQueryBytes: 8 << 10, MaxBodyBytes: 64 << 10}
	options.LoginIDResolver = resolve
	options.Clients = map[string]sso.Client{
		"orders": {
			ClientSecret: exchangeKey,
			SecretKey:    signingKey,
			RedirectURIs: []string{"https://orders.example.com/sso/callback"},
			Modes:        []sso.Mode{sso.ModeTicket},
			Scopes:       []string{"profile"},
		},
	}
	handler, err := sso.NewHTTPServerChecked(server, options)
	if err != nil {
		_ = server.Close()
		return nil, nil, nil, err
	}

	config := sso.DefaultClientConfig()
	config.RequestLimits = options.RequestLimits
	config.ClientID = "orders"
	config.ClientSecret = exchangeKey
	config.SecretKey = signingKey
	config.ServerURL = "https://login.example.com"
	app, err := sso.NewClientAppChecked(config)
	if err != nil {
		_ = server.Close()
		return nil, nil, nil, err
	}
	return server, handler.Handler(), app, nil
}

// main leaves authentication, secret injection and HTTP listening to the application. main 将身份核验、密钥注入及 HTTP 监听交给应用。
func main() {}
```

Use persistent shared storage for deployment; the example above uses the default in-memory store only to show assembly. `ClientSecret` authenticates credential exchange; `SecretKey` signs protocol requests. A client-specific signing key takes precedence over the server fallback. A center does not need a global fallback when all relevant clients have independent keys.

The application supplies its trusted identity resolver and managed secrets, mounts the returned handler only after successful construction, and closes the returned Server after requests have drained. The example is included in the repository's documentation compilation checks; that check does not run `main` or verify authentication and deployment.

## Validation rules

- Optional `VerificationKey` requires `SecretKey` in the same configuration. Checked HTTP/ClientApp construction also requires signing to be enabled when a companion is configured. Dedicated client keys never combine with fallback keys; see [rotation](SIGNING_ROTATION.md). Cookie signing is unaffected by this companion field.

- Six center routes (`Authorize`, `Token`, `Introspect`, `UserInfo`, `Revoke`, `Logout`) must be distinct, clean absolute literal paths. Root/trailing slashes, wildcards, method/host patterns, queries, fragments, controls, backslashes, encoded separators, and decoded duplicates are rejected. Ordinary escapes such as `/sso/%61uth` are allowed. Reserved message and client-route fields are not mounted or validated as center routes.
- Parameter names are checked after default completion and must be distinct, nonempty, and free of whitespace/control characters. Custom routes and parameters must match on both sides. A gateway prefix in `ServerURL` is preserved.
- Client IDs must be nonempty without whitespace/control characters. Modes must be implemented; scopes must be individual nonempty names without whitespace or commas. Redirects must be absolute HTTP(S) URLs without userinfo or fragments. Allowed origins may have no path or `/`, and no query. `RequirePKCE` requires at least one Ticket/Code mode; omitted modes retain the Ticket default. Empty redirect lists remain valid for direct core API use but do not authorize an arbitrary HTTP redirect.
- A center must supply `LoginIDResolver` or a signed center-cookie key (`Cookie.SecretKey`, falling back to `ServerOptions.SecretKey`). Cookie names/domains/paths, secure prefixes, and `SameSite=None` with `Secure` are checked. A custom resolver does not disable checks on configured cookie attributes.
- Signing requires a key for each inline client, either independent or fallback. When anonymous access and signing are both enabled, supply the fallback or an explicit inline anonymous client with its own key. Anonymous allow-list URLs are checked when anonymous access is enabled. Existing clients in external storage cannot be enumerated by static validation; request-time client/key checks still apply. An existing stored anonymous key alone does not satisfy this static rule.
- Client `ServerURL` must be absolute HTTP(S), without userinfo, query, fragment, or path whitespace/backslashes. `RegisterCallback` requires a valid absolute `LogoutCallbackURL`; an unused callback field is not required. Replay protection requires signing and a valid replay component.

Existing defaults remain in effect. Omitted route/parameter fields are filled individually. Existing nonpositive defaults for client response size, callback age, center callback timeout/concurrency/retries, and cookie age remain in effect. A zero-valued options struct does not enable boolean defaults; start with `DefaultHTTPOptions()` / `DefaultClientConfig()` when their defaults are desired. Browser and RemoteAuth retain their own documented negative-value rules.

## Errors, compatibility, and ownership

`errors.Is(err, sso.ErrSSOConfig)` identifies static validation errors. `errors.As` into `*sso.ConfigurationError` exposes `Field` and a fixed `Reason`; diagnostics do not echo URLs, keys, or registration values. Existing domain classifications such as `ErrInvalidRedirectURI`, `ErrSignSecretRequired`, and `ErrModeUnsupported` are preserved where applicable. Validation returns the first error; inline client keys are sorted for stable reporting. Nil/uninitialized core errors and storage failures retain their separate classifications. Custom storage errors may contain backend details, so do not expose raw initialization errors in public HTTP responses.

All inline settings are checked before initial registrations are written, and validation does not fill IDs into the caller's map. Actual registration remains sequential and nontransactional. A storage failure may leave some registrations persisted, including overwritten existing entries; there is no rollback. Constructors do not close the supplied server, HTTP client, or borrowed storage. Keep ownership and cleanup in the application.

Legacy `NewClientApp`, `NewHTTPServer`, and `RegisterClient` signatures remain available. The legacy HTTP constructor retains route, parameter, replay, and registration errors; invalid endpoint configurations do not panic during its own mux assembly. `Handler()` returns a bounded HTTP 500 handler on initialization failure; `Register(mux)` adds no routes for invalid endpoint configurations or absent cores. Use checked construction or `InitializationError()` before mounting individual handlers. Conflicts with routes already present in an application-owned mux still follow normal `http.ServeMux` panic behavior and remain the application's responsibility. Legacy constructors do not perform all the stricter checks described above.

The four HTTP/Gin examples check their configuration before listening. Regression sources in `config_validation_test.go` cover rejection before writes, ownership, legacy routing, registration checks, diagnostics, constructor integration, and a signed authorize/exchange flow. See [testing instructions](TESTING.md) for commands to run locally.
