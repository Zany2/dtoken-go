# SSO 签名通信与防重放

`ReplayProtection` 为 SSO HTTP 协议增加时间窗口、一次性 nonce 和操作绑定。覆盖授权、Ticket／Code 兑换、四种凭证模式的自省与用户信息查询、撤销、中心注销和客户端注销回调。通过 `HTTPOptions.ReplayProtection` 与 `ClientConfig.ReplayProtection` 显式启用；默认 `nil` 保留原有协议行为，`Signer` 的现有方法不变。

## 接入

在现有客户端注册、登录身份解析、签名密钥和 HTTP 配置上增加以下配置。这里的 `serverOptions` 是 `sso.HTTPOptions`，`clientConfig` 是 `sso.ClientConfig`；`centerStore`、`clientStore` 是应用已经初始化的 `contract.Storage`。中心与子应用分别部署时，在对应进程创建各自的组件。

```go
centerReplay, err := sso.NewReplayProtection(sso.ReplayProtectionOptions{
    Storage: centerStore,
    KeyPrefix: "company:sso:center:replay:",
    MaxAge: 5 * time.Minute,
})
if err != nil {
    return err
}
serverOptions.CheckSign = true
serverOptions.ReplayProtection = centerReplay
handler := sso.NewHTTPServer(server, serverOptions)

clientReplay, err := sso.NewReplayProtection(sso.ReplayProtectionOptions{
    Storage: clientStore,
    KeyPrefix: "company:sso:app-a:replay:",
    MaxAge: 5 * time.Minute,
})
if err != nil {
    return err
}
clientConfig.CheckSign = true
clientConfig.ReplayProtection = clientReplay
app := sso.NewClientApp(clientConfig)
```

继续把 `handler.Handler()` 注册到中心路由，把 `app.LogoutCallbackHandler(...)` 注册到子应用回调路由。Gin 等集成通过同一标准 HTTP 协议层使用这些配置。

- 必须启用 `CheckSign`，并配置客户端签名密钥。中心优先使用注册项 `Client.SecretKey`，否则使用 `ServerOptions.SecretKey`；子应用 `ClientConfig.SecretKey` 必须匹配。`ClientSecret` 仍按现有凭证交换规则配置。
- 保护模式要求显式、单值的 `client`，子应用必须设置 `ClientID`。匿名兼容场景也须显式发送已注册的 `ClientAnonymous`。
- 组件借用存储，不创建默认存储、不关闭调用方存储。`MaxAge` 零值为五分钟，负值或无法安全计算双倍保留窗口的值返回配置错误。
- 同一中心的所有副本使用相同存储、前缀和时间窗口；同一子应用的所有副本也如此。不同中心或环境使用不同前缀，并保持机器时钟同步。
- 支持 `contract.AtomicStorage` 的存储使用 `SetIfAbsent` 去重；现有 Redis 存储支持此能力。普通存储使用 `Get`／`Set` 与组件实例内锁，同进程应共享同一个组件；这种回退不保证跨进程或跨组件实例并发去重。
- 内存存储适合本地调试，重启会丢失 nonce 记录。生产环境需选择在整个接受窗口内保留记录的共享存储。

## 协议规则

每次调用客户端 URL 构造或请求方法时，会覆盖保留元数据并重新签名：

| 参数 | 规则 |
| --- | --- |
| `timestamp` | UTC RFC3339Nano 时间戳；必须满足 `now - MaxAge < timestamp <= now + MaxAge` |
| `nonce` | `crypto/rand.Text()` 生成；接收端允许 16–128 个 ASCII 字母、数字、`_`、`-` |
| `action` | 当前操作的固定值，见下表 |
| `client` | 显式客户端标识，参与签名与去重命名空间 |

四个字段均必须单值且非空。`ParamNames.Action`、`Timestamp`、`Nonce` 等支持自定义；保护模式要求规范化后的协议参数名互不重复。

| 接口 | action |
| --- | --- |
| authorize | `sso.authorize` |
| token | `sso.token` |
| introspect | `sso.introspect` |
| userinfo | `sso.userinfo` |
| revoke | `sso.revoke` |
| logout | `sso.logout` |
| 子应用注销回调 | `sso.logout_callback` |

操作值不随自定义 URL 路径变化。接收方先验签，再检查元数据、占用 nonce，最后执行业务。错误签名和错误操作不会占用 nonce。去重键由 `(client, action, nonce)` 编码后哈希生成；时间戳不在去重键中。记录保留 `2 * MaxAge`，覆盖被接受的未来时间偏差。

时间过期或请求重放返回 HTTP 401；缺失、重复或错误的操作参数返回 400；配置错误、存储读写失败返回 500。存储失败不会降级放行。直接调用回调校验方法时可用 `errors.Is` 判断 `ErrRequestExpired`、`ErrRequestReplayed`、`ErrReplayProtectionConfig` 或 `ErrStorageUnavailable`。

## 登录往返与重试

未登录访问授权入口时只校验签名和时间，不占用 nonce。登录成功回到原链接后，中心在签发凭证及调用 `AuthorizationExtra` 前占用 nonce；再次使用该链接会被拒绝。登录过程超出时间窗口时应从子应用重新发起授权。保护模式不支持把授权 URL 缓存为长期登录入口。

其他请求在业务处理前占用 nonce，业务失败不会回滚占用。重新调用 `AuthURL`、`SignoutURL`、`ExchangeCredential`、`Introspect`、`UserInfo` 或 `Revoke` 才会生成新请求；原样重发缓存的 URL／请求体仍会被拒绝。生成新 nonce 不能恢复已经消费的 Ticket／Code，响应丢失时需要重新发起登录流程。

中心每次发送注销通知都生成新 nonce。子应用回调处理失败后，同一签名回调仍不能再次执行；中心重新发送的新通知可以执行。注销业务应保持幂等。启用保护后，回调时效统一使用 `ReplayProtectionOptions.MaxAge`；旧的 `LogoutCallbackMaxAge` 只用于未启用保护的回调。

## 升级与验证

先将中心和所有子应用升级到支持该协议的版本，再协调启用双方配置。启用后的接收方会拒绝旧请求：只启用中心会拒绝旧客户端请求，只启用子应用会拒绝旧中心注销回调。没有自动降级或兼容放行开关。共享密钥、参数名和操作值属于 DToken-Go 协议，不能将 Sa-Token 的签名请求直接当作兼容请求。

`replay_test.go` 和 `replay_http_test.go` 提供元数据边界、存储故障、并发去重、四种凭证模式、登录往返、操作隔离和回调重试的回归测试。执行方式见 [测试指南](./TESTING_zh.md)。
