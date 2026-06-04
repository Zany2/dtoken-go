# 高级能力

本页整理 DToken-Go 在普通登录、登出、权限和角色 API 之外可以直接使用的高级能力。

## Token Introspection

Token Introspection 用来无续期副作用地检查 token 当前是否活跃，并返回归属信息、TTL、权限、角色、扩展数据和非活跃原因。

```go
info, err := dtoken.IntrospectToken(ctx, token)
if err != nil {
	return err
}
if !info.Active {
	fmt.Println("invalid reason:", info.Error)
	return nil
}
fmt.Println(info.LoginID, info.ExpiresIn, info.Permissions, info.Roles)
```

各框架包也会导出同名 API。例如 GoFrame 项目只引入 `github.com/Zany2/dtoken-go/integrations/gf` 后，可以直接调用 `gfdt.IntrospectToken(...)`。

## Refresh Token

普通登录流程支持 access token + refresh token 双令牌。刷新时会轮换 refresh token：旧 access token 和旧 refresh token 会被撤销，然后签发一组新的令牌。

```go
pair, err := dtoken.LoginWithRefreshToken(ctx, "user-1001")
if err != nil {
	return err
}

nextPair, err := dtoken.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}

ttl, err := dtoken.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
if err != nil {
	return err
}
fmt.Println(ttl)

_ = dtoken.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

默认 refresh token 有效期是 `30` 天。可以通过 `RefreshTokenTimeout(...)` 设置全局有效期，也可以通过 `LoginWithRefreshTokenOptions(...)` 为单次登录覆盖有效期。

## Ticket 临时凭证

用于一次性票据、临时授权和系统间换票。

```go
createdTicket, err := dtoken.CreateTicket(ctx, "user-1001")
if err != nil {
	return err
}

result, err := dtoken.ConsumeTicket(ctx, createdTicket.Ticket)
if err != nil {
	return err
}
fmt.Println(result.Ticket.LoginID)
```

## Short-Key 访问凭证

用于短链访问、扫码确认、临时授权和系统间换票。

```go
createdKey, err := dtoken.CreateShortKey(ctx)
if err != nil {
	return err
}

confirmedKey, err := dtoken.ConfirmShortKey(ctx, createdKey.Key, "user-1001")
if err != nil {
	return err
}

result, err := dtoken.ConsumeShortKey(ctx, confirmedKey.Key)
if err != nil {
	return err
}
fmt.Println(result.ShortKey.LoginID)
```

## SSO 单点登录

用于统一登录、票据交换、跨系统登录态共享和统一登出。SSO 位于独立模块 `github.com/Zany2/dtoken-go/sso`，不绑定基础认证鉴权架构，只依赖存储和编解码适配器；当前已提供 Ticket、共享 Token、远程会话和 OAuth2 授权码模式原语，也提供 redirect、token exchange、introspection、revoke、userinfo、logout 和 callback 处理等 HTTP 服务封装。更多说明见 [SSO 单点登录](../../../sso/README_zh.md)。

```go
server := sso.NewServerWithConfig("sso:", "dtoken:", storage, codec, sso.DefaultConfig())

ticket, err := server.GenerateTicket(
	ctx,
	"app-a",
	"user-1001",
	"https://app.example.com/sso/callback",
	nil,
	nil,
)
if err != nil {
	return err
}

info, err := server.ConsumeTicket(
	ctx,
	ticket.Ticket,
	"app-a",
	"secret-a",
	"https://app.example.com/sso/callback",
)
if err != nil {
	return err
}
fmt.Println(info.LoginID)
```

## Nonce 防重放

用于一次性随机值生成、校验和消费，防止请求重放。

```go
nonce, err := dtoken.GenerateNonce(ctx)
if err != nil {
	return err
}

err = dtoken.VerifyAndConsumeNonce(ctx, nonce)
```

## 事件监听

事件系统可以监听登录、登出、续期、权限、角色、封禁、解封等核心生命周期事件。

```go
eventMgr := mgr.GetEventManager()
eventMgr.RegisterFunc(listener.EventAll, func(data *listener.EventData) {
	fmt.Println(data.Event, data.LoginID, data.Token)
})
```
