# Hertz DToken Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/hertz` with Hertz middleware.

## Run

```bash
cd examples/hertz
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: logs in a demo user. Password must be `123456`.
- `GET /me`: returns current login information.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `hertzdt.NewBuilder()`, so no Redis service is required.

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
h.Use(hertzdt.RegisterDTokenContextMiddleware(ctx))
auth.Use(hertzdt.AuthMiddleware(ctx))
auth.GET("/admin", hertzdt.RoleMiddleware(ctx, []string{"admin"}), handleAdmin)
auth.GET("/articles", hertzdt.PermissionMiddleware(ctx, []string{"article:read"}), handleArticles)
```
