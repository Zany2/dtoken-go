# 快速开始

[English](quick-start.md) | 中文文档

应用通过根包构建 Auth 并显式注入 HTTP 中间件。实例调用无需全局注册；Nonce、OAuth2、Ticket 和 ShortKey 按需创建独立服务。

## 安装

根模块要求 Go 1.25+；仓库 go.work 开发环境要求 Go 1.27+。先添加根模块，使用 Gin 时再添加对应适配模块：

```bash
go get github.com/Zany2/dtoken-go
go get github.com/Zany2/dtoken-go/integrations/gin
```

## 创建实例并调用

默认 Builder 已包含内存存储及基础组件。下面是可独立保存为 main.go 的示例；实际应用先验证账号密码等身份凭据，再调用 LoginID。

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

## 配置实例

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

Build 返回 *dtoken.Auth。Timeout、ActiveTimeout 使用秒数；WithAuth 绑定哪个实例，请求就使用哪个实例。默认内存存储不跨进程共享，需要共享数据时配置 Redis 或其他存储实现。运行期间不要修改底层 Manager 配置。

## 接入 Gin

以下片段放在 Gin 路由注册之前，假设 auth 是上面构建的实例，router 是 gin.Engine。Context 绑定请求实例，RequireLogin 对受保护路由校验登录。

```go
import gindt "github.com/Zany2/dtoken-go/integrations/gin"

router.Use(gindt.Context(auth))
users := router.Group("/user")
users.Use(gindt.RequireLogin())
```

将 Token 放入 Authorization 请求头。登录接口应注册在受保护路由组之外。完整服务与身份验证示例见根 README 和 examples/quick_start。退出时先排空 HTTP 请求，再关闭 Auth。

## 可选全局入口

需要全局调用时，显式导入 global 并注册已有实例。注册表共享引擎；删除或替换注册项会关闭对应引擎，不能作为与 Auth 相互独立的副本使用。

```go
import "github.com/Zany2/dtoken-go/global"

global.SetAuth(auth)
defer global.DeleteManager(auth.Config().AuthType)
token, err := global.Login(ctx, "1000", "", "", auth.Config().AuthType)
```

HTTP 若通过全局注册表选择实例，还需显式配置 WithManagerResolver(global.GetManager)。直接 WithAuth(auth) 注入无需注册。

## 继续阅读

- [Auth 实例 API](../api/auth_zh.md)
- [框架集成](../guide/integration/framework-integration_zh.md)
- [配置指南](../guide/reference/configuration_zh.md)
- [独立能力](../guide/security/advanced-features_zh.md)
- [资源生命周期](../design/api-lifecycle_zh.md)
- [Quick Start](../../examples/quick_start/)
- [Gin](../../examples/gin/)
