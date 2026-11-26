package engine

import (
	"testing"

	"github.com/Zany2/dtoken-go/config"
)

// TestManagerConfigSnapshotKeepsNamespace verifies caller configuration changes cannot redirect storage or alter cookie settings. TestManagerConfigSnapshotKeepsNamespace 验证调用方修改配置不能改变存储命名空间或 Cookie 设置。
func TestManagerConfigSnapshotKeepsNamespace(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetAuthType("user")
	cfg.CookieConfig.Path = "/original"
	mgr := NewManager(cfg, nil, nil, nil, nil, nil, nil)
	t.Cleanup(mgr.CloseManager)
	originalKey := mgr.getTokenKey("token")

	cfg.SetAuthType("admin")
	cfg.KeyPrefix = "other:"
	cfg.CookieConfig.Path = "/input-changed"
	snapshot := mgr.GetConfig()
	snapshot.SetAuthType("changed")
	snapshot.KeyPrefix = "snapshot:"
	snapshot.CookieConfig.Path = "/snapshot-changed"

	got := mgr.GetConfig()
	if got.AuthType != "user:" || got.KeyPrefix != config.DefaultKeyPrefix || got.CookieConfig.Path != "/original" {
		t.Fatalf("configuration escaped snapshot boundary: %+v", got)
	}
	if key := mgr.getTokenKey("token"); key != originalKey {
		t.Fatalf("storage namespace changed: %q, want %q", key, originalKey)
	}
}
