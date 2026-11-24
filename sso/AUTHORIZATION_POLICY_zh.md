# SSO 动态授权策略

`HTTPOptions.AuthorizationPolicy` 在中心的签发、兑换和身份检查三个阶段判断“当前用户是否可以访问目标子应用，以及实际允许哪些 Scopes”。它适合接入应用自己的成员关系、租户权限、账号状态或用户授权记录，无需向 SSO 模块引入权限数据库。

这是 HTTP 中心的可选策略。默认 `nil` 保留现有行为；直接调用 `Server.Generate*`、`Consume*`、`Validate*` 等可信底层 API 不会自动执行该策略。使用这些 API 对外提供自定义流程时，应用必须自行实施同等授权检查。

## 接入方式

```go
options := sso.DefaultHTTPOptions()
// 配置客户端、签名密钥以及可信的中心登录态解析器。
options.LoginIDResolver = resolveCenterLogin
options.AuthorizationPolicy = func(ctx context.Context, request sso.AuthorizationRequest) (sso.AuthorizationDecision, error) {
    // 应用实现：按可信用户与客户端查询当前成员关系及权限。
    membership, err := loadMembership(ctx, request.ClientID, request.LoginID)
    if err != nil {
        return sso.AuthorizationDecision{}, err
    }
    if !membership.Enabled {
        return sso.AuthorizationDecision{}, nil
    }
    granted := make([]string, 0, len(request.Scopes))
    for _, scope := range request.Scopes {
        if slices.Contains(membership.Scopes, scope) {
            granted = append(granted, scope)
        }
    }
    return sso.AuthorizationDecision{Allowed: true, Scopes: granted}, nil
}
handler := sso.NewHTTPServer(server, options)
```

`resolveCenterLogin` 和 `loadMembership` 是应用函数。租户选择应由应用可信的客户端／用户映射确定，不要直接相信浏览器提交的租户 ID。策略只收到 Context 和结构化的身份、模式、阶段、范围，不接收原始 HTTP 请求、客户端密钥、Ticket、Token、PKCE verifier 或业务 Extra。

可独立阅读的示例见 [example_authorization_policy_test.go](example_authorization_policy_test.go)。使用 `sso/httpserver` 外观时，同样设置 `Options.AuthorizationPolicy`。

## 策略输入与阶段

| 阶段 | 触发入口 | 身份与范围来源 | 允许后的行为 |
| --- | --- | --- | --- |
| `AuthorizationIssue` | `/authorize`，四种模式 | 已验证中心登录态、已注册客户端、通过静态白名单检查的请求范围 | 将缩减范围保存在新凭证中 |
| `AuthorizationExchange` | `/token`，Ticket／授权码 | 已完成客户端认证、有效期、归属、redirect 和 PKCE 检查的存储凭证 | 消费一次性凭证，将缩减范围返回子应用 |
| `AuthorizationInspect` | `/introspect`、`/userinfo`，四种模式 | 已认证客户端、有效凭证及其原始范围，并通过可选应用状态检查 | 返回本次允许的范围，不改写存储凭证 |

签发前仍检查 `Client.Modes`、`Client.Scopes`、回调地址、签名及 PKCE。策略不能修复一个本身无效的请求，也不能通过返回允许重新激活过期、撤销或已消费的凭证。请求里伪造的 `loginId` 或兑换时附加的 `scope` 不会成为策略身份或授予范围。

开启防重放时，请求 nonce 在策略执行前已经消费。策略拒绝或故障后，应重新生成签名请求；`ClientApp` 会按现有机制生成新 nonce。浏览器组件仍遵守“一次回调消费一次浏览器状态”的规则，失败后应重新发起登录。

## 授权结果

- `AuthorizationDecision{}`：明确拒绝。即使 Scopes 不为空，`Allowed: false` 也不会继续。
- `Allowed: true, Scopes: nil`：允许并保留输入的全部范围。
- `Allowed: true, Scopes: []string{}`：允许身份访问但不授予任何范围。
- `Allowed: true, Scopes: subset`：仅保留输入范围的子集。使用精确匹配，不展开通配符；返回新范围会导致失败，不会静默扩大授权。

显式范围选择保持输入顺序并去重。传给策略的范围切片和返回给流程的范围切片均独立复制，策略修改输入切片不能改变原始授权上限。策略实现必须支持并发调用，不能异步修改本次输入或返回值。

签发时缩减的范围是新凭证的永久上限。例如请求 `profile email`，签发策略仅授予 `profile`，后续兑换或检查策略不能重新加入 `email`。检查阶段的缩减只影响本次响应：若存储凭证原有 `profile email`，本次允许 `profile`，以后策略恢复时仍可返回原有的 `email`。兑换成功后凭证已消费，不能用新的策略重放兑换。

`Allowed: true` 且范围为空仍表示身份有效；子应用必须用 `RemoteAuth.RequiredScopes` 或业务鉴权检查需要的权限。Scopes 不替代订单归属等具体资源校验。`Extra` 不按范围自动脱敏，应用应控制元数据内容；不要在 Extra 中保存另一套绕过 Scopes 的授权结果。

## 失败与生命周期

| 结果 | 授权／兑换 HTTP | introspect HTTP | userinfo HTTP |
| --- | --- | --- | --- |
| 策略明确拒绝 | 403，`ErrAuthorizationDenied` | 200，只有 inactive 结果 | 401，不返回身份 |
| 策略查询错误或 Context 取消 | 503，`ErrAuthorizationPolicyUnavailable` | 503 | 503 |
| 策略扩大范围或实际消费身份与策略输入不一致 | 500，`ErrAuthorizationDecision` | 500（非法范围） | 500（非法范围） |

错误返回优先于 Allowed。将 `ErrAuthorizationDenied` 当作 error 返回也属于策略求值失败；业务拒绝应返回 `Allowed: false, nil`。协议响应仅包含固定分类，不暴露策略错误原因。应用需要诊断时，应在策略函数中记录经过脱敏的内部信息；不要把查询错误转换成“允许”。

签发策略在 `AuthorizationExtra` 和凭证生成之前执行，拒绝时不生成凭证、不登记注销回调。兑换策略在消费之前执行，正常流程中的拒绝、故障、非法范围均保留凭证；随后消费仍会复核凭证状态。消费成功后会核对实际身份及原始范围与求值时一致，再应用缩减结果。

`EnrichExchangeResult` 在策略通过及消费完成后执行，看到的是缩减后的范围，仍不能改变身份或范围。这个钩子失败时凭证已经消费；策略不会回滚它。兑换成功才新增兑换阶段的注销回调登记，之前授权阶段登记的绑定不会因策略拒绝而清除。

`CheckCredentialActive` 在身份检查策略之前执行。该钩子返回 false 时直接 inactive，策略不会再次激活凭证；查询故障继续返回错误。不要在策略中递归调用同一个 HTTP 兑换流程、登记回调或修改凭证；带注销回调的兑换流程会持有原有的登记互斥锁，策略应只做权限判断。

## 与远程鉴权和注销配合

上一轮的 `RemoteAuth` 每次请求都会重新 introspect，因此可以直接接收本模块的动态决策：

| 当前策略结果 | 子应用默认响应 |
| --- | --- |
| 不允许当前用户访问应用 | 401 |
| 允许，但缩减后缺少业务所需范围 | 403 |
| 策略服务故障、非法结果或请求超时 | 503 |
| 允许且具备所需范围 | 进入业务处理器 |

策略拒绝不删除凭证、不撤销中心登录态、不清理已有子应用会话。权限恢复后，仍有效的存储凭证可以重新通过。需要永久失效时，使用 `RevokeLoginCredentials`、单凭证撤销或现有中心注销流程；已撤销凭证不会重新进入策略。

通过 Ticket 建立的本地会话只获得登录时的授权快照。后续策略变化不会自动删除它；需要使用现有单点注销、本地授权检查或适用的远程会话方案。已经进入业务处理器的请求也不会被策略变化回滚。

## 配置与部署边界

所有中心实例应配置相同策略；漏配策略的实例仍保留默认允许的既有行为。策略数据由应用持有，缓存有效期和一致性也由应用决定。本模块不缓存求值结果，不引入存储接口、依赖、权限管理端或新的 HTTP 端点。

策略函数应遵守传入 Context，并为权限数据库或远程查询设置合理超时。组件拒绝取消后才返回的允许结果，但不会启动后台 goroutine 强制中断不合作的实现。授权检查与后续操作不是分布式事务；求值后发生权限变更不会撤回已经完成的签发或业务请求。

## 验证

回归源码覆盖四种模式签发与检查、范围缩减与扩张拒绝、策略输入隔离、普通／原子存储消费、失败保留凭证、PKCE 与客户端认证顺序、无 SLO 接入、远程鉴权、账号撤销、浏览器签名防重放及取消后的拒绝。已执行格式化及静态检查；未运行测试、构建、示例或真实存储联调。手动命令见 [测试指南](TESTING_zh.md)。
