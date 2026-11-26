package gin_core_flow_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// BenchmarkAuthTerminalQueries measures stable terminal indexes with missing credentials. BenchmarkAuthTerminalQueries 度量包含缺失凭据的稳定终端索引查询。
func BenchmarkAuthTerminalQueries(b *testing.B) {
	backend := authBenchmarkBackend(b)
	ctx := context.Background()
	for _, terminals := range []int{10, 100, 1000} {
		for _, missingPercent := range []int{0, 50, 100} {
			for _, operation := range []string{"raw", "alive", "device-alive"} {
				b.Run(fmt.Sprintf("%s/terminals=%d/missing=%d/%s", backend, terminals, missingPercent, operation), func(b *testing.B) {
					f := newAuthBenchmark(b, backend, terminals, 0, false, false, nil)
					tokens, err := f.auth.GetTokenValueListByLoginID(ctx, "benchmark", false)
					if err != nil || len(tokens) != terminals {
						b.Fatalf("initial terminal count = %d, %v; want %d", len(tokens), err, terminals)
					}

					// Model completed expiry without sleeping or invoking logout, which would edit the index. 模拟已过期状态，不等待时钟，也不调用会修改索引的退出操作。
					cfg := f.auth.Config()
					namespace := utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType)
					missing := terminals * missingPercent / 100
					keys := make([]string, missing)
					for i := range missing {
						keys[i] = namespace + config.TokenKeyPrefix + tokens[i]
					}
					if len(keys) > 0 {
						if err := f.auth.Storage().Delete(ctx, keys...); err != nil {
							b.Fatal(err)
						}
					}

					// Select a concrete device without relying on index ordering to identify its token. 选择具体设备，不依赖索引顺序判断其 Token。
					deviceID := fmt.Sprint(terminals - 1)
					want := tokens
					if operation == "alive" {
						want = tokens[missing:]
					} else if operation == "device-alive" {
						want = nil
						if !slices.Contains(tokens[:missing], f.token) {
							want = []string{f.token}
						}
					}
					var got []string
					b.ReportAllocs()
					for b.Loop() {
						if operation == "device-alive" {
							got, err = f.auth.GetTokenValueListByDeviceAndDeviceID(ctx, "benchmark", "web", deviceID, true)
						} else {
							got, err = f.auth.GetTokenValueListByLoginID(ctx, "benchmark", operation == "alive")
						}
						if err != nil || len(got) != len(want) {
							b.Fatalf("query count = %d, %v; want %d", len(got), err, len(want))
						}
					}
					if !slices.Equal(got, want) {
						b.Fatal("query returned unexpected token values or order")
					}
					if after, err := f.auth.GetTokenValueListByLoginID(ctx, "benchmark", false); err != nil || !slices.Equal(after, tokens) {
						b.Fatalf("read-only query changed the stored index: %v", err)
					}
					f.verify(b, 0)
				})
			}
		}
	}
}

// BenchmarkAuthPermissionProvider compares provider hits, Session fallback and authoritative denial. BenchmarkAuthPermissionProvider 比较提供器命中、Session 回退及明确拒绝。
func BenchmarkAuthPermissionProvider(b *testing.B) {
	backend := authBenchmarkBackend(b)
	ctx := context.Background()
	for _, terminals := range []int{1, 100, 1000} {
		for _, mode := range []string{"hit", "fallback", "empty-deny"} {
			b.Run(fmt.Sprintf("%s/terminals=%d/%s", backend, terminals, mode), func(b *testing.B) {
				values := []string{"article:read"}
				var wantErr error
				switch mode {
				case "fallback":
					values = nil
				case "empty-deny":
					values = []string{}
					wantErr = derror.ErrPermissionDenied
				}
				calls := 0
				provider := adapter.AccessProviderFunc{
					PermissionFunc: func(context.Context, adapter.AccessSubject) ([]string, error) {
						calls++
						return values, nil
					},
				}
				f := newAuthBenchmark(b, backend, terminals, 0, false, false, provider)
				// Seed the same Session permission even for denial, so accidental fallback cannot pass. 拒绝场景也写入相同 Session 权限，防止错误回退被当成正确结果。
				if err := f.auth.AddPermissions(ctx, dtoken.PermissionOptions{LoginID: "benchmark", Permissions: []string{"article:read"}}); err != nil {
					b.Fatal(err)
				}
				calls = 0
				b.ReportAllocs()
				for b.Loop() {
					if err := f.auth.CheckPermissionByToken(ctx, f.token, "article:read"); !errors.Is(err, wantErr) {
						b.Fatalf("permission check = %v; want %v", err, wantErr)
					}
				}
				if calls != b.N {
					b.Fatalf("provider calls = %d; want %d", calls, b.N)
				}
				b.ReportMetric(float64(calls)/float64(b.N), "provider-calls/op")
				f.verify(b, 0)
			})
		}
	}
}
