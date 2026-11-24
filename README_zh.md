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

DToken-Go 是一个面向 Go 应用的认证与授权框架，提供登录校验、自动续期、角色权限、账号封禁和多端会话管理，支持多认证体系隔离、并发登录控制及终端下线。框架提供统一 API 和主流 Web 框架适配，支持按需替换存储、编解码、日志等组件，并可组合 Nonce、OAuth2、Ticket、ShortKey 及独立的 SSO 模块，满足单体应用以及跨系统登录的认证需求。

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

# 使用 Gin 时添加框架适配模块
go get github.com/Zany2/dtoken-go/integrations/gin
```

根模块包含 Auth、公共契约、内部引擎、基础组件、共享 HTTP 层，以及 Nonce、OAuth2、Ticket、ShortKey 能力包。能力包独立组合，但随根模块一起发布。

框架集成、Redis 存储、MsgPack、JSON v2、GoFrame 日志、SSO 和 SSO Redis 存储使用独立模块，按需添加。旧目录结构的包路径及部分 API 已变更，升级步骤和发布版本约定见[迁移指南](docs/design/module-migration_zh.md)。

## 5 分钟上手

默认 Builder 使用进程内存储并装配基础组件，无需启动外部服务。以下示例演示为已确认身份的用户签发 Token；账号密码或其他身份凭据的验证由应用完成。

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
    if err := auth.AddRoles(ctx, dtoken.RoleOptions{LoginID: "user-1001", Roles: []string{"admin"}}); err != nil {
        panic(err)
    }
    fmt.Println(auth.IsLogin(ctx, token))
    if err := auth.LogoutByToken(ctx, token); err != nil {
        panic(err)
    }
}
```

Build 自动注册并返回 Auth；BuildInstance 创建不注册的独立实例。默认内存存储不跨进程共享；多实例共享认证数据时，可配置 [Redis 存储](docs/guide/integration/redis-storage_zh.md)，或实现 `contract.Storage`。

### AuthType 构建与调用

构建时通过 `AuthType` 区分用户、管理员等认证体系。`Build()` 自动注册并返回 `*Auth`，无需再调用 `SetAuth`；`MustBuild()` 行为相同，失败时 panic。下面是独立示例，用户 Token 有效期为 1 天，管理员 Token 有效期为 30 分钟，`Timeout` 的单位为秒：

<!-- dtoken-compile: multi-auth -->
```go
package main

import (
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go"
)

func main() {
    defer dtoken.CloseAll()
    userAuth := dtoken.NewBuilder().
        AuthType("user").TokenName("user-token").Timeout(86400).
        IsPrintBanner(false).MustBuild()
    dtoken.NewBuilder().
        AuthType("admin").TokenName("admin-token").Timeout(1800).
        IsPrintBanner(false).MustBuild()

    ctx := context.Background()
    token, err := dtoken.AuthType("user").Login(ctx, dtoken.LoginOptions{
        LoginID: "10001",
        Device: "web",
    })
    if err != nil {
        panic(err)
    }
    if err := userAuth.CheckLogin(ctx, token); err != nil {
        panic(err)
    }
    fmt.Println(dtoken.AuthType("user").IsLogin(ctx, token))  // true
    fmt.Println(dtoken.AuthType("admin").IsLogin(ctx, token)) // false
}
```

| 调用方式 | 使用的认证体系 |
| --- | --- |
| `dtoken.CheckLogin(ctx, token)` | Builder 未设置 AuthType 时注册的默认体系 `auth:` |
| `dtoken.AuthType("user").CheckLogin(ctx, token)` | 当前注册的 user 体系 |
| `userAuth.CheckLogin(ctx, token)` | 构建返回的固定实例，与该体系的注册项共享引擎 |

上例只注册 user 和 admin，不包含默认体系。未注册的体系不会回退到默认体系，`CheckLogin` 返回 `ErrManagerNotFound`，`IsLogin` 返回 false。同名 `Build` 返回 `ErrManagerAlreadyRegistered`；需要不注册的独立实例时使用 `BuildInstance()`。根包业务函数与实例方法签名一致，通过调用入口选择体系，`LoginOptions.AuthType` 不改变这些入口的体系选择。

应用退出时先停止接收并排空请求，再调用 `dtoken.CloseAll()` 关闭当前注册实例；未注册实例单独调用 `auth.Close()`。旧 global 包共用这张注册表。更多用法见[多认证体系](docs/guide/core/multi-auth_zh.md)和 [API 与生命周期](docs/design/api-lifecycle_zh.md)。

## Web 框架接入

支持 Gin、Echo、Fiber、Chi、GoFrame、Hertz、Kratos 和 Beego。根包构建实例，框架通过 `WithAuth` 注入；下面的 Gin 示例使用 `Context(auth)` 绑定请求实例，再用 `RequireLogin()` 保护路由。

示例的 `/login` 固定使用演示账号，只展示 Token 签发。实际应用应先验证身份，再调用 `LoginID`。

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

请求 `/user/me` 时，将登录响应中的 Token 放入 `Authorization` 请求头。框架中间件通过绑定的实例完成鉴权；需要从全局注册表选择实例时，显式配置 `WithManagerResolver(dtoken.GetManager)`。更多接入方式见[框架集成指南](docs/guide/integration/framework-integration_zh.md)。

## 组合独立能力

Nonce、OAuth2、Ticket 和 ShortKey 通过各自包的构造器创建；`NewService` 可接收 Auth 作为事件发布器。能力服务复用实例的命名空间、存储及所需编解码器，Builder 不提供能力开关。HTTP 通过 `contract` 下的服务接口绑定能力，也支持自定义实现。

在上面的 Gin 示例中，补充以下导入，并用后面的绑定片段替换 `router.Use(gindt.Context(auth))`，放在所有路由注册之前：

```go
import (
    "context"

    requestcontext "github.com/Zany2/dtoken-go/httpauth/context"
    "github.com/Zany2/dtoken-go/nonce"
)
```

```go
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

请求现在可以通过绑定的 Nonce 服务执行操作；OAuth2、Ticket 和 ShortKey 按同样方式组合。未绑定能力时，返回错误的能力操作报告 `ErrModuleNotEnabled`，布尔检查返回 false。服务借用实例组件，使用期间应保持 Auth 可用。完整用法见 [Nonce 指南](docs/guide/security/nonce_zh.md)和[高级能力指南](docs/guide/security/advanced-features_zh.md)。

## 架构与目录

实例调用沿 `应用 → Auth → 认证引擎 → 组件接口` 执行；HTTP 调用沿 `框架适配器 → 公共 HTTP 层 → 引擎或能力服务接口` 执行。具体组件和能力服务由应用装配。

| 包 | 职责 |
| --- | --- |
| 根包 `dtoken` | 实例/默认 API、AuthType 选择器、共享注册表、Builder 与高级扩展类型 |
| `global` | 转发到根共享注册表的旧版兼容 API |
| `contract`、`config`、`errors`、`event` | 公共契约与共享模型、核心配置、错误、事件发布与监听分发 |
| `internal/engine` | 内部认证引擎，应用不直接导入 |
| `internal/builder`、`internal/defaults` | 通用组件装配与内置默认组件工厂 |
| `nonce`、`oauth2`、`ticket`、`shortkey` | 独立组合的能力与服务 |
| `httpauth`、`httpauth/context` | 共享 HTTP 鉴权与请求上下文 |
| `codec`、`generator`、`logging`、`pool`、`storage` | 可插拔组件实现 |
| `integrations/<framework>` | 各框架的中间件、请求适配与快捷操作 |
| `sso` | 独立 SSO 子系统及其协议、客户端和服务端 |
| `examples`、`tests`、`docs` | 接入示例、应用级验证与文档 |
| `.github` | CI、仓库检查脚本与协作配置 |

引擎依赖组件契约，HTTP 公共层依赖能力服务契约；它们不选择具体组件或隐式查找全局实例。包目录划分职责，`go.mod` 划分发布模块，`go.work` 组合本地开发工作区。

应用优先使用 Auth；自定义集成可从 `auth.Manager()` 获取受支持的高级 API。多个门面或中间件绑定同一引擎时共享生命周期：关闭 Auth 会使这些绑定不可用。应用退出时先停止接收请求并等待处理完成，再关闭 Auth；组件所有权及全局替换规则见生命周期文档。

维护者可先阅读[架构设计](docs/design/architecture_zh.md)，再通过[源码目录](docs/design/source-layout_zh.md)定位文件和放置改动。[模块化设计](docs/design/modular_zh.md)定义依赖与发布边界，[API 与生命周期](docs/design/api-lifecycle_zh.md)定义公开契约和资源所有权。

## 文档与示例

### 文档

- [文档中心](docs/README_zh.md)
- [快速开始](docs/tutorial/quick-start_zh.md)
- [登录认证](docs/guide/core/authentication_zh.md)
- [权限管理](docs/guide/core/permission_zh.md)
- [多认证体系](docs/guide/core/multi-auth_zh.md)
- [Session 与终端](docs/guide/core/session-terminal_zh.md)
- [封禁体系](docs/guide/core/disable_zh.md)
- [Token 风格](docs/guide/core/token-style_zh.md)
- [AccessProvider](docs/guide/core/access-provider_zh.md)
- [框架集成](docs/guide/integration/framework-integration_zh.md)
- [事件监听](docs/guide/core/listener_zh.md)
- [Nonce 防重放](docs/guide/security/nonce_zh.md)
- [JWT 集成](docs/guide/security/jwt_zh.md)
- [Redis 存储](docs/guide/integration/redis-storage_zh.md)
- [OAuth2](docs/guide/security/oauth2_zh.md)
- [Ticket、ShortKey 等高级能力](docs/guide/security/advanced-features_zh.md)
- [SSO 单点登录](sso/README_zh.md)
- [SSO Redis 存储](sso/storage/redis/README_zh.md)
- [SSO 测试说明](sso/TESTING_zh.md)
- [Refresh Token](docs/guide/security/refresh-token_zh.md)
- [全局 API 参考](docs/api/dtoken_zh.md)

### 示例项目

| 示例 | 说明 |
| --- | --- |
| [examples/quick_start](examples/quick_start/) | Auth 实例配合自定义 Gin 处理器，演示登录、角色与权限校验 |
| [examples/gin](examples/gin/) | Gin 中间件、登录校验和角色校验 |
| [examples/sso_gin_server](examples/sso_gin_server/) | Gin SSO 统一登录中心示例 |
| [examples/sso_gin_client](examples/sso_gin_client/) | Gin SSO 子系统接入示例 |
| [examples/sso_server](examples/sso_server/) | 标准库 SSO 统一登录中心示例 |
| [examples/sso_client](examples/sso_client/) | 标准库 SSO 子系统接入示例 |
| [examples/sso_deployment](examples/sso_deployment/README_zh.md) | 完整签名 SSO 中心与多子应用、Redis、PKCE、防重放及部署指南 |
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
