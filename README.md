<p align="center">
  <img src="docs/assets/logo.png" alt="DToken-Go" width="180">
</p>

<h1 align="center">DToken-Go</h1>

<p align="center">
  一个面向 Go 应用的认证、授权与会话管理框架。
</p>

<p align="center">
  中文 | <a href="README_EN.md">English</a>
</p>

---

## 简介

DToken-Go 是一个模块化的 Go 认证授权框架，提供 Token 登录认证、会话管理、角色和权限校验、账号封禁、Nonce 防重放、OAuth2 服务端等能力。

它把核心能力、默认组件和 Web 框架集成拆分成独立模块：

- `dtoken` 提供全局门面 API，适合业务代码直接调用。
- `defaults` 提供开箱即用的默认 Builder，默认使用内存存储、JSON 编解码、默认 Token 生成器和日志组件。
- `core` 提供核心接口、配置、Manager、上下文、事件、Nonce、OAuth2 等基础能力。
- `com` 提供可插拔组件实现，例如存储、编解码、日志、Token 生成器和协程池。
- `integrations` 提供 Gin、Echo、Fiber、Chi、GoFrame、Hertz、Kratos 等框架集成。

## 功能特性

- Token 认证：登录、登出、Token 校验、自动续期、多终端管理。
- 会话管理：Session 存储、终端信息追踪、在线终端统计。
- 权限管理：权限增删查、AND/OR 校验、按 Token 校验、自定义权限回调。
- 角色管理：角色增删查、AND/OR 校验、按 Token 校验、自定义角色回调。
- 账号封禁：定时封禁、解封、封禁原因、剩余封禁时间查询。
- 在线状态控制：踢人下线、顶人下线、按账号/设备/设备 ID 操作。
- Nonce：一次性随机值生成、校验和消费，用于防重放场景。
- OAuth2：授权码模式、客户端凭证模式、密码模式、刷新令牌、Token 校验和撤销。
- 多认证体系：通过 `authType` 同时维护多套独立认证体系。
- 可插拔组件：存储、编解码、日志、Token 生成器、协程池均可替换。
- 多框架集成：为常见 Go Web 框架提供中间件、注解式校验和请求上下文适配。

## 安装

核心能力：

```bash
go get github.com/Zany2/dtoken-go/defaults
go get github.com/Zany2/dtoken-go/dtoken
```

按需安装存储组件：

```bash
go get github.com/Zany2/dtoken-go/com/storage/redis
```

按需安装框架集成：

```bash
go get github.com/Zany2/dtoken-go/integrations/gin
go get github.com/Zany2/dtoken-go/integrations/echo
go get github.com/Zany2/dtoken-go/integrations/fiber
go get github.com/Zany2/dtoken-go/integrations/chi
go get github.com/Zany2/dtoken-go/integrations/gf
go get github.com/Zany2/dtoken-go/integrations/hertz
go get github.com/Zany2/dtoken-go/integrations/kratos
```

## 快速开始

`defaults.NewBuilder()` 已经装配默认内存存储，因此最小示例不需要 Redis 或数据库。

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
	_ = dtoken.AddPermissions(ctx, "user-1001", []string{"article:read"})

	loginID, _ := dtoken.GetLoginID(ctx, token)
	hasRole := dtoken.HasRole(ctx, loginID, "admin")
	hasPermission := dtoken.HasPermission(ctx, loginID, "article:read")

	fmt.Println(token, loginID, hasRole, hasPermission)
	_ = dtoken.Logout(ctx, token)
}
```

完整快速开始示例见 [examples/quick_start](examples/quick_start/)。

## Gin 集成示例

框架示例推荐只导入对应的 `integrations/*` 包作为 DToken 入口。下面以 Gin 为例：

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
		_ = gindt.AddRoles(c.Request.Context(), "user-1001", []string{"admin"})
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

更多框架示例：

| 框架 | 示例 | 集成包 |
| --- | --- | --- |
| Gin | [examples/gin](examples/gin/) | `github.com/Zany2/dtoken-go/integrations/gin` |
| Echo | [examples/echo](examples/echo/) | `github.com/Zany2/dtoken-go/integrations/echo` |
| Fiber | [examples/fiber](examples/fiber/) | `github.com/Zany2/dtoken-go/integrations/fiber` |
| Chi | [examples/chi](examples/chi/) | `github.com/Zany2/dtoken-go/integrations/chi` |
| GoFrame | [examples/gf](examples/gf/) | `github.com/Zany2/dtoken-go/integrations/gf` |
| Hertz | [examples/hertz](examples/hertz/) | `github.com/Zany2/dtoken-go/integrations/hertz` |
| Kratos | [examples/kratos](examples/kratos/) | `github.com/Zany2/dtoken-go/integrations/kratos` |

## 常用 API

### 登录与 Token

| API | 说明 |
| --- | --- |
| `Login(ctx, loginID, params...)` | 登录并返回 Token，`params` 可传 `device`、`deviceId`、`authType` |
| `LoginByToken(ctx, tokenValue)` | 根据已有 Token 续登录 |
| `Logout(ctx, tokenValue)` | 按 Token 登出 |
| `LogoutByLoginID(ctx, loginID)` | 登出指定账号的所有终端 |
| `IsLogin(ctx, tokenValue)` | 判断 Token 是否已登录 |
| `CheckLogin(ctx, tokenValue)` | 校验登录状态，失败返回错误 |
| `GetLoginID(ctx, tokenValue)` | 根据 Token 获取登录 ID |
| `GetTokenInfo(ctx, tokenValue)` | 获取 Token 详情 |
| `GetTokenTTL(ctx, tokenValue)` | 获取 Token 剩余有效时间 |

### 权限与角色

| API | 说明 |
| --- | --- |
| `AddPermissions(ctx, loginID, permissions)` | 添加权限 |
| `RemovePermissions(ctx, loginID, permissions)` | 移除权限 |
| `GetPermissions(ctx, loginID)` | 获取权限列表 |
| `HasPermission(ctx, loginID, permission)` | 判断是否拥有指定权限 |
| `HasPermissionsAnd(ctx, loginID, permissions)` | 判断是否拥有全部权限 |
| `HasPermissionsOr(ctx, loginID, permissions)` | 判断是否拥有任一权限 |
| `AddRoles(ctx, loginID, roles)` | 添加角色 |
| `RemoveRoles(ctx, loginID, roles)` | 移除角色 |
| `GetRoles(ctx, loginID)` | 获取角色列表 |
| `HasRole(ctx, loginID, role)` | 判断是否拥有指定角色 |
| `HasRolesAnd(ctx, loginID, roles)` | 判断是否拥有全部角色 |
| `HasRolesOr(ctx, loginID, roles)` | 判断是否拥有任一角色 |

### 在线状态与封禁

| API | 说明 |
| --- | --- |
| `Kickout(ctx, tokenValue)` | 踢出指定 Token |
| `KickoutByLoginID(ctx, loginID)` | 踢出指定账号所有终端 |
| `Replace(ctx, tokenValue)` | 顶替指定 Token |
| `ReplaceByLoginID(ctx, loginID)` | 顶替指定账号所有终端 |
| `Disable(ctx, loginID, duration, reason...)` | 封禁账号 |
| `Untie(ctx, loginID)` | 解封账号 |
| `IsDisable(ctx, loginID)` | 判断账号是否封禁 |
| `GetDisableInfo(ctx, loginID)` | 获取封禁信息 |

### Nonce 与 OAuth2

| API | 说明 |
| --- | --- |
| `GenerateNonce(ctx)` | 生成一次性 Nonce |
| `VerifyNonce(ctx, nonce)` | 校验 Nonce |
| `VerifyAndConsumeNonce(ctx, nonce)` | 校验并消费 Nonce |
| `RegisterOAuth2Client(client)` | 注册 OAuth2 客户端 |
| `GenerateOAuth2AuthorizationCode(...)` | 生成授权码 |
| `ExchangeOAuth2CodeForToken(...)` | 授权码换取 Token |
| `OAuth2ClientCredentialsToken(...)` | 客户端凭证模式获取 Token |
| `OAuth2PasswordGrantToken(...)` | 密码模式获取 Token |
| `RefreshOAuth2AccessToken(...)` | 刷新访问令牌 |
| `ValidateOAuth2AccessToken(ctx, accessToken)` | 校验访问令牌 |
| `RevokeOAuth2Token(ctx, accessToken)` | 撤销访问令牌 |

## Builder 配置

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

常用配置：

| 配置 | 说明 |
| --- | --- |
| `AuthType` | 认证体系标识，用于多套认证并行 |
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

## 可插拔组件

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
└── integrations/     # Web 框架集成包
```

## 文档

- [文档中心](docs/README_zh.md)
- [快速开始](docs/tutorial/quick-start_zh.md)
- [登录认证](docs/guide/authentication_zh.md)
- [权限管理](docs/guide/permission_zh.md)
- [框架集成](docs/guide/framework-integration_zh.md)
- [Redis 存储](docs/guide/redis-storage_zh.md)
- [OAuth2](docs/guide/oauth2_zh.md)
- [API 参考](docs/api/dtoken_zh.md)

## License

MIT
