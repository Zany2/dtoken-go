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

登录回调先于刷新令牌签发执行。签发会在账号锁内重新检查原登录身份和有效会话。如果回调中登出并复用同值 access token 创建了其他登录，外层调用会失败，不会为替代登录绑定刷新令牌。失败清理也仅作用于原生命周期，不会撤销后续登录。

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
3. 一次性消费旧 refresh token，仅移除属于它的反向索引
4. 签发新的 access token 和 refresh token，不重复执行常规并发顶替
5. 确认旧 access token 仍属于原登录生命周期后，再将其下线

如果新令牌对签发失败，旧 refresh token 已被消费，但不会主动登出旧 access token。

Refresh token 不依赖旧 access token 的 TTL。只要 refresh token 仍有效，即使 access token 已过期，也可以刷新成功。

轮换会分别保留登录时的 `Extra`（Token 扩展数据）和 `TerminalExtra`（终端扩展数据），即使旧 access token 和 Session 已过期也不受影响。终端数据作为可选字段保存在内部刷新记录中，现有公开类型和存储键不变。旧记录缺少该字段时仍可刷新，但不会携带终端扩展数据。

新登录会在内部访问记录中生成随机生命周期标识，并将其保存到刷新记录；续期和共享登录保留原标识。即使在同一秒内复用相同的 access token 文本，新登录也有不同标识：轮换或撤销旧 refresh token 不会删除新登录、其附属元数据或刷新绑定。清理在账号锁内重新核对标识；访问记录已不存在或已失效时，跳过访问侧清理。过期终端条目仍由正常 Session 过期或终端清理处理。

没有生命周期标识的旧令牌对仍可读取和刷新。两个旧记录之间的访问清理按账号、设备和创建时间尽力匹配，无法可靠识别升级前已经发生的同值 Token 复用；旧刷新记录绝不匹配新格式访问记录。共享存储的写入节点应统一升级，旧版本重写 Token 元数据可能丢失标识。本机制不引入分布式事务，也不要求自定义存储提供 CAS 能力。

## 撤销流程

```go
err := dtoken.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

撤销 refresh token 时，只有生命周期绑定仍匹配，才会同时登出原 access token；后续复用同一 Token 值的新登录不受影响。

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
