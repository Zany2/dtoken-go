# DToken Beego Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/beego` with Beego filters.

## Run

```bash
cd examples/beego
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: accepts a nonempty username and password `123456`, then grants the demo user the `admin` role and `article:read` permission.
- `GET /me`: returns current login information.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `beegodt.NewBuilder()`, so no Redis service is required. Login state and authorization data are lost on restart. Tokens expire after two hours; automatic renewal is enabled with a one-hour remaining-lifetime threshold.

This is a demonstration credential check. Real applications must verify credentials against their user data and load the user's actual roles and permissions. By default, repeated logins for the same user and device share a token; logging out that token invalidates all clients using it.

## Input and authentication

Login accepts URL-encoded forms, multipart forms, and Query parameters through Beego's `Input.Query`. URL-encoded body fields take precedence over Query fields. JSON login bodies are not supported. Use a form body as shown below to avoid putting credentials in URLs and access logs; Query support is retained for compatibility.

The login input filter runs at `BeforeStatic`, before Beego's automatic form parser. It rejects parsing/read errors and oversized form bodies with HTTP 400, retains Beego's `BConfig.MaxMemory` limit for ordinary forms and `BConfig.MaxUploadSize` limit for multipart forms, and prevents partial parse results from being used to log in.

Authentication tokens are read from the `dtoken` or `Authorization` header, either raw or prefixed with `Bearer `. Query, Cookie, and Body token extraction are disabled by default; login credentials and authentication tokens use separate input rules.

## Responses and lifecycle

Handlers and DToken filters use a common JSON envelope: `code`, `message`, and optional `data`.

- HTTP 400: missing credentials or invalid login input.
- HTTP 401: incorrect demo password or missing/invalid/inactive token.
- HTTP 403: account/device restrictions or insufficient roles/permissions.
- HTTP 500: unavailable Manager or internal errors, including reported storage failures. Internal details are logged server-side and replaced with a fixed response message.

Every DToken filter uses `WithFailFunc(writeAuthError)` and stops routing on failure. The example enables Beego's graceful shutdown so that normal interrupt/termination signals drain HTTP requests before `Run` returns and the deferred Manager cleanup runs.

## Try

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "username=admin" \
  --data-urlencode "password=123456"
curl http://localhost:8080/me \
  -H "Authorization: Bearer <token>"
curl http://localhost:8080/admin \
  -H "Authorization: Bearer <token>"
curl http://localhost:8080/articles \
  -H "Authorization: Bearer <token>"
curl -X POST http://localhost:8080/logout \
  -H "Authorization: Bearer <token>"
```

## Key APIs

```go
// registerRoutes installs the example's filters and handlers together.
registerRoutes(web.BeeApp.Handlers)

// Pass the same error callback to each DToken filter.
beegodt.AuthMiddleware(ctx, beegodt.WithFailFunc(writeAuthError))
beegodt.RoleMiddleware(ctx, []string{"admin"}, beegodt.WithFailFunc(writeAuthError))
beegodt.PermissionMiddleware(ctx, []string{"article:read"}, beegodt.WithFailFunc(writeAuthError))
```
