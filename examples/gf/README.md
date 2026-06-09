# GoFrame DToken Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/gf` with GoFrame middleware.

## Run

```bash
cd examples/gf
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
- `GET /access/public`: skips authentication through route access rules.
- `GET /access/me`: requires login only through route access rules.
- `GET /access/articles`: requires `article:read` through route access rules.
- `GET /access/admin`: requires `admin` through route access rules.

The example uses bundled memory storage through `gfdt.NewBuilder()`, so no Redis service is required.

## Try

```bash
curl -X POST http://localhost:8080/login \
  -d "username=admin&password=123456"

curl http://localhost:8080/me \
  -H "Authorization: Bearer <access-token>"

curl -X POST http://localhost:8080/refresh \
  -d "refreshToken=<refresh-token>"

curl http://localhost:8080/introspect \
  -H "Authorization: Bearer <access-token>"

curl http://localhost:8080/access/articles \
  -H "Authorization: Bearer <access-token>"
```

## Key APIs

```go
s.Use(gfdt.RegisterDTokenContextMiddleware(ctx))

group.Middleware(gfdt.AuthMiddleware(ctx))
group.GET("/admin", gfdt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdmin, nil))
group.GET("/articles", gfdt.CheckPermissionMiddleware(ctx, []string{"article:read"}, handleArticles, nil))

group.Middleware(gfdt.AccessMiddleware(ctx,
	gfdt.WithRouteAccessHandler(resolveRouteAccess),
	gfdt.WithFailFunc(handleAuthFail),
))

pair, err := gfdt.LoginWithRefreshToken(ctx, loginID, "web", "gf-example")
pair, err = gfdt.RefreshToken(ctx, refreshToken)
info, err := gfdt.IntrospectTokenByCtx(ctx)
```
