# Kratos DToken Example

This example shows how to use `github.com/Zany2/dtoken-go/integrations/kratos` with Kratos HTTP middleware.

## Run

```bash
cd examples/kratos
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: accepts a nonempty username and password `123456`, then grants the demo user the `admin` role and `article:read` permission.
- `GET /me`: returns current login information.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `kratosdt.NewBuilder()`, so no Redis service is required. Login state and authorization data are lost on restart. Tokens expire after two hours; automatic renewal is enabled with a one-hour remaining-lifetime threshold. Managers are closed when the application exits.

This is a demonstration credential check. Real applications must verify credentials against their own user data and load the user's actual roles and permissions. By default, repeated logins for the same user and device share a token; logging out that token also invalidates other clients using it.

## Authentication and errors

Send the token in the `dtoken` or `Authorization` header, either as the raw token or with the `Bearer ` prefix. Query, Cookie, and Body token extraction are disabled by default. The login JSON body contains credentials, not an authentication token.

The routes above use a common JSON envelope (`code`, `message`, and optional `data`) for handler and middleware responses:

- HTTP 400: malformed login input or missing credentials.
- HTTP 401: incorrect demo password, missing token, or inactive/invalid token.
- HTTP 403: account/device restrictions or insufficient roles/permissions.
- HTTP 500: unavailable Manager, storage failures, or other internal errors. Detailed errors are logged server-side; responses contain a fixed message.

`wrapHandler` applies the registered context middleware and route-specific authentication middleware, then converts their errors into this envelope. Custom routes should use the same wrapper to retain this behavior.

## Proto scaffolding

The `api/` and `third_party/` folders are retained for later proto-based expansion. The generated service is not registered by this example: `/api/login` and `/api/user/info` are not exposed, and no gRPC server is started. Adding those services requires explicit registration and authentication/authorization rules for their endpoints.

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
srv := khttp.NewServer(
	khttp.Address(":8080"),
	khttp.Middleware(kratosdt.RegisterDTokenContextMiddleware()),
)

r.GET("/admin", wrapHandler(handleAdmin, kratosdt.AuthMiddleware(), kratosdt.RoleMiddleware([]string{"admin"})))
r.GET("/articles", wrapHandler(handleArticles, kratosdt.AuthMiddleware(), kratosdt.PermissionMiddleware([]string{"article:read"})))
```
