# Complete SSO integration example

[中文](README_zh.md) · [Deployment and operations](../../sso/DEPLOYMENT.md)

Run the same entry point as `-role center` or `-role client` for a center and multiple independent applications. The example composes signed requests, replay protection, PKCE S256, browser state, Auth sessions, strict single logout, and shared storage. The four existing HTTP/Gin examples remain available for smaller integration steps.

The center uses an **explicitly enabled fixed-account demonstration authenticator**. It does not implement an account database, password reset, MFA, risk controls, or login rate limiting. Replace the credential check in `center.go` with your identity system before production. There are no default keys, and development mode is disabled unless requested.

## Three-process local flow

From the repository root, run one command in each of three PowerShell terminals:

```powershell
./examples/sso_deployment/local.ps1 -Role center
./examples/sso_deployment/local.ps1 -Role alpha
./examples/sso_deployment/local.ps1 -Role beta
```

The script runs in the current terminal, sets process environment variables, and forces loopback development mode with memory storage and public local-only credentials. Ctrl+C stops the service. The workspace currently requires Go 1.27; the module declares Go 1.25. This implementation pass has not executed, built, or tested the example; run these commands for local verification.

1. Open [Alpha](http://localhost:9201/protected) and sign in at the center with `alice` / `local-example-password`.
2. Open [Beta](http://localhost:9202/protected) in the same browser. It reuses the center login and establishes an independent application session.
3. Select **Local logout** in Alpha. Only the current Alpha browser session is retired. Visiting Alpha again can sign in automatically through the center.
4. Select **Global logout** in either app. The app logs out locally before redirecting to the center. The center sends freshly signed callbacks, revokes the account's center sessions and SSO credentials, and returns a JSON result. Both applications reject their old session cookies.
5. Stop an application before global logout to observe strict callback failure. The center retains its session and callback records for retry; the initiating application's local logout is already complete. Restart the app and initiate a new login/global-logout flow to generate a new nonce. Do not refresh the previous logout URL.

Memory storage loses sessions, pending flows, and replay records on restart. With Redis and stable keys/namespaces, replicas of an application share these records. Regression sources cover starting login on one replica and completing the callback on another.

## Deployment acceptance

See [deployment acceptance](ACCEPTANCE.md) for live HTTP checks, browser checks and service/Redis failure recovery. The `acceptance` subpackage skips live traffic by default; explicit targets, test credentials and `SSO_ACCEPTANCE_RUN=true` are required. It starts/stops no services. HTTP automation does not replace browser or failure acceptance.

## Configuration

Optional `SSO_VERIFICATION_KEY` accepts one additional logout-callback key on the client; nonempty values require at least 32 bytes. Centers use optional `verificationKey` in each `SSO_CLIENTS_JSON` registration with the same length rule. Signing still uses `SSO_SIGNING_KEY` / `secretKey`. Follow the [three-stage rotation guide](../../sso/SIGNING_ROTATION.md); keep registration writers coordinated across replicas.

All settings come from process environment variables. Configuration errors do not echo secrets or raw registration JSON. `SSO_DEV` accepts only `true` or `false`; omission means `false`.

| Variable | Used by | Rule |
| --- | --- | --- |
| `SSO_NAMESPACE` | All | Environment ID: 1–64 letters, digits, `_`, `-`; consistent within an environment |
| `SSO_LISTEN` | All | Explicit `host:port`; development requires loopback |
| `SSO_PUBLIC_URL` | All | Public root origin, such as `https://alpha.example.com`, without path or trailing slash |
| `SSO_REDIS_URL` | All | Required outside development; existing Redis adapter URL, including `rediss://` |
| `SSO_COOKIE_KEY` | All | At least 32 random bytes; identical across a service's replicas and managed independently per app |
| `SSO_COOKIE_VERIFICATION_KEY` | All, optional | At least 32 bytes when set; verifies old/new session and state cookies during [cookie rotation](../../sso/SIGNING_ROTATION.md#browser-session-and-state-cookie-rotation); never used to issue cookies |
| `SSO_CENTER_URL` | Clients | Center root origin matching its public URL |
| `SSO_CLIENT_ID` | Clients | 1–64 letters, digits, `_`, `-`; matches registration |
| `SSO_CLIENT_SECRET` | Clients | At least 32 bytes; matches `clientSecret` |
| `SSO_SIGNING_KEY` | Clients | At least 32 bytes; matches `secretKey`; generate separately from cookie keys |
| `SSO_CLIENTS_JSON` | Center | One nonempty registration array, at most 1 MiB; unknown fields, duplicate IDs and trailing JSON rejected |
| `SSO_ENABLE_DEMO_LOGIN` | Center | Explicit `true` for the current demonstration; replace the authenticator for real use |
| `SSO_DEMO_USER` / `SSO_DEMO_PASSWORD` | Center | Demo username and password of at least 16 bytes; never logged |

Outside development, public origins must use HTTPS. The process listens on HTTP behind a trusted TLS proxy. Development public URLs must use `localhost` or loopback IPs. This example mounts at root origins and does not implement path-prefix deployment; framework gateway prefixes are covered in [configuration guidance](../../sso/CONFIGURATION.md).

Example center registration (replace secret placeholders using your environment/secret manager; `${...}` interpolation is not implemented):

```json
[
  {
    "clientId": "alpha",
    "clientSecret": "<32-byte-or-longer-random-exchange-secret>",
    "secretKey": "<32-byte-or-longer-independent-signing-key>",
    "redirectUris": ["https://alpha.example.com/sso/callback"],
    "modes": ["ticket"],
    "requirePkce": true,
    "scopes": ["profile"]
  }
]
```

Each app has one exact `/sso/callback` URL and must use Ticket with PKCE. These are example-specific restrictions. Logout callbacks use the same origin at `/sso/logout-callback`; do not add unrelated `allowOrigins`. Every center startup registers/overwrites listed clients. Removing a client from the array **does not delete existing Redis registrations**; use `UnregisterClient` and handle existing sessions when removing access. Center replicas must use the same registration version to avoid overwriting one another during rollout.

## Source map

| File | Responsibility |
| --- | --- |
| `config.go` | Local configuration, registrations, namespace validation |
| `application.go` | Auth sessions, BrowserClient, health routes, response protections |
| `center.go` | Demo credentials, center-session binding, exchange lifetime, logout delivery |
| `main.go` | Storage ownership, HTTP timeouts, Ctrl+C/SIGTERM shutdown |
| `application_test.go` | Actual-handler multi-app/replica lifecycle and failure boundaries |
| `lifecycle_review_test.go` | Canonical origins, absolute center deadlines, and handler completion after connection closure |
| `account_logout_test.go` | Trusted account logout, real Auth revocation, signed multi-app notification failure and retry across replicas |
| `nginx.conf.example` | Three independent HTTPS origins |

See [deployment and operations](../../sso/DEPLOYMENT.md) for Redis data responsibilities, deployment steps, scaling, and failure handling.

The center retains `centerHTTP` for application-owned administrative operations using [LogoutAccount](../../sso/ACCOUNT_LOGOUT.md). No management route or account suspension system is exposed; the caller authorizes targets and controls new login/issuance.

Origin configuration normalizes host case, IP spelling, and default ports before generating callback registrations. Center and app sessions enforce absolute deadlines. Shutdown waits for handlers even after forced connection closure; see the [release checklist](../../sso/RELEASE_CHECKLIST.md) for the remaining runtime checks. Tests have been authored but not executed in this implementation pass.
