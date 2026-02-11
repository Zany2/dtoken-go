# dtoken-go

一个功能完备的 Go 语言认证授权框架，提供 Token 认证、权限管理、角色管理、会话管理、账号封禁、OAuth2 服务端等能力，支持多种 Web 框架集成。

## 功能特性

- **Token 认证** — 登录/登出、Token 校验、自动续期、多设备登录管理
- **权限管理** — 权限的增删查、AND/OR 逻辑判断、自定义权限获取函数
- **角色管理** — 角色的增删查、AND/OR 逻辑判断、自定义角色获取函数
- **会话管理** — Session 数据存储、终端信息追踪、在线终端统计
- **账号封禁** — 定时封禁/解封、封禁原因记录、剩余封禁时间查询
- **在线状态管理** — 踢人下线（Kickout）、顶人下线（Replace）、按设备/账号维度操作
- **OAuth2 服务端** — 授权码模式、客户端凭证模式、密码模式、刷新令牌
- **Nonce 管理** — 一次性令牌生成与验证
- **多认证体系** — 通过 `authType` 支持多套独立的认证体系并行运行
- **可插拔架构** — 存储、编解码、日志、Token 生成器、协程池均可替换

## 项目结构

```
dtoken-go/
├── core/                        # 核心框架
│   ├── adapter/                 # 适配器接口定义（Storage, Log, Codec, Generator, Pool）
│   ├── builder/                 # Builder 构建器
│   ├── config/                  # 配置结构体与校验
│   ├── context/                 # 上下文处理
│   ├── derror/                  # 错误定义
│   ├── listener/                # 事件监听器
│   ├── manager/                 # Manager 核心实现
│   ├── nonce/                   # Nonce 管理
│   ├── oauth2/                  # OAuth2 实现
│   └── utils/                   # 工具函数
├── com/                         # 组件实现
│   ├── codec/                   # 编解码器（base64, json, jsonv2, msgpack）
│   ├── generator/dgenerator/    # Token 生成器（UUID, JWT）
│   ├── log/                     # 日志（dlog 默认日志, gf 适配, nop 空日志）
│   ├── pool/ants/               # 协程池（基于 ants）
│   └── storage/                 # 存储（memory 内存, redis）
├── integrations/                # Web 框架集成
│   ├── gin/                     # Gin 中间件
│   ├── fiber/                   # Fiber 中间件
│   ├── gf/                      # GoFrame 中间件
│   └── zero/                    # Go-Zero 中间件
├── dtoken/                      # 全局 API 门面
└── examples/                    # 使用示例
    ├── quick_start/             # 快速开始（完整 API 演示）
    ├── gin/                     # Gin 集成示例
    ├── gf/                      # GoFrame 示例
    ├── gf_v2/                   # GoFrame v2 示例
    └── zero/                    # Go-Zero 示例
```

## 快速开始

### 安装

```bash
# 核心模块
go get github.com/Zany2/dtoken-go/core
go get github.com/Zany2/dtoken-go/dtoken

# 存储（按需选择）
go get github.com/Zany2/dtoken-go/com/storage/redis
go get github.com/Zany2/dtoken-go/com/storage/memory

# Web 框架集成（按需选择）
go get github.com/Zany2/dtoken-go/integrations/gin
go get github.com/Zany2/dtoken-go/integrations/fiber
go get github.com/Zany2/dtoken-go/integrations/gf
go get github.com/Zany2/dtoken-go/integrations/zero
```

### 基础用法

```go
package main

import (
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go/core/builder"
    "github.com/Zany2/dtoken-go/com/storage/redis"
    "github.com/Zany2/dtoken-go/dtoken"
)

func main() {
    ctx := context.Background()

    // 1. 创建存储
    storage, err := redis.NewStorage("redis://:password@localhost:6379/0")
    if err != nil {
        panic(err)
    }

    // 2. 构建 Manager
    mgr := builder.NewBuilder().
        TokenName("token").
        Timeout(7200).          // Token 有效期 2 小时
        IsConcurrent(true).     // 允许并发登录
        MaxLoginCount(5).       // 最多 5 个设备同时在线
        SetStorage(storage).
        Build()

    // 3. 注册全局 Manager
    dtoken.SetManager(mgr)

    // 4. 登录
    token, _ := dtoken.Login(ctx, "user001")
    fmt.Println("Token:", token)

    // 5. 校验登录状态
    fmt.Println("IsLogin:", dtoken.IsLogin(ctx, token))

    // 6. 获取登录 ID
    loginID, _ := dtoken.GetLoginID(ctx, token)
    fmt.Println("LoginID:", loginID)

    // 7. 登出
    _ = dtoken.Logout(ctx, token)
}
```

### Gin 框架集成

```go
package main

import (
    "context"
    gindt "github.com/Zany2/dtoken-go/integrations/gin"
    "github.com/Zany2/dtoken-go/com/storage/redis"
    "github.com/gin-gonic/gin"
)

func main() {
    ctx := context.Background()

    // 初始化存储和 Manager
    storage, _ := redis.NewStorage("redis://:password@localhost:6379/0")
    mgr := gindt.NewDefaultBuilder().
        SetStorage(storage).
        Timeout(3600).
        Build()
    gindt.SetManager(mgr)

    r := gin.Default()

    // 注册 DToken 上下文中间件
    r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

    // 公开路由
    r.POST("/login", handleLogin)

    // 需要登录的路由
    user := r.Group("/user")
    user.Use(gindt.AuthMiddleware(ctx))
    {
        user.GET("/info", handleUserInfo)
    }

    // 需要角色的路由
    admin := r.Group("/admin")
    admin.Use(gindt.RoleMiddleware(ctx, []string{"admin"}))
    {
        admin.GET("/dashboard", handleDashboard)
    }

    // 需要权限的路由
    resource := r.Group("/resource")
    resource.Use(gindt.PermissionMiddleware(ctx, []string{"resource:read"}))
    {
        resource.GET("/list", handleResourceList)
    }

    r.Run(":8080")
}
```

## 核心 API

### 登录与认证

| 方法 | 说明 |
|------|------|
| `Login(ctx, loginID, params...)` | 登录，返回 Token。params 可选：device, deviceId, authType |
| `Logout(ctx, tokenValue)` | 根据 Token 登出 |
| `LogoutByLoginID(ctx, loginID)` | 登出指定用户的所有终端 |
| `LogoutByDevice(ctx, loginID, device)` | 登出指定设备类型的所有终端 |
| `IsLogin(ctx, tokenValue)` | 检查是否已登录 |
| `CheckLogin(ctx, tokenValue)` | 检查登录状态，未登录返回 error |
| `GetLoginID(ctx, tokenValue)` | 根据 Token 获取登录 ID |
| `GetTokenInfo(ctx, tokenValue)` | 获取 Token 详细信息 |

### 在线状态管理

| 方法 | 说明 |
|------|------|
| `Kickout(ctx, tokenValue)` | 踢人下线 |
| `KickoutByLoginID(ctx, loginID)` | 踢出指定用户所有终端 |
| `Replace(ctx, tokenValue)` | 顶人下线 |
| `ReplaceByLoginID(ctx, loginID)` | 顶替指定用户所有终端 |

### 权限管理

| 方法 | 说明 |
|------|------|
| `AddPermissions(ctx, loginID, permissions)` | 添加权限 |
| `RemovePermissions(ctx, loginID, permissions)` | 移除权限 |
| `GetPermissions(ctx, loginID)` | 获取权限列表 |
| `HasPermission(ctx, loginID, permission)` | 检查单个权限 |
| `HasPermissionsAnd(ctx, loginID, permissions)` | 检查是否拥有全部权限 |
| `HasPermissionsOr(ctx, loginID, permissions)` | 检查是否拥有任一权限 |

### 角色管理

| 方法 | 说明 |
|------|------|
| `AddRoles(ctx, loginID, roles)` | 添加角色 |
| `RemoveRoles(ctx, loginID, roles)` | 移除角色 |
| `GetRoles(ctx, loginID)` | 获取角色列表 |
| `HasRole(ctx, loginID, role)` | 检查单个角色 |
| `HasRolesAnd(ctx, loginID, roles)` | 检查是否拥有全部角色 |
| `HasRolesOr(ctx, loginID, roles)` | 检查是否拥有任一角色 |

### 账号封禁

| 方法 | 说明 |
|------|------|
| `Disable(ctx, loginID, duration, reason)` | 封禁账号 |
| `Untie(ctx, loginID)` | 解封账号 |
| `IsDisable(ctx, loginID)` | 检查是否被封禁 |
| `GetDisableInfo(ctx, loginID)` | 获取封禁详情 |

### OAuth2

| 方法 | 说明 |
|------|------|
| `RegisterOAuth2Client(client)` | 注册 OAuth2 客户端 |
| `GenerateOAuth2AuthorizationCode(...)` | 生成授权码 |
| `ExchangeOAuth2CodeForToken(...)` | 授权码换取 Token |
| `OAuth2ClientCredentialsToken(...)` | 客户端凭证模式获取 Token |
| `OAuth2PasswordGrantToken(...)` | 密码模式获取 Token |
| `RefreshOAuth2AccessToken(...)` | 刷新 Token |
| `ValidateOAuth2AccessToken(ctx, accessToken)` | 验证 Token |
| `RevokeOAuth2Token(ctx, accessToken)` | 撤销 Token |

### Nonce

| 方法 | 说明 |
|------|------|
| `GenerateNonce(ctx)` | 生成 Nonce |
| `VerifyNonce(ctx, nonce)` | 验证并消费 Nonce |
| `IsNonceValid(ctx, nonce)` | 检查 Nonce 是否有效（不消费） |

## 配置项

通过 Builder 链式调用进行配置：

```go
builder.NewBuilder().
    AuthType("user").              // 认证体系类型，默认 "login:"
    TokenName("token").            // Token 名称，默认 "dtoken"
    Timeout(7200).                 // Token 过期时间（秒），-1 永不过期
    AutoRenew(true).               // 是否自动续期
    RenewMaxRefresh(3600).         // 续期触发阈值（秒）
    RenewInterval(60).             // 最小续期间隔（秒）
    ActiveTimeout(1800).           // 最大不活跃时长（秒），-1 不限制
    IsConcurrent(true).            // 是否允许并发登录
    IsShare(false).                // 并发登录是否共用 Token
    MaxLoginCount(5).              // 最大并发登录数，-1 不限制
    ConcurrencyScope("account").   // 并发控制作用域：account / device
    IsReadHeader(true).            // 从 Header 读取 Token
    IsReadCookie(false).           // 从 Cookie 读取 Token
    IsReadBody(false).             // 从请求体读取 Token
    TokenStyle(adapter.TokenStyleUUID). // Token 风格：UUID / JWT
    JwtSecret("your-secret").      // JWT 模式密钥
    IsLog(true).                   // 开启日志
    AsyncEvent(true).              // 异步触发事件
    SetStorage(storage).           // 设置存储适配器
    SetCodec(codec).               // 设置编解码器
    SetLog(logger).                // 设置日志适配器
    SetGenerator(generator).       // 设置 Token 生成器
    SetPool(pool).                 // 设置协程池
    Build()
```

## 可插拔组件

| 组件 | 可选实现 | 模块路径 |
|------|---------|---------|
| 存储 | Memory, Redis | `com/storage/memory`, `com/storage/redis` |
| 编解码 | JSON, JSON v2, MessagePack, Base64 | `com/codec/*` |
| 日志 | DLog（默认）, GoFrame 适配, Nop | `com/log/*` |
| Token 生成器 | UUID, JWT | `com/generator/dgenerator` |
| 协程池 | Ants | `com/pool/ants` |

## Web 框架集成

| 框架 | 模块路径 |
|------|---------|
| Gin | `integrations/gin` |
| Fiber | `integrations/fiber` |
| GoFrame | `integrations/gf` |
| Go-Zero | `integrations/zero` |

每个集成模块提供：
- 上下文注册中间件
- 登录认证中间件
- 角色校验中间件
- 权限校验中间件
- 注解式路由校验（CheckLogin / CheckRole / CheckPermission / CheckAll）

## 多认证体系

通过 `authType` 参数支持多套独立认证体系并行运行：

```go
// 创建用户认证管理器
userMgr := builder.NewBuilder().AuthType("user").SetStorage(storage).Build()
dtoken.SetManager(userMgr)

// 创建管理员认证管理器
adminMgr := builder.NewBuilder().AuthType("admin").SetStorage(storage).Build()
dtoken.SetManager(adminMgr)

// 使用时指定 authType
userToken, _ := dtoken.Login(ctx, "user001", "", "", "user")
adminToken, _ := dtoken.Login(ctx, "admin001", "", "", "admin")
```

## License

MIT
