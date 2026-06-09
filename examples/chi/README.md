# Chi DToken Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/chi` with Chi middleware.

## Run

```bash
cd examples/chi
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: logs in a demo user. Password must be `123456`.
- `GET /me`: returns current login information.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `chidt.NewBuilder()`, so no Redis service is required.

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
r.Use(chidt.RegisterDTokenContextMiddleware())
auth.Use(chidt.AuthMiddleware())
auth.With(chidt.RoleMiddleware([]string{"admin"})).Get("/admin", handleAdmin)
auth.With(chidt.PermissionMiddleware([]string{"article:read"})).Get("/articles", handleArticles)
```
