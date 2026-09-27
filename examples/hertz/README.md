# Hertz DToken Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/hertz` with Hertz middleware.

## Run

```bash
cd examples/hertz
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: accepts JSON credentials with any nonempty demo username and password `123456`, grants the `admin` role and `article:read` permission, and returns a token.
- `GET /me`: returns current login information.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `hertzdt.NewBuilder()`, so no Redis service is required. Login and authorization data are lost on restart. Demo credentials and permission grants illustrate the API; real applications must verify their own accounts and load their own authorization rules.

Tokens have a two-hour timeout, with automatic renewal during authentication checks when one hour or less remains. Repeated logins for the same username may reuse a token under the default sharing policy; logging out invalidates that shared token.

Protected routes read access tokens from the configured `dtoken` header, falling back to `Authorization`. Both raw tokens and `Bearer <token>` are supported. Cookie, query, and body access-token extraction are disabled by default; `/login` separately accepts credentials in its JSON request body.

Malformed or incomplete login payloads return HTTP `400`, invalid credentials return `401`, account/device restrictions and insufficient permissions return `403`, and server failures return `500`. Internal error details are logged instead of being included in responses.

Registered Managers are released after Hertz's `Spin` returns, including shutdown and startup-error paths.

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

## Key APIs

```go
h.Use(hertzdt.RegisterDTokenContextMiddleware(ctx, hertzdt.WithFailFunc(writeAuthError)))
auth.Use(hertzdt.AuthMiddleware(ctx, hertzdt.WithFailFunc(writeAuthError)))
auth.GET("/admin", hertzdt.RoleMiddleware(ctx, []string{"admin"}, hertzdt.WithFailFunc(writeAuthError)), handleAdmin)
auth.GET("/articles", hertzdt.PermissionMiddleware(ctx, []string{"article:read"}, hertzdt.WithFailFunc(writeAuthError)), handleArticles)
```
