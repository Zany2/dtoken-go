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

- `TestSessionFlow`: tests session lookup.
  - Login.
  - Request `/api/session`.
  - Assert login ID and terminal count.

- `TestDisableFlow`: tests account and service disable.
  - Account disable rejects old token and new login.
  - Service disable rejects only `/api/payment`.

- `TestNonceFlow`: tests nonce.
  - Generate nonce through `/nonce`.
  - Verify once through `/nonce/verify`, expect success.
  - Verify the same nonce again, expect failure.

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
