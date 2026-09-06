package msgpack

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestMsgPackManagerRefreshRecords verifies actual manager records survive codec round trips and legacy decoding. TestMsgPackManagerRefreshRecords 验证实际 Manager 记录经过编解码及旧格式读取后仍可完成刷新。
func TestMsgPackManagerRefreshRecords(t *testing.T) {
	ctx := context.Background()
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%v", legacy), func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.IsLog, cfg.IsPrintBanner, cfg.AsyncEvent, cfg.AutoRenew = false, false, false, false
			cfg.Timeout, cfg.RefreshTokenTimeout = 60, 120
			cfg.ActiveTimeout, cfg.RenewInterval = config.NoLimit, config.NoLimit
			storage := &codecManagerStorage{items: make(map[string]codecManagerEntry)}
			codec := NewMsgPackSerializer()
			mgr := manager.NewManager(cfg, &codecManagerGenerator{}, storage, codec, adapter.NewNopLogger(), nil, nil)
			t.Cleanup(mgr.CloseManager)
			tokenExtra := map[string]any{"kind": "token"}
			terminalExtra := map[string]any{"kind": "terminal"}
			pair, err := mgr.LoginWithRefreshTokenOptions(ctx, manager.RefreshTokenOptions{LoginOptions: manager.LoginOptions{
				LoginID: "msgpack-manager", Device: "web", DeviceID: "browser", Extra: tokenExtra, TerminalExtra: terminalExtra,
			}})
			if err != nil {
				t.Fatalf("login with MsgPack error = %v", err)
			}
			if err = mgr.RenewTimeout(ctx, pair.AccessToken, 2*time.Minute); err != nil {
				t.Fatalf("manual renewal error = %v", err)
			}
			shared, err := mgr.Login(ctx, pair.LoginID, "web", "browser")
			if err != nil || shared != pair.AccessToken {
				t.Fatalf("shared login = %q, %v, want %q", shared, err, pair.AccessToken)
			}
			wantTerminalExtra := terminalExtra
			if legacy {
				// Re-encode real records through the unchanged public types to simulate old storage payloads. 通过未改变的公开类型重编码真实记录，模拟旧存储载荷。
				accessConverted, refreshConverted := false, false
				for key, entry := range storage.items {
					raw, ok := entry.value.([]byte)
					if !ok {
						continue
					}
					var refresh manager.RefreshTokenInfo
					if codec.Decode(raw, &refresh) == nil && refresh.AccessToken == pair.AccessToken {
						entry.value, err = codec.Encode(refresh)
						refreshConverted = true
					} else {
						var access manager.TokenInfo
						if codec.Decode(raw, &access) != nil || access.LoginID != pair.LoginID || access.Timeout <= 0 {
							continue
						}
						entry.value, err = codec.Encode(access)
						accessConverted = true
					}
					if err != nil {
						t.Fatalf("legacy encode error = %v", err)
					}
					storage.items[key] = entry
				}
				if !accessConverted || !refreshConverted {
					t.Fatal("current records did not decode through the public legacy types")
				}
				wantTerminalExtra = nil
			}
			next, err := mgr.RefreshToken(ctx, pair.RefreshToken)
			if err != nil {
				t.Fatalf("refresh with MsgPack error = %v", err)
			}
			info, err := mgr.GetTokenInfo(ctx, next.AccessToken)
			if err != nil || !reflect.DeepEqual(info.Extra, tokenExtra) {
				t.Fatalf("rotated token metadata = %+v, %v", info, err)
			}
			terminal, err := mgr.GetTerminalInfoByToken(ctx, next.AccessToken)
			if err != nil || !reflect.DeepEqual(terminal.Extra, wantTerminalExtra) {
				t.Fatalf("rotated terminal metadata = %+v, %v, want %+v", terminal, err, wantTerminalExtra)
			}
			if err = mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, derror.ErrInvalidToken) {
				t.Fatalf("old access state = %v, want ErrInvalidToken", err)
			}
			if err = mgr.RevokeRefreshToken(ctx, next.RefreshToken); err != nil {
				t.Fatalf("revoke with MsgPack error = %v", err)
			}
			if err = mgr.CheckLogin(ctx, next.AccessToken); !errors.Is(err, derror.ErrInvalidToken) {
				t.Fatalf("revoked access state = %v, want ErrInvalidToken", err)
			}
		})
	}
}

// codecManagerGenerator provides distinct tokens for the synchronous compatibility fixture. codecManagerGenerator 为同步兼容性用例提供不同的 Token。
type codecManagerGenerator struct{ sequence int }

// Generate returns the next fixture token. Generate 返回下一个用例 Token。
func (g *codecManagerGenerator) Generate(string, string, string) (string, error) {
	g.sequence++
	return fmt.Sprintf("codec-token-%d", g.sequence), nil
}

// codecManagerEntry stores a value and its optional deadline. codecManagerEntry 保存值及可选截止时间。
type codecManagerEntry struct {
	value   any
	expires time.Time
}

// codecManagerStorage is a synchronous basic-storage fixture requiring no extra module dependencies. codecManagerStorage 是无需额外模块依赖的同步基础存储用例。
type codecManagerStorage struct{ items map[string]codecManagerEntry }

// Set stores a fixture value with optional expiration. Set 保存用例值及可选过期时间。
func (s *codecManagerStorage) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	entry := codecManagerEntry{value: value}
	if ttl > 0 {
		entry.expires = time.Now().Add(ttl)
	}
	s.items[key] = entry
	return nil
}

// Get discards expired fixture entries before returning a value. Get 返回值前移除过期用例条目。
func (s *codecManagerStorage) Get(_ context.Context, key string) (any, error) {
	entry, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	if !entry.expires.IsZero() && !time.Now().Before(entry.expires) {
		delete(s.items, key)
		return nil, nil
	}
	return entry.value, nil
}

// Exists reports whether a non-expired fixture entry remains. Exists 判断未过期用例条目是否存在。
func (s *codecManagerStorage) Exists(ctx context.Context, key string) bool {
	value, _ := s.Get(ctx, key)
	return value != nil
}

// Delete removes the selected fixture entries. Delete 移除指定用例条目。
func (s *codecManagerStorage) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(s.items, key)
	}
	return nil
}

// Expire follows the storage contract for missing keys and non-positive lifetimes. Expire 遵守缺失键及非正有效期的存储约定。
func (s *codecManagerStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	value, _ := s.Get(ctx, key)
	if value == nil {
		return errors.New("missing key")
	}
	if ttl <= 0 {
		return s.Delete(ctx, key)
	}
	return s.Set(ctx, key, value, ttl)
}

// TTL reports the remaining lifetime or the standard storage sentinel. TTL 返回剩余有效期或标准存储哨兵值。
func (s *codecManagerStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if !s.Exists(ctx, key) {
		return adapter.TTLNotFound, nil
	}
	deadline := s.items[key].expires
	if deadline.IsZero() {
		return adapter.TTLNoExpire, nil
	}
	return time.Until(deadline), nil
}

// Ping reports the in-memory fixture as available. Ping 表示内存用例可用。
func (*codecManagerStorage) Ping(context.Context) error { return nil }

var _ adapter.Storage = (*codecManagerStorage)(nil)
