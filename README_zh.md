<p align="center">
  <img src="docs/assets/logo.png" alt="DToken-Go" width="100" height="100">
</p>

<h1 align="center">DToken-Go</h1>

<p align="center">
  一个面向 Go 应用的认证、授权、会话管理与 Token 生命周期管理框架。
</p>

<p align="center">
  <a href="https://github.com/Zany2/dtoken-go"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-blue?style=flat-square" alt="License"></a>
  <a href="docs/README_zh.md"><img src="https://img.shields.io/badge/Docs-中文文档-brightgreen?style=flat-square" alt="Docs"></a>
  <a href="https://pkg.go.dev/github.com/Zany2/dtoken-go"><img src="https://img.shields.io/badge/pkg.go.dev-dtoken-007D9C?style=flat-square&logo=go" alt="pkg.go.dev"></a>
</p>

<p align="center">
  简体中文 | <a href="README.md">English</a>
</p>

---

## DToken-Go 是什么

DToken-Go 是一个模块化、可插拔的 Go 认证授权框架，已提供登录认证、Token 管理、Session 管理、终端管理、角色权限校验、账号与设备封禁、Nonce 防重放、OAuth2 服务端、事件监听、Ticket 临时凭证、短 Key 访问凭证、Token Introspection 和独立 Refresh Token 等核心能力，SSO 单点登录模块仍在开发中。框架支持插件化组件替换与自定义扩展，并适配主流 Go Web 框架，既可以作为独立认证核心使用，也可以快速接入现有业务项目。

你可以把它用于：

- 后台管理系统、用户中心、开放平台等需要统一认证授权的业务系统。
- Gin、Echo、Fiber、Chi、GoFrame、Hertz、Kratos、Beego 等 Go Web 项目的登录与权限接入。
- 微服务网关、统一认证中心、跨系统单点登录和统一登出场景。
- App、小程序、Web、多设备、多终端的登录态与会话管理。
- 扫码确认、一次性访问、临时授权、短链接凭证和第三方系统 Token 校验。

## 核心特性

| 能力 | 说明 |
| --- | --- |
| 登录认证 | 登录、续登、登出、登录态校验、Token 信息查询、TTL 查询、手动续期和自动续期 |
| Session 管理 | 按账号、Token、设备、设备 ID 查询和管理登录态 |
| 终端管理 | 多端登录、终端追踪、在线终端统计、终端清理、踢下线、顶下线 |
| 角色权限 | 角色和权限增删查、AND/OR 校验、Token 维度校验、通配符权限匹配 |
| 并发控制 | 支持同账号并发登录控制、共享 Token、最大在线终端数限制、账号级/设备级作用域 |
| 账号与设备封禁 | 账号封禁、服务封禁、设备封禁、解封、封禁原因和剩余封禁时间查询 |
| 多认证体系 | 支持通过 AuthType 隔离多套认证体系，Token、Session、权限和角色互不串用 |
| Nonce 防重放 | 一次性随机值生成、校验、消费，防止请求重放 |
| OAuth2 | 授权码、客户端凭证、密码模式、刷新令牌、Token 校验和撤销 |
| 事件系统 | 登录、登出、续期、权限、角色、封禁、解封等核心生命周期事件监听 |
| 可插拔组件 | 存储、编解码、日志、Token 生成器、协程池等组件可替换 |
| 多框架集成 | 为主流 Go Web 框架提供中间件、上下文适配和 API 导出 |
| SSO 单点登录（开发中） | 统一登录、票据交换、跨系统登录态共享、统一登出、应用维度管理 |
| Ticket 临时凭证 | Ticket 创建、校验、一次性消费、撤销、TTL 查询和状态识别 |
| 短 Key 访问凭证 | 生成随机短 Key，用于短链接访问、扫码确认、临时授权和系统间换票 |
| Token Introspection | 标准化查询 Token 是否有效、归属信息、TTL 和失效原因 |
| Refresh Token | 独立刷新令牌签发、刷新、撤销、过期、轮换和安全校验 |

## 安装

根模块要求 Go 1.25+；仓库工作区要求 Go 1.27+。可选 `codec/jsonv2` 需要 `GOEXPERIMENT=jsonv2`。

```bash
go get github.com/Zany2/dtoken-go
go get github.com/Zany2/dtoken-go/integrations/gin
```

框架集成、SSO、Redis 存储、MsgPack、JSON v2 和 GoFrame 日志各自使用独立模块。路径与版本升级见[迁移指南](docs/design/module-migration_zh.md)。

## 5 分钟上手

<!-- dtoken-compile: instance -->
```go
package main

import (
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go"
)

func main() {
    auth, err := dtoken.NewBuilder().
        AuthType("user").
        TokenName("Authorization").
        Timeout(7200).
        IsPrintBanner(false).
        Build()
    if err != nil {
        panic(err)
    }
    defer auth.Close()

    ctx := context.Background()
    token, err := auth.LoginID(ctx, "user-1001")
    if err != nil {
        panic(err)
    }
    _ = auth.AddRoles(ctx, dtoken.RoleOptions{LoginID: "user-1001", Roles: []string{"admin"}})
    fmt.Println(auth.IsLogin(ctx, token))
    _ = auth.LogoutByToken(ctx, token)
}
```

需要全局 API 时，显式导入 `github.com/Zany2/dtoken-go/global`，调用 `global.SetAuth(auth)`，并由 `global.DeleteManager` 或 `global.DeleteAllManager` 清理注册项。实例之间无需全局注册。

## Web 框架接入

根包构建实例，框架通过 `WithAuth` 或 `Context` 注入。

<!-- dtoken-compile: gin -->
```go
package main

import (
    "net/http"

    "github.com/Zany2/dtoken-go"
    gindt "github.com/Zany2/dtoken-go/integrations/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    auth, err := dtoken.NewBuilder().TokenName("Authorization").Build()
    if err != nil {
        panic(err)
    }
    defer auth.Close()

    router := gin.Default()
    router.Use(gindt.Context(auth))
    router.POST("/login", func(c *gin.Context) {
        token, err := auth.LoginID(c.Request.Context(), "user-1001")
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"token": token})
    })
    users := router.Group("/user")
    users.Use(gindt.RequireLogin())
    users.GET("/me", func(c *gin.Context) {
        loginID, err := gindt.GetLoginIDByContext(c)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"loginId": loginID})
    })
    if err := router.Run(":8080"); err != nil {
        panic(err)
    }
}
```

## 组合独立能力

Nonce、OAuth2、Ticket 和 ShortKey 通过各自包的构造器创建；`NewService` 可接收认证实例作为事件发布器。使用相同的配置命名空间和存储即可与认证实例组合，Builder 不再提供能力开关。请求上下文通过 `WithCapabilities` 显式接收服务；未注入的能力返回 `ErrModuleNotEnabled`。以下片段在注册路由前使用，需同时导入标准库 `context`。

```go
// Imports: nonce "github.com/Zany2/dtoken-go/nonce",
// requestcontext "github.com/Zany2/dtoken-go/httpauth/context".
cfg := auth.Config()
service := nonce.NewService(
    nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage()),
    auth,
)
options := []gindt.AuthOption{
    gindt.WithAuth(auth),
    gindt.WithCapabilities(requestcontext.Capabilities{Nonce: service}),
}
router.Use(gindt.RegisterDTokenContextMiddleware(context.Background(), options...))
```

## 包边界

| 包 | 职责 |
| --- | --- |
| `dtoken` | 实例门面与默认 Builder |
| `global` | 全局认证 API 与注册表 |
| `contract`、`config`、`errors`、`event` | 公共契约、配置、错误、事件 |
| `internal/engine` | 内部认证引擎，应用不直接导入 |
| `nonce`、`oauth2`、`ticket`、`shortkey` | 独立组合的能力与服务 |
| `httpauth`、`httpauth/context` | 共享 HTTP 鉴权与请求上下文 |
| `codec`、`generator`、`logging`、`pool`、`storage` | 可插拔组件实现 |
| `integrations/<framework>`、`sso` | 框架适配与独立 SSO 模块 |

## 文档与示例

### 文档

- [文档中心](docs/README_zh.md)
- [快速开始](docs/tutorial/quick-start_zh.md)
- [登录认证](docs/guide/core/authentication_zh.md)
- [权限管理](docs/guide/core/permission_zh.md)
- [多认证体系](docs/guide/core/multi-auth_zh.md)
- [封禁体系](docs/guide/core/disable_zh.md)
- [Token 风格](docs/guide/core/token-style_zh.md)
- [AccessProvider](docs/guide/core/access-provider_zh.md)
- [框架集成](docs/guide/integration/framework-integration_zh.md)
- [事件监听](docs/guide/core/listener_zh.md)
- [Nonce 防重放](docs/guide/security/nonce_zh.md)
- [JWT 集成](docs/guide/security/jwt_zh.md)
- [Redis 存储](docs/guide/integration/redis-storage_zh.md)
- [OAuth2](docs/guide/security/oauth2_zh.md)
- [SSO 单点登录](sso/README_zh.md)
- [SSO 测试说明](sso/TESTING_zh.md)
- [Refresh Token](docs/guide/security/refresh-token_zh.md)
- [API 参考](docs/api/dtoken_zh.md)

### 示例项目

| 示例 | 说明 |
| --- | --- |
| [examples/quick_start](examples/quick_start/) | 默认 Builder + 全局 API 最小使用方式 |
| [examples/gin](examples/gin/) | Gin 中间件、登录校验和角色校验 |
| [examples/sso_gin_server](examples/sso_gin_server/) | Gin SSO 统一登录中心示例 |
| [examples/sso_gin_client](examples/sso_gin_client/) | Gin SSO 子系统接入示例 |
| [examples/sso_server](examples/sso_server/) | 标准库 SSO 统一登录中心示例 |
| [examples/sso_client](examples/sso_client/) | 标准库 SSO 子系统接入示例 |
| [examples/echo](examples/echo/) | Echo 框架接入示例 |
| [examples/fiber](examples/fiber/) | Fiber 框架接入示例 |
| [examples/chi](examples/chi/) | Chi 框架接入示例 |
| [examples/gf](examples/gf/) | GoFrame 框架接入示例 |
| [examples/hertz](examples/hertz/) | Hertz 框架接入示例 |
| [examples/kratos](examples/kratos/) | Kratos 框架接入示例 |
| [examples/beego](examples/beego/) | Beego 框架接入示例 |

## 贡献

欢迎提交 Issue、Pull Request 或使用反馈。提交代码前请尽量保持改动聚焦，并遵循项目现有的模块边界和代码风格。

开发与验证流程见[贡献指南](.github/CONTRIBUTING.md)，漏洞报告方式见[安全报告说明](.github/SECURITY.md)。

## Star History

<picture>
  <source
    media="(prefers-color-scheme: dark)"
    srcset="https://api.star-history.com/svg?repos=Zany2/dtoken-go&type=Date&theme=dark"
  />
  <source
    media="(prefers-color-scheme: light)"
    srcset="https://api.star-history.com/svg?repos=Zany2/dtoken-go&type=Date"
  />
  <img
    alt="Star History Chart"
    src="https://api.star-history.com/svg?repos=Zany2/dtoken-go&type=Date"
  />
</picture>

## License

DToken-Go 使用 [Apache-2.0](LICENSE) 协议开源。
