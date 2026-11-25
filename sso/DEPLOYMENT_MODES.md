# Three SSO deployment patterns

[中文](DEPLOYMENT_MODES_zh.md)

The center authenticates the user, applications accept a trusted login result, subsequent applications reuse the center session, and global logout revokes associated sessions. Deployment determines how identity travels. `ModeTicket`, `ModeOAuth2`, `ModeSharedToken`, and `ModeRemoteSession` are credential types, not deployment patterns.

| Deployment | Identity transfer | Sessions and logout | Integration |
| --- | --- | --- | --- |
| Shared cookie domain and authentication Redis | Shared cookie Token resolves the same Auth session | Revocation is visible on subsequent session checks | Explicit composition using root Auth and cookie helpers; no dedicated all-in-one pattern-one component yet |
| Different domains, shared authentication Redis | Browser-delivered Ticket is exchanged directly against shared storage | Independent application sessions still require logout notifications | `ClientConfig.SharedStoreServer` with `BrowserClient` |
| Different domains, independent Redis | Browser-delivered Ticket is exchanged over HTTP | Applications establish and revoke their own sessions | Default `ClientApp` with `BrowserClient`; see the [deployment example](../examples/sso_deployment/README.md) |

The shared domain must actually allow cookie sharing under the configured domain and path. Business Redis may differ; authentication storage is what matters. Same-domain applications with independent authentication Redis can also use HTTP exchange. Matching Redis addresses alone is insufficient: database, key prefix, AuthType, and codec settings must match.

## Shared-store exchange

There is still one login center. It serves the login page, authenticates users, and exposes `/sso/authorize`. An application composes a local protocol service connected to the center's storage solely for exchange. `HTTPServer` already owns these protocol rules; the local instance needs neither `Register` nor a listener.

Here `sharedStore` is an initialized shared `contract.Storage`, `centerExchangeOptions` supplies the same exchange rules as the center, and `clientConfig` is the existing application configuration:

```go
peer := sso.NewServer(
    sso.WithStorage(sharedStore),
    sso.WithKeyPrefix("company:sso:"), // Match the center.
    sso.WithAuthType("sso"),           // Match the center.
)

// Read registrations maintained by the center; do not overwrite them.
centerExchangeOptions.Clients = nil
centerExchangeOptions.AllowAnonymousClient = false
localExchange, err := sso.NewHTTPServerChecked(peer, centerExchangeOptions)
if err != nil {
    _ = peer.Close()
    return err
}
clientConfig.SharedStoreServer = localExchange
clientConfig.UseHTTPCheck = false
clientApp, err := sso.NewClientAppChecked(clientConfig)
if err != nil {
    _ = peer.Close()
    return err
}
// Pass clientApp to the existing NewBrowserClient and retain the same routes.
// Close peer after draining application requests; sharedStore remains owner-managed.
```

Prefix and AuthType values are illustrative and must match the actual center. See the complete [public API example](example_shared_store_test.go) and [browser integration](BROWSER_CLIENT.md).

| SharedStoreServer | UseHTTPCheck | Ticket/code exchange |
| --- | --- | --- |
| nil | false or true | HTTP, preserving historical behavior |
| non-nil | false | Local shared storage |
| non-nil | true | Force HTTP; ignore the local service |

Neither path falls back to the other on failure. `ServerURL` remains necessary for browser authorization and logout redirects. `Introspect`, `UserInfo`, and `Revoke` still use HTTP. Local exchange supports Ticket and code only; checked client configuration rejects an active local service with Shared Token or Remote Session mode.

## One login and logout lifecycle

1. `BrowserClient.BeginLogin` stores state and PKCE proof before redirecting to the center.
2. The center resolves its session, requests authentication if missing, or reuses the existing session to issue a Ticket.
3. The application callback validates and consumes browser state.
4. `ClientApp.ExchangeCredential` locally executes the same parsing, signing, replay, client authentication, PKCE, authorization policy, consumption, enrichment, and logout registration pipeline as `/sso/token`.
5. The application hook establishes its local session. A second application repeats this flow using the existing center login.
6. Global logout remains center-owned: notify applications, revoke their sessions through hooks, revoke the center session and old SSO credentials, and clear bindings.

Shared protocol storage does not automatically revoke independent Auth sessions. Configure `EnableSLO`, `RegisterCallback`, a valid `LogoutCallbackURL`, and a callback that actually revokes local sessions. `CenterLogout` revokes the business-owned center session. Clearing a cookie alone does not revoke a stored session. Direct exchange writes callback bindings visible to the center.

## Configuration and runtime boundaries

- Keep protocol parameters, signing trust, replay namespace, client limits, authorization policy, and enrichment consistent between the center and local service. Checked client configuration rejects mismatched parameter names and uninitialized services. It cannot copy functions from another process or verify remote configuration equality.
- Shared authentication storage places applications in the same trust boundary. Applications should not overwrite registrations at startup. Use HTTP exchange when application storage permissions must be isolated.
- Local policy and enrichment execute in the application process. `EnrichExchangeResult` receives a synthetic POST with form values, the center URL, and the caller's Context, without browser cookies, network provenance, or HTTP middleware state. Hooks needing center-only resources must use shared resources or retain HTTP exchange.
- Request limits, signatures, and nonces still apply. `HTTPClient.Timeout` and `MaxResponseBytes` do not govern local calls. Supply a Context deadline; storage and hooks must honor it. Browser flows preserve request Context.
- Local calls return original Go errors and skip JSON encoding and `OnHTTPError`. Browser handlers still produce their existing bounded error responses. Do not render raw errors to users.
- Policy, proof, and callback ownership rejection leave credentials intact, although a nonce may already be consumed. Enrichment failure after consumption does not restore credentials or browser state; restart login.
- Atomic storage supports one-time consumption across instances. Ordinary storage retains Get/Delete fallback with instance-local locking. Prefer an atomic-capable Redis adapter for concurrent processes.

## Verification status

New regression sources cover separate issuer/consumer instances, Ticket and code, ordinary/atomic storage, PKCE, reduced grants, invalid requests, nonces, capacity, cancellation, failure lifecycles, two-domain center-session reuse, and global logout. They use memory and controlled transports, without listeners or real Redis.

Only static checks were performed; no builds, tests, examples, or services were run. User-run commands:

```powershell
go test ./sso/... -run 'TestSharedStore|ExampleClientApp_sharedStore' -count=1
go test ./sso/... -count=1
```

Shared-memory tests do not establish real Redis, cross-process deployment, or network logout acceptance; verify those in the intended environment.
