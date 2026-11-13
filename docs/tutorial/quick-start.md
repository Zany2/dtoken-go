# Quick Start

[中文文档](quick-start_zh.md) | English

## Get Started with DToken-Go in 5 Minutes

This page is based on the current version of the codebase and walks through the minimum steps for installation, initialization, and basic usage.

## Step 1: Install

### Option 1: Root Module with Default Memory Storage

```bash
go get github.com/Zany2/dtoken-go
```

### Option 2: Use a Framework Integration Package Directly

If you are already using a web framework, you can also import an integration package directly:

```bash
go get github.com/Zany2/dtoken-go/integrations/gin
```

## Step 2: Initialize

<!-- dtoken-compile: quick-start -->
```go
package main

import (
    dt "github.com/Zany2/dtoken-go"
    dtoken "github.com/Zany2/dtoken-go/global"
    "context"
)

var ctx = context.Background()

func initDToken() *dtoken.Auth {
    auth, err := dt.NewBuilder().
        TokenName("Authorization").
        Timeout(86400).
        Build()
    if err != nil {
        panic(err)
    }
    dtoken.SetAuth(auth)
    return auth
}
```

### Initialization Notes

- `dt.NewBuilder().Build()` creates an authentication instance using memory storage by default.
- `Storage(...)` can replace the storage implementation.
- `dtoken.SetAuth(auth)` registers the instance for the global APIs used below.
- `auth.Close()` unregisters the instance and releases manager-owned resources.

## Step 3: Use It

<!-- dtoken-compile: quick-start -->
```go
func main() {
    auth := initDToken()
    defer auth.Close()
    defer dtoken.DeleteAllManager()

    // Login
    token, _ := dtoken.Login(ctx, "1000")

    // Check login state
    isLogin := dtoken.IsLogin(ctx, token)

    // Add permissions
    _ = dtoken.AddPermissions(ctx, "1000", []string{"user:read"})

    // Check permission
    hasPermission := dtoken.HasPermission(ctx, "1000", "user:read")

    // Logout
    _ = dtoken.Logout(ctx, token)

    _, _, _ = token, isLogin, hasPermission
}
```

## Step 4: Common Configuration

You can continue adjusting common settings through the Builder:

```go
auth, err := dt.NewBuilder().
    TokenName("token").
    Timeout(7200).
    ActiveTimeout(1800).
    AutoRenew(true).
    IsReadHeader(true).
    IsPrintBanner(true).
    Build()
if err != nil {
    panic(err)
}

defer auth.Close()
    defer dtoken.DeleteAllManager()
dtoken.SetAuth(auth)
```

Common config meanings:

- `TokenName` - token name
- `Timeout` - absolute token timeout
- `ActiveTimeout` - maximum inactive duration
- `AutoRenew` - whether auto-renew is enabled
- `IsReadHeader` - whether to read token from HTTP header
- `IsPrintBanner` - whether to print the startup banner

## Step 5: Read Complete Examples

If you want to see more complete examples, you can directly refer to:

- [Quick Start example](../../examples/quick_start/)
- [Gin example](../../examples/gin/)
- [GoFrame example](../../examples/gf/)
- [Beego example](../../examples/beego/)
- [Echo example](../../examples/echo/)
- [Fiber example](../../examples/fiber/)
- [Chi example](../../examples/chi/)
- [Hertz example](../../examples/hertz/)
- [Kratos example](../../examples/kratos/)

After these steps, you already understand the most basic usage of the current DToken-Go version.

## Next Steps

- [Authentication Guide](../guide/core/authentication.md)
- [Permission Guide](../guide/core/permission.md)
- [Annotation Guide](../guide/integration/annotation.md)
- [Framework Integration Guide](../guide/integration/framework-integration.md)
