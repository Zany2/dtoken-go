# Memory 性能测量记录

2026-11-26，按用户授权运行 Memory 基准，不连接 Redis，不执行实际迁移。环境为 Go 1.27.0、Windows/amd64、AMD Ryzen 7 7840HS，`-cpu=4`；测量命令依次运行。工作区基于 `fff8fe862d73c4a322d82062e12f9d148032ef05`，包含尚未提交的重构，不能用该提交单独复现，也不是与 develop 的完整版本对照。

## 范围与原始记录

| 记录 | 内容与结果 |
| --- | --- |
| [Memory 全场景](2026-11-26-memory-smoke.txt) | 全部 120 个公开 Auth 基准各执行 20 次，通过工作负载及结果断言；仅作场景检查 |
| [局部算法初测](2026-11-26-algorithms-before.txt) | 主记录解析、权限匹配、权限删除，每场景 300 ms、3 次采样；发现大删除请求的退步 |
| [删除算法首次调整](2026-11-26-access-removal-after.txt) | 限制集合大小后的中间结果，仍有扩容和重复扫描成本；不是最终实现 |
| [删除算法最终复测](2026-11-26-access-removal-final.txt) | 按较小输入建立集合，延迟分配保留列表，3 次采样通过 |
| [基础校验](2026-11-26-memory-validation.txt) | 8 个配置组合各 300 ms、3 次采样，通过有效性及后台任务数量断言 |
| [维护成本](2026-11-26-memory-maintenance.txt) | 活跃前台、完整活跃维护、续期加活跃维护的 12 个配置组合各 300 ms、3 次采样通过 |
| [真实池并发回归](2026-11-26-maintenance-regression.txt) | 7 个维护场景在 GOMAXPROCS=1/4/8 下各重复 20 次通过，不是性能基准或 race |

下表时间取三次样本的中位数，仅描述本机结果。样本存在波动，未做统计显著性检验；20 次场景检查尤其不能用来推断稳定性能。局部基准的 `previous`、`split` 是测试内保留的算法，`legacy` 是旧完整 Token JSON 解码，不代表运行了旧版框架。

## 基础登录校验

直接调用公开 Auth 实例，使用内置 Memory，关闭活跃超时、自动续期、共享和日志，TTL 一小时。数据准备与清理不计时；不经过 HTTP、Redis 或网络。

| 终端数量 | Extra 字符串大小 | IsLogin ns/op | CheckLogin ns/op | B/op | allocs/op |
| --- | --- | --- | --- | --- | --- |
| 1 | 0 | 269.9 | 268.3 | 224 | 2 |
| 1 | 64 KiB | 333.1 | 298.5 | 224 | 2 |
| 1000 | 0 | 300.4 | 279.0 | 224 | 2 |
| 1000 | 64 KiB | 273.3 | 285.0 | 224 | 2 |

所有组合均为 `0 tasks/op`。分配量没有随终端数和 Extra 大小增长，符合基础校验只读小型主记录、不加载 Session 和 Extra 的设计。这里仍有固定字段解析和分配，不是完全不解码或零分配；也不包含开启活跃检查、续期和封禁组合后的成本。

## 权限与角色删除

初测发现，按请求长度建立集合会在“8 个已有权限、1032 项删除请求”场景产生 54,912 B/op、8 次分配，中位耗时 36,199 ns。最终实现按较小输入建集合：大请求索引已有值，命中后从集合移除；小请求继续索引待删除值。删除结果保持请求首次出现顺序，重复存储值全部移除，空字符串不作为删除请求，输入数组不变，未命中不复制原列表。

最终同次运行与测试内旧算法对照如下，时间单位为 ns/op。所有请求和已有值的规模均列出，避免只展示有利场景。

| 已有值 | 删除请求 | 最终 / 旧算法 ns/op | 最终 / 旧算法 B/op | 最终 / 旧算法 allocs/op |
| --- | --- | --- | --- | --- |
| 8 | 1 项未命中 | 96.46 / 124.3 | 0 / 16 | 0 / 1 |
| 8 | 首尾 2 项 | 303.6 / 369.0 | 144 / 224 | 2 / 3 |
| 8 | 全部 8 项 | 463.7 / 803.6 | 128 / 128 | 1 / 1 |
| 8 | 1032 项，含全部已有值 | 3,027 / 4,957 | 128 / 18,432 | 1 / 1 |
| 1024 | 1 项未命中 | 14,886 / 19,993 | 0 / 54,624 | 0 / 6 |
| 1024 | 首尾 2 项 | 24,249 / 39,276 | 16,416 / 89,776 | 2 / 15 |
| 1024 | 全部 1024 项 | 116,227 / 161,726 | 73,040 / 127,648 | 6 / 11 |
| 1024 | 2048 项，含全部已有值 | 98,182 / 149,530 | 73,040 / 141,984 | 6 / 11 |

原始标签 `request_set` 保留以便对齐初测与复测，最终集合大小已不再固定取请求长度。部分窄请求的最终耗时高于初测，同时对照组也有波动，不能据此声称所有场景相对本轮修改前都提速；最终同次对照的中位数更低，内存分配改善可直接核对。这里仅测切片处理，不包含 Session 读写、Codec 或事件投递。

新增大请求兼容测试覆盖未命中、空值、稀疏命中、重复值、Unicode、输入保护和不变列表复用。最终实现通过根模块与 `tests/gin_core_flow` 的 vet 和完整回归（`-count=1`）。此前全部 24 个非 SSO 模块已通过 vet 和非联网回归；本次私有算法改动后未重复运行其他独立模块。

## 解析和匹配局部对照

| 场景 | 当前 / 对照 ns/op | 当前 / 对照 B/op | 当前 / 对照 allocs/op |
| --- | --- | --- | --- |
| 冒号分段权限匹配 | 66.24 / 118.7 | 0 / 128 | 0 / 2 |
| 斜线分段权限匹配 | 63.88 / 115.1 | 0 / 128 | 0 / 2 |
| 主记录解析，Extra 为 0 | 162.0 / 764.2 | 128 / 128 | 1 / 1 |
| 主记录解析，Extra 为 1 KiB | 162.1 / 2,562 | 128 / 1,552 | 1 / 8 |
| 主记录解析，Extra 为 64 KiB | 160.3 / 67,075 | 128 / 66,077 | 1 / 8 |

权限匹配由兼容性用例验证语义一致。主记录对照体现的是存储职责拆分：紧凑记录不包含 Extra，旧 JSON 对照解码整个 TokenInfo；不能解释为相同完整对象的解码速度提升，更不能据此宣称 Redis 或完整登录流程提升同样倍数。

## 复现

从仓库根目录运行，使用当前工作区源码：

```powershell
$env:GOCACHE = Join-Path $env:TEMP 'dtoken-nonsso-go-cache'
$env:GOTOOLCHAIN = 'local'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:GOEXPERIMENT = ''
$env:DTOKEN_REDIS_URL = ''
$env:DTOKEN_MANAGER_TEST_STORAGE = ''
$env:DTOKEN_BENCH_STORAGE = 'memory'
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuth' -benchtime=20x -benchmem -cpu=4 -count=1 -timeout=10m
go test ./internal/engine -run '^$' -bench '^(BenchmarkTokenRecordDecode|BenchmarkPermissionSegments|BenchmarkAccessRemoval)$' -benchmem -benchtime=300ms -cpu=4 -count=3 -timeout=5m
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthValidation$/^memory$/^terminals=(1|1000)$/^extra=(0|65536)$/^(is-login|check-login)$' -benchmem -benchtime=300ms -cpu=4 -count=3 -timeout=5m
```

每条命令应单独检查退出码。删除算法调整期间额外两次只选择 `^BenchmarkAccessRemoval$`，其他参数保持一致。当前源码只能复现最终算法，初测及中间结果用于保留发现和修复过程。

最终关键源码 SHA256（原始文件字节，换行变化也会改变摘要）：

```text
internal/engine/internal_access_values.go 9C23010C54D947F0CA9686088BF8134B32C7732F165DCEBC71FEF1C8CAFDE247
internal/engine/access_algorithms_test.go 7211AA3205294C80AA14A4520C22A6FDD35A5A61EE665BF581A381280936A50A
internal/engine/internal_token_record.go 0D30620201166C90FECBD2C3110954336EB56744A384A70F1F37B0E0CC171733
internal/engine/internal_token_record_test.go E3E4DDBBB878CF1BC07AB6A1D3507C320F8146227CD77DBB66EB4DCA996D2640
tests/gin_core_flow/auth_benchmark_test.go 1F0394E47D641BA8B01A2B1B7B213FE13A272424D9397140CC1E592E39F8A8ED
```

## 真实池维护补充验收

`auth_maintenance_load_test.go` 通过公开 Auth API、内置 Memory 和固定容量为 1 的真实 Ants 池，使用通道控制交错顺序。复用已有任务及错误计数器；只有续期事件启用，避免登录日志事件消耗测试池。没有修改生产逻辑，也没有增加生产配置或依赖。

- 同一个 Token 先接收一个任务，再发起 64 次并发校验；被暂缓的维护任务保持唯一，活跃时间合并，最终只续期一次。
- 维护和异步监听器共用单工作线程池；监听器的嵌套提交遭拒后同步执行，既不死锁也不丢事件。关闭拒绝新请求，等待维护及被屏障暂缓的监听器全部完成。最终计数为 1 次接收、1 次监听器拒绝回退、1 次续期，无维护错误日志。
- 池被外部任务占满时，32 个不同 Token 并发校验；分别覆盖同账号／多账号及仅活跃、仅续期、两者均启用，共 6 个场景。32 次维护提交均拒绝，可选续期不执行，活跃值在调用返回前更新且不延长 TTL；释放工作线程后可重试并成功接收，续期事件与配置一致。

7 个场景在 `-count=20 '-cpu=1,4,8'` 下共执行 420 次，通过；随后 Gin 流程模块、engine、event、pool/ants 的 vet 与完整回归通过。初次测试包含了准备数据时产生的登录事件日志任务，修正测试观察范围后通过，并非生产逻辑缺陷。测试强制使用 Memory，不受 Redis 环境变量影响。

串行维护基准继续使用默认 Ants 池，沿用上文工具链和 `-cpu=4`，每项 300 ms、3 次采样。下表为中位数，单位 ns/op；全部 `tasks/op=1`，自动续期逐次核对成功事件。

| 终端数 | Extra | 活跃前台 | 完整活跃维护 | 完整续期与活跃维护 |
| --- | --- | --- | --- | --- |
| 1 | 0 | 2,851 | 4,081 | 4,930 |
| 1 | 64 KiB | 2,671 | 4,030 | 5,308 |
| 1000 | 0 | 2,767 | 4,183 | 5,029 |
| 1000 | 64 KiB | 2,624 | 4,133 | 5,440 |

完整活跃维护均为 1,472 B/op、20 次分配；完整续期与活跃维护无 Extra 时为 2,184 B/op、30 次分配，64 KiB Extra 时为 2,280 B/op、31 次分配。Extra 会增加附属键生命周期维护，但这里没有按其内容大小重新解码。

前台基准在返回后暂停计时并等待后台排空，因此墙钟耗时较长，报告的约 851—866 B/op、11 次分配不能当作请求及后台总成本。工作线程与请求可能同时执行，也不能视为完全隔离的前台成本。完整维护包括任务排空同步开销；这里无旧版同场景对照，不计算版本提升比例。真实池测试验证固定交错下的行为，串行基准测量正常维护成本，两者都不能代表生产持续过载或 p95/p99。

复现新增检查与测量，沿用上文环境变量：

```powershell
go test ./tests/gin_core_flow -run '^TestAuthRealPool' -count=20 '-cpu=1,4,8' -timeout=2m
go test ./tests/gin_core_flow -run '^$' -bench '^BenchmarkAuthValidation$/^memory$/^terminals=(1|1000)$/^extra=(0|65536)$/^(active-foreground|active-complete|renew-active-complete)$' -benchmem -benchtime=300ms -cpu=4 -count=3 -timeout=5m
go vet ./tests/gin_core_flow/... ./internal/engine ./event ./pool/ants
go test ./tests/gin_core_flow/... ./internal/engine ./event ./pool/ants -count=1 -timeout=3m
```

新增测试 SHA256：`3EDB7F1EE6150FFB151EABE20CBCE824F48B246C065CFEF1474B2847E34A53A3`。基准源码没有变化，沿用上面的摘要。

## 尚未完成

后续已完成全部 120 个现有 Auth 场景各 3 次计时采样，覆盖串并发读写、清理和 Provider，并补测并发线程数，详见[完整工作负载报告](2026-11-26-memory-workloads_zh.md)。本页保留之前的局部测量及修复证据。

真实 Redis 生命周期与测量、具备 C 编译器的核心 race、实际部署的持续过载及 p95/p99 负载、隔离数据部署迁移演练仍未完成。合并、池满回退和关闭的真实池组合验证已完成。暂不据此扩展 Session 拆分、全局缓存或强制事务；完整状态见[验收记录](../performance-acceptance_zh.md)。
