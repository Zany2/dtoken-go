# Gin DToken Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/gin` with Gin middleware.

## Run

```bash
cd examples/gin
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: logs in a demo user and returns an access token plus refresh token. Password must be `123456`.
- `POST /refresh`: rotates a refresh token and returns a new token pair.
- `GET /me`: returns current login information.
- `GET /introspect`: returns current token introspection.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `gindt.NewBuilder()`, so no Redis service is required.

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

curl http://localhost:8080/introspect \
  -H "Authorization: Bearer <access-token>"

curl http://localhost:8080/admin \
  -H "Authorization: Bearer <access-token>"

curl http://localhost:8080/articles \
  -H "Authorization: Bearer <access-token>"

curl -X POST http://localhost:8080/logout \
  -H "Authorization: Bearer <access-token>"
```

## Key APIs

```go
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

auth := r.Group("/")
auth.Use(gindt.AuthMiddleware(ctx))
auth.GET("/admin", gindt.RoleMiddleware(ctx, []string{"admin"}), handleAdmin)
auth.GET("/articles", gindt.PermissionMiddleware(ctx, []string{"article:read"}), handleArticles)

pair, err := gindt.LoginWithRefreshToken(ctx, loginID, "web", "gin-example")
pair, err = gindt.RefreshToken(ctx, refreshToken)
info, err := gindt.IntrospectTokenByContext(c)
```
