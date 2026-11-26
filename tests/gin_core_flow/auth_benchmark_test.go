package gin_core_flow_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/pool/ants"
	"github.com/Zany2/dtoken-go/storage/memory"
	redisstorage "github.com/Zany2/dtoken-go/storage/redis"
)

// benchmarkTaskPool observes a real asynchronous pool and drains each serial benchmark iteration. benchmarkTaskPool 观察真实异步池，并排空每个串行基准迭代的任务。
type benchmarkTaskPool struct {
	adapter.Pool
	pending  sync.WaitGroup
	accepted atomic.Int64
	rejected atomic.Int64
}

// Submit retains asynchronous admission while tracking completion outside the engine. Submit 保留异步接收语义，在引擎外跟踪完成状态。
func (p *benchmarkTaskPool) Submit(task func()) error {
	p.pending.Add(1)
	if err := p.Pool.Submit(func() { defer p.pending.Done(); task() }); err != nil {
		p.pending.Done()
		p.rejected.Add(1)
		return err
	}
	p.accepted.Add(1)
	return nil
}

// benchmarkErrorLog detects maintenance failures that do not propagate through login checks. benchmarkErrorLog 检测不通过登录校验返回的维护错误。
type benchmarkErrorLog struct {
	adapter.Log
	errors atomic.Int64
}

// Error records asynchronous errors without writing benchmark output. Error 记录异步错误而不写入基准输出。
func (l *benchmarkErrorLog) Error(...any) { l.errors.Add(1) }

// Errorf records formatted asynchronous errors. Errorf 记录格式化异步错误。
func (l *benchmarkErrorLog) Errorf(string, ...any) { l.errors.Add(1) }

// authBenchmarkFixture owns a fixed account dataset and observes maintenance outcomes. authBenchmarkFixture 持有固定账号数据集并观察维护结果。
type authBenchmarkFixture struct {
	auth     *dtoken.Auth
	pool     *benchmarkTaskPool
	log      *benchmarkErrorLog
	token    string
	renewals *atomic.Int64
}

// authBenchmarkBackend requires explicit Redis selection rather than silently falling back or skipping. authBenchmarkBackend 要求显式选择 Redis，不静默回退或跳过。
func authBenchmarkBackend(b *testing.B) string {
	b.Helper()
	backend := strings.TrimSpace(os.Getenv("DTOKEN_BENCH_STORAGE"))
	if backend == "" {
		backend = "memory"
	}
	if backend != "memory" && backend != "redis" {
		b.Fatalf("DTOKEN_BENCH_STORAGE must be memory or redis")
	}
	if backend == "redis" && strings.TrimSpace(os.Getenv("DTOKEN_REDIS_URL")) == "" {
		b.Fatal("Redis benchmarks require DTOKEN_REDIS_URL")
	}
	return backend
}

// newAuthBenchmark creates a standalone Auth with production adapters and a fixed terminal count. newAuthBenchmark 使用内置适配器及固定终端数量创建独立 Auth。
func newAuthBenchmark(b *testing.B, backend string, terminals, extraBytes int, active, renew bool, provider adapter.AccessProvider) authBenchmarkFixture {
	b.Helper()
	return newAuthBenchmarkWithTimeout(b, backend, terminals, extraBytes, active, renew, provider, 3600)
}

// newAuthBenchmarkWithTimeout selects finite or permanent credentials before seeding. newAuthBenchmarkWithTimeout 在准备数据前选择有限期或永久凭据。
func newAuthBenchmarkWithTimeout(b *testing.B, backend string, terminals, extraBytes int, active, renew bool, provider adapter.AccessProvider, timeout int64) authBenchmarkFixture {
	b.Helper()
	prefix := flowKeyPrefix(b)
	var storage adapter.Storage
	if backend == "redis" {
		value, err := redisstorage.NewStorage(strings.TrimSpace(os.Getenv("DTOKEN_REDIS_URL")))
		if err != nil {
			b.Fatal(err)
		}
		storage = value
		b.Cleanup(func() {
			if err := value.Close(); err != nil {
				b.Error(err)
			}
		})
	} else {
		storage = memory.NewStorage()
	}

	// Auth drains workers before prefix cleanup, and Redis closes last. Auth 先排空任务，再清理前缀，最后关闭 Redis。
	b.Cleanup(func() { clearFlowStorage(b, storage, prefix) })
	worker, err := ants.NewRenewPoolManagerWithConfig(ants.DefaultRenewPoolConfig())
	if err != nil {
		b.Fatal(err)
	}
	pool := &benchmarkTaskPool{Pool: worker}
	b.Cleanup(pool.Stop)
	logger := &benchmarkErrorLog{Log: adapter.NewNopLogger()}
	cfg := config.DefaultConfig()
	cfg.KeyPrefix = prefix
	cfg.Timeout = timeout
	cfg.ActiveTimeout = config.NoLimit
	if active {
		cfg.ActiveTimeout = 3600
	}
	cfg.AutoRenew = renew
	cfg.RenewMaxRefresh = 3600
	cfg.RenewInterval = config.NoLimit
	cfg.IsShare, cfg.IsConcurrent = false, true
	cfg.MaxLoginCount = config.NoLimit
	// Disabled logging replaces custom adapters; keep the error observer for maintenance. 关闭日志会替换自定义适配器，维护场景须保留错误观察器。
	cfg.IsPrintBanner, cfg.IsLog, cfg.AsyncEvent = false, active, false
	auth, err := dtoken.NewBuilder().Config(cfg).Storage(storage).SetPool(pool).SetLog(logger).SetAccessProvider(provider).
		UseManagerOption(dtoken.WithComponentOwnership(dtoken.ComponentOwnership{Logger: true})).BuildInstance()
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(auth.Close)
	var token string
	for i := 0; i < terminals; i++ {
		opts := dtoken.LoginOptions{LoginID: "benchmark", Device: "web", DeviceID: fmt.Sprint(i)}
		if i == terminals-1 && extraBytes > 0 {
			opts.Extra = map[string]any{"payload": strings.Repeat("x", extraBytes)}
		}
		token, err = auth.Login(context.Background(), opts)
		if err != nil {
			b.Fatal(err)
		}
	}
	pool.pending.Wait()
	pool.accepted.Store(0)
	renewals := new(atomic.Int64)
	if renew {
		// Count successful renewal, not merely task admission. 统计成功续期，而非仅统计任务接收。
		auth.EventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) {
			renewals.Add(1)
		}, listener.ListenerConfig{Async: false})
	}
	return authBenchmarkFixture{auth: auth, pool: pool, log: logger, token: token, renewals: renewals}
}

// verify rejects degraded runs and reports work accepted per public operation. verify 拒绝降级运行结果，并报告每次公开操作接收的任务数。
func (f authBenchmarkFixture) verify(b *testing.B, wantTasks int64) {
	b.Helper()
	f.pool.pending.Wait()
	if f.log.errors.Load() != 0 || f.pool.rejected.Load() != 0 || f.pool.accepted.Load() != wantTasks {
		b.Fatalf("maintenance errors/rejected/accepted = %d/%d/%d, want 0/0/%d", f.log.errors.Load(), f.pool.rejected.Load(), f.pool.accepted.Load(), wantTasks)
	}
	b.ReportMetric(float64(f.pool.accepted.Load())/float64(b.N), "tasks/op")
}

// BenchmarkAuthValidation measures public API calls through the selected real storage adapter. BenchmarkAuthValidation 度量经过所选真实存储适配器的公开 API 调用。
func BenchmarkAuthValidation(b *testing.B) {
	backend := authBenchmarkBackend(b)
	ctx := context.Background()
	for _, terminals := range []int{1, 100, 1000} {
		for _, extraBytes := range []int{0, 1024, 65536} {
			for _, operation := range []string{"is-login", "check-login", "details", "active-foreground", "active-complete", "renew-active-complete"} {
				b.Run(fmt.Sprintf("%s/terminals=%d/extra=%d/%s", backend, terminals, extraBytes, operation), func(b *testing.B) {
					active := strings.Contains(operation, "active")
					f := newAuthBenchmark(b, backend, terminals, extraBytes, active, operation == "renew-active-complete", nil)
					b.ReportAllocs()
					for b.Loop() {
						switch operation {
						case "is-login":
							if !f.auth.IsLogin(ctx, f.token) {
								b.Fatal("valid credential rejected")
							}
						case "details":
							info, err := f.auth.GetTokenInfo(ctx, f.token)
							if err != nil || info == nil || info.LoginID != "benchmark" {
								b.Fatalf("details = %+v, %v", info, err)
							}
						default:
							if err := f.auth.CheckLogin(ctx, f.token); err != nil {
								b.Fatal(err)
							}
						}
						if active {
							// Drain every iteration so coalescing cannot hide maintenance cost. 每轮排空任务，避免合并任务掩盖维护成本。
							if operation == "active-foreground" {
								b.StopTimer()
							}
							f.pool.pending.Wait()
							if operation == "active-foreground" {
								b.StartTimer()
							}
						}
					}
					wantTasks := int64(0)
					if active {
						wantTasks = int64(b.N)
					}
					f.verify(b, wantTasks)
					if operation == "renew-active-complete" && f.renewals.Load() != int64(b.N) {
						b.Fatalf("successful renewals = %d; want %d", f.renewals.Load(), b.N)
					}
				})
			}
		}
	}
}

// BenchmarkAuthWrites keeps terminal cardinality stable while measuring complete write operations. BenchmarkAuthWrites 保持终端数量稳定，度量完整写操作。
func BenchmarkAuthWrites(b *testing.B) {
	backend := authBenchmarkBackend(b)
	ctx := context.Background()
	for _, terminals := range []int{1, 100, 1000} {
		for _, extraBytes := range []int{0, 1024, 65536} {
			for _, operation := range []string{"login-logout", "manual-renew"} {
				b.Run(fmt.Sprintf("%s/terminals=%d/extra=%d/%s", backend, terminals, extraBytes, operation), func(b *testing.B) {
					f := newAuthBenchmark(b, backend, terminals, extraBytes, false, false, nil)
					opts := dtoken.LoginOptions{LoginID: "benchmark", Device: "web", DeviceID: "transient"}
					if extraBytes > 0 {
						opts.Extra = map[string]any{"payload": strings.Repeat("x", extraBytes)}
					}
					b.ReportAllocs()
					for b.Loop() {
						if operation == "manual-renew" {
							if err := f.auth.RenewTimeout(ctx, f.token, time.Hour); err != nil {
								b.Fatal(err)
							}
						} else {
							token, err := f.auth.Login(ctx, opts)
							if err != nil {
								b.Fatal(err)
							}
							if err = f.auth.LogoutByToken(ctx, token); err != nil {
								b.Fatal(err)
							}
						}
					}
					f.verify(b, 0)
					if tokens, err := f.auth.GetTokenValueListByLoginID(ctx, "benchmark", false); err != nil || len(tokens) != terminals {
						b.Fatalf("terminal count = %d, %v; want %d", len(tokens), err, terminals)
					}
				})
			}
		}
	}
}
