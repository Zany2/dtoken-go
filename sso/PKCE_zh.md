# SSO 授权凭证绑定（PKCE S256）

Ticket 和 OAuth2 授权码可以绑定一次登录流程的兑换证明。客户端为每次登录生成随机 `code_verifier`，授权请求只发送其 SHA-256 摘要 `code_challenge`，中心将挑战随凭证保存。兑换时，中心验证原始 verifier 后才消费凭证。

该能力采用 RFC 7636 的 S256 算法，同时适用于本模块的 Ticket 模式。它不会把 `ModeOAuth2` 扩展成完整 OAuth2/OIDC 服务，也不会替代客户端认证、请求签名、HTTPS 或浏览器 state 绑定。

## 浏览器完整接入

中心按客户端配置策略：

```go
err := server.RegisterClient(&sso.Client{
    ClientID: "app-a",
    ClientSecret: "secret-a",
    RedirectURIs: []string{"https://app.example.com/sso/callback"},
    Modes: []sso.Mode{sso.ModeTicket},
    RequirePKCE: true,
})
if err != nil {
    return err
}
```

子应用启用自动处理：

```go
browser, err := sso.NewBrowserClient(app, sso.BrowserClientOptions{
    Storage: stateStorage,
    CallbackURL: "https://app.example.com/sso/callback",
    StateCookie: sso.CookieOptions{SecretKey: "replace-with-a-strong-key"},
    PKCE: true,
})
if err != nil {
    return err
}
```

继续使用 `BeginLogin` / `CallbackHandler` 即可。组件为每次登录生成独立的 256 位随机 verifier，和返回地址、浏览器绑定一起保存在服务端状态存储中。浏览器只收到挑战、state 和绑定 Cookie；verifier 不进入授权 URL、Cookie 或登录结果。回调先消费浏览器绑定状态，再通过 POST 表单提交原始 verifier。

不同标签页使用不同证明。把第二个标签页的 Ticket 放进第一个标签页的回调会兑换失败，第二个 Ticket 保留，可以由其原流程完成登录。第一个流程的浏览器状态已经消费，必须重新发起登录。交换发生网络或存储故障时也不重放原流程。

`BeginLogin` 的 `extra` 不接受三个 PKCE 参数，证明由组件管理。`PKCE: true` 仅支持 Ticket 和 OAuth2 授权码；共享 Token、远程会话配置会返回 `ErrModeUnsupported`。切换 PKCE 配置后，旧配置产生的待完成浏览器流程需要重新开始；部署期间各实例应使用一致配置，并共享状态存储与 Cookie 密钥。

## 手动客户端接入

```go
verifier := sso.NewPKCEVerifier()
authURL, err := app.AuthURLWithPKCE(callbackURL, verifier, nil)
if err != nil {
    return err
}
// 将 verifier 保存在当前登录流程的服务端状态中，然后跳转 authURL。

// 回调校验浏览器 state 后，从同一流程读取 verifier。
result, err := app.ExchangeCredential(ctx, sso.CredentialRequest{
    Mode: sso.ModeTicket,
    Ticket: ticket,
    RedirectURI: callbackURL,
    CodeVerifier: verifier,
})
```

授权码模式改用 `ModeOAuth2` 与 `Code`。`AuthURLWithPKCE` 只发送挑战，并保留签名、重放防护、注销回调登记和自定义参数名配置。它不会替调用方保存 verifier 或校验浏览器 state。

`ClientApp.Introspect` 和 `UserInfo` 对绑定证明的一次性凭证也需要在 `CredentialRequest.CodeVerifier` 中提供证明；这两个接口仍不消费凭证。HTTP 查询接口未通过证明检查时不返回身份或扩展数据，防止绕过换票入口读取身份。

## 直接调用中心 API

```go
challenge, err := sso.PKCEChallenge(verifier)
if err != nil {
    return err
}
ticket, err := server.GenerateTicketWithOptions(ctx, clientID, loginID, callbackURL, scopes, extra,
    sso.OneTimeCredentialOptions{
        Timeout: time.Minute,
        CodeChallenge: challenge,
        CodeChallengeMethod: sso.PKCEMethodS256,
    })
if err != nil {
    return err
}
identity, err := server.ConsumeTicketWithVerifier(ctx, ticket.Ticket, clientID, clientSecret, callbackURL, verifier)
```

授权码对应 `GenerateOAuth2CodeWithOptions` / `ConsumeOAuth2CodeWithVerifier`。`Timeout <= 0` 沿用对应模式的默认有效期。挑战保存在凭证的独立字段中，不使用业务 `Extra`；自定义 Codec 必须保留新字段。

普通存储继续采用同一 `Server` 实例内串行化的 Get/Delete 回退，支持 `AtomicStorage` 时采用原子消费并复核实际载荷。此次不增加存储接口、分布式锁或依赖。凭证有效期、客户端认证、回调绑定、账号注销版本和一次性消费规则仍然生效。

`ValidateTicket` 是可信中心内部的只读检查 API，不要求 verifier，不应直接暴露为客户端身份兑换接口。`Revoke` 仍通过现有客户端认证与归属校验撤销凭证，不要求证明，以便清理凭证。

## 协议与错误

| 参数 | 默认名称 | 所在阶段 |
| --- | --- | --- |
| `ParamNames.CodeChallenge` | `code_challenge` | 授权 GET |
| `ParamNames.CodeChallengeMethod` | `code_challenge_method` | 授权 GET，必须为 `S256` |
| `ParamNames.CodeVerifier` | `code_verifier` | 换票、userinfo、introspect 的 POST 表单 |

verifier 长度为 43–128 个 ASCII 非保留字符：字母、数字及 `-._~`。挑战必须为 32 字节摘要的规范无填充 base64url 编码，长度 43。不支持 `plain`、省略算法或自动降级。重复参数、显式空参数、授权阶段携带 verifier、兑换阶段携带挑战，以及在 POST 查询串中提交 verifier 都会被拒绝。所有协议参数名必须互不冲突。

`ErrPKCERequired` 表示当前客户端要求证明而凭证或授权请求没有绑定；`ErrInvalidPKCE` 表示挑战或证明无效。HTTP 返回 400，不回显原始证明。正常消费流程中，错误证明不会删除凭证或登记兑换阶段的注销回调；若请求开启重放防护，该请求的 nonce 可能已消费，重试必须生成新签名和 nonce。

即便客户端不强制 PKCE，带挑战签发的凭证也必须提供匹配证明。向未绑定凭证提交 verifier 同样失败，避免挑战被剥离后静默成功。降低客户端策略不会移除已有凭证上的绑定。

## 兼容与上线顺序

1. 先升级所有中心实例及自定义 Codec，使其能够保存和校验证明字段。旧中心会忽略挑战，因此不能视为已启用保护。
2. 子应用启用 `BrowserClientOptions.PKCE` 或手动接入。`examples/sso_client -browser-flow` 已启用自动证明；普通手动示例流程维持原有配置。
3. 确认子应用接入完成后，对指定客户端设置 `RequirePKCE: true`。此时旧的未绑定 Ticket/Code 不再可兑换或读取身份，需要重新授权。

两个配置均默认 `false`。旧签发和消费方法保留：不要求 PKCE 且未绑定证明时行为不变；要求 PKCE 时，旧签发方法返回 `ErrPKCERequired`，旧消费方法不能绕过绑定。该策略只约束 Ticket/Code；若客户端仅应使用证明绑定流程，应将 `Modes` 限制在这两种模式内。

服务端流程存储包含短期 verifier，应按登录秘密保护；不要把它写入日志、业务 `Extra` 或公开结果。PKCE 不延长本地会话，不改变注销范围，不自动重试一次性凭证兑换。

## 验证范围

回归源码覆盖 RFC 向量、格式与算法、策略切换、旧接口绕过、两种存储路径、并发消费、注销失效、HTTP 身份旁路、签名与自定义参数、多标签页串票及浏览器状态迁移。已执行格式化与静态检查，未运行测试、构建、示例或真实 Redis 联调。手动验证命令见 [测试指南](TESTING_zh.md)。
