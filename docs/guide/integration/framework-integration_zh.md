# 框架集成使用指南

**[English](../integration/framework-integration.md)**

## 概览

DToken-Go 会区分核心认证 API 和框架集成包。登录、登出、会话、权限、角色以及全局 Manager API 使用 `dtoken`；带默认组件的 Manager 构建使用 `defaults`；`integrations/*` 只负责框架中间件、注解式检查和请求上下文辅助方法。

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

    mgr := defaults.NewBuilder().
        SetStorage(storage).
        TokenName("token").
        Timeout(2 * 60 * 60).
        Build()
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
