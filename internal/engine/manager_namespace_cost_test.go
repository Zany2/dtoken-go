package engine

import (
	"testing"

	"github.com/Zany2/dtoken-go/config"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// TestManagerNamespaceConstruction verifies stable key boundaries and configuration snapshots. TestManagerNamespaceConstruction 验证固定键边界与配置快照。
func TestManagerNamespaceConstruction(t *testing.T) {
	for _, input := range []struct{ prefix, authType, want string }{
		{"dtoken:", "user:", "dtoken:user:v2:"},
		{"app:", "admin:user:", `app:admin\:user:v2:`},
		{"app:admin:", "user:", `app\:admin:user:v2:`},
		{`tenant\blue:`, `admin\ops:`, `tenant\\blue:admin\\ops:v2:`},
		{"应用:", "用户:", "应用:用户:v2:"},
	} {
		t.Run(input.want, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.KeyPrefix, cfg.AuthType = input.prefix, input.authType
			mgr := NewManager(cfg, nil, nil, nil, nil, nil, nil)
			t.Cleanup(mgr.CloseManager)
			cfg.KeyPrefix, cfg.AuthType = "changed:", "changed:"
			snapshot := mgr.GetConfig()
			snapshot.KeyPrefix, snapshot.AuthType = "snapshot:", "snapshot:"
			if got := mgr.storageNamespace(); got != input.want {
				t.Fatalf("namespace = %q, want %q", got, input.want)
			}
			for _, key := range []struct{ got, suffix string }{
				{mgr.getTokenKey("credential"), config.TokenKeyPrefix + "credential"},
				{mgr.getTokenExtraKey("life"), "token-extra:life"},
				{mgr.getSessionKey("user"), SessionKeyPrefix + "user"},
				{mgr.getRenewKey("credential"), RenewKeyPrefix + "credential"},
				{mgr.getActiveKey("credential"), ActivePrefix + "credential"},
				{mgr.getRefreshTokenKey("refresh"), RefreshTokenKeyPrefix + "refresh"},
				{mgr.getTokenRefreshKey("credential"), TokenRefreshKeyPrefix + "credential"},
				{mgr.getDisableKey("user:a"), DisableKeyPrefix + `user\:a`},
				{mgr.getDisableServiceKey("user:a", "svc:b"), DisableServiceKeyPrefix + `user\:a:svc\:b`},
				{mgr.getDisableDeviceKey("user:a", "web:b"), DisableDeviceKeyPrefix + `user\:a:web\:b`},
				{mgr.getDisableDeviceAndDeviceIDKey("user:a", "web:b", `id\c`), DisableDeviceIDKeyPrefix + `user\:a:web\:b:id\\c`},
			} {
				if key.got != input.want+key.suffix {
					t.Fatalf("key = %q, want %q", key.got, input.want+key.suffix)
				}
			}
		})
	}
}

var namespaceKeyBenchmarkResult string // namespaceKeyBenchmarkResult keeps generated keys observable. namespaceKeyBenchmarkResult 保留生成键，避免基准被消除。

// BenchmarkManagerTokenKey compares cached construction with the previous per-key namespace encoding. BenchmarkManagerTokenKey 对比固定命名空间与原先逐键编码的成本。
func BenchmarkManagerTokenKey(b *testing.B) {
	for _, prefix := range []string{"dtoken:", `tenant\region:blue:`} {
		cfg := config.DefaultConfig()
		cfg.KeyPrefix, cfg.AuthType = prefix, "user:"
		mgr := NewManager(cfg, nil, nil, nil, nil, nil, nil)
		b.Cleanup(mgr.CloseManager)
		b.Run(prefix+"/cached", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				namespaceKeyBenchmarkResult = mgr.getTokenKey("credential")
			}
		})
		b.Run(prefix+"/encode-each-key", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				namespaceKeyBenchmarkResult = utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType) + config.TokenKeyPrefix + "credential"
			}
		})
	}
}
