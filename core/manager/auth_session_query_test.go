package manager

import (
	"context"
	"errors"
	"path"
	"reflect"
	"sync"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
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

// managerGlobScannerStorage applies glob matching to slash-free search fixtures. managerGlobScannerStorage 对不含斜杠的搜索用例键应用 glob 匹配。
type managerGlobScannerStorage struct {
	adapter.Storage
	keys []string // keys stores deliberately unordered fixture keys. keys 存储刻意打乱顺序的用例键。
}

// Keys matches fixtures with the standard library, independently of manager escaping. Keys 使用标准库匹配用例键，独立于 Manager 的转义实现。
func (s *managerGlobScannerStorage) Keys(_ context.Context, pattern string) ([]string, error) {
	keys := make([]string, 0, len(s.keys))
	for _, key := range s.keys {
		matched, err := path.Match(pattern, key)
		if err != nil {
			return nil, err
		}
		if matched {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// TestManagerSearchTreatsNamespacesAndKeywordsLiterally verifies scan metacharacters cannot widen searches or corrupt pagination. TestManagerSearchTreatsNamespacesAndKeywordsLiterally 验证扫描特殊字符不会扩大搜索范围或扰乱分页。
func TestManagerSearchTreatsNamespacesAndKeywordsLiterally(t *testing.T) {
	ctx := context.Background()
	literals := []struct {
		name    string
		literal string
		other   string
	}{
		{name: "plain", literal: "plain", other: "other"},
		{name: "star", literal: "a*b", other: "axb"},
		{name: "question mark", literal: "a?b", other: "axb"},
		{name: "backslash", literal: `a\b`, other: "ab"},
		{name: "character class", literal: "a[bc]", other: "ab"},
		{name: "open bracket", literal: "a[b", other: "ab"},
		{name: "close bracket", literal: "a]b", other: "ab"},
	}
	for _, field := range []string{"key prefix", "auth type", "keyword"} {
		for _, tt := range literals {
			t.Run(field+"/"+tt.name, func(t *testing.T) {
				cfg := config.DefaultConfig()
				keyword := "item"
				otherKeyword := keyword
				otherNamespace := ""
				switch field {
				case "key prefix":
					cfg.KeyPrefix = "search-" + tt.literal + ":"
					otherNamespace = "search-" + tt.other + ":" + cfg.AuthType
				case "auth type":
					cfg.AuthType = "search-" + tt.literal + ":"
					otherNamespace = cfg.KeyPrefix + "search-" + tt.other + ":"
				case "keyword":
					keyword = tt.literal
					otherKeyword = tt.other
					otherNamespace = cfg.KeyPrefix + cfg.AuthType
				}
				if err := cfg.Validate(); err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				storage := &managerGlobScannerStorage{Storage: newManagerTestStorage()}
				mgr := &Manager{config: cfg, storage: storage}
				queries := []struct {
					name   string
					prefix string
					search func(context.Context, string, int, int) ([]string, error)
				}{
					{name: "tokens", prefix: config.TokenKeyPrefix, search: mgr.SearchTokenValue},
					{name: "sessions", prefix: SessionKeyPrefix, search: mgr.SearchSessionId},
				}
				for _, query := range queries {
					t.Run(query.name, func(t *testing.T) {
						prefix := cfg.KeyPrefix + cfg.AuthType + query.prefix
						want := []string{"value-" + keyword + "-a", "value-" + keyword + "-z"}
						storage.keys = []string{
							prefix + want[1],
							otherNamespace + query.prefix + "value-" + otherKeyword + "-0",
							prefix + want[0],
						}
						got, err := query.search(ctx, keyword, 0, -1)
						if err != nil || !reflect.DeepEqual(got, want) {
							t.Fatalf("search(all) = %v, %v, want %v", got, err, want)
						}
						got, err = query.search(ctx, keyword, 1, 1)
						if err != nil || !reflect.DeepEqual(got, want[1:]) {
							t.Fatalf("search(page) = %v, %v, want %v", got, err, want[1:])
						}
					})
				}
			})
		}
	}
}

// TestManagerOptionalDeviceQueriesValidateBeforeStorage verifies invalid filters do not depend on session presence or storage availability. TestManagerOptionalDeviceQueriesValidateBeforeStorage 验证非法过滤条件不依赖会话存在与否或存储可用性。
func TestManagerOptionalDeviceQueriesValidateBeforeStorage(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"existing session", "missing session", "storage error"} {
		t.Run(state, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			const loginID = "optional-device-query"
			if state == "existing session" {
				if _, err := mgr.Login(ctx, loginID, "web"); err != nil {
					t.Fatalf("Login() error = %v", err)
				}
			}
			if state == "storage error" {
				mgr.storage = &managerFailingStorage{Storage: mgr.storage, getErr: errors.New("storage unavailable")}
			}
			storage := &managerGetCountingStorage{Storage: mgr.storage}
			mgr.storage = storage

			for _, device := range []string{"", " \t\n "} {
				if _, err := mgr.GetTerminalListByLoginID(ctx, loginID, device); !errors.Is(err, derror.ErrInvalidParam) {
					t.Fatalf("GetTerminalListByLoginID(%q) error = %v, want ErrInvalidParam", device, err)
				}
				if _, err := mgr.GetTokenValueByLoginID(ctx, loginID, device); !errors.Is(err, derror.ErrInvalidParam) {
					t.Fatalf("GetTokenValueByLoginID(%q) error = %v, want ErrInvalidParam", device, err)
				}
			}
			if got := storage.getCount(mgr.getSessionKey(loginID)); got != 0 {
				t.Fatalf("session reads for invalid device = %d, want 0", got)
			}
		})
	}
}
