package manager

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerRefreshReverseLifetimeAllowsRevocationAfterRenewal checks the index survives the initial access lifetime. TestManagerRefreshReverseLifetimeAllowsRevocationAfterRenewal 验证撤销索引不会随访问令牌初始有效期结束。
func TestManagerRefreshReverseLifetimeAllowsRevocationAfterRenewal(t *testing.T) {
	ctx := context.Background()
	for _, unlimited := range []bool{false, true} {
		name := "finite"
		if unlimited {
			name = "unlimited"
		}
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 10
				cfg.RefreshTokenTimeout = 3600
				if unlimited {
					cfg.RefreshTokenTimeout = config.NoLimit
				}
			})
			pair, err := mgr.LoginWithRefreshToken(ctx, "reverse-lifetime", "web")
			if err != nil {
				t.Fatal(err)
			}
			ttl, err := mgr.storage.TTL(ctx, mgr.getTokenRefreshKey(pair.AccessToken))
			if err != nil {
				t.Fatal(err)
			}
			if (unlimited && ttl != adapter.TTLNoExpire) || (!unlimited && (ttl <= time.Minute || ttl > time.Hour)) {
				t.Fatalf("reverse TTL = %v, want refresh lifetime", ttl)
			}
			if err := mgr.RenewTimeout(ctx, pair.AccessToken, 20*time.Minute); err != nil {
				t.Fatal(err)
			}
			if err := mgr.Logout(ctx, pair.AccessToken); err != nil {
				t.Fatal(err)
			}
			if _, err := mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
				t.Fatalf("logout retained refresh capability: %v", err)
			}
		})
	}
}

// TestManagerRefreshAfterAccessExpiryCleansTerminals preserves independent refresh credentials while pruning stale terminals. TestManagerRefreshAfterAccessExpiryCleansTerminals 清理陈旧终端时保留独立有效的刷新凭证。
func TestManagerRefreshAfterAccessExpiryCleansTerminals(t *testing.T) {
	ctx := context.Background()
	for _, ordinary := range []bool{false, true} {
		for _, cleanupFirst := range []bool{false, true} {
			name := "atomic/rotate"
			if ordinary {
				name = "ordinary/rotate"
			}
			if cleanupFirst {
				name += "/login-cleanup"
			}
			t.Run(name, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsShare = false
					cfg.MaxLoginCount = config.NoLimit
				})
				if ordinary {
					mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
				}
				id := "expired-refresh-terminals"
				old, err := mgr.LoginWithRefreshToken(ctx, id, "web", "old")
				if err != nil {
					t.Fatal(err)
				}
				other, err := mgr.LoginWithRefreshToken(ctx, id, "web", "other-expired")
				if err != nil {
					t.Fatal(err)
				}
				kept, err := mgr.Login(ctx, id, "mobile", "kept")
				if err != nil {
					t.Fatal(err)
				}
				if err := mgr.SetSessionValue(ctx, id, "theme", "dark"); err != nil {
					t.Fatal(err)
				}
				// Keep the session and reverse indexes alive while access keys expire. 保留 Session 与反向索引，仅模拟访问令牌自然过期。
				if err := mgr.storage.Delete(ctx, mgr.getTokenKey(old.AccessToken), mgr.getTokenKey(other.AccessToken)); err != nil {
					t.Fatal(err)
				}
				before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getRefreshTokenKey(other.RefreshToken), mgr.getTokenRefreshKey(other.AccessToken)})
				if cleanupFirst {
					if _, err := mgr.Login(ctx, id, "desktop"); err != nil {
						t.Fatal(err)
					}
				}
				next, err := mgr.RefreshToken(ctx, old.RefreshToken)
				if err != nil {
					t.Fatal(err)
				}
				assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
				sess, err := mgr.getSession(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				if sess.hasTerminalToken(old.AccessToken) || sess.hasTerminalToken(other.AccessToken) || !sess.hasTerminalToken(next.AccessToken) || !sess.hasTerminalToken(kept) || sess.Data["theme"] != "dark" {
					t.Fatalf("rotation left stale terminals or lost session data: %+v", sess)
				}
				if _, err := mgr.RefreshToken(ctx, old.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
					t.Fatalf("replay = %v", err)
				}
				if _, err := mgr.RefreshToken(ctx, other.RefreshToken); err != nil {
					t.Fatalf("unrelated expired access lost refresh: %v", err)
				}
			})
		}
	}
}

// TestManagerRefreshCleanupChecksReverseOwner protects another account from a misplaced reverse index. TestManagerRefreshCleanupChecksReverseOwner 防止错位的反向索引影响其他账号。
func TestManagerRefreshCleanupChecksReverseOwner(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	first, err := mgr.LoginWithRefreshToken(ctx, "reverse-first", "web")
	if err != nil {
		t.Fatal(err)
	}
	kept, err := mgr.LoginWithRefreshToken(ctx, "reverse-kept", "web")
	if err != nil {
		t.Fatal(err)
	}
	before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getRefreshTokenKey(kept.RefreshToken), mgr.getTokenKey(kept.AccessToken), mgr.getTokenRefreshKey(kept.AccessToken), mgr.getSessionKey(kept.LoginID)})
	if err := mgr.storage.Set(ctx, mgr.getTokenRefreshKey(first.AccessToken), kept.RefreshToken, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Logout(ctx, first.AccessToken); err != nil {
		t.Fatal(err)
	}
	assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
	if mgr.storage.Exists(ctx, mgr.getTokenRefreshKey(first.AccessToken)) {
		t.Fatal("misplaced index retained")
	}
	if _, err := mgr.RefreshToken(ctx, kept.RefreshToken); err != nil {
		t.Fatalf("foreign refresh was revoked: %v", err)
	}
}

// TestManagerRefreshIssuanceRejectsExpiryDuringTTLRead rejects expired pair responses. TestManagerRefreshIssuanceRejectsExpiryDuringTTLRead 拒绝返回签发过程中已过期的令牌对。
func TestManagerRefreshIssuanceRejectsExpiryDuringTTLRead(t *testing.T) {
	ctx := context.Background()
	for _, side := range []string{"access", "refresh"} {
		t.Run(side, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			prefix := mgr.getTokenKey("issuance-ttl-token")
			wantErr := derror.ErrInvalidToken
			if side == "refresh" {
				prefix = mgr.getRefreshTokenKey("")
				wantErr = derror.ErrInvalidRefreshToken
			}
			mgr.storage = &managerExpireOnRefreshTTLStorage{Storage: mgr.storage, prefix: prefix}
			creates := 0
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRefreshTokenCreate, func(*listener.EventData) { creates++ }, listener.ListenerConfig{Async: false})
			pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{LoginID: "issuance-ttl", Token: "issuance-ttl-token"}})
			if pair != nil || !errors.Is(err, wantErr) || creates != 0 {
				t.Fatalf("expired issuance = %+v, %v, creates=%d", pair, err, creates)
			}
			if mgr.storage.Exists(ctx, mgr.getTokenRefreshKey("issuance-ttl-token")) {
				t.Fatal("failed issuance retained reverse binding")
			}
		})
	}
}

// managerExpireOnRefreshTTLStorage models natural expiry immediately before a TTL response. managerExpireOnRefreshTTLStorage 模拟 TTL 查询前的自然过期。
type managerExpireOnRefreshTTLStorage struct {
	adapter.Storage
	prefix string
}

func (s *managerExpireOnRefreshTTLStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if strings.HasPrefix(key, s.prefix) {
		if err := s.Storage.Delete(ctx, key); err != nil {
			return 0, err
		}
	}
	return s.Storage.TTL(ctx, key)
}
