<p align="center">
  <img src="docs/assets/logo.png" alt="DToken-Go" width="100" height="100">
</p>

<h1 align="center">DToken-Go</h1>

<p align="center">
  一个面向 Go 应用的认证、授权、会话管理与 Token 生命周期管理框架。
</p>

<p align="center">
  <a href="https://github.com/Zany2/dtoken-go"><img src="https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-blue?style=flat-square" alt="License"></a>
  <a href="docs/README_zh.md"><img src="https://img.shields.io/badge/Docs-中文文档-brightgreen?style=flat-square" alt="Docs"></a>
  <a href="https://pkg.go.dev/github.com/Zany2/dtoken-go/dtoken"><img src="https://img.shields.io/badge/pkg.go.dev-dtoken-007D9C?style=flat-square&logo=go" alt="pkg.go.dev"></a>
</p>

<p align="center">
  简体中文 | <a href="README.md">English</a>
</p>

---

## DToken-Go 是什么

DToken-Go 是一个模块化、可插拔的 Go 认证授权框架，已提供登录认证、Token 管理、Session 管理、终端管理、角色权限校验、账号与设备封禁、Nonce 防重放、OAuth2 服务端和事件监听等核心能力；SSO 单点登录、Ticket 临时凭证、短 Key 访问凭证、Token Introspection 和独立 Refresh Token 等能力正在持续开发中。框架支持插件化组件替换与自定义扩展，并适配主流 Go Web 框架，既可以作为独立认证核心使用，也可以快速接入现有业务项目。

你可以把它用于：

- 后台管理系统、用户中心、开放平台等需要统一认证授权的业务系统。
- Gin、Echo、Fiber、Chi、GoFrame、Hertz、Kratos 等 Go Web 项目的登录与权限接入。
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
| SSO 单点登录 🚧 | 统一登录、票据交换、跨系统登录态共享、统一登出、应用维度管理 |
| Ticket 临时凭证 🚧 | Ticket 创建、校验、一次性消费、撤销、TTL 查询和状态识别 |
| 短 Key 访问凭证 🚧 | 生成随机短 Key，用于短链接访问、扫码确认、临时授权和系统间换票 |
| Token Introspection 🚧 | 标准化查询 Token 是否有效、归属信息、TTL 和失效原因 |
| Refresh Token 🚧 | 独立刷新令牌签发、刷新、撤销、过期、轮换和安全校验 |

> 🚧 表示功能正在开发中。

## 安装

### 使用默认核心能力

```bash
go get github.com/Zany2/dtoken-go/defaults
go get github.com/Zany2/dtoken-go/dtoken
```

### 使用 Web 框架集成

如果项目已经使用某个 Web 框架，可以直接使用对应的 DToken 集成包。集成包封装了 Builder、中间件、上下文适配和常用 DToken API，方便在具体框架中快速接入认证、鉴权和登录态管理。

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
		TokenName("Authorization"). // 从 Authorization 中读取 Token
		Timeout(7200).              // Token 有效期：7200 秒
		AutoRenew(true).            // 登录态校验时自动续期
		IsPrintBanner(false).       // 示例中关闭启动 Banner
		Build()
	if err != nil {
		panic(err)
	}

	// 注册全局 Manager，之后即可使用 dtoken 全局 API。
	dtoken.SetManager(mgr)

	// 登录并签发 Token。
	token, err := dtoken.Login(ctx, "user-1001")
	if err != nil {
		panic(err)
	}

	// 给用户绑定角色和权限。
	_ = dtoken.AddRoles(ctx, "user-1001", []string{"admin"})
	_ = dtoken.AddPermissions(ctx, "user-1001", []string{"article:read"})

	// 使用 Token 获取登录账号，并进行权限判断。
	loginID, _ := dtoken.GetLoginID(ctx, token)

	fmt.Println("token:", token)
	fmt.Println("loginID:", loginID)
	fmt.Println("is login:", dtoken.IsLogin(ctx, token))
	fmt.Println("is admin:", dtoken.HasRole(ctx, loginID, "admin"))
	fmt.Println("can read article:", dtoken.HasPermission(ctx, loginID, "article:read"))

	// 登出后 Token 失效。
	_ = dtoken.Logout(ctx, token)
}
```

完整示例见 [examples/quick_start](examples/quick_start/)。

## Web 框架接入

下面以 Gin 为例，业务代码只需要引入 `integrations/gin`。接入时通常分为三步：初始化 DToken、注册中间件、在业务路由中使用登录态和权限能力。

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
		TokenName("Authorization"). // 从 Authorization Header 中读取 Token
		Timeout(7200).              // Token 有效期：7200 秒
		AutoRenew(true).            // 登录态校验时自动续期
		IsPrintBanner(false).       // 示例中关闭启动 Banner
		Build()
	if err != nil {
		panic(err)
	}

	// 注册全局 Manager，之后即可使用 gindt 暴露的 DToken API。
	gindt.SetManager(mgr)

	r := gin.Default()

	// 注册上下文中间件，后续可以通过 GetDTokenContext 读取当前请求的认证上下文。
	r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

	// 自定义认证或鉴权失败时的响应格式。
	failFunc := gindt.WithFailFunc(func(c *gin.Context, err error) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    gindt.CodeNotLogin,
			"message": err.Error(),
		})
	})

	r.POST("/login", func(c *gin.Context) {
		// 登录成功后签发 Token，客户端后续请求携带 Authorization: Bearer <token>。
		token, err := gindt.Login(c.Request.Context(), "user-1001")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 示例：给当前用户绑定角色，实际项目中通常来自数据库或权限中心。
		_ = gindt.AddRoles(c.Request.Context(), "user-1001", []string{"admin"})

		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	user := r.Group("/user")

	// AuthMiddleware 会校验请求是否已登录，未登录会直接拦截。
	user.Use(gindt.AuthMiddleware(ctx, failFunc))
	user.GET("/me", func(c *gin.Context) {
		// 从请求上下文中获取当前 Token 对应的登录账号。
		dCtx, _ := gindt.GetDTokenContext(c)
		loginID, _ := dCtx.GetLoginID(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"loginId": loginID})
	})

	admin := r.Group("/admin")

	// RoleMiddleware 会在登录校验之后继续校验角色。
	admin.Use(gindt.AuthMiddleware(ctx, failFunc), gindt.RoleMiddleware(ctx, []string{"admin"}, failFunc))
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

## 深入阅读

README 只保留最小上手路径，更多 API、配置和组件说明可以查看下面的专题文档：

- [Core API 速查](docs/guide/reference/core-api-cheatsheet_zh.md)
- [高级能力](docs/guide/security/advanced-features_zh.md)
- [配置示例](docs/guide/reference/configuration_zh.md)
- [组件生态](docs/guide/integration/component-ecosystem_zh.md)
- [多认证体系](docs/guide/core/multi-auth_zh.md)
- [封禁体系](docs/guide/core/disable_zh.md)
- [Token 风格](docs/guide/core/token-style_zh.md)
- [AccessProvider](docs/guide/core/access-provider_zh.md)

## 项目结构

```text
dtoken-go/
├── core/                         # 框架核心模块，按能力拆分为独立 Go module
│   ├── adapter/                  # 存储、编解码、日志、Token 生成器、请求上下文等接口契约
│   ├── builder/                  # Manager 构建器、组件装配和配置校验入口
│   ├── config/                   # 核心配置项、默认值和配置校验
│   ├── context/                  # 请求级 DTokenContext，承载当前请求的 Token 和 Manager
│   ├── derror/                   # 统一错误、错误码和核心错误定义
│   ├── listener/                 # 事件模型、事件管理器和监听回调
│   ├── manager/                  # 登录、Session、终端、权限、角色、封禁等核心逻辑
│   ├── nonce/                    # Nonce 防重放能力
│   ├── oauth2/                   # OAuth2 服务端能力
│   └── utils/                    # 内部通用工具
├── dtoken/                       # 对外门面 API：全局模式、实例模式和能力分组封装
├── sso/                          # 可选 SSO 模块：Ticket、共享 Token、远程会话和授权码模式
├── defaults/                     # 默认 Builder，内置默认存储、编解码、日志和 Token 生成器装配
├── com/                          # 可插拔组件实现
│   ├── codec/                    # 编解码组件，如 JSON、MessagePack、Base64
│   ├── generator/                # Token 生成器
│   ├── log/                      # 日志组件
│   ├── pool/                     # 协程池组件
│   └── storage/                  # 存储组件，如 Memory、Redis
├── integrations/                 # Web 框架集成包
│   ├── gin/                      # Gin 中间件、上下文适配和 API 导出
│   ├── echo/                     # Echo 中间件、上下文适配和 API 导出
│   ├── fiber/                    # Fiber 中间件、上下文适配和 API 导出
│   ├── chi/                      # Chi 中间件、上下文适配和 API 导出
│   ├── gf/                       # GoFrame 中间件、上下文适配和 API 导出
│   ├── hertz/                    # Hertz 中间件、上下文适配和 API 导出
│   └── kratos/                   # Kratos 中间件、上下文适配和 API 导出
├── examples/                     # 快速开始与框架接入示例
│   ├── quick_start/              # 默认 Builder + 全局 API 最小示例
│   ├── gin/                      # Gin 示例
│   ├── echo/                     # Echo 示例
│   ├── fiber/                    # Fiber 示例
│   ├── chi/                      # Chi 示例
│   ├── gf/                       # GoFrame 示例
│   ├── hertz/                    # Hertz 示例
│   └── kratos/                   # Kratos 示例
├── tests/                        # 流程测试与测试应用
│   ├── gin_core_app/             # Gin 核心流程测试应用
│   └── gin_core_flow/            # 基于 HTTP 流程的核心功能测试
├── docs/                         # 文档、指南、API 参考和设计说明
│   ├── guide/core/               # 登录、权限、Session、终端、封禁等核心能力
│   ├── guide/security/           # Nonce、OAuth2、SSO、JWT、Refresh Token 等安全与协议能力
│   ├── guide/integration/        # Web 框架、注解、组件、Redis、流程测试
│   ├── guide/reference/          # 配置示例和 Core API 速查
│   ├── api/                      # API 参考
│   ├── design/                   # 架构与设计文档
│   └── tutorial/                 # 快速开始教程
├── FEATURE.md                    # 功能规划和待完善能力记录
├── LICENSE                       # Apache-2.0 开源协议
├── README_zh.md                  # 中文项目说明
├── README.md                     # 英文项目说明
├── go.work                       # Go workspace，统一管理多模块
└── go.work.sum                   # Workspace 依赖校验文件
```

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
- [Refresh Token](docs/guide/security/refresh-token_zh.md)
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
