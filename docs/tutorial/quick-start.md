# Quick Start

[中文文档](quick-start_zh.md) | English

Build Auth through the root package and explicitly inject it into HTTP middleware. Build automatically registers and returns Auth; BuildInstance creates an unregistered instance. Compose Nonce, OAuth2, Ticket, and ShortKey as independent services when needed.

## Installation

The root module requires Go 1.25+; repository go.work development requires Go 1.27+. Add the root module, then the adapter if using Gin:

```bash
go get github.com/Zany2/dtoken-go
go get github.com/Zany2/dtoken-go/integrations/gin
```

## Build And Use An Instance

The default builder includes memory storage and basic components. Save the following as main.go. Applications verify passwords or other identity credentials before calling LoginID.

<!-- dtoken-compile: quick-start -->
```go
package main

import (
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go"
)

func main() {
    auth, err := dtoken.NewBuilder().
        TokenName("Authorization").
        Timeout(86400).
        Build()
    if err != nil {
        panic(err)
    }
    defer auth.Close()

    ctx := context.Background()
    token, err := auth.LoginID(ctx, "1000")
    if err != nil {
        panic(err)
    }
    if err := auth.AddPermissions(ctx, dtoken.PermissionOptions{
        LoginID: "1000", Permissions: []string{"user:read"},
    }); err != nil {
        panic(err)
    }
    fmt.Println(auth.IsLogin(ctx, token))
    fmt.Println(auth.HasPermission(ctx, "1000", "user:read"))
    if err := auth.LogoutByToken(ctx, token); err != nil {
        panic(err)
    }
}
```

## Configure The Instance

```go
auth, err := dtoken.NewBuilder().
    AuthType("user").
    TokenName("Authorization").
    Timeout(7200).
    ActiveTimeout(1800).
    AutoRenew(true).
    IsReadHeader(true).
    IsPrintBanner(false).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

Build automatically registers and returns *dtoken.Auth; duplicate names are rejected. Timeout and ActiveTimeout use seconds. WithAuth selects the instance used by requests. Default memory storage is process-local; use Redis or another storage implementation for shared data. Do not mutate the underlying Manager configuration while serving requests.

## Connect Gin

Place this fragment before registering Gin routes. It assumes auth is the built instance and router is a gin.Engine. Context binds the request instance; RequireLogin validates protected routes.

```go
import gindt "github.com/Zany2/dtoken-go/integrations/gin"

router.Use(gindt.Context(auth))
users := router.Group("/user")
users.Use(gindt.RequireLogin())
```

Send the token in the Authorization header. Register login outside the protected group. See the root README and examples/quick_start for complete server and credential-validation examples. Drain HTTP requests before closing Auth.

## Default And Named Namespace Calls

Build needs no SetAuth. Without a configured AuthType, call `dtoken.LoginID(ctx, "1000")` or `dtoken.CheckLogin(ctx, token)` directly. For the user namespace configured above:

```go
user := dtoken.AuthType("user")
token, err := user.Login(ctx, dtoken.LoginOptions{LoginID: "1000"})
if err != nil {
    panic(err)
}
err = user.CheckLogin(ctx, token)
```

The registry shares the engine returned as Auth. Missing namespaces never fall back; duplicate Build does not replace an instance. Use BuildInstance for isolated instances and close them through their owner. For explicit replacement, use BuildInstance followed by SetAuth.

HTTP registry lookup requires `WithManagerResolver(dtoken.GetManager)`; WithAuth binds a fixed engine. Root calls and Auth methods share typed option signatures.

On shutdown, stop accepting and drain requests before dtoken.CloseAll. It excludes unregistered instances. The library does not install signal handlers; see the [application lifecycle example](../design/api-lifecycle.md).

## Next Steps

- [Auth instance API](../api/auth.md)
- [Framework integration](../guide/integration/framework-integration.md)
- [Configuration](../guide/reference/configuration.md)
- [Independent capabilities](../guide/security/advanced-features.md)
- [Resource lifecycle](../design/api-lifecycle.md)
- [Quick Start](../../examples/quick_start/)
- [Gin](../../examples/gin/)
