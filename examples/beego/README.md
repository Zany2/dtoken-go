# DToken Beego Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/beego` with Beego filters.

## Run

```bash
cd examples/beego
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: logs in a demo user. Password must be `123456`.
- `GET /me`: returns current login information.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `beegodt.NewBuilder()`, so no Redis service is required.

## Try

```bash
curl -X POST "http://localhost:8080/login?username=admin&password=123456"
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
web.InsertFilter("/*", web.BeforeRouter, beegodt.RegisterDTokenContextMiddleware(ctx))
web.InsertFilter("/me", web.BeforeRouter, beegodt.AuthMiddleware(ctx))
web.InsertFilter("/admin", web.BeforeRouter, beegodt.RoleMiddleware(ctx, []string{"admin"}))
web.InsertFilter("/articles", web.BeforeRouter, beegodt.PermissionMiddleware(ctx, []string{"article:read"}))
```
