package gin_core_flow_test

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// BenchmarkAuthPermanentValidation compares finite and permanent credentials without maintenance. BenchmarkAuthPermanentValidation 比较不启用维护的有限期和永久凭据。
func BenchmarkAuthPermanentValidation(b *testing.B) {
	backend := authBenchmarkBackend(b)
	ctx := context.Background()
	for _, timeout := range []int64{3600, config.NoLimit} {
		b.Run(fmt.Sprintf("%s/timeout=%d", backend, timeout), func(b *testing.B) {
			f := newAuthBenchmarkWithTimeout(b, backend, 1, 0, false, false, nil, timeout)
			b.ReportAllocs()
			for b.Loop() {
				if err := f.auth.CheckLogin(ctx, f.token); err != nil {
					b.Fatal(err)
				}
			}
			cfg := f.auth.Config()
			key := utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType) + config.TokenKeyPrefix + f.token
			ttl, err := f.auth.Storage().TTL(ctx, key)
			if err != nil || (timeout == config.NoLimit && ttl != adapter.TTLNoExpire) || (timeout > 0 && ttl <= 0) {
				b.Fatalf("unexpected TTL = %v, %v for timeout %d", ttl, err, timeout)
			}
			f.verify(b, 0)
		})
	}
}

// BenchmarkAuthParallelValidation measures aggregate throughput with shared or distinct account identities. BenchmarkAuthParallelValidation 度量共享或不同账号身份的并发总吞吐。
func BenchmarkAuthParallelValidation(b *testing.B) {
	backend := authBenchmarkBackend(b)
	ctx := context.Background()
	const credentials = 64
	for _, scope := range []string{"same-account", "many-accounts"} {
		b.Run(backend+"/"+scope, func(b *testing.B) {
			f := newAuthBenchmark(b, backend, 0, 0, false, false, nil)
			tokens := make([]string, credentials)
			for i := range tokens {
				loginID := "benchmark"
				if scope == "many-accounts" {
					loginID = fmt.Sprintf("benchmark-%d", i)
				}
				token, err := f.auth.Login(ctx, dtoken.LoginOptions{LoginID: loginID, Device: "web", DeviceID: fmt.Sprint(i)})
				if err != nil {
					b.Fatal(err)
				}
				tokens[i] = token
			}

			var nextWorker atomic.Uint64
			var failures atomic.Int64
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				index := int(nextWorker.Add(1)-1) % len(tokens)
				var localFailures int64
				for pb.Next() {
					if err := f.auth.CheckLogin(ctx, tokens[index]); err != nil {
						localFailures++
					}
					index = (index + 1) % len(tokens)
				}
				failures.Add(localFailures)
			})
			b.StopTimer()
			if failures.Load() != 0 {
				b.Fatalf("parallel validation failures = %d", failures.Load())
			}
			f.verify(b, 0)
		})
	}
}

// BenchmarkAuthParallelWrites measures complete login/logout cycles under account contention. BenchmarkAuthParallelWrites 度量账号竞争下的完整登录退出循环。
func BenchmarkAuthParallelWrites(b *testing.B) {
	backend := authBenchmarkBackend(b)
	ctx := context.Background()
	for _, scope := range []string{"same-account", "many-accounts"} {
		b.Run(backend+"/"+scope, func(b *testing.B) {
			f := newAuthBenchmark(b, backend, 0, 0, false, false, nil)
			var workers atomic.Uint64
			var failures atomic.Int64
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				id := workers.Add(1)
				loginID := "benchmark"
				if scope == "many-accounts" {
					loginID = fmt.Sprintf("benchmark-%d", id)
				}
				opts := dtoken.LoginOptions{LoginID: loginID, Device: "web", DeviceID: fmt.Sprint(id)}
				var localFailures int64
				for pb.Next() {
					token, err := f.auth.Login(ctx, opts)
					if err != nil {
						localFailures++
						continue
					}
					if err := f.auth.LogoutByToken(ctx, token); err != nil {
						localFailures++
					}
				}
				failures.Add(localFailures)
			})
			b.StopTimer()
			if failures.Load() != 0 {
				b.Fatalf("parallel login/logout failures = %d", failures.Load())
			}
			// No accepted cycle may leave a terminal behind. 已完成的登录退出循环不能遗留终端。
			for id := uint64(1); id <= workers.Load(); id++ {
				loginID := "benchmark"
				if scope == "many-accounts" {
					loginID = fmt.Sprintf("benchmark-%d", id)
				}
				if tokens, err := f.auth.GetTokenValueListByLoginID(ctx, loginID, false); err != nil || len(tokens) != 0 {
					b.Fatalf("remaining terminals for %s = %d, %v", loginID, len(tokens), err)
				}
				if scope == "same-account" {
					break
				}
			}
			f.verify(b, 0)
		})
	}
}

// BenchmarkAuthLoginPruning restores the same stale index before each measured login. BenchmarkAuthLoginPruning 在每次计时登录前恢复相同的残留索引。
func BenchmarkAuthLoginPruning(b *testing.B) {
	backend := authBenchmarkBackend(b)
	ctx := context.Background()
	for _, terminals := range []int{10, 100, 1000} {
		for _, missingPercent := range []int{50, 100} {
			b.Run(fmt.Sprintf("%s/terminals=%d/missing=%d", backend, terminals, missingPercent), func(b *testing.B) {
				f := newAuthBenchmark(b, backend, terminals, 0, false, false, nil)
				storage := f.auth.Storage()
				cfg := f.auth.Config()
				namespace := utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType)

				// This fixture deliberately restores serialized storage state, not application Session data. 此数据集刻意恢复序列化存储状态，而非修改应用 Session 数据。
				sessionKey := namespace + "session:benchmark"
				snapshot, err := storage.Get(ctx, sessionKey)
				if err != nil || snapshot == nil {
					b.Fatalf("load Session snapshot: %v", err)
				}
				tokens, err := f.auth.GetTokenValueListByLoginID(ctx, "benchmark", false)
				if err != nil || len(tokens) != terminals {
					b.Fatalf("initial terminal count = %d, %v; want %d", len(tokens), err, terminals)
				}
				missing := terminals * missingPercent / 100
				keys := make([]string, missing)
				for i := range keys {
					keys[i] = namespace + config.TokenKeyPrefix + tokens[i]
				}
				if err := storage.Delete(ctx, keys...); err != nil {
					b.Fatal(err)
				}
				limit := int64(terminals)
				opts := dtoken.LoginOptions{LoginID: "benchmark", Device: "web", DeviceID: "transient", MaxLoginCount: &limit}
				want := append(slices.Clone(tokens[missing:]), "")
				b.ReportAllocs()
				for b.Loop() {
					// Restore outside timing so every measured login must reconsider all stale slots. 计时外恢复数据，让每次计时登录都必须重新检查残留名额。
					b.StopTimer()
					if err := storage.Set(ctx, sessionKey, snapshot, time.Hour); err != nil {
						b.Fatal(err)
					}
					b.StartTimer()
					token, err := f.auth.Login(ctx, opts)
					b.StopTimer()
					if err != nil {
						b.Fatal(err)
					}
					want[len(want)-1] = token
					if got, err := f.auth.GetTokenValueListByLoginID(ctx, "benchmark", false); err != nil || !slices.Equal(got, want) {
						b.Fatalf("login did not persist the expected pruned index: %v", err)
					}
					if err := f.auth.CheckLogin(ctx, token); err != nil {
						b.Fatal(err)
					}
					if err := f.auth.LogoutByToken(ctx, token); err != nil {
						b.Fatal(err)
					}
					b.StartTimer()
				}
				for _, token := range tokens[missing:] {
					if err := f.auth.CheckLogin(ctx, token); err != nil {
						b.Fatalf("pruning retired a live credential: %v", err)
					}
				}
				f.verify(b, 0)
			})
		}
	}
}
