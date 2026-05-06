# 单包导入使用指南

**[English](single-import.md)**

## 概览

DToken-Go 的框架集成包会重新导出常用 core 类型、Builder 辅助方法、错误、常量以及全局 `dtoken` API。如果你已经在使用某个框架集成包，通常只需要导入该集成包和所选存储包。

这样业务代码可以集中使用一个 DToken 命名空间，例如 Gin 项目中使用 `gindt`。

## 安装

```bash
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
```

## Gin 示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/Zany2/dtoken-go/com/storage/memory"
    gindt "github.com/Zany2/dtoken-go/integrations/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    ctx := context.Background()
    storage := memory.NewStorage()

    mgr := gindt.NewDefaultBuilder().
        SetStorage(storage).
        TokenName("token").
        Timeout(2 * 60 * 60).
        Build()
    gindt.SetManager(mgr)

    r := gin.Default()
    r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

    r.POST("/login", func(c *gin.Context) {
        token, err := gindt.Login(ctx, "1000")
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
        loginID, _ := gindt.GetLoginID(ctx, token)
        c.JSON(http.StatusOK, gin.H{"loginId": loginID})
    })

    _ = r.Run(":8080")
}
```

## 常用 API

集成包提供和 `dtoken` 一致风格的快捷入口：

```go
gindt.SetManager(mgr)
mgr, err := gindt.GetManager()

token, err := gindt.Login(ctx, "1000")
ok := gindt.IsLogin(ctx, token)
loginID, err := gindt.GetLoginID(ctx, token)
err = gindt.Logout(ctx, token)

err = gindt.AddPermissions(ctx, "1000", []string{"user:read"})
hasPermission := gindt.HasPermission(ctx, "1000", "user:read")

err = gindt.AddRoles(ctx, "1000", []string{"admin"})
hasRole := gindt.HasRole(ctx, "1000", "admin")
```

也可以直接使用框架中间件：

```go
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
r.Use(gindt.AuthMiddleware(ctx))
r.Use(gindt.PermissionMiddleware(ctx, []string{"user:read"}))
r.Use(gindt.RoleMiddleware(ctx, []string{"admin"}))
```

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

## 什么时候直接导入 core

如果你在编写与框架无关的基础设施、测试或共享库，可以直接导入 `core/builder`、`core/manager` 或 `dtoken`。普通 Web Handler 中通常使用集成包就够了。

## 相关文档

- [DToken API](../api/dtoken_zh.md)
- [登录认证](authentication_zh.md)
- [注解使用](annotation_zh.md)
