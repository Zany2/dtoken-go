# SSO 完整接入示例

[English](README.md) · [部署与运维](../../sso/DEPLOYMENT_zh.md)

同一入口通过 `-role center` 或 `-role client` 运行认证中心、任意数量的独立子应用。示例将签名、防重放、PKCE S256、浏览器状态、Auth 本地会话、严格单点注销及共享存储组合起来。原有四个 HTTP/Gin 示例继续保留，用于查看更小的接入步骤。

认证中心的用户名／密码核验是**显式启用的固定账号演示**，不包含用户数据库、密码重置、MFA、风控或登录限流。真实接入应替换 `center.go` 的认证边界；本文的部署配置不能代替身份系统。默认不启用开发模式，也不提供默认密钥。

## 本地三进程联调

在三个 PowerShell 终端分别执行以下一条命令，命令均从仓库根目录运行：

```powershell
./examples/sso_deployment/local.ps1 -Role center
./examples/sso_deployment/local.ps1 -Role alpha
./examples/sso_deployment/local.ps1 -Role beta
```

脚本在当前终端运行服务，不创建后台进程；Ctrl+C 退出。脚本会设置进程环境并强制开发模式、回环地址和内存存储，公开演示密钥只用于本机。`go.work` 当前要求 Go 1.27；此模块声明 Go 1.25。此批代码未运行、未构建、未执行测试，上述命令由使用者本地验证。

1. 打开 [Alpha](http://localhost:9201/protected)，中心使用 `alice` / `local-example-password` 登录。
2. 在同一浏览器打开 [Beta](http://localhost:9202/protected)，复用中心登录态，获得独立子应用会话。
3. 在 Alpha 点击 **Local logout**，只撤销 Alpha 当前浏览器会话；Beta 不受影响。再次访问 Alpha 会通过中心自动登录。
4. 在 Alpha 或 Beta 点击 **Global logout**。子应用先注销本地会话，再跳转中心；中心向已登记应用发送带签名的新鲜回调，撤销账号的中心会话及 SSO 凭证。成功返回 JSON，两个应用的旧 Cookie 均不能继续使用。
5. 子应用停止时尝试全局退出：严格通知失败会保留中心会话及登记记录，便于重试；发起退出的子应用已完成本地退出，不会恢复。恢复服务后从应用重新登录、点击全局退出，生成新 nonce；不要刷新旧退出 URL。

内存模式重启会丢失会话、流程和 nonce。Redis 模式的记录有各自 TTL，Cookie 密钥、客户端密钥及命名空间稳定时，可在同应用副本间完成登录往返；测试源码覆盖开始登录和回调落到不同副本的场景。

## 部署验收

已启动服务的真实 HTTP 验收、浏览器检查和停服／Redis 故障恢复步骤见[部署验收指南](ACCEPTANCE_zh.md)。`acceptance` 子包默认跳过真实网络用例，只有显式设置目标、测试账号和 `SSO_ACCEPTANCE_RUN=true` 才执行；不会启动或停止服务。HTTP 自动化不替代真实浏览器与故障验收。

## 配置

子应用可用 `SSO_VERIFICATION_KEY` 额外接受一把注销回调验签密钥；非空时至少 32 字节。中心在 `SSO_CLIENTS_JSON` 的各注册项使用可选 `verificationKey`，长度规则相同。签名仍使用 `SSO_SIGNING_KEY`／`secretKey`。按[三阶段轮换指南](../../sso/SIGNING_ROTATION_zh.md)部署，并协调副本间的注册写入。

全部配置从进程环境读取，字段错误不回显密钥和原始 JSON。`SSO_DEV` 只接受 `true` / `false`，省略等同 `false`。

| 变量 | 使用方 | 规则 |
| --- | --- | --- |
| `SSO_NAMESPACE` | 全部 | 环境标识，1–64 个字母、数字、`_`、`-`；同环境各进程一致 |
| `SSO_LISTEN` | 全部 | 明确 `host:port`；开发模式只允许回环地址 |
| `SSO_PUBLIC_URL` | 全部 | 浏览器和对端可访问的根来源，如 `https://alpha.example.com`；无路径及尾斜杠 |
| `SSO_REDIS_URL` | 全部 | 非开发模式必填；支持现有 Redis 适配器的 URL，包括 `rediss://` |
| `SSO_COOKIE_KEY` | 全部 | 至少 32 字节的随机密钥；同一服务所有副本一致，不同应用分别管理 |
| `SSO_COOKIE_VERIFICATION_KEY` | 全部，可选 | 非空时至少 32 字节；在 [Cookie 轮换](../../sso/SIGNING_ROTATION_zh.md#浏览器会话与状态-cookie-轮换)期间验证旧／新会话及 state Cookie，不用于签发 |
| `SSO_CENTER_URL` | 子应用 | 中心根来源，必须与中心公开地址一致 |
| `SSO_CLIENT_ID` | 子应用 | 1–64 个字母、数字、`_`、`-`，与中心注册一致 |
| `SSO_CLIENT_SECRET` | 子应用 | 至少 32 字节，与注册项 `clientSecret` 一致 |
| `SSO_SIGNING_KEY` | 子应用 | 至少 32 字节，与注册项 `secretKey` 一致；与 Cookie 密钥分开生成 |
| `SSO_CLIENTS_JSON` | 中心 | 非空客户端数组，最多 1 MiB；未知字段、重复 ID、尾随 JSON 拒绝 |
| `SSO_ENABLE_DEMO_LOGIN` | 中心 | 显式 `true` 才能运行当前演示认证；真实接入替换认证实现 |
| `SSO_DEMO_USER` / `SSO_DEMO_PASSWORD` | 中心 | 演示账号与至少 16 字节密码，不写入日志 |

非开发模式要求中心及子应用公开地址均为 HTTPS，服务自身在可信网络内监听 HTTP，由 TLS 代理提供公开来源。开发模式的公开地址只能使用 `localhost` 或回环 IP。示例固定使用根路径，不支持在路径前缀下部署；框架本身的网关前缀能力见[配置说明](../../sso/CONFIGURATION_zh.md)。

中心注册示意（秘密占位符必须由环境或密钥管理系统填入，示例不会展开 `${...}`）：

```json
[
  {
    "clientId": "alpha",
    "clientSecret": "<32-byte-or-longer-random-exchange-secret>",
    "secretKey": "<32-byte-or-longer-independent-signing-key>",
    "redirectUris": ["https://alpha.example.com/sso/callback"],
    "modes": ["ticket"],
    "requirePkce": true,
    "scopes": ["profile"]
  }
]
```

每个应用只允许一个精确 `/sso/callback` 地址，必须开启 Ticket + PKCE。此约束属于本示例。注销回调自动使用同源 `/sso/logout-callback`；不要添加无关 `allowOrigins`。每次中心启动会注册／覆盖数组内的客户端；从数组移除客户端**不会删除 Redis 中的既有记录**，撤销接入应显式调用 `UnregisterClient` 并处理已有会话。所有中心副本使用一致注册版本，避免滚动启动互相覆盖。

## 代码入口

| 文件 | 职责 |
| --- | --- |
| `config.go` | 本地配置校验、客户端注册和环境命名空间 |
| `application.go` | Auth 会话、BrowserClient、健康检查、响应保护 |
| `center.go` | 演示身份核验、中心会话绑定、换票有效期和注销通知 |
| `main.go` | 存储所有权、监听超时、Ctrl+C/SIGTERM 退出 |
| `application_test.go` | 实际处理器的多应用／跨副本闭环与失败边界 |
| `lifecycle_review_test.go` | 来源规范化、中心绝对期限及关闭连接后的处理器等待 |
| `account_logout_test.go` | 可信账号下线、实际 Auth 撤销、签名多应用通知失败及跨副本恢复 |
| `nginx.conf.example` | 三个独立 HTTPS 来源的反向代理模板 |

部署命令、Redis 数据职责、扩容及故障处理见[部署与运维](../../sso/DEPLOYMENT_zh.md)。

中心保留 `centerHTTP`，供应用自有后台通过 [LogoutAccount](../../sso/ACCOUNT_LOGOUT_zh.md) 执行账号下线。示例未公开管理路由或实现账号封禁系统；调用方负责目标授权和新登录／签发控制。

来源配置统一主机大小写、IP 写法及默认端口后生成回调注册。中心与子应用均执行绝对会话期限；强制关闭连接后仍等待在途处理器返回。待执行项目见[发布验收清单](../../sso/RELEASE_CHECKLIST_zh.md)。本轮已补充测试源码，尚未执行测试。
