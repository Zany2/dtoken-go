package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerLoginConsolidatesSessionWrites checks pruning, replacement, overflow and sharing at the public login boundary. TestManagerLoginConsolidatesSessionWrites 从公开登录入口检查整理、顶替、超限和共享的写入合并。
func TestManagerLoginConsolidatesSessionWrites(t *testing.T) {
	ctx := context.Background()
	for _, scope := range []config.ConcurrencyScope{config.ConcurrencyScopeAccount, config.ConcurrencyScopeDevice} {
		for _, operation := range []string{"overflow", "replace", "share"} {
			for _, lifetime := range []string{"finite", "unlimited"} {
				t.Run(string(scope)+"/"+operation+"/"+lifetime, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.Timeout = 60
						cfg.IsShare = false
						cfg.MaxLoginCount = config.NoLimit
						cfg.ConcurrencyScope = scope
					})
					id := "session-write-cost"
					var tokens []string
					for _, deviceID := range []string{"expired", "first", "second", "shared"} {
						token, err := mgr.Login(ctx, id, "web", deviceID)
						if err != nil {
							t.Fatal(err)
						}
						tokens = append(tokens, token)
					}
					mobile, err := mgr.Login(ctx, id, "mobile", "other")
					if err != nil {
						t.Fatal(err)
					}
					if err := mgr.SetSessionValue(ctx, id, "theme", "dark"); err != nil {
						t.Fatal(err)
					}
					sess, err := mgr.getSession(ctx, id)
					if err != nil {
						t.Fatal(err)
					}
					ttl := 24 * time.Hour
					if lifetime == "unlimited" {
						ttl = 0
					}
					if err := mgr.saveToStorage(ctx, mgr.getSessionKey(id), *sess, ttl); err != nil {
						t.Fatal(err)
					}
					if err := mgr.storage.Delete(ctx, mgr.getTokenKey(tokens[0])); err != nil {
						t.Fatal(err)
					}
					mgr.config.MaxLoginCount = 2
					mgr.config.IsShare = operation == "share"
					mgr.config.IsConcurrent = operation != "replace"
					mgr.config.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
					storage := &loginSessionWriteStorage{Storage: mgr.storage, sessionKey: mgr.getSessionKey(id)}
					codec := &loginSessionWriteCodec{Codec: mgr.serializer}
					mgr.storage, mgr.serializer = storage, codec
					callbackLocked := false
					mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(*listener.EventData) {
						mgr.loginLocksMu.Lock()
						if entry := mgr.loginLocks[id]; entry != nil {
							if entry.mu.TryLock() {
								entry.mu.Unlock()
							} else {
								callbackLocked = true
							}
						}
						mgr.loginLocksMu.Unlock()
					}, listener.ListenerConfig{Async: false})
					result, err := mgr.Login(ctx, id, "web", "shared")
					if err != nil {
						t.Fatal(err)
					}
					if storage.writes != 1 || storage.deletes != 0 || codec.sessions != 1 || callbackLocked {
						t.Fatalf("session writes/deletes/encodes=%d/%d/%d, callbacks locked=%v", storage.writes, storage.deletes, codec.sessions, callbackLocked)
					}
					stored, err := mgr.getSession(ctx, id)
					if err != nil {
						t.Fatal(err)
					}
					wantCount, wantHistory := 2, int64(6)
					if operation == "replace" {
						wantCount = 1
					} else if operation == "share" {
						wantHistory = 5
						if result != tokens[3] {
							t.Fatal("sharing replaced its original token")
						}
					}
					if scope == config.ConcurrencyScopeDevice {
						wantCount++
						if !sessionHasTerminalToken(stored, mobile) {
							t.Fatal("device policy removed an unrelated terminal")
						}
					}
					if len(stored.TerminalInfos) != wantCount || stored.HistoryTerminalCount != wantHistory || stored.Data["theme"] != "dark" || !sessionHasTerminalToken(stored, result) || sessionHasTerminalToken(stored, tokens[0]) {
						t.Fatalf("final session = %+v, want %d terminals and history %d", stored, wantCount, wantHistory)
					}
					remaining, err := storage.TTL(ctx, storage.sessionKey)
					if err != nil || lifetime == "finite" && remaining < 23*time.Hour || lifetime == "unlimited" && remaining != adapter.TTLNoExpire {
						t.Fatalf("preserved session TTL = %v, %v", remaining, err)
					}
				})
			}
		}
	}
}

// TestManagerSharedLoginLeavesUnchangedSessionUnwritten verifies shared renewal does not serialize unchanged account data. TestManagerSharedLoginLeavesUnchangedSessionUnwritten 验证共享续期不序列化未变化的账号数据。
func TestManagerSharedLoginLeavesUnchangedSessionUnwritten(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) { cfg.IsShare = true })
	token, err := mgr.Login(ctx, "shared-unchanged", "web")
	if err != nil {
		t.Fatal(err)
	}
	storage := &loginSessionWriteStorage{Storage: mgr.storage, sessionKey: mgr.getSessionKey("shared-unchanged")}
	codec := &loginSessionWriteCodec{Codec: mgr.serializer}
	mgr.storage, mgr.serializer = storage, codec
	got, err := mgr.Login(ctx, "shared-unchanged", "web")
	if err != nil || got != token || storage.writes != 0 || storage.deletes != 0 || codec.sessions != 0 {
		t.Fatalf("unchanged sharing = %q, %v; writes/deletes/encodes=%d/%d/%d", got, err, storage.writes, storage.deletes, codec.sessions)
	}
}

// TestManagerLoginResetsExpiredSessionOnce preserves fresh-session identity and TTL without an intermediate delete. TestManagerLoginResetsExpiredSessionOnce 验证过期会话重建保留新身份和 TTL 语义，且无需中间删除。
func TestManagerLoginResetsExpiredSessionOnce(t *testing.T) {
	ctx := context.Background()
	for _, outcome := range []string{"success", "strategy failure"} {
		t.Run(outcome, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 60
				cfg.IsShare = false
				cfg.MaxLoginCount = 1
			})
			old, err := mgr.Login(ctx, "session-reset", "web")
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.SetSessionValue(ctx, "session-reset", "old", true); err != nil {
				t.Fatal(err)
			}
			sess, err := mgr.getSession(ctx, "session-reset")
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.saveToStorage(ctx, mgr.getSessionKey(sess.LoginID), *sess, 0); err != nil {
				t.Fatal(err)
			}
			if err := mgr.storage.Delete(ctx, mgr.getTokenKey(old)); err != nil {
				t.Fatal(err)
			}
			if outcome == "strategy failure" {
				mgr.strategy.CreateSession = func(string, string, int64) *Session { return nil }
			}
			storage := &loginSessionWriteStorage{Storage: mgr.storage, sessionKey: mgr.getSessionKey(sess.LoginID)}
			mgr.storage = storage
			events := registerTerminalLifecycleEvents(mgr)
			token, err := mgr.Login(ctx, sess.LoginID, "web")
			if outcome == "strategy failure" {
				if !errors.Is(err, derror.ErrInvalidParam) || token != "" || storage.writes != 0 || storage.deletes != 1 || storage.Exists(ctx, storage.sessionKey) {
					t.Fatalf("failed reset = %q, %v, writes/deletes=%d/%d", token, err, storage.writes, storage.deletes)
				}
				return
			}
			if err != nil || storage.writes != 1 || storage.deletes != 0 {
				t.Fatalf("reset = %v, writes/deletes=%d/%d", err, storage.writes, storage.deletes)
			}
			stored, err := mgr.getSession(ctx, sess.LoginID)
			if err != nil || stored.HistoryTerminalCount != 1 || stored.Data["old"] != nil {
				t.Fatalf("fresh session = %+v, %v", stored, err)
			}
			ttl, err := storage.TTL(ctx, storage.sessionKey)
			if err != nil || ttl <= 0 || ttl > time.Minute {
				t.Fatalf("fresh session inherited old permanent TTL: %v, %v", ttl, err)
			}
			got := events()
			if len(got) != 3 || got[0].Event != listener.EventDestroySession || got[1].Event != listener.EventCreateSession || got[2].Event != listener.EventLogin {
				t.Fatalf("reset events = %+v", got)
			}
		})
	}
}

// TestManagerLoginFlushesCompletedEvictions covers early errors and token-write rollback after deferred cleanup. TestManagerLoginFlushesCompletedEvictions 覆盖延迟清理后的提前失败及 Token 写入回滚。
func TestManagerLoginFlushesCompletedEvictions(t *testing.T) {
	ctx := context.Background()
	for _, failure := range []string{"second eviction", "new token", "final session"} {
		t.Run(failure, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsShare = false
				cfg.MaxLoginCount = config.NoLimit
				cfg.OverflowLogoutMode = config.LogoutModeKickout
			})
			var tokens []string
			for _, token := range []string{"evict-first", "evict-second", "evict-third"} {
				value, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "evict-failure", Device: "web", Token: token})
				if err != nil {
					t.Fatal(err)
				}
				tokens = append(tokens, value)
			}
			mgr.config.MaxLoginCount = 2
			storage := &loginSessionWriteStorage{Storage: mgr.storage, sessionKey: mgr.getSessionKey("evict-failure"), failKey: mgr.getTokenKey("new-token")}
			if failure == "second eviction" {
				storage.failKey = mgr.getTokenKey(tokens[1])
			} else if failure == "final session" {
				storage.failKey = storage.sessionKey
			}
			mgr.storage = storage
			events := registerTerminalLifecycleEvents(mgr)
			token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "evict-failure", Token: "new-token", Device: "web"})
			if token != "" || !errors.Is(err, derror.ErrStorageUnavailable) {
				t.Fatalf("failed login = %q, %v", token, err)
			}
			sess, err := mgr.getSession(ctx, "evict-failure")
			if err != nil {
				t.Fatal(err)
			}
			wantCount, wantWrites, wantEvents := 1, 2, 2
			if failure == "second eviction" {
				wantCount, wantWrites, wantEvents = 2, 1, 1
			}
			if len(sess.TerminalInfos) != wantCount || storage.writes != wantWrites || storage.deletes != 0 || sessionHasTerminalToken(sess, tokens[0]) || sessionHasTerminalToken(sess, "new-token") || storage.Exists(ctx, mgr.getTokenKey("new-token")) {
				t.Fatalf("failed login session = %+v, writes/deletes=%d/%d", sess, storage.writes, storage.deletes)
			}
			if !sessionHasTerminalToken(sess, tokens[2]) || (failure == "second eviction" && !sessionHasTerminalToken(sess, tokens[1])) {
				t.Fatal("failed or unprocessed terminal disappeared")
			}
			got := events()
			if len(got) != wantEvents {
				t.Fatalf("completed eviction events = %d, want %d", len(got), wantEvents)
			}
			for _, event := range got {
				if event.Event != listener.EventKickout {
					t.Fatalf("failed login emitted %s", event.Event)
				}
			}
		})
	}
}

type loginSessionWriteStorage struct {
	adapter.Storage
	sessionKey, failKey string
	writes, deletes     int
	failed              bool
}

// Set counts session writes and fails a selected write once. Set 统计会话写入，并让选定写入失败一次。
func (s *loginSessionWriteStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if key == s.sessionKey {
		s.writes++
	}
	if key == s.failKey && !s.failed {
		s.failed = true
		return errors.New("write unavailable")
	}
	return s.Storage.Set(ctx, key, value, ttl)
}

// Delete counts session deletion without counting credential cleanup. Delete 统计会话删除，不计凭据清理。
func (s *loginSessionWriteStorage) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if key == s.sessionKey {
			s.deletes++
		}
	}
	return s.Storage.Delete(ctx, keys...)
}

type loginSessionWriteCodec struct {
	adapter.Codec
	sessions int
}

// Encode counts whole-session serialization. Encode 统计完整会话序列化次数。
func (c *loginSessionWriteCodec) Encode(value any) ([]byte, error) {
	switch value.(type) {
	case Session, *Session:
		c.sessions++
	}
	return c.Codec.Encode(value)
}
