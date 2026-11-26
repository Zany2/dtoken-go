# 非 SSO 性能重构验收

本记录用于收敛[优化计划](performance-refactor-plan_zh.md)的 F 批验证，区分源码交付、行为回归和实际性能收益。SSO、SSO Redis 与全部 `examples/sso_*` 不在本次范围内。

## 本轮状态

2026-11-26，用户授权执行非 SSO 的 vet、回归与核心 race 检查，随后另行授权 Memory 基准。工具链为 `go1.27.0 windows/amd64`，Memory 阶段显式清空 `DTOKEN_REDIS_URL` 和 `DTOKEN_MANAGER_TEST_STORAGE`。后续真实 Redis 适配器测试通过，引擎回归按用户要求中止，转回内存方法审核；未启动示例服务或执行部署迁移。Go 缓存使用临时目录，未修改全局 Go 配置。

方法审核阶段补充了 14 个方法生命周期与配置测试，修正刷新过期测试的准备时序，并再次通过 24 个非 SSO 模块的 vet 和回归。随后完成关键分支收尾：新增 4 个顶层测试、16 个场景，删除 4 个确认无调用的私有方法；根模块与 Gin 流程的 vet、完整回归再次通过，没有发现需改变业务行为的新缺陷。最新根模块语句覆盖为 89.3%，引擎为 90.4%；911 个含语句的导出函数／方法均命中，10 个空日志方法不适用，仍不代表全部分支和并发组合正确。最新逐方法清单见[分支收尾记录](benchmarks/2026-11-26-memory-branches_zh.md)，前次行为断言及 Redis 的部分进度见 [Memory 方法逻辑验收](benchmarks/2026-11-26-memory-methods_zh.md)。

`go.work` 中全部 **24 个非 SSO 模块的 vet 与回归测试已通过**。根模块和 Gin HTTP 流程测试首次失败，修复下列问题后重跑通过。各模块均使用 `-count=1`，不是沿用测试结果缓存。随后通过全部 120 个 Memory Auth 基准的固定次数检查、基础校验及局部算法重复测量，并修复测量发现的批量权限删除退步；最终又通过根模块和 Gin 流程模块的 vet、完整回归。原始结果和范围见[Memory 测量记录](benchmarks/2026-11-26-memory-report_zh.md)。

此前已完成现有 **120 个 Auth 场景各 300 ms、重复 3 次的完整 Memory 矩阵**，以及并发读写在 GOMAXPROCS=1/8 下的补充测量，合计 384 个样本通过。详见[完整工作负载报告](benchmarks/2026-11-26-memory-workloads_zh.md)。该测量阶段没有修改生产代码或基准实现；其后的方法审核新增了测试，已重新执行模块回归。

| 范围 | 本轮结果 |
| --- | --- |
| 根模块（含引擎、HTTP 公共层、事件、能力及 migration） | vet、完整测试通过 |
| codec/jsonv2、codec/msgpack | vet、测试通过；jsonv2 单独启用实验特性 |
| integrations 下 Beego、Chi、Echo、Fiber、GF、Gin、Hertz、Kratos | 8 个模块 vet、测试通过 |
| examples 下上述 8 个框架及 quick_start | 9 个模块 vet、测试通过；未执行示例入口 |
| logging/gf、storage/redis | vet、非联网测试通过；另有先前真实 Redis 适配器全套通过记录 |
| tests/gin_core_app、tests/gin_core_flow | vet、测试通过；HTTP 流程使用内存存储及测试内 HTTP 服务 |
| 核心 race | 执行返回 `go: -race requires cgo`；CGO 未启用，PATH 未找到 C 编译器，未自动安装 |
| Memory 基准 | 120 个场景各 20 次检查及各 3 次计时采样通过；并发读写补充 GOMAXPROCS=1/8，局部算法对照通过，原始输出已保留 |
| 真实池维护并发 | 7 个合并、池满回退、恢复及关闭场景，在 GOMAXPROCS=1/4/8 下各重复 20 次通过；非 race、非持续过载基准 |
| 真实 Redis | 适配器全套通过；引擎回归未完成，按用户要求停止，未执行 HTTP Redis 流程或基准 |
| 部署负载、部署迁移 | 未执行，不计为通过 |

代码修正、非联网模块回归及本轮授权的 Memory 测量已完成；完整性能／发布验收尚未完成。已测收益只适用于报告中的场景，不能推断 race 无问题、Redis 生命周期通过、整个框架提升相同倍数或实际数据已经迁移。

当前主要源码改造和现有 Memory 验收已收尾，无新增失败要求继续重构。基础校验成本稳定，但大 Session 的登录、手动续期、管理查询和 Provider 回退仍有显著数据处理成本；增加并发数也不保证写入吞吐提高。先完成剩余环境验收，再根据实际负载决定是否另立会话拆分或存储竞争优化任务。

维护补充验收仅新增测试与测量记录，没有继续修改生产实现。使用真实单工作线程 Ants 池，验证 64 次同 Token 并发合并、32 个不同 Token 的饱和回退、恢复后重新接收，以及共享池监听器的同步回退和关闭等待；随后 Gin 流程模块、engine、event、pool/ants 的 vet 与完整回归通过。细节、命令和原始输出见上述 Memory 测量记录。

本轮处理的回归问题：

- 维护任务测试仍导入已删除的公共 `utils` 路径，改为内部工具包。
- 迁移用例将字符串传给类型化 `Auth.Login`，改用 `LoginID`。
- Session 数据成本测试缩短 Token 有效期后未同步续期阈值，导致配置校验失败，补齐有效阈值。
- 残留列表断言同步懒清理约定：拒绝接收或未触达数量限制时允许保留索引，但有效查询必须排除失效、跨账号凭据，损坏活跃时间不得通过校验。
- HTTP Session 写入测试在首次 Token 读取后撤销凭据，验证锁内复核会拒绝写入；不再将直接修改底层存储模拟成框架账号锁内的跨键事务。
- HTTP 账号封禁用例区分已撤销凭据的 401 与新登录被封禁的 403，并验证解封不恢复旧 Token、新登录可以使用。
- Redis 适配器测试在显式配置地址后连接失败会报错；未配置地址仍可跳过，TLS 地址不能用于仅支持普通 Redis 配置的构造器测试，仍明确跳过该项。

## 复现模块回归

从仓库根目录，在独立 PowerShell 会话运行。以下清空 Redis 设置；测试只使用内存路径，Redis 集成用例的跳过不是 Redis 验收通过。

```powershell
$env:GOTOOLCHAIN = 'local'
$env:DTOKEN_REDIS_URL = ''
$env:DTOKEN_MANAGER_TEST_STORAGE = ''
$taskRoot = (Get-Location).Path
$env:GOWORK = Join-Path $taskRoot 'go.work'
$workspace = go work edit -json | ConvertFrom-Json
if ($LASTEXITCODE -ne 0) { throw 'Cannot read workspace' }
foreach ($entry in $workspace.Use) {
    $relative = $entry.DiskPath.Replace('\', '/').TrimStart('./')
    if ($relative -eq 'sso' -or $relative.StartsWith('sso/') -or $relative.StartsWith('examples/sso_')) { continue }
    $directory = if ($relative -eq '') { $taskRoot } else { Join-Path $taskRoot $relative }
    Push-Location -LiteralPath $directory
    try {
        $env:GOEXPERIMENT = if ($relative -eq 'codec/jsonv2') { 'jsonv2' } else { '' }
        go vet ./...
        if ($LASTEXITCODE -ne 0) { throw "vet failed: $directory" }
        go test -count=1 -timeout=5m ./...
        if ($LASTEXITCODE -ne 0) { throw "tests failed: $directory" }
    } finally { Pop-Location }
}
```

本轮 Windows 执行环境使用 `GOOS=windows`、`GOARCH=amd64`；其他环境应使用能在当前主机执行的目标，避免交叉编译测试程序无法运行。`codec/jsonv2` 单独启用实验特性。根模块 `./...` 不覆盖工作区的独立子模块，必须遍历 `go.work`。

## race 与运行边界

在具备当前平台所需 CGO 工具链的环境运行：

```powershell
$env:GOEXPERIMENT = ''
$env:CGO_ENABLED = '1'
go test -race -count=1 -timeout=10m ./internal/engine ./event ./migration ./nonce ./oauth2 ./ticket ./shortkey ./httpauth/... ./storage/memory ./pool/ants
```

仅设置 `CGO_ENABLED=1` 不会安装 C 编译器。此命令不覆盖所有 Web 框架适配器的 race 检查，也不覆盖跨进程分布式竞态。

真实 Redis 验收需使用独立测试数据库，分别运行引擎、`storage/redis` 与 `tests/gin_core_flow`。适配器共享契约包含清空存储的测试，不能套用 HTTP 流程“只删除随机前缀”的承诺。引擎设置 `DTOKEN_MANAGER_TEST_STORAGE=redis`；Redis 适配器和 HTTP 流程读取 `DTOKEN_REDIS_URL`；公开 API 基准还需 `DTOKEN_BENCH_STORAGE=redis`。逐项核对跳过原因，不能只看进程退出码。详见[基准说明](../../tests/gin_core_flow/README_zh.md)。

## 完成标准

| 交付项 | 验收要求 |
| --- | --- |
| 功能与模块 | 非 SSO 模块 vet 和回归通过，失败已修复；独立发布模块与消费者解析另按既有发布流程检查 |
| 并发 | 核心 race 通过；现有池拒绝、关闭等待、任务代次、刷新消费等用例通过 |
| Redis | 真实连接及核心生命周期验证通过；明确哪些用例仍在内存运行或被跳过 |
| 性能 | 运行现有串行、并发读写、终端清理和 Provider 基准；保留版本、配置、数据规模、原始输出及同语义对比结果 |
| 部署负载 | 维护合并、过载与 p95/p99 依据实际应用测试，不能由串行平均值或 RunParallel 的 ns/op 推断 |
| 迁移 | 隔离数据演练凭据／封禁转换、TTL 扣减、刷新关系及停写切换；单元测试不等于部署迁移完成 |

当前阶段不以新增缓存、拆分 Session 或分布式事务来替代验收。若检查没有发现新的正确性问题，应先完成运行验证；可选优化由测量结果决定。
