<p align="center">
  <img src="docs/assets/logo.png" alt="DToken-Go" width="100" height="100">
</p>

<h1 align="center">DToken-Go</h1>

<p align="center">
  一个面向 Go 应用的认证、授权、会话管理与单点登录框架。
</p>

<p align="center">
  <a href="https://github.com/Zany2/dtoken-go"><img src="https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-blue?style=flat-square" alt="License"></a>
  <a href="docs/README_zh.md"><img src="https://img.shields.io/badge/Docs-中文文档-brightgreen?style=flat-square" alt="Docs"></a>
</p>

<p align="center">
  中文 | <a href="README.md">English</a>
</p>

---

## DToken-Go 是什么

DToken-Go 是一个模块化、可插拔的 Go 认证授权框架，提供登录认证、Token 管理、Session 管理、终端管理、角色权限校验、账号封禁、SSO 单点登录、短 Key 登录、Ticket 临时凭证、Token Introspection、Refresh Token、Nonce 防重放、OAuth2 服务端和事件监听等能力。

你可以把它用于：

- 后台管理系统、用户中心、开放平台。
- Gin、Echo、Fiber、Chi、GoFrame、Hertz、Kratos 等 Web 项目。
- 微服务网关、统一认证中心、跨系统单点登录。
- App、小程序、Web、多设备、多终端登录态管理。
- 扫码登录、一次性登录、临时授权、第三方系统 Token 校验。

## 目录

- [核心特性](#核心特性)
- [安装](#安装)
- [5 分钟上手](#5-分钟上手)
- [Web 框架接入](#web-框架接入)
- [Core API 速查](#core-api-速查)
- [高级能力](#高级能力)
- [配置示例](#配置示例)
- [组件生态](#组件生态)
- [项目结构](#项目结构)
- [文档与示例](#文档与示例)

## 核心特性

| 能力 | 说明 |
| --- | --- |
| 登录认证 | 登录、续登、登出、登录态校验、Token 信息查询、TTL 查询、手动续期和自动续期 |
| Session 管理 | 按账号、Token、设备、设备 ID、应用维度查询和管理登录态 |
| 终端管理 | 多端登录、终端追踪、在线终端统计、终端清理、踢下线、顶下线 |
| 角色权限 | 角色和权限增删查、AND/OR 校验、Token 维度校验、通配符权限匹配 |
| 并发控制 | 支持同账号并发登录控制、共享 Token、最大在线终端数限制 |
| 账号封禁 | 账号封禁、设备封禁、解封、封禁原因和剩余封禁时间查询 |
| SSO 单点登录 | 统一登录、票据交换、跨系统登录态共享、统一登出、应用维度管理 |
| Ticket 临时凭证 | Ticket 创建、校验、一次性消费、撤销、TTL 查询和状态识别 |
| 短 Key 登录 | 适合扫码登录、一次性登录、临时授权和系统间换票 |
| Token Introspection | 标准化查询 Token 是否有效、归属信息、TTL 和失效原因 |
| Refresh Token | 访问令牌刷新、刷新令牌撤销、过期、轮换和安全校验 |
| Nonce 防重放 | 一次性随机值生成、校验、消费，防止请求重放 |
| OAuth2 | 授权码、客户端凭证、密码模式、刷新令牌、Token 校验和撤销 |
| 事件系统 | 登录、登出、续期、Ticket、短 Key、SSO 等事件监听 |
| 可插拔组件 | 存储、编解码、日志、Token 生成器、协程池均可替换 |
| 多框架集成 | 为主流 Go Web 框架提供中间件、上下文适配和 API 导出 |

## 安装

### 使用默认核心能力

```bash
go get github.com/Zany2/dtoken-go/defaults
go get github.com/Zany2/dtoken-go/dtoken
```

### 使用 Web 框架集成

如果项目已经使用某个 Web 框架，推荐直接引入对应集成包。集成包会导出 Builder、中间件、上下文适配和常用 DToken API。

```bash
go get github.com/Zany2/dtoken-go/integrations/gin
```

可选框架：

```bash
go get github.com/Zany2/dtoken-go/integrations/echo
go get github.com/Zany2/dtoken-go/integrations/fiber
go get github.com/Zany2/dtoken-go/integrations/chi
go get github.com/Zany2/dtoken-go/integrations/gf
go get github.com/Zany2/dtoken-go/integrations/hertz
go get github.com/Zany2/dtoken-go/integrations/kratos
```

### 使用可插拔组件

```bash
go get github.com/Zany2/dtoken-go/com/storage/redis
go get github.com/Zany2/dtoken-go/com/generator/dgenerator
go get github.com/Zany2/dtoken-go/com/pool/ants
```

## 5 分钟上手

`defaults.NewBuilder()` 已经装配默认内存存储、JSON 编解码、默认 Token 生成器和日志组件，适合快速体验。

```go
package main

import (
	"context"
	"fmt"

	"github.com/Zany2/dtoken-go/defaults"
	"github.com/Zany2/dtoken-go/dtoken"
)

func main() {
	ctx := context.Background()

	mgr, err := defaults.NewBuilder().
		TokenName("Authorization").
		Timeout(7200).
		ActiveTimeout(1800).
		AutoRenew(true).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}
	dtoken.SetManager(mgr)

	token, err := dtoken.Login(ctx, "user-1001")
	if err != nil {
		panic(err)
	}

	_ = dtoken.AddRoles(ctx, "user-1001", []string{"admin"})
	_ = dtoken.AddPermissions(ctx, "user-1001", []string{"article:read", "article:write"})

	loginID, _ := dtoken.GetLoginID(ctx, token)
	hasRole := dtoken.HasRole(ctx, loginID, "admin")
	hasPermission := dtoken.HasPermission(ctx, loginID, "article:read")

	fmt.Println("token:", token)
	fmt.Println("loginID:", loginID)
	fmt.Println("has role:", hasRole)
	fmt.Println("has permission:", hasPermission)

	_ = dtoken.Logout(ctx, token)
}
```

完整示例见 [examples/quick_start](examples/quick_start/)。

## Web 框架接入

下面以 Gin 为例，业务代码只需要引入 `integrations/gin`：

```go
package main

import (
	"context"
	"net/http"

	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	mgr, err := gindt.NewBuilder().
		TokenName("Authorization").
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}
	gindt.SetManager(mgr)

	r := gin.Default()
	r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

	r.POST("/login", func(c *gin.Context) {
		token, err := gindt.Login(c.Request.Context(), "user-1001")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	user := r.Group("/user")
	user.Use(gindt.AuthMiddleware(ctx))
	user.GET("/me", func(c *gin.Context) {
		dCtx, _ := gindt.GetDTokenContext(c)
		loginID, _ := dCtx.GetLoginID(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"loginId": loginID})
	})

	admin := r.Group("/admin")
	admin.Use(gindt.AuthMiddleware(ctx), gindt.RoleMiddleware(ctx, []string{"admin"}))
	admin.GET("/dashboard", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	_ = r.Run(":8080")
}
```

| 框架 | 示例 | 集成包 |
| --- | --- | --- |
| Gin | [examples/gin](examples/gin/) | `github.com/Zany2/dtoken-go/integrations/gin` |
| Echo | [examples/echo](examples/echo/) | `github.com/Zany2/dtoken-go/integrations/echo` |
| Fiber | [examples/fiber](examples/fiber/) | `github.com/Zany2/dtoken-go/integrations/fiber` |
| Chi | [examples/chi](examples/chi/) | `github.com/Zany2/dtoken-go/integrations/chi` |
| GoFrame | [examples/gf](examples/gf/) | `github.com/Zany2/dtoken-go/integrations/gf` |
| Hertz | [examples/hertz](examples/hertz/) | `github.com/Zany2/dtoken-go/integrations/hertz` |
| Kratos | [examples/kratos](examples/kratos/) | `github.com/Zany2/dtoken-go/integrations/kratos` |

## Core API 速查

### 登录认证

```go
token, err := dtoken.Login(ctx, "user-1001")
token, err = dtoken.Login(ctx, "user-1001", "web", "browser-001", "user")

isLogin := dtoken.IsLogin(ctx, token)
loginID, err := dtoken.GetLoginID(ctx, token)
tokenInfo, err := dtoken.GetTokenInfo(ctx, token)
ttl, err := dtoken.GetTokenTTL(ctx, token)

err = dtoken.RenewTimeout(ctx, token, 7200)
err = dtoken.Logout(ctx, token)
err = dtoken.LogoutByLoginID(ctx, "user-1001")
```

### 角色权限

```go
_ = dtoken.AddRoles(ctx, "user-1001", []string{"admin", "auditor"})
_ = dtoken.AddPermissions(ctx, "user-1001", []string{"order:read", "order:*"})

hasRole := dtoken.HasRole(ctx, "user-1001", "admin")
hasAnyRole := dtoken.HasRolesOr(ctx, "user-1001", []string{"admin", "owner"})
hasPermission := dtoken.HasPermission(ctx, "user-1001", "order:read")
hasAllPermission := dtoken.HasPermissionsAnd(ctx, "user-1001", []string{"order:read", "order:write"})

_, _, _, _ = hasRole, hasAnyRole, hasPermission, hasAllPermission
```

### 在线状态控制

```go
_ = dtoken.Kickout(ctx, token)
_ = dtoken.KickoutByLoginID(ctx, "user-1001")
_ = dtoken.Replace(ctx, token)
_ = dtoken.ReplaceByLoginID(ctx, "user-1001")
```

### 封禁控制

```go
_ = dtoken.Disable(ctx, "user-1001", 3600, "risk_control")
disabled := dtoken.IsDisable(ctx, "user-1001")
disableInfo, err := dtoken.GetDisableInfo(ctx, "user-1001")
_ = dtoken.Untie(ctx, "user-1001")

_, _ = disabled, disableInfo
```

## 高级能力

### Token Introspection

```go
info, err := dtoken.IntrospectToken(ctx, token)
if err != nil {
	return err
}
if !info.Active {
	fmt.Println("invalid reason:", info.Reason)
}
```

### Refresh Token

```go
pair, err := dtoken.LoginWithRefreshToken(ctx, "user-1001")
if err != nil {
	return err
}

nextPair, err := dtoken.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}
_ = dtoken.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

### Ticket 临时凭证

```go
ticket, err := dtoken.CreateTicket(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ConsumeTicket(ctx, ticket)
if err != nil {
	return err
}
fmt.Println(token)
```

### 短 Key 登录

```go
shortKey, err := dtoken.CreateShortKey(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ConsumeShortKey(ctx, shortKey)
if err != nil {
	return err
}
fmt.Println(token)
```

### SSO 单点登录

```go
ssoTicket, err := dtoken.CreateSSOTicket(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ExchangeSSOTicket(ctx, ssoTicket)
if err != nil {
	return err
}

_ = dtoken.LogoutAllApps(ctx, "user-1001")
```

### Nonce 防重放

```go
nonce, err := dtoken.GenerateNonce(ctx)
if err != nil {
	return err
}

ok, err := dtoken.VerifyAndConsumeNonce(ctx, nonce)
_, _ = ok, err
```

### 事件监听

```go
mgr.AddListener(func(event dtoken.Event) {
	fmt.Println(event.Type, event.LoginID, event.TokenValue)
})
```

## 配置示例

```go
mgr, err := defaults.NewBuilder().
	AuthType("user").
	KeyPrefix("dtoken").
	TokenName("Authorization").
	Timeout(7200).
	ActiveTimeout(1800).
	AutoRenew(true).
	RenewMaxRefresh(3600).
	RenewInterval(60).
	IsConcurrent(true).
	IsShare(false).
	MaxLoginCount(5).
	IsReadHeader(true).
	IsReadCookie(false).
	IsReadBody(false).
	IsLog(false).
	IsPrintBanner(false).
	SetStorage(storage).
	Build()
```

| 配置 | 说明 |
| --- | --- |
| `AuthType` | 认证体系标识，可同时维护多套认证体系 |
| `KeyPrefix` | 存储 key 前缀 |
| `TokenName` | Token 名称，通常对应 Header 或 Cookie 名 |
| `Timeout` | Token 绝对过期时间，单位秒 |
| `ActiveTimeout` | 最大不活跃时间，单位秒 |
| `AutoRenew` | 是否自动续期 |
| `RenewMaxRefresh` | 续期触发阈值 |
| `RenewInterval` | 最小续期间隔 |
| `IsConcurrent` | 是否允许同账号并发登录 |
| `IsShare` | 并发登录时是否共享 Token |
| `MaxLoginCount` | 最大在线终端数 |
| `IsReadHeader` | 是否从 Header 读取 Token |
| `IsReadCookie` | 是否从 Cookie 读取 Token |
| `IsReadBody` | 是否从请求体读取 Token |
| `SetStorage` | 设置自定义存储适配器 |

## 组件生态

| 类型 | 实现 | 模块 |
| --- | --- | --- |
| 存储 | Memory、Redis、PostgreSQL | `com/storage/*` |
| 编解码 | JSON、JSON v2、MessagePack、Base64 | `com/codec/*` |
| 日志 | DLog、GoFrame、Nop | `com/log/*` |
| Token 生成器 | UUID、JWT | `com/generator/dgenerator` |
| 协程池 | Ants | `com/pool/ants` |

## 项目结构

```text
dtoken-go/
├── com/              # 可插拔组件实现
├── core/             # 核心接口、配置、Manager、上下文、Nonce、OAuth2
├── defaults/         # 默认 Builder 和默认组件装配
├── docs/             # 详细文档和图片资源
├── dtoken/           # 全局 API 门面
├── examples/         # 快速开始和框架集成示例
├── integrations/     # Web 框架集成包
└── go.work           # Go workspace
```

## 文档与示例

### 文档

- [文档中心](docs/README_zh.md)
- [快速开始](docs/tutorial/quick-start_zh.md)
- [登录认证](docs/guide/authentication_zh.md)
- [权限管理](docs/guide/permission_zh.md)
- [框架集成](docs/guide/framework-integration_zh.md)
- [事件监听](docs/guide/listener_zh.md)
- [Nonce 防重放](docs/guide/nonce_zh.md)
- [JWT 集成](docs/guide/jwt_zh.md)
- [Redis 存储](docs/guide/redis-storage_zh.md)
- [OAuth2](docs/guide/oauth2_zh.md)
- [Refresh Token](docs/guide/refresh-token_zh.md)
- [API 参考](docs/api/dtoken_zh.md)

### 示例项目

| 示例 | 说明 |
| --- | --- |
| [examples/quick_start](examples/quick_start/) | 默认 Builder + 全局 API 最小使用方式 |
| [examples/gin](examples/gin/) | Gin 中间件、登录校验和角色校验 |
| [examples/echo](examples/echo/) | Echo 框架接入示例 |
| [examples/fiber](examples/fiber/) | Fiber 框架接入示例 |
| [examples/chi](examples/chi/) | Chi 框架接入示例 |
| [examples/gf](examples/gf/) | GoFrame 框架接入示例 |
| [examples/hertz](examples/hertz/) | Hertz 框架接入示例 |
| [examples/kratos](examples/kratos/) | Kratos 框架接入示例 |

## 贡献

欢迎提交 Issue、Pull Request 或使用反馈。提交代码前请尽量保持改动聚焦，并遵循项目现有的模块边界和代码风格。

## License

DToken-Go 使用 [Apache-2.0](LICENSE) 协议开源。
