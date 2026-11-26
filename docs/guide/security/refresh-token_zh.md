[English](../security/refresh-token.md) | 中文文档

# Refresh Token 指南

本文以根包 Auth 实例为主；dt 是根包别名。先通过 dt.NewBuilder().Build() 构建 auth，并在使用结束后关闭。以下操作片段放在持有该实例的函数中，ctx 使用当前请求上下文。

## 概览

DToken-Go 在两个地方支持 Refresh Token：

- 普通业务登录：`LoginWithRefreshToken(...)`
- OAuth2 令牌流程：`RefreshOAuth2AccessToken(...)`

本文主要说明普通业务登录流程。

## 登录并返回双令牌

```go
pair, err := auth.LoginWithRefreshTokenOptions(ctx, dt.RefreshTokenOptions{
    LoginOptions: dt.LoginOptions{LoginID: "user-1001", Device: "web", DeviceID: "browser-1"},
})
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
pair, err := auth.LoginWithRefreshTokenOptions(ctx, dt.RefreshTokenOptions{
	LoginOptions: dt.LoginOptions{
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
nextPair, err := auth.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}
```

刷新是一次轮换操作：

1. 校验 refresh token
2. 拒绝已封禁账号或已封禁设备；原访问生命周期已不活跃超时时，执行超时下线并撤销刷新凭据
3. 一次性消费旧 refresh token，仅移除属于它的反向索引
4. 签发新的 access token 和 refresh token；仅替换仍挂载的原终端时跳过重复并发顶替，否则按当前账号或设备作用域执行登录数量及非并发策略
5. 确认旧 access token 仍属于原登录生命周期后，再将其下线

如果新令牌对签发失败，旧 refresh token 已被消费，但不会主动登出旧 access token。

Refresh token 不依赖旧 access token 的 TTL。即使 access token 已自然过期，只要 refresh token 仍有效且当前封禁、登录策略允许，仍可刷新。原访问记录仍存在且已不活跃超时时，直接刷新与先鉴权都会使原凭据失效；超时事件在释放账号锁后触发。

轮换会分别保留登录时的 `Extra`（Token 扩展数据）和 `TerminalExtra`（终端扩展数据），即使旧 access token 和 Session 已过期也不受影响。终端数据作为可选字段保存在内部刷新记录中，现有公开类型和存储键不变。旧记录缺少该字段时仍可刷新，但不会携带终端扩展数据。

新登录会在内部访问记录中生成随机生命周期标识，并将其保存到刷新记录；续期和共享登录保留原标识。即使在同一秒内复用相同的 access token 文本，新登录也有不同标识：轮换或撤销旧 refresh token 不会删除新登录、其附属元数据或刷新绑定。清理在账号锁内重新核对标识；访问记录已不存在或已失效时，跳过访问侧清理。过期终端条目仍由正常 Session 过期或终端清理处理。

新登录复用已到期的 access token 文本时，会解除旧反向索引，但保留旧生命周期尚未到期的独立刷新凭据。新登录的签发或登出不会通过旧索引误撤销该凭据；旧凭据后续刷新时仍须遵守当前登录策略。

旧令牌对必须先通过 PrepareAuthUpgrade 转换，再切换到版本化空间。离线转换按账号、设备和创建时间尽力匹配匿名令牌对并分配生命周期标识，无法重建旧版本从未记录的身份。运行时访问记录必须包含非空生命周期标识和当前 AuthType，旧 Token 载荷直接拒绝，不回退 Codec 解码；刷新元数据仍使用 Codec。同一逻辑认证体系的写入节点须统一切换。本机制不引入分布式事务，也不要求自定义存储提供 CAS 能力。

## 撤销流程

```go
err := auth.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

撤销 refresh token 时，只有生命周期绑定仍匹配，才会同时登出原 access token；后续复用同一 Token 值的新登录不受影响。

## 存储开销与消费边界

刷新元数据包含独立有效期、设备和扩展数据，仍通过 Codec 解码。以下仅统计成功路径中**旧 refresh token 主键**的读取、消费及其解码次数，不包含封禁、访问令牌、Session、反向索引和新令牌签发：

| 操作 | 普通 Storage | 支持 AtomicStorage | 刷新记录解码次数（普通 / 原子） |
| --- | --- | --- | --- |
| 轮换 | 2 GET + 1 Delete | 2 GET + 1 GetAndDelete | 2 / 3 |
| 撤销 | 2 GET + 1 Delete | 1 GET + 1 GetAndDelete | 2 / 2 |

第一次读取用于定位账号锁。轮换在锁内重新读取刷新记录，检查当前限制及原访问状态后再消费；其访问状态检查复用本次已读取的主记录，不再重复 GET。普通存储消费直接复用锁内刷新记录，省去第三次 GET 和解码。原子撤销直接核对 GetAndDelete 返回的记录，省去一次中间复读；原子消费结果的账号、访问令牌、生命周期标识和创建时间必须与预期一致，才能继续处理关联数据。

这些复用仅限单次操作，不缓存跨请求认证结果，也不跨越登录回调复用签发状态。普通存储依靠同一 Manager 的账号锁协调本地请求，不能保证跨 Manager 或跨进程的严格一次性消费；需要该保证时应实现 AtomicStorage。多个键的后续更新仍按顺序执行，不构成分布式事务。

## 有效期

```go
ttl, err := auth.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
```

默认 refresh token 有效期是 `30` 天。可以设置全局有效期：

```go
auth, err := dt.NewBuilder().
	RefreshTokenTimeout(30 * 24 * 60 * 60).
	Build()
```

也可以使用 `time.Duration`：

```go
auth, err := dt.NewBuilder().
	RefreshTokenTimeoutDuration(30 * 24 * time.Hour).
	Build()
```

## 全局与框架入口

Build 已自动注册实例，可通过根包选择器调用；以下 auth 是已构建的实例：

```go
selected := dt.AuthType(auth.Config().AuthType)
pair, err := selected.LoginWithRefreshTokenOptions(ctx, dt.RefreshTokenOptions{LoginOptions: dt.LoginOptions{LoginID: "user-1001"}})
if err != nil {
	return err
}

nextPair, err := selected.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}

ttl, err := selected.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
if err != nil {
	return err
}
fmt.Println(ttl)

if err := selected.RevokeRefreshToken(ctx, nextPair.RefreshToken); err != nil {
	return err
}
```

GoFrame 中先通过 WithAuth(auth) 绑定请求实例，再使用 gfdt（integrations/gf）的请求快捷方法；这些方法不查询全局注册表：

```go
func (c *AuthController) Login(r *ghttp.Request) {
	pair, err := gfdt.LoginWithRefreshTokenByCtx(r.Context(), "user-1001", "web", "browser-1")
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": err.Error()})
	}
	r.Response.WriteJsonExit(pair)
}

func (c *AuthController) Refresh(r *ghttp.Request) {
	pair, err := gfdt.RefreshTokenByCtx(r.Context(), r.Get("refreshToken").String())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": err.Error()})
	}
	r.Response.WriteJsonExit(pair)
}
```

## OAuth2 中的 Refresh Token

OAuth2 刷新通过独立 OAuth2 Service 调用。下例的 service 由 oauth2.NewService(server, auth) 创建，初始化见 OAuth2 指南；这是能力服务方法：

```go
newToken, err := service.RefreshOAuth2AccessToken(
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
