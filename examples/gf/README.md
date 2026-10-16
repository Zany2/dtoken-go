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
- `POST /refresh`: rotates both tokens and returns a new token pair; the old access and refresh tokens become invalid.
- `GET /me`: returns current login information.
- `GET /introspect`: returns current token introspection.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current access token and revokes its associated refresh token.
- `GET /access/public`: skips authentication through route access rules.
- `GET /access/me`: requires login only through route access rules.
- `GET /access/articles`: requires `article:read` through route access rules.
- `GET /access/admin`: requires `admin` through route access rules.

The example uses bundled memory storage through `gfdt.NewBuilder()`, so no Redis service is required. Login state and authorization data are lost on restart. Access tokens last two hours, automatic renewal uses a one-hour threshold, and refresh tokens last 30 days. Registered Managers are closed when `main` returns.

This is a demonstration: any nonempty username can log in with password `123456`, and every successful login receives the `admin` role and `article:read` permission. A real application must verify credentials and load each user's own authorization rules.

Login and refresh credentials must be strings in the request body, using `application/json`, `application/x-www-form-urlencoded`, or `multipart/form-data`. Query parameters are not used as credentials. Malformed bodies and missing fields return 400, invalid credentials return 401, account/device restrictions and insufficient permissions return 403, and backend failures return 500 without exposing internal error details.

Protected routes read access tokens from the `dtoken` header, falling back to `Authorization`; both raw tokens and `Bearer <access-token>` are supported. Cookie, Query, and Body access-token extraction are disabled by default. The JSON/form fields on `/login` and `/refresh` are separate from that access-token extraction policy.

The `/access` rules use GoFrame's matched handler route. Repeated slashes or an `X-Url-Path` routing override therefore retain the target handler's login, role, and permission checks.

## Try

```bash
curl -X POST http://localhost:8080/login \
  -d "username=admin&password=123456"

curl http://localhost:8080/me \
  -H "Authorization: Bearer <access-token>"

curl -X POST http://localhost:8080/refresh \
  -d "refreshToken=<refresh-token>"

# Save both tokens from the refresh response; use the new access token below.
curl http://localhost:8080/introspect \
  -H "Authorization: Bearer <new-access-token>"

curl http://localhost:8080/access/articles \
  -H "Authorization: Bearer <new-access-token>"

curl -X POST http://localhost:8080/logout \
  -H "Authorization: Bearer <new-access-token>"
```

After logout, both the new access token and its associated refresh token are invalid.

## Key APIs

```go
s.Use(gfdt.RegisterDTokenContextMiddleware(ctx, gfdt.WithFailFunc(handleAuthFail)))

group.Middleware(gfdt.AuthMiddleware(ctx, gfdt.WithFailFunc(handleAuthFail)))
group.GET("/admin", gfdt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdmin, handleAuthFail))
group.GET("/articles", gfdt.CheckPermissionMiddleware(ctx, []string{"article:read"}, handleArticles, handleAuthFail))

group.Middleware(gfdt.AccessMiddleware(ctx,
	gfdt.WithRouteAccessHandler(resolveRouteAccess),
	gfdt.WithFailFunc(handleAuthFail),
))

pair, err := gfdt.LoginWithRefreshToken(ctx, loginID, "web", "gf-example")
pair, err = gfdt.RefreshToken(ctx, refreshToken)
info, err := gfdt.IntrospectTokenByCtx(ctx)
```
