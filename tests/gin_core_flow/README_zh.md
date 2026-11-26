# Gin 核心流程测试

本目录包含 `tests/gin_core_app` 的 HTTP 流程测试，以及公开 Auth API 到存储的串行与并发性能基准。

HTTP 流程测试通过 `httptest.NewServer` 在测试进程内启动 Gin 应用，然后按照真实 HTTP 流程调用接口。

## Auth 性能基准

`auth_benchmark_test.go` 直接调用公开 Auth API，使用内置 Memory 或 Redis 适配器；不经过 HTTP。基准默认使用 Memory，Redis 必须显式选择；选择 Redis 后缺少地址或连接失败会报错，不回退或跳过。

2026-11-26 已完成全部 120 个 Auth 场景各 3 次 Memory 计时采样及并发线程数补充测量，见[完整工作负载报告](../../docs/design/benchmarks/2026-11-26-memory-workloads_zh.md)。先前局部对照与真实池并发验证见[测量记录](../../docs/design/benchmarks/2026-11-26-memory-report_zh.md)；Redis 尚未测量。以下是从仓库根目录运行的通用命令：

```powershell
$env:DTOKEN_BENCH_STORAGE='memory'
go test ./tests/gin_core_flow -run '^$' -bench 'BenchmarkAuth(Validation|Writes)' -benchmem -count=3

$env:DTOKEN_BENCH_STORAGE='redis'
$env:DTOKEN_REDIS_URL='redis://localhost:6379/0'
go test ./tests/gin_core_flow -run '^$' -bench 'BenchmarkAuth(Validation|Writes)' -benchmem -count=3
```

完整矩阵较慢，可先只测基础校验：

```powershell
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthValidation$/^redis$/^terminals=1$/^extra=0$/^(is-login|check-login)$' -benchmem -count=3
```

| 场景 | 测量内容 |
| --- | --- |
| `is-login` / `check-login` | 关闭活跃超时和自动续期的基础有效性校验 |
| `details` | 完整 Token 详情，包括存在时的 Extra 读取与解码 |
| `active-foreground` | 活跃超时校验及任务提交；返回后的排空等待不计时 |
| `active-complete` | 校验直到活跃维护任务完成，包括排空等待 |
| `renew-active-complete` | 校验直到自动续期和活跃维护完成；每次都满足续期条件，并核对成功续期事件数 |
| `login-logout` | 一次完整登录加退出，临时终端退出后恢复原终端数量 |
| `manual-renew` | 对既有 Token 手动续期一小时 |

每组使用 1、100、1000 个终端和 0、1 KiB、64 KiB 的 Extra 字符串。Extra 只放在测量目标上，登录退出场景的临时 Token 也使用同等大小的 Extra；大小指字符串内容，不含编码开销。账号关闭共享、允许并发登录且不限制终端数量。基础 TTL 和启用时的活跃超时均为一小时，单个样本的准备与测量应在其有效期内完成。

构建、数据准备、结果核对和清理不计时。每个样本使用随机前缀，关闭 Auth 并排空任务后只删除该前缀的键；Redis 建议使用专用测试数据库。登录退出保持终端列表长度稳定，但历史终端计数仍会递增。

活跃场景每轮排空任务，避免任务合并造成少做后台工作的虚假收益；`tasks/op` 应为 1，其余场景应为 0。活跃场景启用仅计数、不输出日志的错误观察器，自动续期场景另有同步事件计数器；这些观察与同步开销属于测量结果。普通校验和写入场景关闭日志与事件观察。

`active-foreground` 的工作线程可能与前台同时执行，因此其 `B/op`、`allocs/op` **不是纯前台分配量**，返回前已发生的后台工作也可能影响耗时。完整场景包含排空同步成本；这些都是串行平均值，不代表并发吞吐、池饱和表现或 p95/p99。基础并发校验与永久 TTL 对比见下文。比较版本时应保存工具链、Redis 版本与部署位置、配置、数据规模及原始输出，并保持语义一致。

### 终端残留与 Provider 基准

`auth_query_benchmark_test.go` 复用相同存储选择和清理方式。在仓库根目录执行：

```powershell
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuth(TerminalQueries|PermissionProvider)$' -benchmem -count=3
```

- `BenchmarkAuthTerminalQueries`：10、100、1000 个终端，凭据缺失比例为 0%、50%、100%。`raw` 返回所有索引条目；`alive` 筛选全部存活条目；`device-alive` 仅查询最后登录的具体设备。每轮检查数量，结束后核对 Token 内容、顺序及索引未被清扫。
- `BenchmarkAuthPermissionProvider`：1、100、1000 个终端，比较 `hit`（Provider 返回权限）、`fallback`（返回 nil）、`empty-deny`（返回非 nil 空列表）。三个场景的 Session 都有同一权限，前两者必须通过，空列表必须拒绝；每次都应调用一次 Provider，结果显示 `provider-calls/op`。

终端基准在计时前直接删除索引前半段或全部条目对应的主记录，保留 Session；不创建 Extra、活跃或刷新记录。这是在实际存储上构造自然过期后的缺失状态，**不测量 TTL 到期过程、Redis 后台清理或登录触发的索引清扫**。准备代码使用当前版本的命名空间函数，移植到历史版本时须适配其键格式。100% 缺失场景的 Session 仍保留一小时有效期。

两组基准均关闭活跃维护、自动续期和日志，`tasks/op` 应为 0。Provider 是返回固定切片的本地回调，计数开销包含在结果中；这些数值不能代表应用数据库或远程权限服务的耗时。测量循环保留错误和数量断言，其开销计入结果；准备数据、最终完整结果核对与清理不计时。

### 永久 TTL、并发校验与登录清理

`auth_lifecycle_benchmark_test.go` 继续使用相同的存储选择和随机前缀隔离：

```powershell
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthPermanentValidation$' -benchmem -count=3
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthParallelValidation$' -benchmem '-cpu=1,4,8' -count=3
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthParallelWrites$' -benchmem '-cpu=1,4,8' -count=3
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthLoginPruning$' -benchtime=20x -benchmem -count=3
```

- `PermanentValidation`：比较一小时 TTL 与永久配置下签发的 Token，计时调用 `CheckLogin`，结束后检查存储 TTL。只覆盖基础校验，不测量有限期转永久、Extra 或活跃标记的永久化。
- `ParallelValidation`：同一个 Auth 内固定准备 64 个凭据；`same-account` 为同账号的 64 个终端，`many-accounts` 为 64 个账号各一个终端。`RunParallel` 的每个工作协程轮询这些凭据，分别使用不同起点；`-cpu` 控制各轮 GOMAXPROCS，默认工作协程数随之变化。汇总失败后由主协程判定结果。`ns/op` 是总墙钟时间除以总操作数，表示吞吐对应的平均成本，**不是单请求延迟或尾延迟**；结果也包含工作调度和轮询开销。
- `LoginPruning`：10、100、1000 个终端，模拟 50% 或 100% 主记录缺失；每次登录将账号上限设为原终端数量，强制执行名额检查和残留清理。每轮在计时外恢复相同 Session 原始快照，计时完整登录，然后停止计时核对持久化列表、新 Token 有效性并退出临时终端；最终核对原有存活凭据。50% 残留场景保留存活项，100% 残留场景覆盖清空旧索引后重新登录。没有可观察的事件监听器。

三组均关闭活跃超时、自动续期及日志，`tasks/op` 应为 0。除永久凭据外，数据有效期为一小时；单个样本须在有效期内完成。登录清理每轮都恢复数据，实际耗时会显著长于报告的计时耗时，因此推荐使用有限次数 `-benchtime=20x`。反复恢复的测试数据会影响缓存热度，该基准也不模拟真实的 TTL 到期过程。它没有开启共享登录、设备作用域或仍有活跃终端时的超限淘汰。

`ParallelWrites` 补充同账号与多账号的并发登录／退出：每个工作协程使用独立设备 ID；同账号模式竞争同一账号锁，多账号模式每个协程持有独立账号。一次操作包含登录和退出，结束后确认没有遗留终端；失败次数在工作协程结束后汇总。准备独立账号的成本也包含在计时内，不能与预先创建全部账号的负载直接等同。共享、数量上限、活跃维护和自动续期均关闭；`tasks/op` 应为 0。

运行以上场景不等于完成性能验收：后台维护合并与池过载、尾延迟及迁移演练仍须结合实际部署验证。全部 Auth 基准可用 `-bench '^BenchmarkAuth'` 选择，但完整矩阵耗时较长，建议分组执行并保存原始结果。模块回归与发布状态见[非 SSO 验收记录](../../docs/design/performance-acceptance_zh.md)。

### 真实池维护并发验证

`auth_maintenance_load_test.go` 强制使用内置 Memory 和单工作线程 Ants 池，不连接 Redis，也不经过 HTTP。通道控制任务和监听器的执行时序，分别检查：

- `TestAuthRealPoolCoalescingAndClose`：64 次同 Token 并发校验复用在途任务；共享池的异步监听器遭拒后同步执行；关闭等待任务和监听器完成，并拒绝新校验。
- `TestAuthRealPoolSaturationAndRecovery`：32 个不同 Token，组合同账号／多账号及仅活跃／仅续期／两者启用；池满时更新活跃值且不延长 TTL、跳过可选续期，池恢复后可以重试。

```powershell
go test ./tests/gin_core_flow -run '^TestAuthRealPool' -count=20 '-cpu=1,4,8' -timeout=2m
```

2026-11-26，上述 7 个场景共 420 次执行通过；已有活跃／续期基准的 12 个配置各 3 次采样也通过，详见[测量记录](../../docs/design/benchmarks/2026-11-26-memory-report_zh.md)。这验证受控交错及正常维护成本，不替代 race、持续过载或 p95/p99 测试。

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
  - 验证登录后的初始续期间隔会阻止已经满足阈值的请求续期。
  - 间隔结束后访问受保护接口，等待实际 TTL 增长和续期事件。
  - 立即再次访问，验证间隔内不会产生额外的续期事件。
  - 通过 Manager 只读接口观察 TTL，避免测量动作触发续期。

- `TestRenewConfigurationMatrixFlow`：测试关闭续期、刷新阈值和无间隔时的重复续期。
  - 使用有超时限制的存储 TTL 轮询，替代异步续期后的固定等待。

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

- `TestSessionExpiredTokenFilterFlow`：测试自然过期过滤。
  - 保持移动端登录有效，等待短有效期的 web token 过期。
  - 验证 `alive=false` 保留两个终端条目，`alive=true` 仅返回移动端 token。

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
  - 刷新后验证旧 access token 和 refresh token 均不可再用，并在撤销前确认新 access token 有效。
  - 撤销刷新后的 token 并验证其无效。

- `TestOAuth2AuthorizationCodeBindingFlow`：测试授权码与客户端、回调地址的绑定。
  - 拒绝其他已注册客户端换码，以及不匹配的回调地址。
  - 验证上述请求被拒绝后，原客户端仍可使用同一授权码合法换码。

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
  - 读取两侧身份，对比权限列表和角色列表，验证 AuthType 隔离。

- `TestDecodeFlowData`：拒绝缺失、null、格式错误及类型错误的数据，并在复用解码目标时清除旧字段。
- `TestFlowClientDefaultsToMemory`：在未设置 Redis 环境变量时执行真实登录和受保护 HTTP 请求。
- `TestFlowStorageCleanup`：删除真实测试实例的键，检查认证和 nonce 失效，并保留同一后端的无关键。

## 运行

自动化流程测试默认使用内存存储。设置 `DTOKEN_REDIS_URL` 后，流程场景会使用 Redis，覆盖其存储和扫描行为。默认命令无需外部服务，也会实际执行流程测试。

每个测试实例使用随机的 `dt-gcf-<hex>:` 键前缀，不同测试进程之间也保持隔离。清理时只扫描并删除该实例前缀下的键。建议使用专门的测试 Redis 数据库。

Redis URL 示例：

```text
redis://localhost:6379/0
redis://:password@localhost:6379/0
```

未设置 `DTOKEN_REDIS_URL` 时，流程测试使用内存存储。配置 Redis 后若初始化失败，或应用配置、创建失败，测试会明确失败，不再静默跳过。

在当前目录运行：

```powershell
go test ./...
```

如需启用 Redis 流程测试，请先设置地址：

```powershell
$env:DTOKEN_REDIS_URL='redis://:password@localhost:6379/0'
go test ./...
```

如果你的本地 Go 环境在 Windows 上设置了 `GOOS=linux`，请切换为 Windows 目标后再运行：

```powershell
$env:GOOS='windows'
$env:GOARCH='amd64'
go test ./...
```
