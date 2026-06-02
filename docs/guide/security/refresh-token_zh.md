[English](../security/refresh-token.md) | 中文文档

# Refresh Token 指南

## 概览

DToken-Go 在两个地方支持 Refresh Token：

- 普通业务登录：`LoginWithRefreshToken(...)`
- OAuth2 令牌流程：`RefreshOAuth2AccessToken(...)`

本文主要说明普通业务登录流程。

## 登录并返回双令牌

```go
pair, err := dtoken.LoginWithRefreshToken(ctx, "user-1001", "web", "browser-1")
if err != nil {
	return err
}

fmt.Println(pair.AccessToken)
fmt.Println(pair.RefreshToken)
fmt.Println(pair.ExpiresIn)
fmt.Println(pair.RefreshExpiresIn)
```

`AccessToken` 用于访问受保护接口。`RefreshToken` 由客户端保存，只用于换取新的令牌对。

## 使用选项登录

如果单次登录需要设置自定义有效期、设备、扩展数据或并发登录策略，可以使用 `LoginWithRefreshTokenOptions(...)`。

```go
pair, err := dtoken.LoginWithRefreshTokenOptions(ctx, dtoken.RefreshTokenOptions{
	LoginOptions: dtoken.LoginOptions{
		LoginID: "user-1001",
		Device:  "app",
		Extra: map[string]any{
			"tenant": "main",
		},
	},
	RefreshTimeout: 30 * 24 * time.Hour,
})
```

## 刷新流程

```go
nextPair, err := dtoken.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}
```

刷新是一次轮换操作：

1. 校验 refresh token
2. 拒绝已封禁账号或已封禁设备
3. 撤销旧 access token 和旧 refresh token
4. 签发新的 access token 和 refresh token

Refresh token 不依赖旧 access token 的 TTL。只要 refresh token 仍有效，即使 access token 已过期，也可以刷新成功。

## 撤销流程

```go
err := dtoken.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

撤销 refresh token 时，会同时登出它关联的 access token。

## 有效期

```go
ttl, err := dtoken.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
```

默认 refresh token 有效期是 `30` 天。可以设置全局有效期：

```go
mgr, err := dtoken.NewBuilder().
	RefreshTokenTimeout(30 * 24 * 60 * 60).
	Build()
```

也可以使用 `time.Duration`：

```go
mgr, err := dtoken.NewBuilder().
	RefreshTokenTimeoutDuration(30 * 24 * time.Hour).
	Build()
```

## 框架门面示例

各框架包会导出这些 API。例如 GoFrame：

```go
import gfdt "github.com/Zany2/dtoken-go/integrations/gf"

pair, err := gfdt.LoginWithRefreshToken(ctx, "user-1001")
nextPair, err := gfdt.RefreshToken(ctx, pair.RefreshToken)
ttl, err := gfdt.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
_ = gfdt.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

在 GoFrame 控制器中，登录与刷新接口也可以只使用同一个框架包：

```go
func (c *AuthController) Login(r *ghttp.Request) {
	pair, err := gfdt.LoginWithRefreshToken(r.Context(), "user-1001", "web", "browser-1")
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": err.Error()})
	}
	r.Response.WriteJsonExit(pair)
}

func (c *AuthController) Refresh(r *ghttp.Request) {
	pair, err := gfdt.RefreshToken(r.Context(), r.Get("refreshToken").String())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": err.Error()})
	}
	r.Response.WriteJsonExit(pair)
}
```

## OAuth2 中的 Refresh Token

OAuth2 的 refresh token 能力仍然通过 OAuth2 API 使用：

```go
newToken, err := dtoken.RefreshOAuth2AccessToken(
	ctx,
	"web-app",
	oldToken.RefreshToken,
	"secret",
)
```

OAuth2 专属行为见 [OAuth2 指南](../security/oauth2_zh.md)。

## 相关文档

- [OAuth2 指南](../security/oauth2_zh.md)
- [登录认证](../core/authentication_zh.md)
- [高级能力](../security/advanced-features_zh.md)
