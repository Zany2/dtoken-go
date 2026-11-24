# SSO 请求参数一致性

[English](REQUEST_VALIDATION.md) · [测试指南](TESTING_zh.md)

中心、ClientApp 和浏览器组件对已配置的协议字段采用单值语义，避免重复参数及混合凭证被不同接入层按不同规则解释。此检查是请求格式校验，不替代签名、客户端密钥、PKCE、凭证归属或业务授权。

## 单值字段与扩展参数

`ParamNames` 中的全部配置字段至多出现一次，包括 `client`、`clientSecret`、`mode`、`redirect`、`callback`、`back`、`scope`、`loginId`、四种凭证字段和签名／防重放字段。重复值相同也会拒绝。自定义参数名按配置后的名称校验，默认旧名称不会自动成为别名。

POST 的 Query 和表单 Body 仍按标准库规则合并；同一协议字段同时存在于两处属于重复参数。此变更没有将所有 POST 参数改为仅允许 Body，也没有改变 PKCE verifier 禁止出现在兑换／身份检查 URL 中的既有规则。客户端兑换密钥应通过 POST 提交，签名密钥不作为协议参数发送。

单个空字符串仍交由各字段原有规则处理；如缺失凭证、空回调与空 mode 的原有判断保持在对应入口。直接传入 `url.Values` 时，存在但长度为零的协议值切片会被拒绝。

多个权限应使用 `scope=profile%20email`（一个值），不要使用 `scope=profile&scope=email`。未占用协议字段名的业务扩展参数仍允许重复，例如 `tag=a&tag=b`。底层 `Signer` 的多值排序／编码及验签算法保持不变：它可以对多值参数签名，但签名正确并不表示协议请求格式有效。

## 凭证与模式

| mode | 允许携带的凭证字段 | 可用操作 |
| --- | --- | --- |
| `ticket` | `ticket` | 兑换、检查、用户信息、撤销 |
| `oauth2` | `code` | 兑换、检查、用户信息、撤销 |
| `shared_token` | `tokenValue` | 检查、用户信息、撤销 |
| `remote_session` | `sessionId` | 检查、用户信息、撤销 |

兑换／检查／用户信息／撤销请求不能同时携带其他模式的凭证字段，即使额外字段是空值也会拒绝。例如 `mode=ticket&ticket=...&tokenValue=` 返回 `ErrInvalidRequest`。`/token` 缺失 mode 时保留已有规则：有非空 code 则推断为 OAuth2，否则 Ticket；其他三个端点缺失 mode 时仍默认 Ticket。缺失当前模式必需凭证的错误分类仍由既有凭证逻辑决定。

`ClientApp` 的 `CredentialRequest.Mode` 为空时使用其配置的 Mode；它不会根据结构体中的其他凭证字段自动切换模式。仅填写当前模式的凭证字段。SDK 会在传输前拒绝混合凭证；标准生成函数覆盖其拥有的配置字段后，才检查最终请求。启用防重放时仍先生成新的 action／timestamp／nonce，再校验和签名。

## 顺序与副作用

- 中心六个 HTTP 入口在验签及 nonce 消费前进行格式检查，重复参数或混合凭证返回 HTTP 400／`ErrInvalidRequest`。PKCE 字段沿用原有 `ErrInvalidPKCE` 分类与校验顺序，未知或不支持的模式返回 `ErrModeUnsupported`。
- 因上述重复字段或混合凭证检查被拒绝的请求，不会触发身份解析器、授权／状态钩子、兑换消费、撤销或注销通知，也不登记新的回调绑定。
- 子应用 `VerifyLogoutCallback` 在验签与 nonce 消费前拒绝重复协议字段；`LogoutCallbackHandler` 不调用业务注销函数。
- `BrowserClient.CompleteLogin` 保留原有 state／凭证检查分类，并在消费浏览器状态前检查其他协议字段。拒绝后原合法回调仍可使用，前提是状态未过期且尚未被其他请求消费。
- 由于格式错误未占用 nonce，修正参数并重新签名后仍可使用未消费的 nonce；正常 SDK 调用会自动生成新 nonce。已经接受的请求仍按原规则防重放。

直接调用可信 `Server` 方法使用结构化参数，不经过这层 HTTP 校验。自定义 HTTP 适配器应实施同样的单值和模式校验，不能仅依赖签名算法。

## 兼容与验证

这是对歧义输入的显式收紧。已有单值协议请求不变；过去依赖重复字段“取首值”、同时提交所有凭证字段，或重复提交 scope 的调用方需要调整。检查代理、表单组件是否又在 URL 中追加了 Body 已有的协议参数。不要通过关闭签名来修复格式错误，未签名模式同样检查格式。

新增 [request_validation_test.go](request_validation_test.go) 包含 8 组参数化回归：全部协议字段、四种模式、六个动作的 nonce 顺序、中心生命周期、SDK 和注销回调、浏览器状态保留、授权／注销副作用，以及自定义参数名下的签名 Query／Body 重复与重放。

本轮仅做格式化及静态审查，未运行构建、测试或竞态检查。由使用者在仓库根目录执行：

```powershell
go test ./sso/... -run TestRequestValidation -count=1
go test -race ./sso/... -run TestRequestValidation -count=1
```

完整模块和真实部署验收仍见[发布验收清单](RELEASE_CHECKLIST_zh.md)。
