# DToken Quick Start

This example shows the core `dtoken` facade without a framework-specific integration package. It uses Gin only as a small HTTP wrapper and calls `github.com/Zany2/dtoken-go/dtoken` directly.

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

The example uses bundled memory storage through `defaults.NewBuilder()`, so no Redis service is required.

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
mgr, err := defaults.NewBuilder().
	TokenName("Authorization").
	Timeout(7200).
	Build()
if err != nil {
	panic(err)
}
dtoken.SetManager(mgr)

token, err := dtoken.Login(ctx, loginID)
err = dtoken.CheckLogin(ctx, token)
loginID, err = dtoken.GetLoginID(ctx, token)
err = dtoken.Logout(ctx, token)
```
