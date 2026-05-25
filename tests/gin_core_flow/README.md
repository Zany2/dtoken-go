# Gin Core Flow Tests

This directory contains HTTP flow tests for `examples/gin_core_app`.

这些测试不会手动请求外部端口，而是用 `httptest.NewServer` 在测试进程里启动 Gin app，然后按真实 HTTP 流程调用接口。

## Test List

- `TestAuthFlow`: tests authentication.
  - No token requests `/api/me`, expect unauthorized.
  - Login through `/login`, expect token.
  - Use token to request `/api/me`, expect current user.
  - Logout through `/api/logout`, expect old token rejected.

- `TestTokenMetadataAndStatusFlow`: tests token metadata and status APIs.
  - Check `IsLogin` behavior with and without token.
  - Read token info, device, device ID, create time, and timeout.
  - Login with a custom timeout and verify TTL.
  - Call LoginByToken with an existing token.

- `TestPermissionFlow`: tests permission checks.
  - Login without `article:read`.
  - Request `/api/articles`, expect forbidden.
  - Grant permission through `/api/permissions`.
  - Request `/api/articles` again, expect success.

- `TestPermissionMutationAndLogicFlow`: tests permission mutation and logic checks.
  - Remove granted permissions and verify access is revoked.
  - Verify AND permission checks require all values.
  - Verify OR permission checks and wildcard permissions.

- `TestAccessStatusFlow`: tests boolean permission and role checks.
  - Verify HasPermission and HasRole by login ID.
  - Verify HasPermission and HasRole by token.

- `TestRoleFlow`: tests role checks.
  - Login without `admin`.
  - Request `/api/admin`, expect forbidden.
  - Grant role through `/api/roles`.
  - Request `/api/admin` again, expect success.

- `TestRoleMutationAndLogicFlow`: tests role mutation and logic checks.
  - Remove granted roles and verify access is revoked.
  - Verify AND role checks require all values.
  - Verify OR role checks require any value.

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

- `TestTerminalInspectionFlow`: tests terminal metadata queries.
  - Login from web and mobile.
  - Request `/api/terminal`.
  - Assert terminal device info and online counts.

- `TestSessionQueryFlow`: tests session query APIs.
  - Query token lists by login ID, device, and concrete device ID.
  - Query terminal lists and traverse terminals.
  - Search token values and session IDs.

- `TestTerminalOperationFlow`: tests terminal-scoped operations.
  - Logout one concrete device and keep another terminal online.
  - Kick out all terminals for a device type.
  - Replace all terminals for an account.

- `TestConcurrencyPolicyFlow`: tests login concurrency policies.
  - Shared token reuse for the same device.
  - Account-level max login count overflow.
  - Device-level max login count overflow.
  - Non-concurrent replacement and new-device rejection.

- `TestDisableFlow`: tests account and service disable.
  - Account disable rejects old token and new login.
  - Account disable info and TTL can be queried.
  - Service disable rejects only `/api/payment`.
  - Service disable info, level, and TTL can be queried.

- `TestUntieFlow`: tests removing disable states.
  - Untie account disable and login again.
  - Untie service disable and access service again.
  - Untie device disable and login from that device again.

- `TestDeviceDisableFlow`: tests device disable.
  - Disable `web` device for current account.
  - Device disable info and TTL can be queried.
  - Verify web login is rejected.
  - Verify mobile login still succeeds.

- `TestConcreteDeviceDisableFlow`: tests concrete device ID disable.
  - Disable only `web/browser-1`.
  - Concrete device disable info and TTL can be queried.
  - Verify `web/browser-1` is rejected.
  - Verify `web/browser-2` is still accepted.

- `TestNonceFlow`: tests nonce.
  - Generate nonce through `/nonce`.
  - Check nonce validity and TTL without consuming it.
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

- `TestMultiAuthIsolationFlow`: tests multiple auth systems.
  - Login the same ID into user-auth and admin-auth.
  - Verify tokens cannot cross auth systems.
  - Verify permissions and roles are isolated by AuthType.

## Run

By default, `examples/gin_core_app` uses in-memory storage. To run the same Gin demo app with Redis, pass a Redis URL through `DTOKEN_REDIS_URL` when starting the example server, or set `gincoreapp.Config.RedisURL` from code.

Example Redis URLs:

```text
redis://localhost:6379/0
redis://:password@localhost:6379/0
```

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
