# Fiber DToken Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/fiber` with Fiber middleware.

## Run

```bash
cd examples/fiber
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: accepts JSON or form credentials with any nonempty demo username and password `123456`, grants the `admin` role and `article:read` permission, and returns a token.
- `GET /me`: returns current login information.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `fiberdt.NewBuilder()`, so no Redis service is required. Login and authorization data are lost on restart. Demo credentials and permission grants illustrate the API; real applications must verify their own accounts and load their own authorization rules.

Tokens have a two-hour timeout, with automatic renewal during authentication checks when one hour or less remains. Repeated logins for the same username may reuse a token under the default sharing policy; logging out invalidates that shared token.

Use `Authorization: Bearer <token>` as shown below, or send the raw token in the `Authorization` or configured `dtoken` header. Cookie, query, and body token extraction are disabled; this does not prevent `/login` from accepting credentials in its request body. Malformed login payloads return HTTP `400`, invalid credentials return `401`, account restrictions and insufficient permissions return `403`, and server failures return `500`.

Managers are released when `main` returns or unwinds after a server startup error.

## Try

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'

curl http://localhost:8080/me \
  -H "Authorization: Bearer <token>"

curl http://localhost:8080/admin \
  -H "Authorization: Bearer <token>"

curl http://localhost:8080/articles \
  -H "Authorization: Bearer <token>"

curl -X POST http://localhost:8080/logout \
  -H "Authorization: Bearer <token>"
```

Handlers and authentication middleware share the same error response callback. Reported storage failures return HTTP 500 with a fixed message, and internal details stay in server logs.

## Key APIs

```go
fail := func(c *gofiber.Ctx, err error) { _ = writeAuthError(c, err) }
app.Use(fiberdt.RegisterDTokenContextMiddleware(ctx, fiberdt.WithFailFunc(fail)))
auth.Use(fiberdt.AuthMiddleware(ctx, fiberdt.WithFailFunc(fail)))
auth.Get("/admin", fiberdt.RoleMiddleware(ctx, []string{"admin"}, fiberdt.WithFailFunc(fail)), handleAdmin)
auth.Get("/articles", fiberdt.PermissionMiddleware(ctx, []string{"article:read"}, fiberdt.WithFailFunc(fail)), handleArticles)
```
