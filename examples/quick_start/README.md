# DToken Quick Start

This example shows the core `dtoken` facade without a framework-specific integration package. It uses Gin only as a small HTTP wrapper and calls `github.com/Zany2/dtoken-go` directly.

## Run

```bash
cd examples/quick_start
go run .
```

The server listens on `http://localhost:8080`.

## Endpoints

- `POST /login`: logs in a demo user. Password must be `123456`.
- `GET /me`: returns current login information.
- `GET /admin`: requires the `admin` role.
- `GET /articles`: requires the `article:read` permission.
- `POST /logout`: logs out the current token.

The example uses bundled memory storage through `dtoken.NewBuilder()`, so no Redis service is required.

Tokens have a two-hour timeout and a one-hour automatic-renewal threshold. Login requires a single JSON object; additional values or trailing non-whitespace content are rejected before login state changes.

Any non-empty username can log in with the fixed demo password. Every successful demo login receives the `admin` role and `article:read` permission; this demonstrates the DToken APIs rather than a real account database. In an application, verify the account credentials and load its own access rules before granting access. Restarting this example clears its in-memory login state.

Protected routes read the raw token from `Authorization: <token>`; this example does not parse the `Bearer` prefix or read tokens from cookies, query parameters, or request bodies. The custom middleware uses each HTTP request's context. Missing or invalid login credentials return HTTP 401; a logged-in user without the required role or permission receives HTTP 403.

Account/device restrictions also return HTTP 403. Backend and Manager failures return HTTP 500 with a fixed message; internal errors are recorded in Gin's request error log. Build automatically registers the default Auth, and handlers share that returned instance. The application handles Ctrl+C and SIGTERM: it stops accepting requests, waits for active HTTP handlers, then calls `dtoken.CloseAll()`. The authentication library does not install process signal handlers. Shutdown waits for handlers without a fixed timeout; forced process termination cannot guarantee cleanup.

## Try

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'

curl http://localhost:8080/me \
  -H "Authorization: <token>"

curl http://localhost:8080/admin \
  -H "Authorization: <token>"

curl http://localhost:8080/articles \
  -H "Authorization: <token>"

curl -X POST http://localhost:8080/logout \
  -H "Authorization: <token>"
```

## Key APIs

```go
auth, err := dtoken.NewBuilder().
    TokenName("Authorization").
    Timeout(7200).
    RenewMaxRefresh(3600).
    Build()
if err != nil { panic(err) }
defer auth.Close()

token, err := auth.Login(ctx, dtoken.LoginOptions{LoginID: loginID})
err = auth.CheckLogin(ctx, token)
loginID, err = auth.GetLoginID(ctx, token)
err = auth.Logout(ctx, dtoken.LogoutOptions{Token: token})
```

Build also makes the same engine available through `dtoken.CheckLogin(ctx, token)`. For named systems, build with `.AuthType("user")` and call `dtoken.AuthType("user").CheckLogin(ctx, token)`. Use `BuildInstance()` when constructing unregistered, independent instances.
