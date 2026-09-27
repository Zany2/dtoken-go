# 框架集成使用指南

**[English](../integration/framework-integration.md)**

## 概览

DToken-Go 会区分核心认证 API 和框架集成包。登录、登出、会话、权限、角色以及全局 Manager API 使用 `dtoken`；带默认组件的 Manager 构建使用 `defaults`；`integrations/*` 只负责框架中间件、注解式检查和请求上下文辅助方法。

各框架的上下文注册与鉴权中间件遵循同一 Manager 选择顺序：显式 `WithManager`、显式 `WithAuthType`、已有请求 Manager、全局默认实例。未指定实例的重复注册会保留请求 Manager。请求上下文不存在（包括带类型的 nil）时允许回退；已有请求上下文中的 Manager 为空或已关闭时返回 `ErrManagerNotFound`，不会隐式切换到全局实例。显式选择仍可覆盖已有请求作用域。

## 安装

```bash
go get github.com/Zany2/dtoken-go/defaults
go get github.com/Zany2/dtoken-go/integrations/gin
go get github.com/Zany2/dtoken-go/com/storage/memory
```

其他集成包同理：

```bash
go get github.com/Zany2/dtoken-go/integrations/echo
go get github.com/Zany2/dtoken-go/integrations/fiber
go get github.com/Zany2/dtoken-go/integrations/chi
go get github.com/Zany2/dtoken-go/integrations/gf
go get github.com/Zany2/dtoken-go/integrations/hertz
go get github.com/Zany2/dtoken-go/integrations/kratos
go get github.com/Zany2/dtoken-go/integrations/beego
```

## 最小框架接入示例

所有框架包都会转发常用 `dtoken` 门面 API，并提供对应框架的中间件。下面是登录接口加受保护接口的最小接入方式，完整可运行示例位于 `examples/gf`、`examples/gin`、`examples/echo` 和 `examples/fiber`。

### GoFrame

```go
ctx := context.Background()
mgr, err := gfdt.NewBuilder().IsPrintBanner(false).Build()
if err != nil {
    panic(err)
}
gfdt.SetManager(mgr)

s := g.Server()
s.Use(gfdt.RegisterDTokenContextMiddleware(ctx))
s.Group("/", func(group *ghttp.RouterGroup) {
    group.POST("/login", func(r *ghttp.Request) {
        token, _ := gfdt.Login(r.Context(), "1000")
        r.Response.WriteJson(g.Map{"token": token})
    })
    group.Group("/", func(group *ghttp.RouterGroup) {
        group.Middleware(gfdt.AuthMiddleware(ctx))
        group.GET("/me", func(r *ghttp.Request) {
            dCtx, _ := gfdt.GetDTokenContext(r)
            loginID, _ := dCtx.Auth().GetLoginID(r.Context())
            r.Response.WriteJson(g.Map{"loginId": loginID})
        })
    })
})
```

### Gin

Gin 鉴权中间件、注解检查及鉴权钩子使用当前 `c.Request.Context()`，保留请求上下文值、截止时间和取消信号；注册函数的 `ctx` 参数保留以兼容现有调用。

钩子替换 `c.Request` 后，后续鉴权读取替换后的请求上下文。钩子的 `req.Next()` 与 `req.Exit()` 只执行首次决定，后续重复调用或相反选择不会再次推进或中止处理链。

Query 和 Body Token 同样从当前请求读取，不沿用替换前 Gin 缓存中的凭据。Body Token 仅使用表单字段，multipart 解析仍遵循 Gin 的 `MaxMultipartMemory` 配置。

通过 `RegisterDTokenContextMiddleware(ctx, WithManager(mgr))` 注入 Manager 后，未指定 Manager 或 `AuthType` 的鉴权中间件会沿用请求中的实例。显式 `WithManager` 优先，其次是 `WithAuthType` 指定的注册实例；没有请求缓存时才使用全局默认实例。`...ByContext` 方法也使用请求中的 Manager，并拒绝已关闭的实例。

鉴权失败时会先中止 Gin 处理链，再执行失败回调，因此失败回调中的 `c.Next()` 不会进入业务处理器。前置鉴权钩子或路由规则钩子调用 `c.Abort()` 后，鉴权流程直接结束。

```go
ctx := context.Background()
mgr, err := gindt.NewBuilder().IsPrintBanner(false).Build()
if err != nil {
    panic(err)
}
gindt.SetManager(mgr)

r := gin.Default()
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
r.POST("/login", func(c *gin.Context) {
    token, _ := gindt.Login(c.Request.Context(), "1000")
    c.JSON(http.StatusOK, gin.H{"token": token})
})

auth := r.Group("/")
auth.Use(gindt.AuthMiddleware(ctx))
auth.GET("/me", func(c *gin.Context) {
    dCtx, _ := gindt.GetDTokenContext(c)
    loginID, _ := dCtx.Auth().GetLoginID(c.Request.Context())
    c.JSON(http.StatusOK, gin.H{"loginId": loginID})
})
```

### Echo

```go
ctx := context.Background()
mgr, err := echodt.NewBuilder().IsPrintBanner(false).Build()
if err != nil {
    panic(err)
}
echodt.SetManager(mgr)

e := echo4.New()
e.Use(echodt.RegisterDTokenContextMiddleware(ctx))
e.POST("/login", func(c echo4.Context) error {
    token, _ := echodt.Login(c.Request().Context(), "1000")
    return c.JSON(http.StatusOK, echo4.Map{"token": token})
})

auth := e.Group("")
auth.Use(echodt.AuthMiddleware(ctx))
auth.GET("/me", func(c echo4.Context) error {
    dCtx, _ := echodt.GetDTokenContext(c)
    loginID, _ := dCtx.Auth().GetLoginID(c.Request().Context())
    return c.JSON(http.StatusOK, echo4.Map{"loginId": loginID})
})
```

### Fiber

```go
ctx := context.Background()
mgr, err := fiberdt.NewBuilder().IsPrintBanner(false).Build()
if err != nil {
    panic(err)
}
fiberdt.SetManager(mgr)

app := gofiber.New()
app.Use(fiberdt.RegisterDTokenContextMiddleware(ctx))
app.Post("/login", func(c *gofiber.Ctx) error {
    token, _ := fiberdt.Login(c.UserContext(), "1000")
    return c.JSON(gofiber.Map{"token": token})
})

auth := app.Group("")
auth.Use(fiberdt.AuthMiddleware(ctx))
auth.Get("/me", func(c *gofiber.Ctx) error {
    dCtx, _ := fiberdt.GetDTokenContext(c)
    loginID, _ := dCtx.Auth().GetLoginID(c.UserContext())
    return c.JSON(gofiber.Map{"loginId": loginID})
})
```
## Gin 示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/Zany2/dtoken-go/com/storage/memory"
    "github.com/Zany2/dtoken-go/defaults"
    "github.com/Zany2/dtoken-go/dtoken"
    gindt "github.com/Zany2/dtoken-go/integrations/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    ctx := context.Background()
    storage := memory.NewStorage()

    mgr, err := defaults.NewBuilder().
        SetStorage(storage).
        TokenName("token").
        Timeout(2 * 60 * 60).
        Build()
    if err != nil {
        panic(err)
    }
    dtoken.SetManager(mgr)

    r := gin.Default()
    r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

    r.POST("/login", func(c *gin.Context) {
        token, err := dtoken.Login(ctx, "1000")
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"token": token})
    })

    user := r.Group("/user")
    user.Use(gindt.AuthMiddleware(ctx))
    user.GET("/info", func(c *gin.Context) {
        dCtx, _ := gindt.GetDTokenContext(c)
        token := dCtx.GetTokenValue()
        loginID, _ := dtoken.GetLoginID(ctx, token)
        c.JSON(http.StatusOK, gin.H{"loginId": loginID})
    })

    _ = r.Run(":8080")
}
```

## 常用 API

核心认证 API 来自 `dtoken`：

```go
dtoken.SetManager(mgr)
mgr, err := dtoken.GetManager()

token, err := dtoken.Login(ctx, "1000")
ok := dtoken.IsLogin(ctx, token)
loginID, err := dtoken.GetLoginID(ctx, token)
err = dtoken.Logout(ctx, token)

err = dtoken.AddPermissions(ctx, "1000", []string{"user:read"})
hasPermission := dtoken.HasPermission(ctx, "1000", "user:read")

err = dtoken.AddRoles(ctx, "1000", []string{"admin"})
hasRole := dtoken.HasRole(ctx, "1000", "admin")
```

框架集成包提供中间件：

```go
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
r.Use(gindt.AuthMiddleware(ctx))
r.Use(gindt.PermissionMiddleware(ctx, []string{"user:read"}))
r.Use(gindt.RoleMiddleware(ctx, []string{"admin"}))
```

## 中间件选项

各框架集成包都提供类似的中间件选项，用来控制认证体系、权限逻辑和失败返回。

### 自定义失败返回

默认失败返回适合快速验证。正式项目通常建议通过 `WithFailFunc` 统一业务错误结构：

```go
failFunc := gindt.WithFailFunc(func(c *gin.Context, err error) {
    c.JSON(http.StatusUnauthorized, gin.H{
        "code":    401,
        "message": err.Error(),
    })
})

r.Use(gindt.AuthMiddleware(ctx, failFunc))
```

权限、角色中间件也可以使用同一个失败处理：

```go
r.GET("/admin",
    gindt.RoleMiddleware(ctx, []string{"admin"}, failFunc),
    adminHandler,
)
```

### 指定认证体系

多认证体系场景下，可以通过 `WithAuthType` 让不同路由组使用不同 Manager：

```go
userGroup := r.Group("/api")
userGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuthType("user")))

adminGroup := r.Group("/admin")
adminGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuthType("admin")))
adminGroup.Use(gindt.PermissionMiddleware(ctx, []string{"admin:read"}, gindt.WithAuthType("admin")))
```

如果应用不使用全局 Manager 注册表，可以通过 `WithManager` 直接注入 Manager。显式注入的 Manager 优先于 `WithAuthType`，并适用于上下文、认证、访问、权限和角色中间件：

```go
auth.Use(gindt.RegisterDTokenContextMiddleware(ctx, gindt.WithManager(mgr)))
auth.Use(gindt.AuthMiddleware(ctx, gindt.WithManager(mgr)))
```

当注解的 `AuthType` 为空时，注解处理器会复用请求上下文中已经保存的 Manager；显式填写 `AuthType` 时，仍通过全局注册表解析，适合多认证体系路由。

### 权限和角色逻辑

权限、角色中间件支持不同逻辑类型。常见用法是：

```go
r.GET("/reports",
    gindt.PermissionMiddleware(
        ctx,
        []string{"report:read", "report:export"},
        gindt.WithLogicType(gindt.LogicAnd),
    ),
    reportHandler,
)

r.GET("/console",
    gindt.RoleMiddleware(
        ctx,
        []string{"admin", "operator"},
        gindt.WithLogicType(gindt.LogicOr),
    ),
    consoleHandler,
)
```

`LogicAnd` 表示必须全部满足，`LogicOr` 表示满足任意一个即可。

## Chi 请求处理

通过 `RegisterDTokenContextMiddleware(WithManager(mgr))` 注册后，Chi 鉴权中间件未显式指定 Manager 或认证类型时，会继承请求中的 Manager。`WithManager` 优先级最高；显式 `WithAuthType` 则从注册表选择 Manager。鉴权中间件和注解处理器会将上下文绑定到当前请求和响应写入器，包括上游中间件做出的修改。替换请求或包装响应写入器的中间件应放在业务处理器使用的 DToken 中间件之前。

`GetPostForm` 只读取请求体字段，因此关闭 `IsReadQuery` 后，即使启用了 `IsReadBody`，也不会接受仅在查询参数中的 Token。`GetClientIP` 读取 `RemoteAddr`，不会直接信任 `X-Real-IP` 或 `X-Forwarded-For`。代理部署应在 DToken 之前使用中间件验证可信代理地址，并规范化 `RemoteAddr`。

在 Chi 鉴权钩子中，`req.Next()` 最多执行一次后续处理，`req.Exit()` 在自定义处理后终止流程。对已注册的请求适配器调用 `Abort()` 同样会终止 DToken 处理链。显式传入空注解会返回参数错误响应。

## Echo 请求处理

Echo 鉴权中间件、注解处理器和钩子使用 `c.Request().Context()` 获取请求值、截止时间与取消信号。注册处理器时的 `ctx` 参数保留以兼容现有 API。通过 `WithManager(mgr)` 注册后，未显式指定 Manager 或认证类型的鉴权会继承请求 Manager。`WithManager` 优先于 `WithAuthType`，显式认证类型则从注册表选择 Manager。上下文门面操作（包括续期）使用请求中的 Manager。

Token 提取在 `SetRequest` 后读取当前请求 URL，表单字段只从请求体读取。关闭 `IsReadQuery` 后，即使启用了请求体读取，也不会接受仅来自查询参数的 Token。钩子中的 `req.Next()` 最多调用一次后续处理并保留其返回错误；`req.Exit()` 或对已注册请求适配器调用 `Abort()` 会停止 DToken 处理。显式空注解会触发参数错误处理。

客户端 IP 遵循 Echo 的 `IPExtractor`。直接接收客户端连接时，配置 `e.IPExtractor = echo.ExtractIPDirect()`；通过代理部署时，配置带有适当信任规则的代理提取器。Echo 未配置时的旧策略会信任转发头，因此用客户端 IP 做安全判断前应显式配置该策略。

## Fiber 请求处理

Fiber 鉴权、注解、钩子和上下文门面使用 `c.UserContext()`。需要传递请求值、截止时间和取消信号时，在上游中间件通过 `c.SetUserContext(ctx)` 设置标准上下文；未设置时 Fiber 返回 `context.Background()`，客户端断开连接不会自动取消该上下文。注册时的 `ctx` 参数保留以兼容现有 API。`Locals` 仍可通过请求适配器访问，不会自动合并到 `UserContext`。

鉴权默认继承已注册的请求 Manager，显式 `WithManager` 或 `WithAuthType` 可以覆盖选择，其中 `WithManager` 优先。上下文门面的续期也使用该请求 Manager。请求头、查询参数、Cookie 和表单中的 Token 值会被复制，避免异步维护任务持有 Fiber 可复用的请求缓冲区。请求体 Token 提取不包含查询参数，multipart 表单同样遵循这一规则。`GetBody` 返回原始请求体的副本，不自动解压，也不会将解压错误文本当作正文返回。

前置钩子使用 `req.Next()` 放行一次并保留下游错误，或使用 `req.Exit()` 在自定义处理后终止。对已注册请求适配器调用 `Abort()`，DToken 入口和钩子会遵循该状态。Fiber 没有中止整条处理链的原生机制，因此失败回调只应输出响应，不应调用 `c.Next()`。显式空注解进入参数错误处理流程。客户端 IP 和代理协议识别继续遵循 Fiber 配置的代理信任策略。

## GoFrame 请求处理

GoFrame 鉴权中间件、注解处理器及钩子使用当前 `r.Context()`，保留请求上下文值、截止时间和取消信号；注册时的 `ctx` 参数保留以兼容现有调用。通过 `RegisterDTokenContextMiddleware(ctx, WithManager(mgr))` 注入实例后，未显式指定 Manager 或认证类型的检查会继承请求 Manager。显式 `WithManager` 优先，其次是 `WithAuthType`；没有请求实例时使用全局默认实例。上下文门面（包括续期）使用请求 Manager，并拒绝缓存中的空实例或已关闭实例。

鉴权失败时，在执行失败回调前通过 `ExitAll()` 标记整条 GoFrame 处理链已退出，因此回调中的 `r.Middleware.Next()` 无法继续执行受保护处理器。默认失败响应使用 JSON 和对应的 HTTP 状态码。钩子的 `req.Next()` 最多放行一次，`req.Exit()` 退出处理链。调用已注册适配器的 `Abort()` 会停止 DToken 处理，但不会立即退出当前 Go 函数；写完响应后应返回，不要再调用原生 `r.Middleware.Next()`。显式传入空注解会返回参数错误。

响应头写入 `r.Response`，设置状态码不会向响应体追加状态文本。`GetBody()` 保留 GoFrame 的重复读取能力，并将框架的请求体读取异常转换为错误返回。Body Token 读取保留 GoFrame `GetForm` 的语义，包括其支持的 JSON/XML 请求体，不会回退到查询参数。GoFrame 的 `GetClientIp()` 不验证代理地址就信任转发头，因此适配器的 `GetClientIP()` 改为使用连接的 `RemoteAddr`。部署在代理后方时，应在 DToken 之前通过上游中间件验证可信代理地址并规范化 `RemoteAddr`。

## Hertz 请求处理

Hertz 鉴权、注解及钩子使用当前处理器收到的标准 `context.Context`，保留上下文值、截止时间和取消信号；注册时的 `ctx` 参数保留以兼容现有调用。DToken 中间件和注解还会为下游 `...ByContext` 门面绑定该标准上下文，嵌套调用返回后恢复外层绑定。应将 DToken 中间件放在派生标准上下文的中间件之后；如果后续处理器又派生新上下文且没有再进入 DToken 中间件或注解，请显式传给 `dCtx.Auth()` 等上下文门面分组。没有 DToken 绑定时，`...ByContext` 保留 `context.Background()` 兜底行为。Hertz 的请求键值不会自动合并到标准上下文，该绑定也不会自行产生客户端断开连接的取消信号。

通过 `RegisterDTokenContextMiddleware(ctx, WithManager(mgr))` 注入实例后，未显式选择实例的鉴权会继承请求 Manager。显式 `WithManager` 优先于 `WithAuthType`，没有请求实例时使用全局默认实例。上下文门面拒绝缓存中的空 Manager 或已关闭 Manager。鉴权失败时先 `Abort()` 再执行失败回调，回调中的 `Next()` 无法进入受保护处理器。钩子的 `req.Next()` 最多放行一次，原生或适配器 `Abort()` 都会停止 DToken 检查。显式空注解返回参数错误。

Cookie 读取会解码 Hertz 写入时的转义，使包含 `+`、`/`、`%` 的自定义 Token 能正确往返。Body Token 只读取表单字段（含 multipart），不会回退到 Query。`IsTLS()` 依据实际连接是否实现 Hertz TLS 接口判断，URL 协议名或转发头不能证明连接已经加密。

客户端 IP 沿用 Hertz 配置的 `ClientIP` 函数。其默认配置会信任所有代理地址；直接接入时可使用 `h.SetClientIPFunc(app.ClientIPWithOption(app.ClientIPOptions{}))`，通过代理接入时应显式配置 `TrustedCIDRs` 和 `RemoteIPHeaders`。

## Kratos 请求处理

Kratos HTTP 与一元 gRPC 鉴权中间件、注解、钩子和上下文门面使用当前处理器的 `context.Context`，保留值、截止时间及取消信号。通过 `RegisterDTokenContextMiddleware(WithManager(mgr))` 注册后，未显式选择实例的检查继承请求 Manager。优先级依次为显式 `WithManager`、`WithAuthType`、请求缓存、全局默认实例。上下文门面拒绝缓存中的空 Manager 或已关闭 Manager，续期也使用所选实例。切换 Manager 时保留请求适配器中的值及中止状态。

钩子的 `req.Next()` 最多调用一次下游处理器，并保留其返回值和错误；`req.Exit()` 在自定义处理后退出。DToken 入口及钩子遵守已注册适配器的 `Abort()` 状态。失败分发会先将适配器标为中止，再调用 `FailFunc`；回调返回的错误交给 Kratos，返回 nil 表示回调已自行处理响应，两种情况都不会执行受保护处理器。默认失败保留原始错误链，并转换为 Kratos 的 HTTP/gRPC 状态错误。显式空注解返回参数错误；未传注解仍直接放行。默认 AND 逻辑只应用到请求内副本，不修改共享路由注解配置。

HTTP 请求和 Cookie 操作支持 Kratos 的公开传输接口，包括传输包装器。替换传输对象的中间件应放在 DToken 注册之前，让适配器捕获所需传输对象。Body Token 支持 URL 编码和 multipart 表单字段，不回退到 Query。Cookie 写入保持标准 `net/http` 的值及 SameSite 行为。Cookie 和原始 HTTP 响应写入仅适用于 HTTP；gRPC 使用请求及响应元数据，路径权限检查采用 RPC 的操作名。

HTTP 客户端 IP 使用 `RemoteAddr`，gRPC 使用连接对端地址，不自动信任转发头。HTTP 代理部署应在上游校验可信代理并规范化 `RemoteAddr`。TLS 判断依据 HTTP 请求的 TLS 状态或 gRPC 对端 TLS 凭据，不依据转发头或 URL 协议名。

流式 RPC 需要单独接入鉴权：Kratos v2.9.1 中，仅通过 `grpc.Middleware` 注册这些处理器不会在建立流时执行鉴权。`grpc.StreamMiddleware` 作用于逐条消息的发送和接收；若要求在进入流式服务处理器前鉴权，应使用流拦截器。

## Beego 请求处理

Beego 鉴权过滤器及钩子使用当前 `c.Request.Context()`，保留请求值、截止时间和取消信号；注册时的 `ctx` 参数保留以兼容现有 API。通过 `RegisterDTokenContextMiddleware(ctx, WithManager(mgr))` 注册后，未显式指定 Manager 或认证类型的过滤器继承请求 Manager。优先级依次为显式 `WithManager`、`WithAuthType`、请求缓存、全局默认实例。上下文门面拒绝缓存中的空 Manager 或已关闭 Manager，续期使用请求 Manager。

鉴权失败会先标记 `ResponseWriter.Started`，再调用 `FailFunc`。即使回调不输出响应，原生过滤器链也会停止；回调仍可写入自己的状态码和响应体。适配器 `Abort()` 和钩子 `req.Exit()` 使用相同的原生状态，上游钩子已输出响应时，DToken 过滤器也会停止。Beego 过滤器没有嵌套的 `Next` 调用：`req.Next()` 仅跳过当前 DToken 检查，后续过滤器仍由 Beego 分发。`Next()` 和 `Exit()` 只遵循第一次选择。

鉴权过滤器应注册在受保护处理器之前，并保留 Beego 默认的 `WithReturnOnOutput(true)`。`RegisterAuthFilter`、`RegisterAccessFilter`、`RegisterPermissionFilter` 和 `RegisterRoleFilter` 均在 `BeforeRouter` 显式设置此选项。设为 false 可能使 Beego 在拒绝请求后仍继续进入控制器。静态资源可能在 `BeforeRouter` 之前已被响应，需要保护时应配置适当的 `BeforeStatic` 过滤器。

Query Token 仅读取 URL 查询参数，Body Token 仅读取 URL 编码或 multipart 表单字段，两者不相互回退，也不使用路由参数。客户端 IP 来自 `RemoteAddr`；代理部署应在上游校验可信代理并规范化 `RemoteAddr`。TLS 判断使用 HTTP 请求的真实 TLS 状态。`SetStatusCode` 后调用适配器 `Write` 会提交 Beego 暂存的状态码。`GetBody` 保留 Beego 的缓存正文语义：存在 `Input.RequestBody` 时返回副本，否则读取并恢复当前请求体；正文可能已由 Beego 按 `CopyRequestBody` 配置处理。

`RouteAccessHandlerFromAnnotations(nil)` 通过配置的 `FailFunc` 返回参数错误。未传注解时保留 `AccessMiddleware` 的登录检查；空注解或 `Ignore: true` 跳过鉴权，保持 Beego 原有注解策略。

## 推荐别名

| 框架 | 导入路径 | 推荐别名 |
| --- | --- | --- |
| Gin | `github.com/Zany2/dtoken-go/integrations/gin` | `gindt` |
| Echo | `github.com/Zany2/dtoken-go/integrations/echo` | `echodt` |
| Fiber | `github.com/Zany2/dtoken-go/integrations/fiber` | `fiberdt` |
| Chi | `github.com/Zany2/dtoken-go/integrations/chi` | `chidt` |
| GoFrame | `github.com/Zany2/dtoken-go/integrations/gf` | `gfdt` |
| Hertz | `github.com/Zany2/dtoken-go/integrations/hertz` | `hertzdt` |
| Kratos | `github.com/Zany2/dtoken-go/integrations/kratos` | `kratosdt` |
| Beego | `github.com/Zany2/dtoken-go/integrations/beego` | `beegodt` |

## 什么时候直接导入 core

如果你在编写与框架无关的基础设施、测试、共享库或业务 Handler，可以导入 `defaults`、`core/builder`、`core/manager` 或 `dtoken`。只有想自己注入全部适配器时才直接使用 `core/builder`。只有需要框架请求处理时，才导入对应的集成包。

## 相关文档

- [DToken API](../../api/dtoken_zh.md)
- [登录认证](../core/authentication_zh.md)
- [注解使用](../integration/annotation_zh.md)
- [多认证体系](../core/multi-auth_zh.md)
- [AccessProvider](../core/access-provider_zh.md)
