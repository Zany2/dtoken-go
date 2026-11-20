package dtoken

import (
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// TestBuilderPoolStatusLifecycle verifies builder-driven reporting and shutdown for owned and borrowed pools. TestBuilderPoolStatusLifecycle 验证构建器启动状态报告及自有、借用任务池的关闭行为。
func TestBuilderPoolStatusLifecycle(t *testing.T) {
	for _, owned := range []bool{true, false} {
		name := "caller-owned pool"
		if owned {
			name = "factory-created pool"
		}
		t.Run(name, func(t *testing.T) {
			// Keep the cache janitor outside the virtual-time bubble; this test only observes reporting tasks. 将缓存清理协程留在虚拟时间区域之外，本测试只观察报告任务。
			storage := memory.NewStorage()
			synctest.Test(t, func(t *testing.T) {
				logger := &builderStatusLogger{NopLogger: contract.NewNopLogger()}
				pool := &builderStatusPool{}
				b := NewBuilder().Storage(storage).IsPrintBanner(false).IsLog(true).
					SetLog(logger).RenewPoolPrintStatusInterval(time.Second)
				if owned {
					b.SetPoolFactory(func(*Config) (contract.Pool, error) { return pool, nil })
				} else {
					b.AutoRenew(false).SetPool(pool).
						UseManagerOption(WithComponentOwnership(ComponentOwnership{
							Storage: true, Logger: true, Pool: false,
						}))
				}
				auth, err := b.Build()
				if err != nil {
					t.Fatal(err)
				}
				defer auth.Close()

				// Let the reporting task create its ticker before advancing virtual time. 推进虚拟时间前等待报告任务创建定时器。
				synctest.Wait()
				time.Sleep(3 * time.Second)
				synctest.Wait()
				if logger.reports.Load() == 0 || pool.reads.Load() == 0 {
					t.Fatal("builder did not start pool status reporting")
				}

				// Reporting stops with Auth even when the application retains the pool. 即使任务池由应用持有，状态报告也随 Auth 关闭而停止。
				auth.Close()
				auth.Close()
				reports, reads := logger.reports.Load(), pool.reads.Load()
				time.Sleep(3 * time.Second)
				synctest.Wait()
				if logger.reports.Load() != reports || pool.reads.Load() != reads {
					t.Fatal("pool status reporting continued after Auth.Close")
				}
				wantStops := int64(0)
				if owned {
					wantStops = 1
				}
				if got := pool.stops.Load(); got != wantStops {
					t.Fatalf("pool stops = %d, want %d", got, wantStops)
				}
			})
		})
	}
}

// TestBuilderPoolStatusDisabled verifies disabled or failed builds never poll the pool. TestBuilderPoolStatusDisabled 验证禁用状态报告或构建失败时不会轮询任务池。
func TestBuilderPoolStatusDisabled(t *testing.T) {
	for _, name := range []string{"zero interval", "logging disabled", "missing pool", "build failure"} {
		t.Run(name, func(t *testing.T) {
			// Keep unrelated cache maintenance outside the virtual-time bubble. 将无关的缓存维护任务留在虚拟时间区域之外。
			storage := memory.NewStorage()
			synctest.Test(t, func(t *testing.T) {
				logger := &builderStatusLogger{NopLogger: contract.NewNopLogger()}
				pool := &builderStatusPool{}
				b := NewBuilder().Storage(storage).IsPrintBanner(false).IsLog(true).AutoRenew(false).
					SetLog(logger).SetPool(pool).RenewPoolPrintStatusInterval(time.Second)
				switch name {
				case "zero interval":
					b.RenewPoolPrintStatusInterval(0)
				case "logging disabled":
					b.IsLog(false)
				case "missing pool":
					b.SetPool(nil)
				case "build failure":
					b.TokenName("")
				}
				auth, err := b.Build()
				if auth != nil {
					defer auth.Close()
				}
				if name == "build failure" {
					if err == nil || auth != nil {
						t.Fatalf("Build() = (%v, %v), want nil Auth and error", auth, err)
					}
				} else if err != nil {
					t.Fatal(err)
				}

				synctest.Wait()
				time.Sleep(3 * time.Second)
				synctest.Wait()
				if logger.reports.Load() != 0 || pool.reads.Load() != 0 {
					t.Fatal("pool status reporting started without its prerequisites")
				}
			})
		})
	}
}

// builderStatusLogger records status reports without file or console output. builderStatusLogger 记录状态报告，不输出到文件或控制台。
type builderStatusLogger struct {
	*contract.NopLogger
	reports atomic.Int64 // reports counts pool status messages. reports 统计池状态消息数。
}

// Infof records pool status messages. Infof 记录池状态消息。
func (l *builderStatusLogger) Infof(format string, _ ...any) {
	if strings.Contains(format, "renew pool status") {
		l.reports.Add(1)
	}
}

// builderStatusPool observes statistics reads and resource ownership. builderStatusPool 观察统计读取与资源所有权。
type builderStatusPool struct {
	reads atomic.Int64 // reads counts statistics queries. reads 统计状态查询次数。
	stops atomic.Int64 // stops counts shutdown calls. stops 统计关闭调用次数。
}

// Submit runs the supplied task. Submit 执行传入的任务。
func (p *builderStatusPool) Submit(task func()) error {
	task()
	return nil
}

// Stop records pool shutdown. Stop 记录任务池关闭。
func (p *builderStatusPool) Stop() { p.stops.Add(1) }

// Stats reports a fixed capacity and records the query. Stats 返回固定容量并记录查询。
func (p *builderStatusPool) Stats() (int, int, float64) {
	p.reads.Add(1)
	return 1, 2, 0.5
}
