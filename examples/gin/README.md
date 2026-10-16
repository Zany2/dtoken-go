# Gin DToken Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/gin` with Gin middleware.

## Run

```bash
cd examples/gin
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: accepts any nonempty demo username with password `123456`, grants the `admin` role and `article:read` permission, and returns an access token plus refresh token.
- `POST /refresh`: rotates a refresh token, returns a new token pair, and invalidates the old pair.
- `GET /me`: returns current login information.
- `GET /introspect`: returns current token introspection.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current access token and revokes its associated refresh token.

The example uses bundled memory storage through `gindt.NewBuilder()`, so no Redis service is required. Login and authorization data are lost on restart. Demo credentials and permission grants are for illustrating the API; real applications must verify their own accounts and load their own authorization rules.

Access tokens have a two-hour timeout, with automatic renewal during authentication checks when one hour or less remains. Refresh tokens have a 30-day timeout. The example reads tokens from the `Authorization` header and accepts both `Bearer <access-token>` and the raw access token. Cookie, query, and body access-token extraction are disabled; `/refresh` separately reads `refreshToken` from its JSON payload.

Credential failures return HTTP `401`, account or device restrictions and insufficient permissions return `403`, and server failures return `500`. Managers are released when `main` returns or unwinds after a startup error.

Login and refresh bodies must contain one JSON object. Invalid trailing input is rejected with HTTP 400 before issuing or rotating tokens.

## Try

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'

curl http://localhost:8080/me \
  -H "Authorization: Bearer <access-token>"

curl -X POST http://localhost:8080/refresh \
  -H "Content-Type: application/json" \
  -d '{"refreshToken":"<refresh-token>"}'

# Save both tokens from the refresh response; the previous pair is no longer usable.
curl http://localhost:8080/introspect \
  -H "Authorization: Bearer <new-access-token>"

curl http://localhost:8080/admin \
  -H "Authorization: Bearer <new-access-token>"

curl http://localhost:8080/articles \
  -H "Authorization: Bearer <new-access-token>"

curl -X POST http://localhost:8080/logout \
  -H "Authorization: Bearer <new-access-token>"
```

Handlers and authentication middleware share the same error response callback. Reported storage failures return HTTP 500 with a fixed message, and internal details stay in server logs.

## Key APIs

```go
r.Use(gindt.RegisterDTokenContextMiddleware(ctx, gindt.WithFailFunc(writeAuthError)))

auth := r.Group("/")
auth.Use(gindt.AuthMiddleware(ctx, gindt.WithFailFunc(writeAuthError)))
auth.GET("/admin", gindt.RoleMiddleware(ctx, []string{"admin"}, gindt.WithFailFunc(writeAuthError)), handleAdmin)
auth.GET("/articles", gindt.PermissionMiddleware(ctx, []string{"article:read"}, gindt.WithFailFunc(writeAuthError)), handleArticles)

pair, err := gindt.LoginWithRefreshToken(ctx, loginID, "web", "gin-example")
pair, err = gindt.RefreshToken(ctx, refreshToken)
info, err := gindt.IntrospectTokenByContext(c)
```
