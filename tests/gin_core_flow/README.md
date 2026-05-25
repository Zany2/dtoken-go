# Gin Core Flow Tests

This directory contains HTTP flow tests for `examples/gin_core_app`.

这些测试不会手动请求外部端口，而是用 `httptest.NewServer` 在测试进程里启动 Gin app，然后按真实 HTTP 流程调用接口。

## Test List

- `TestAuthFlow`: tests authentication.
  - No token requests `/api/me`, expect unauthorized.
  - Login through `/login`, expect token.
  - Use token to request `/api/me`, expect current user.
  - Logout through `/api/logout`, expect old token rejected.

- `TestPermissionFlow`: tests permission checks.
  - Login without `article:read`.
  - Request `/api/articles`, expect forbidden.
  - Grant permission through `/api/permissions`.
  - Request `/api/articles` again, expect success.

- `TestRoleFlow`: tests role checks.
  - Login without `admin`.
  - Request `/api/admin`, expect forbidden.
  - Grant role through `/api/roles`.
  - Request `/api/admin` again, expect success.

- `TestRenewFlow`: tests token renewal.
  - Login with short token timeout.
  - Read initial TTL through `/api/token/ttl`.
  - Wait until TTL decreases.
  - Renew through `/api/token/renew`, expect TTL extended.

- `TestTokenExpiredFlow`: tests token expiration.
  - Login with one-second timeout.
  - Wait until timeout passes.
  - Request protected API, expect unauthorized.

- `TestActiveTimeoutFlow`: tests inactive timeout.
  - Login with long absolute TTL and short active timeout.
  - Wait until inactive timeout passes.
  - Request protected API, expect active-timeout code.

- `TestKickoutAndReplaceFlow`: tests token state changes.
  - Kick out current token, expect old token rejected.
  - Replace current token, expect old token rejected.

- `TestSessionFlow`: tests session lookup.
  - Login.
  - Request `/api/session`.
  - Assert login ID and terminal count.

- `TestMultiTerminalSessionFlow`: tests multiple terminals.
  - Login same account from web and mobile.
  - Request `/api/session` with either token.
  - Assert terminal count is two.

- `TestDisableFlow`: tests account and service disable.
  - Account disable rejects old token and new login.
  - Service disable rejects only `/api/payment`.

- `TestDeviceDisableFlow`: tests device disable.
  - Disable `web` device for current account.
  - Verify web login is rejected.
  - Verify mobile login still succeeds.

- `TestNonceFlow`: tests nonce.
  - Generate nonce through `/nonce`.
  - Verify once through `/nonce/verify`, expect success.
  - Verify the same nonce again, expect failure.

- `TestOAuth2AuthorizationCodeFlow`: tests OAuth2 authorization code flow.
  - Generate authorization code.
  - Exchange code for access and refresh tokens.
  - Verify authorization code is single-use.
  - Introspect access token.
  - Refresh token and verify old access token is invalid.
  - Revoke refreshed token and verify it is invalid.

- `TestOAuth2PasswordAndClientCredentialsFlow`: tests additional OAuth2 grants.
  - Password grant returns user token.
  - Client credentials grant returns client token.
  - Wrong client secret is rejected.

## Run

From this directory:

```powershell
go test ./...
```

If your local Go environment has `GOOS=linux` on Windows, run with Windows target:

```powershell
$env:GOOS='windows'
$env:GOARCH='amd64'
go test ./...
```
