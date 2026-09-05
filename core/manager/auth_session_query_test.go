package manager

import (
	"context"
	"sync"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
)

// managerGetCountingStorage counts reads while forwarding storage operations. managerGetCountingStorage 统计读取次数并转发存储操作。
type managerGetCountingStorage struct {
	adapter.Storage
	mu   sync.Mutex     // mu protects read counters. mu 保护读取计数。
	gets map[string]int // gets stores read counts by key. gets 按键存储读取次数。
}

// Get records one read and delegates to the wrapped storage. Get 记录一次读取并转发到被包装存储。
func (s *managerGetCountingStorage) Get(ctx context.Context, key string) (any, error) {
	s.mu.Lock()
	if s.gets == nil {
		s.gets = make(map[string]int)
	}
	s.gets[key]++
	s.mu.Unlock()
	return s.Storage.Get(ctx, key)
}

// getCount returns the read count for one key. getCount 返回指定键的读取次数。
func (s *managerGetCountingStorage) getCount(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gets[key]
}

// TestManagerAggregateSessionQueryCachesDisableChecks verifies repeated terminals reuse disable-state reads. TestManagerAggregateSessionQueryCachesDisableChecks 验证聚合查询中的重复终端会复用封禁状态读取。
func TestManagerAggregateSessionQueryCachesDisableChecks(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
		cfg.AutoRenew = false
		cfg.ActiveTimeout = config.NoLimit
	})

	deviceIDs := []string{"browser-a", "browser-b", "browser-c"}
	for _, deviceID := range deviceIDs {
		if _, err := mgr.Login(ctx, "aggregate-query-cache", "web", deviceID); err != nil {
			t.Fatalf("Login() error = %v", err)
		}
	}

	storage := &managerGetCountingStorage{Storage: mgr.storage}
	mgr.storage = storage
	count, err := mgr.GetOnlineTerminalCount(ctx, "aggregate-query-cache")
	if err != nil {
		t.Fatalf("GetOnlineTerminalCount() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("GetOnlineTerminalCount() = %d, want 3", count)
	}

	keys := []string{
		mgr.getDisableKey("aggregate-query-cache"),
		mgr.getDisableDeviceKey("aggregate-query-cache", "web"),
	}
	for _, deviceID := range deviceIDs {
		keys = append(keys, mgr.getDisableDeviceAndDeviceIDKey("aggregate-query-cache", "web", deviceID))
	}
	for _, key := range keys {
		if got := storage.getCount(key); got != 1 {
			t.Fatalf("storage.Get(%q) count = %d, want 1", key, got)
		}
	}
}
