# SSO 远程会话鉴权

`RemoteAuth` 将中心凭证检查接入子应用的每个受保护请求，支持 `ModeRemoteSession` 和 `ModeSharedToken`。每次调用都通过现有 `ClientApp.Introspect` 查询中心，不缓存成功结果、不创建本地登录态、不自动续期或重试。

适用于希望后续请求跟随中心凭证撤销和账号注销状态的应用。Ticket 和 OAuth2 授权码只能兑换一次，不适用于逐请求远程鉴权；它们继续使用 `BrowserClient` 或现有换票 API。

## API 接入

```go
app := sso.NewClientApp(sso.ClientConfig{
    Mode: sso.ModeRemoteSession,
    ClientID: "orders",
    ClientSecret: "replace-client-secret",
    ServerURL: "https://sso.example.com",
    CheckSign: true,
    SecretKey: "replace-signing-secret",
})

guard, err := sso.NewRemoteAuth(app, sso.RemoteAuthOptions{
    RequiredScopes: []string{"orders:read"},
    Timeout: 2 * time.Second,
})
if err != nil {
    return err
}

mux.Handle("/orders", guard.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    identity, _ := sso.RemoteIdentityFromContext(r.Context())
    // 身份已校验，可用于本请求的业务逻辑；授权范围不替代具体资源的归属检查。
    _, _ = fmt.Fprintf(w, "Orders for %s", identity.LoginID)
})))
```

默认从唯一的 `Authorization: Bearer <凭证>` 请求头提取凭证。远程会话模式使用远程会话 ID，共享 Token 模式使用 Token 值；模式由服务端应用配置确定，不能由业务请求切换。默认解析器拒绝多个或合并的 Authorization 值、控制字符、空值和错误认证方案，不从 URL 或 Cookie 中查找替代凭证。不要把中心登录 Cookie、中心 Auth Token 或一次性 Ticket 当作这里的凭证。

`NewRemoteAuth` 不访问网络，会检查模式、中心检查地址、客户端 ID、签名与防重放配置、参数名冲突、超时和范围配置。现有 `HTTPClient`、响应大小上限、端点、自定义参数名、签名和防重放设置继续生效。默认不会跟随中心重定向；显式自定义重定向策略继续由调用方负责。

`Middleware` 在校验成功后写入请求上下文，再调用业务处理器；上下文不额外保存原始凭证。返回的身份、Scopes 和 Extra 应只读，不能作为长期登录态保存。`ExpiresIn` 是中心报告的剩余秒数快照，组件已保守扣除本次网络耗时检查是否仍可使用，但不会修改此字段。鉴权只发生在业务处理开始前，不中止已经运行的长请求或 WebSocket。

## 中心签发与状态接入

中心需要注册对应的模式和范围。例如，在可信的应用登录流程验证用户后：

```go
err := server.RegisterClient(&sso.Client{
    ClientID: "orders",
    ClientSecret: "replace-client-secret",
    SecretKey: "replace-signing-secret",
    Modes: []sso.Mode{sso.ModeRemoteSession},
    Scopes: []string{"orders:read"},
})
if err != nil {
    return err
}
session, err := server.CreateRemoteSession(ctx, "orders", authenticatedLoginID,
    []string{"orders:read"}, nil)
```

`authenticatedLoginID` 必须来自可信认证结果。`session.SessionID` 是可复用登录秘密，应由应用通过受保护的交付流程保存；这段代码不是供匿名调用的发票接口。HTTP 授权流程也可签发这两类凭证，仍须使用已注册回调地址和浏览器状态校验。自动浏览器组件创建本地会话的结果不包含远程凭证，已有 `BrowserClient` 登录不会自动变成逐请求鉴权。

中心的 `HTTPOptions.CheckCredentialActive` 继续在每次 introspect 时执行。若需要账号禁用、中心会话过期等应用状态立即反映到后续请求，应在这个钩子中接入应用自己的状态来源。仅撤销应用持有的中心 Auth 会话，而未配置状态钩子、未执行 SSO 凭证撤销，不会自动撤销独立签发的 SSO 凭证。

也可配置 `HTTPOptions.AuthorizationPolicy`，在应用状态检查通过后判断当前成员是否仍有访问权限，并缩减本次返回的范围。明确拒绝会使远程鉴权返回 401；允许但缺少 RequiredScopes 时返回 403；策略故障返回 503。此策略不删除存储凭证，权限恢复后有效凭证可重新通过。详见 [动态授权策略](AUTHORIZATION_POLICY_zh.md)。

以下状态变化会在之后的检查中生效：

- `RevokeRemoteSession` / `RevokeSharedToken` 或 `ClientApp.Revoke`：撤销指定凭证。
- `RevokeLoginCredentials` 或成功完成凭证撤销阶段的中心 HTTP 注销：使该账号旧凭证失效。
- `CheckCredentialActive` 返回 `false`：拒绝当前凭证；返回错误：报告中心校验不可用。
- 凭证过期、客户端移除或模式不再允许：拒绝请求。中心拒绝客户端配置时可能返回 503，具体分类见下表。

这不改变上一轮的严格／尽力注销策略。严格注销若在客户端通知阶段停止，尚未执行账号凭证撤销；远程鉴权不会提前把这些凭证标为无效。状态检查完成后再发生的注销也不回滚已经进入业务处理器的请求。

## 从应用会话读取凭证

可以把远程凭证留在子应用的服务端会话存储，只让浏览器持有应用自己的会话标识：

```go
guard, err := sso.NewRemoteAuth(app, sso.RemoteAuthOptions{
    ResolveCredential: func(r *http.Request) (string, error) {
        // 应用实现：验证本地会话标识、过期时间及归属，并返回保存的远程会话 ID。
        return resolveStoredRemoteSession(r.Context(), r)
    },
    RequiredScopes: []string{"orders:read"},
})
```

`resolveStoredRemoteSession` 是业务函数。缺失或无效输入返回 `sso.ErrRemoteCredential`；存储故障返回实际错误，组件将其归类为校验不可用。组件会给解析器传入带鉴权截止时间的请求，请遵守其 Context。Cookie 接入的 CSRF、会话固定和本地注销由应用管理；远程鉴权本身不设置或删除 Cookie。

## 权限、超时与错误

`RequiredScopes` 是全部满足规则，按大小写敏感的精确字符串匹配；`*` 没有通配含义。空列表只检查身份。空范围或含空白／控制字符的配置会在构造时被拒绝。构造器复制范围切片，外部修改原切片不会改变鉴权策略。

`Timeout` 为零时默认三秒，负值无效，正值限制凭证解析和中心请求的整体时间。更早的上游请求截止时间以及 `HTTPClient.Timeout` 仍然生效。自定义解析器和传输必须遵守 Context；组件不通过额外 goroutine 强制中断不合作的实现。即使传输在取消后返回成功，组件也不放行。此超时不会被带入业务处理器的 Context。

| 情况 | 错误标识 | 默认 HTTP 状态 |
| --- | --- | --- |
| 无凭证、错误 Bearer、中心明确返回 inactive、期限耗尽 | `ErrRemoteCredential` | 401 |
| 有效凭证缺少任一所需范围 | `ErrRemoteScope` | 403 |
| 网络、解析存储、中心业务错误、非法响应、身份归属不一致、超时或取消 | `ErrRemoteUnavailable` | 503 |
| 未正确初始化组件或未提供下游 Handler | `ErrRemoteAuthConfig` | 500 |

中心协议的 401 可能意味着子应用自己的客户端密钥或签名配置错误，所以不会直接映射成终端用户 401。默认响应使用固定 JSON 消息，不包含远端响应体、密钥或原始凭证，并设置 `Cache-Control: no-store`、`Pragma: no-cache`、`Referrer-Policy: no-referrer`。所有失败分支都不会调用下游业务处理器。

通过 `OnError` 可以定制返回体或业务侧错误观测；默认响应会被完全替换。钩子应根据 `errors.Is` 分类，不能直接把 `err.Error()` 发给浏览器。诊断错误保留上下文取消和上游原因，可能包含上游响应里的敏感信息，记录日志时也应脱敏。

## 其他框架与直接调用

`guard.Authenticate(r)` 返回本次已校验的 `*sso.CredentialInfo` 或错误，不写 HTTP 响应，也不自动注入上下文，适合框架适配器使用。应用负责错误响应、缓存头和将身份传给自己的业务处理链。Gin 可以用 `gin.WrapH(guard.Middleware(handler))` 挂载标准 HTTP 处理器；该模块不向核心认证引擎或通用 HTTP 能力绑定中注入 SSO 状态。

示例源码见 [example_remote_auth_test.go](example_remote_auth_test.go)。客户端本身没有本地成功缓存；如果配置的中心存储或应用钩子有缓存，其一致性仍由相应组件决定。每个受保护请求会产生一次中心请求，部署时需要为相应延迟和负载预留容量。

## 验证

新增回归源码覆盖两种可复用模式、中心签名和防重放、自定义参数与端点、重复请求重新检查、应用状态变化、单凭证撤销、账号注销、账号隔离、输入与结果校验、范围检查、上下文取消、超时、错误脱敏以及传输响应限制。已做格式化和静态检查；未运行测试、构建、服务或示例。手动验证命令见 [测试指南](TESTING_zh.md)。
