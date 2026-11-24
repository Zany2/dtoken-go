# SSO 部署验收与故障恢复

[English](ACCEPTANCE.md) · [启动配置](README_zh.md) · [发布验收清单](../../sso/RELEASE_CHECKLIST_zh.md)

本指南针对此目录的中心和两个子应用示例。新增 `acceptance` 测试包通过真实 HTTP 地址访问已启动的服务，不启动进程、不连接 Redis 管理接口、不修改配置或停止服务。本文和验收源码尚未运行，不能视为部署通过。

## 自动 HTTP 验收

先按 README 启动 center、alpha、beta。使用专用测试账号：全局退出会撤销该账号在参与应用中的会话，包括其他浏览器中的会话。验收不适配任意生产登录页、验证码或 MFA；替换示例身份系统后需相应修改登录步骤。

在第四个 PowerShell 终端，从仓库根目录运行：

```powershell
$env:SSO_ACCEPTANCE_RUN = 'true'
$env:SSO_ACCEPTANCE_CENTER = 'http://localhost:9200'
$env:SSO_ACCEPTANCE_ALPHA = 'http://localhost:9201'
$env:SSO_ACCEPTANCE_BETA = 'http://localhost:9202'
$env:SSO_ACCEPTANCE_USER = 'alice'
$env:SSO_ACCEPTANCE_PASSWORD = 'local-example-password'
try {
    go test ./examples/sso_deployment/acceptance -run '^TestLiveDeployment$' -count=1 -v -timeout=3m
    if ($LASTEXITCODE -ne 0) { throw 'SSO HTTP acceptance failed; inspect the named stage.' }
} finally {
    Remove-Item Env:SSO_ACCEPTANCE_RUN -ErrorAction SilentlyContinue
    Remove-Item Env:SSO_ACCEPTANCE_PASSWORD -ErrorAction SilentlyContinue
}
```

HTTPS 验收时将三个地址替换为各自公开根来源，不能带路径、尾斜杠、查询串或凭据；密码通过当前终端的安全配置注入，不使用演示密码。系统需信任服务证书，验收不跳过 TLS 校验。三个来源必须不同，HTTP 仅允许 localhost／回环 IP。中心和子应用进程还必须能相互访问配置的公开地址。

只有 `SSO_ACCEPTANCE_RUN=true` 才会运行真实请求；未设置或其他值时该用例跳过。普通 `go test ./examples/sso_deployment/...` 仍会运行无需网络的验收客户端、退出报告及清理请求边界用例。不要在通用 CI 环境中长期设置这个启用变量。发布 CI 尚未自动部署服务，也不会替代本步骤。

| 步骤 | 成功断言 |
| --- | --- |
| Alpha 首次登录 | 跳转中心登录页，登录后返回原授权地址，经 PKCE 回调创建本地会话 |
| Beta 登录 | 同一 Cookie 容器复用中心会话，不再提交用户名／密码 |
| Cookie 与响应 | 创建的会话／state Cookie 为 HttpOnly、Host-only、SameSite=Lax；HTTPS 时 Secure；响应 no-store |
| 回调边界 | 超大查询串返回 413，随后原合法回调仍成功；已消费回调返回 400 |
| 伪造退出 | GET 退出返回 405，错误 Origin 返回 403，未签名注销通知返回 401 |
| 本地退出 | Alpha 旧会话失效，Beta 仍可访问；Alpha 可复用中心会话重新登录 |
| 全局退出 | HTTP 200、code=0、result=ok、阶段及通知均完成；两个实际应用 ID 各出现一次且为 delivered；两个子应用旧 Cookie 均失效；旧中心 Cookie 再次授权会回到登录页 |

请求不自动跟随跳转。每一跳先核对预期来源和路径，密码只 POST 到配置的中心 `/login`；Cookie、签名 URL、响应体和密码不写日志或文件。每个请求超时 15 秒，响应体最多读取 1 MiB。失败输出阶段名和状态，不回显带凭据的网络错误。清理阶段尝试以新请求执行全局退出；若失败会提示手动恢复清理。验收生成但未完成的 state／nonce 按原 TTL 到期，不执行 Redis 清库。

应用 ID 从各自授权跳转读取，必须彼此不同，且重复登录时保持一致。退出报告必须包含这两个 ID，重复、缺失或无关 ID 不能代替预期应用。允许额外目标，但报告中的每个目标都必须为 delivered。清理阶段对已知应用使用相同业务结果检查，HTTP 401 或 HTTP 200 携带部分结果都不代表清理完成。只有主流程已核实退出报告、两个应用旧会话及原中心会话失效后，才跳过再次清理。[logout_boundary_test.go](acceptance/logout_boundary_test.go) 使用注入传输覆盖这些判断，尚未执行。

代理需允许略大于 8 KiB 的请求行到达应用，才能验收库内 413；更新后的 nginx 模板使用 `large_client_header_buffers 4 16k`。其他代理若返回 414 或自身错误页，说明拒绝发生在代理，不能记作库内检查通过。不要通过调高 SSO 参数上限掩盖该差异。

HTTP Cookie 容器不执行浏览器的 SameSite／CSP 策略；本步骤只检查 Cookie 属性。它不证明跨进程重启、多副本实际路由、Redis 故障恢复或账号之间隔离，以下步骤须另行完成。

## 浏览器与多标签页

1. 使用新的浏览器配置或无痕会话，通过 HTTPS 访问 Alpha，检查登录成功及 Cookie 的 Secure／HttpOnly／SameSite 属性。
2. 同时从两个标签页发起登录，再分别完成回调；不能因另一个标签页开始登录而覆盖第一个 state。
3. 打开 Beta 验证中心复用，分别执行本地和全局退出。验证旧页面刷新后访问控制结果，不能仅检查页面按钮是否消失。
4. 在测试环境记录浏览器版本、来源、代理版本和结果。不要导出包含 Cookie、密码、Ticket、nonce 或签名查询串的 HAR；只记录路径、状态及阶段。

## 故障注入与恢复矩阵

由操作者在独立测试环境控制服务和网络；验收代码不接受 shell 命令或自行停止进程。每个场景先恢复健康状态并重新登录 Alpha／Beta，避免前一个场景的退出结果影响下一个。涉及会话连续性时必须使用 Redis 和稳定的命名空间、Cookie 密钥及客户端配置。

| 场景 | 操作 | 失败期预期 | 恢复后验收 |
| --- | --- | --- | --- |
| Beta 注销通知失败 | 两应用登录后停止 Beta，从 Alpha 发起全局退出 | Alpha 已本地退出；中心返回非成功注销结果，阶段为 notifications，记录失败目标；中心会话和回调绑定保留 | 启动 Beta，从 Alpha 重新登录后生成新全局退出请求，通知完成；不刷新旧注销 URL |
| 子应用进程重启 | Redis 模式下登录 Beta，停止并重新启动 Beta，配置不变 | 停机期间访问失败 | 原浏览器会话仍可访问；内存模式不要求保留会话 |
| 跨副本登录回调 | 同一应用部署两个副本，共用 Redis、密钥、ID 与来源；将登录发起路由到 A、回调路由到 B | 正常完成一次登录 | 重放旧回调被拒绝；记录实际副本路由证据，不能靠负载均衡概率推断 |
| Redis 不可达 | 登录后从服务侧阻断专用测试 Redis；不删除数据 | 就绪检查 503；已有会话访问返回错误，不能自动切回内存或作为匿名成功放行 | 恢复连接后重新检查就绪与旧会话；会话未过期且数据未丢失时应恢复，失败的登录流程从头发起 |
| 中心重启 | Redis 模式下停止并重启中心，保持全部注册及密钥一致 | 新授权／兑换暂时不可用 | 未过期中心会话可复用；已消费 nonce／Ticket 不可复用 |
| 退出排空 | 有在途请求时终止服务，观察日志、连接和进程 | 停止接新请求，按宽限期排空；超时后关闭连接，等待处理器返回 | 存储不应早于处理器释放；不响应取消的自有钩子按进程管理器策略处理 |

就绪检查 `/health/ready` 由代理模板限制访问来源。应从允许的运维入口验证，外部探针收到 403 不能被当作 Redis 故障证据。HTTP 全局退出与管理员 `LogoutAccount` 的阶段顺序不同；本矩阵的停服场景针对浏览器退出，管理员恢复另见[账号下线指南](../../sso/ACCOUNT_LOGOUT_zh.md)。

## 记录与结束条件

每个场景记录源码版本、环境、存储模式、步骤、预期、实际状态、是否恢复及脱敏日志。HTTP 自动用例通过、浏览器验收通过、Redis 多副本／故障场景通过应分别记录；跳过不能标记通过。恢复所有服务与网络后再跑一次 HTTP 验收，确认故障注入没有残留影响。

目前完成的是验收入口、边界回归源码与操作说明；运行结果仍待用户执行。最终记录归入[发布验收清单](../../sso/RELEASE_CHECKLIST_zh.md)。
