# Gin Core App

`tests/gin_core_app` is a reusable Gin test fixture for the core DToken flow tests. It exposes real HTTP routes around the framework-agnostic `dtoken` APIs so tests can exercise authentication, authorization, session, terminal, disable, nonce, OAuth2, and multi-auth behavior through HTTP.

This fixture is for local testing only. It deliberately uses fixed demo credentials and exposes operator, OAuth2 authorization, and client management routes without production access controls. Do not expose its router to untrusted networks or reuse these handlers as production authentication endpoints.

## Storage

`NewApp` uses in-memory storage when `Config.RedisURL` is empty. Pass `Config.RedisURL` to use Redis storage instead.

The command-line server in `cmd/server` uses `DTOKEN_REDIS_URL` when it is set. If the environment variable is empty, it uses in-memory storage.

## Run Manually

```powershell
go run ./tests/gin_core_app/cmd/server
```

The manual server binds to loopback at `http://127.0.0.1:8088`. `NewApp` leaves the process-wide Gin mode unchanged; callers can select their mode before creating the app.

## Request And Error Contract

- JSON endpoints accept exactly one JSON object. Trailing whitespace is allowed; malformed bodies, trailing values, and `null` are rejected before the handler changes state.
- Account, service, and device disable routes allow an empty body when only an optional reason is needed. The service-level route requires a positive `level`.
- Custom login, renewal, and nonce lifetimes must be positive seconds that fit in `time.Duration`. `Config.TokenTimeout` defaults to 30 seconds only when zero; negative values are invalid.
- Missing disable records return HTTP 404; invalid OAuth2 client or redirect bindings return HTTP 400. Storage and access-provider failures return HTTP 500 with a generic message.

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
