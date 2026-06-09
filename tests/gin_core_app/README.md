# Gin Core App

`tests/gin_core_app` is a reusable Gin test fixture for the core DToken flow tests. It exposes real HTTP routes around the framework-agnostic `dtoken` APIs so tests can exercise authentication, authorization, session, terminal, disable, nonce, OAuth2, and multi-auth behavior through HTTP.

## Storage

`NewApp` uses in-memory storage when `Config.RedisURL` is empty. Pass `Config.RedisURL` to use Redis storage instead.

The command-line server in `cmd/server` uses `DTOKEN_REDIS_URL` when it is set. If the environment variable is empty, it falls back to:

```text
redis://:root@192.168.19.104:6379/0
```

## Run Manually

```powershell
go run ./tests/gin_core_app/cmd/server
```

The manual server listens on `http://localhost:8088`.

## Common Routes

- `GET /health`
- `POST /login`
- `POST /login/timeout`
- `GET /token/status`
- `GET /api/me`
- `POST /api/logout`
- `GET /api/token/info`
- `GET /api/session`
- `GET /api/terminal`
- `POST /api/permissions`
- `POST /api/roles`
- `POST /api/disable/account`
- `GET /nonce`
- `POST /nonce/verify`
- `POST /oauth2/authorize`
- `POST /oauth2/token`
- `GET /oauth2/introspect`
- `POST /multi-auth/user/login`
- `POST /multi-auth/admin/login`

For the full flow coverage, see `tests/gin_core_flow`.
