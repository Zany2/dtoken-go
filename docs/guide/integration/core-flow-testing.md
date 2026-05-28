# Core Flow Testing Guide

[中文文档](../integration/core-flow-testing_zh.md) | English

## Overview

`tests/gin_core_flow` is a real HTTP flow test suite for core framework behavior. It builds a Gin demo app and starts an in-process temporary HTTP server through `httptest.NewServer`.

It does not start:

```text
tests/gin_core_app/cmd/server/main.go
```

Instead it calls:

```go
gincoreapp.NewApp(cfg)
httptest.NewServer(app.Router())
```

## Running

From the repository root:

```powershell
go test ./tests/gin_core_flow -v
```

If your Windows shell has a non-Windows target configured:

```powershell
$env:GOOS='windows'
$env:GOARCH='amd64'
go clean -cache -testcache
go test ./tests/gin_core_flow -v
```

## Storage

The current `gin_core_flow` suite uses Redis by default:

```text
redis://:root@192.168.19.104:6379/0
```

Each test app gets a short isolated key prefix:

```text
dt:gcf:1:
dt:gcf:2:
```

On cleanup, the test only removes keys under the current prefix:

```text
dt:gcf:1:*
```

It does not clear the whole Redis DB.

## Coverage

The suite covers:

- login, logout, token status, and token metadata
- permissions, roles, wildcards, AND/OR logic, and AccessProvider
- manual renew, auto-renew, expiration, and active timeout
- session, terminals, multi-terminal login, terminal queries, and search
- logout, kickout, and replace
- concurrent login, token sharing, max login count, account scope, and device scope
- account, service, device, and concrete-device disable and untie
- nonce generation, verification, consumption, and expiration
- OAuth2 authorization code, password, client credentials, refresh, revoke, and client management
- multi-auth isolation
- core event dispatch
- all built-in TokenStyle values

## Common Questions

### Why are there still keys in Redis?

The test only deletes keys under the current `dt:gcf:*` prefix. It does not delete other prefixes, for example:

```text
dtoken:gin-core-flow:oauth2:client:demo-client
```

That kind of key usually comes from manually running the example server or from older test runs.

### Why does auto-renew TTL drift?

Redis returns TTL values in seconds, while memory storage can behave slightly differently around timing boundaries. The tests validate a reasonable TTL range instead of exact millisecond timing.

### When should I start gin_core_app manually?

Only start it when you want to manually call APIs through Postman, a browser, or another client:

```powershell
go run ./tests/gin_core_app/cmd/server
```

Automated flow tests do not need a separately started server.

## Related Documentation

- [Redis Storage](../integration/redis-storage.md)
- [Authentication](../core/authentication.md)
- [Concurrent Login Policy](../core/concurrency-login.md)
- [Session And Terminal Management](../core/session-terminal.md)
