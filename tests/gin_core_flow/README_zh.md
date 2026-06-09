# Gin 核心流程测试

本目录包含 `tests/gin_core_app` 的 HTTP 流程测试。

这些测试不会手动请求外部端口，而是通过 `httptest.NewServer` 在测试进程内启动 Gin 应用，然后按照真实 HTTP 流程调用接口。

## 测试列表

- `TestAuthFlow`：测试认证流程。
  - 未携带 token 请求 `/api/me`，期望未授权。
  - 通过 `/login` 登录，期望返回 token。
  - 使用 token 请求 `/api/me`，期望返回当前用户。
  - 通过 `/api/logout` 登出，期望旧 token 被拒绝。

- `TestTokenMetadataAndStatusFlow`：测试 token 元信息和状态接口。
  - 检查有无 token 时的 `IsLogin` 行为。
  - 读取 token 信息、设备、设备 ID、创建时间和超时时间。
  - 使用自定义超时时间登录并验证 TTL。
  - 使用已有 token 调用 LoginByToken。

- `TestPermissionFlow`：测试权限校验。
  - 登录时不授予 `article:read`。
  - 请求 `/api/articles`，期望禁止访问。
  - 通过 `/api/permissions` 授权。
  - 再次请求 `/api/articles`，期望成功。

- `TestPermissionMutationAndLogicFlow`：测试权限变更和逻辑校验。
  - 移除已授权限并验证访问被撤销。
  - 验证 AND 权限校验需要全部权限。
  - 验证 OR 权限校验和通配符权限。

- `TestAccessStatusFlow`：测试布尔型权限和角色校验。
  - 通过 login ID 验证 HasPermission 和 HasRole。
  - 通过 token 验证 HasPermission 和 HasRole。

- `TestAccessListFlow`：测试权限和角色列表接口。
  - 通过 login ID 验证 GetPermissions/GetRoles。
  - 验证 GetPermissionsByToken/GetRolesByToken。

- `TestAccessProviderFlow`：测试外部访问提供者行为。
  - 验证 provider 权限和角色会覆盖 session 中存储的值。
  - 验证按终端区分的 provider 数据可以因设备不同而不同。

- `TestRoleFlow`：测试角色校验。
  - 登录时不授予 `admin`。
  - 请求 `/api/admin`，期望禁止访问。
  - 通过 `/api/roles` 授予角色。
  - 再次请求 `/api/admin`，期望成功。

- `TestRoleMutationAndLogicFlow`：测试角色变更和逻辑校验。
  - 移除已授予角色并验证访问被撤销。
  - 验证 AND 角色校验需要全部角色。
  - 验证 OR 角色校验只需要任意一个角色。

- `TestRenewFlow`：测试 token 续期。
  - 使用较短 token 超时时间登录。
  - 通过 `/api/token/ttl` 读取初始 TTL。
  - 等待 TTL 减少。
  - 通过 `/api/token/renew` 续期，期望 TTL 延长。

- `TestAutoRenewFlow`：测试自动续期。
  - 启用 AutoRenew，并设置刷新阈值和续期间隔。
  - 等待 token TTL 进入刷新窗口。
  - 访问受保护接口并验证 TTL 延长。
  - 立即再次访问并验证续期间隔会阻止重复增长。

- `TestRenewBoundaryFlow`：测试手动续期边界。
  - 在 HTTP 层拒绝 0 和负数续期值。
  - 接受有效续期值并验证新的 TTL。

- `TestTokenExpiredFlow`：测试 token 过期。
  - 使用 1 秒超时时间登录。
  - 等待超时。
  - 请求受保护接口，期望未授权。

- `TestActiveTimeoutFlow`：测试无操作超时。
  - 使用较长绝对 TTL 和较短 active timeout 登录。
  - 等待无操作超时。
  - 请求受保护接口，期望返回 active-timeout 代码。

- `TestKickoutAndReplaceFlow`：测试 token 状态变更。
  - 踢出当前 token，期望旧 token 被拒绝。
  - 替换当前 token，期望旧 token 被拒绝。

- `TestSessionFlow`：测试 session 查询。
  - 登录。
  - 请求 `/api/session`。
  - 断言 login ID 和终端数量。

- `TestMultiTerminalSessionFlow`：测试多终端。
  - 同一账号分别从 web 和 mobile 登录。
  - 使用任一 token 请求 `/api/session`。
  - 断言终端数量为 2。

- `TestTerminalInspectionFlow`：测试终端元数据查询。
  - 从 web 和 mobile 登录。
  - 请求 `/api/terminal`。
  - 断言终端设备信息和在线数量。

- `TestSessionQueryFlow`：测试 session 查询接口。
  - 按 login ID、设备和具体设备 ID 查询 token 列表。
  - 查询终端列表并遍历终端。
  - 搜索 token 值和 session ID。

- `TestSessionAliveFilterFlow`：测试存活 token 过滤。
  - 将一个终端标记为离线。
  - 验证 token 列表查询只返回存活 session token。
  - 验证离线 token 保留准确的失败原因。

- `TestTerminalOperationFlow`：测试终端范围操作。
  - 登出一个具体设备，同时保持另一个终端在线。
  - 踢出某个设备类型的所有终端。
  - 替换某个账号的所有终端。

- `TestTerminalOperationMatrixFlow`：测试账号、设备类型和具体设备操作矩阵。
  - 登出某账号的所有终端。
  - 登出某设备类型的所有终端。
  - 踢出账号终端和具体设备终端。
  - 替换设备类型终端和具体设备终端。

- `TestConcurrencyPolicyFlow`：测试登录并发策略。
  - 同一设备复用共享 token。
  - 不同具体设备 ID 创建新 token。
  - 未提供设备维度时复用账号级 token。
  - 账号级最大登录数量溢出。
  - 溢出模式：logout、kickout、replaced。
  - 设备级最大登录数量溢出。
  - 账号和设备并发范围。
  - 非并发替换，以及账号/设备范围的新设备拒绝。

- `TestDisableFlow`：测试账号禁用和服务禁用。
  - 账号禁用会拒绝旧 token 和新登录。
  - 可以查询账号禁用信息和 TTL。
  - 服务禁用只拒绝 `/api/payment`。
  - 可以查询服务禁用信息、级别和 TTL。

- `TestServiceDisableLevelFlow`：测试服务禁用级别。
  - 以级别 3 禁用某服务。
  - 验证较低或相等级别会被阻止，较高级别允许访问。
  - 解绑服务并验证级别校验被清除。

- `TestUntieFlow`：测试移除禁用状态。
  - 解绑账号禁用并再次登录。
  - 解绑服务禁用并再次访问服务。
  - 解绑设备禁用并从该设备再次登录。

- `TestDeviceDisableFlow`：测试设备类型禁用。
  - 禁用当前账号的 `web` 设备。
  - 可以查询设备禁用信息和 TTL。
  - 验证 web 登录被拒绝。
  - 验证 mobile 登录仍然成功。

- `TestConcreteDeviceDisableFlow`：测试具体设备 ID 禁用。
  - 只禁用 `web/browser-1`。
  - 可以查询具体设备禁用信息和 TTL。
  - 验证 `web/browser-1` 被拒绝。
  - 验证 `web/browser-2` 仍然允许。

- `TestNonceFlow`：测试 nonce。
  - 通过 `/nonce` 生成 nonce。
  - 在不消费 nonce 的情况下检查有效性和 TTL。
  - 通过 `/nonce/verify` 验证一次，期望成功。
  - 再次验证同一个 nonce，期望失败。

- `TestNonceTimeoutFlow`：测试 nonce 自定义 TTL。
  - 生成 1 秒 TTL 的 nonce。
  - 验证它最初有效。
  - 等待过期后验证无法再消费。

- `TestOAuth2AuthorizationCodeFlow`：测试 OAuth2 授权码流程。
  - 生成授权码。
  - 使用授权码换取 access token 和 refresh token。
  - 验证授权码只能使用一次。
  - introspect access token。
  - 刷新 token 并验证旧 access token 无效。
  - 撤销刷新后的 token 并验证其无效。

- `TestOAuth2PasswordAndClientCredentialsFlow`：测试其他 OAuth2 授权方式。
  - password grant 返回用户 token。
  - client credentials grant 返回客户端 token。
  - 错误 client secret 会被拒绝。

- `TestOAuth2ClientManagementFlow`：测试 OAuth2 客户端管理。
  - 注册并查询客户端。
  - 使用注册后的客户端执行 client credentials。
  - 拒绝未允许的 scope。
  - 注销客户端并验证它不能再使用。

- `TestMultiAuthIsolationFlow`：测试多认证体系隔离。
  - 将同一个 ID 分别登录到 user-auth 和 admin-auth。
  - 验证 token 不能跨认证体系使用。
  - 验证权限和角色按 AuthType 隔离。

## 运行

自动化流程测试会强制使用下面这个固定 Redis URL。这样可以让测试行为更接近多进程生产存储语义，也能覆盖 Redis 扫描和清理逻辑。`tests/gin_core_app` 自身在 `Config.RedisURL` 为空时仍然使用内存存储。

Redis URL 示例：

```text
redis://:root@192.168.19.104:6379/0
redis://localhost:6379/0
redis://:password@localhost:6379/0
```

在当前目录运行：

```powershell
go test ./...
```

如果你的本地 Go 环境在 Windows 上设置了 `GOOS=linux`，请切换为 Windows 目标后再运行：

```powershell
$env:GOOS='windows'
$env:GOARCH='amd64'
go test ./...
```
