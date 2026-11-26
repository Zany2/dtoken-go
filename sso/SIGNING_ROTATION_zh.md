# SSO 签名密钥平滑轮换

[English](SIGNING_ROTATION.md) · [配置](CONFIGURATION_zh.md) · [发布验收](RELEASE_CHECKLIST_zh.md)

`SecretKey` 始终是唯一出站签名密钥；可选 `VerificationKey` 仅用于额外验签，最多接受两把密钥。字段为空时保持原有单密钥行为，不改变 HMAC-SHA256、签名规范化、请求参数或路由，也不增加 key ID。它可以在准备阶段保存下一把密钥，在切换后保存上一把密钥。

## 配置与信任范围

| 配置位置 | 验证范围 | 出站行为 |
| --- | --- | --- |
| `Client.SecretKey` / `Client.VerificationKey` | 中心验证该客户端的请求；每次请求读取当前注册记录 | 中心向该客户端发送注销通知，只用该记录的 `SecretKey` |
| `ServerOptions.SecretKey` / `ServerOptions.VerificationKey` | 没有独立密钥组的客户端及相应无客户端身份请求 | 默认组只用其 `SecretKey` 签名 |
| `ClientConfig.SecretKey` / `ClientConfig.VerificationKey` | 子应用验证来自中心的注销回调 | 子应用请求只用其 `SecretKey` 签名 |

客户端独立组整体替代默认组。独立组验签失败不会尝试中心默认主密钥或默认额外密钥，也不会尝试其他客户端密钥。匿名客户端沿用原有选择规则。读取注册记录失败、客户端被移除时不使用缓存或默认密钥绕过错误。

额外密钥要求同一配置中有非空 `SecretKey`。客户端不能只配置 `VerificationKey` 再继承默认主密钥。带校验的 HTTP／ClientApp 构造器也会拒绝配置额外密钥却关闭 `CheckSign` 的情况；旧构造器的 `CheckSign: false` 仍关闭验签。诊断不回显密钥。额外密钥与主密钥相同是合法但没有额外效果的配置。

示意配置中的变量必须来自应用密钥管理：

```go
registration.SecretKey = oldKey
registration.VerificationKey = nextKey
if err := server.RegisterClientChecked(registration); err != nil {
    return err
}

config := sso.DefaultClientConfig()
config.ClientID = "orders"
config.ServerURL = "https://login.example.com"
config.SecretKey = oldKey
config.VerificationKey = nextKey
// 继续设置兑换密钥、防重放及应用需要的其他配置，再使用 NewClientAppChecked。
```

## 三阶段切换

| 阶段 | 所有中心注册／子应用配置的目标状态 | 开始下一阶段的条件 |
| --- | --- | --- |
| 准备 | `SecretKey=old`，`VerificationKey=new` | 两端所有接收副本均支持新字段并部署完成；仍用 old 发送 |
| 切换 | `SecretKey=new`，`VerificationKey=old` | 两端所有发送副本均已切换；新旧接收配置都能接受对端 |
| 退役 | `SecretKey=new`，`VerificationKey=""` | 已排空旧发送方，过渡窗口结束，接受旧请求失败及重新登录的影响 |

先部署能理解新字段的程序，再进入准备阶段。旧程序可能忽略存储中的额外字段，但不会接受新密钥签名，因此不能提前切换发送方。共享存储中的客户端注册是整个记录覆盖；滚动启动的旧副本可能覆盖新密钥组，必须统一注册配置或由应用协调单一注册写入者。

`ClientApp` 持有构造时的配置值；不要并发修改正在使用的对象。用新配置构造新对象后由应用切换。中心默认组也应通过替换 HTTPServer 配置实例切换。客户端独立组可以用 `RegisterClientChecked` 更新，中心请求和每次注销通知重试都会读取最新记录。

## 过渡窗口与重放

额外密钥**不会自动到期**。这里限制的是密钥数量，退役时限由部署流程管理。启用防重放时，保留原来的存储、命名空间和时间窗口；同一个 client/action/nonce 换另一把密钥签名仍然是重放，轮换不能清空 nonce。无效或已退役签名在 nonce 消费前拒绝。

若需要让已签名旧请求自然结束，应从最后一个旧发送方停止后考虑全部请求有效期和允许时钟偏差；当前对未来时间戳的容差意味着可能需要最多 `2 × MaxAge`。同时考虑尚未完成的浏览器流程。未启用防重放时，普通签名请求没有统一过期边界，需要明确切换截止时间，不能无限保留旧密钥。疑似泄露时应直接移除旧密钥并接受旧请求失败。

注销重试始终使用最新注册的主密钥和新 nonce；轮换不改变通知次数、超时或严格／尽力策略。协议密钥轮换不轮换 `ClientSecret`、Cookie 密钥或 PKCE verifier。Cookie 轮换按下节独立配置。

## 浏览器会话与状态 Cookie 轮换

`CookieOptions.VerificationKey` 为 `CookieOptions.SecretKey` 增加一把显式过渡验签密钥。`SetLoginIDCookie` 只用主密钥签发；`LoginIDFromCookie` 接受任一密钥，但仍检查原签名中的过期时间。原有 `v1` 格式及单密钥默认行为不变。验签不会刷新 Cookie、延长期限、写入存储或重建会话。仅配置验签密钥不能签发或认证 Cookie；带校验的 HTTP 构造器会拒绝没有有效主密钥的过渡配置。浏览器 state 仍要求自己的显式主密钥。

默认中心解析器通过 `HTTPOptions.Cookie` 配置，应用会话通过自己管理的 `CookieOptions` 配置，浏览器 state 通过 `BrowserClientOptions.StateCookie` 配置。自定义中心解析器需在调用 `LoginIDFromCookie` 时显式传入这组配置。Cookie 不继承协议的 `ServerOptions.VerificationKey`。原有 `HTTPOptions.Cookie.SecretKey` 为空时回退到 `ServerOptions.SecretKey` 的行为保留，但显式独立 Cookie 密钥能避免两套轮换计划互相影响。不同应用和环境应独立管理密钥。

| 阶段 | 同一服务全部副本的 Cookie 配置 | 必须达到的结果 |
| --- | --- | --- |
| 准备 | 主密钥=old，过渡密钥=new | 所有读取方均支持过渡密钥后，才允许任何写入方切换 |
| 切换 | 主密钥=new，过渡密钥=old | 新旧代际均能读取两种会话及待完成 state Cookie；新 Cookie 使用 new |
| 退役 | 主密钥=new，过渡密钥为空 | 旧 Cookie 验签失败，剩余旧会话／登录流程需要重新登录 |

用新配置重建解析器、BrowserClient 或应用，不并发修改已使用的共享实例。保持 Cookie 名、路径、域、存储命名空间和后端存储一致。轮换不会跨进程搬迁内存状态。部署示例中，同一服务的本地会话与 state 使用相同 Cookie 密钥组；中心与各应用独立轮换。待完成 PKCE 证明及 state 记录继续保存在共享存储中，已消费 state 不会因换钥恢复。Cookie 验签通过后，服务端会话撤销及绝对截止时间仍生效。

过渡密钥不会自动到期。计划内退役应从最后一次旧密钥签发开始，等待旧会话或 state Cookie 的最长可能有效期，并考虑时钟偏差及历史上更长的配置。切换配置不会自动续发用户已有 Cookie，提前移除旧密钥会有意要求受影响用户重新登录。若密钥疑似泄露，应立即移除而不是为连续性保留，并按应用策略撤销受影响的服务端会话。Cookie 密钥退役不能代替服务端注销。

完整部署示例中，`SSO_COOKIE_KEY` 配置主密钥，可选 `SSO_COOKIE_VERIFICATION_KEY` 配置过渡密钥；非空值均要求至少 32 字节。它们独立于 `SSO_SIGNING_KEY`／`SSO_VERIFICATION_KEY`。本地启动脚本清空可选 Cookie 过渡密钥，避免继承无关环境设置。

[cookie_rotation_test.go](cookie_rotation_test.go) 覆盖准备／切换／退役、仅主密钥签发、过期与篡改、配置及 PKCE state 连续性。[部署回归](../examples/sso_deployment/cookie_rotation_test.go) 覆盖实际中心会话复用、旧副本发起／新副本完成登录、会话退役及账号下线。这些源码尚未执行。定向命令：`go test ./sso/... ./examples/sso_deployment/... -run 'TestCookieRotation|TestDeploymentCookieRotation' -count=1`。真实 Redis 滚动部署仍需单独验收。

## 完整部署示例

中心在 `SSO_CLIENTS_JSON` 的对应注册项使用 `secretKey` 和可选 `verificationKey`。子应用使用 `SSO_SIGNING_KEY` 和可选 `SSO_VERIFICATION_KEY`；示例要求非空密钥至少 32 字节。开发脚本清空额外密钥，避免继承外部环境中的轮换设置。不要把密钥放进 URL、日志或错误响应。

## 验证

新增 `signing_keys_test.go` 六组用例覆盖密钥组隔离、六个动作的切换／退役和 nonce 连续性、回调重放、四种凭证模式、通知重试以及配置诊断；部署示例另有交错密钥下的实际登录／注销和配置校验用例。仅完成静态审查和格式化，未运行：

```powershell
go test ./sso/... ./examples/sso_deployment/... -run 'TestSigningRotation|TestDeploymentSigningRotation' -count=1
go test -race ./sso/... ./examples/sso_deployment/... -run 'TestSigningRotation|TestDeploymentSigningRotation' -count=1
```

这些用例使用内存存储和进程内传输，不能代替真实 Redis、多副本滚动更新及浏览器验收。
