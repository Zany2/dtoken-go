package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// memoryReadFailureStorage preserves Memory capabilities while failing one selected read. memoryReadFailureStorage 保留 Memory 能力，仅让指定键的读取失败。
type memoryReadFailureStorage struct {
	*memory.Storage
	failKey  string
	failures int
}

// Get injects a selected read error without changing stored data. Get 注入指定读取错误，不改变存储数据。
func (s *memoryReadFailureStorage) Get(ctx context.Context, key string) (any, error) {
	if key == s.failKey && s.failKey != "" {
		s.failures++
		return nil, errors.New("injected read failure")
	}
	return s.Storage.Get(ctx, key)
}

// TestMemoryRefreshReadFailuresPreserveCredentials checks failures before consumption leave the original lifecycle intact. TestMemoryRefreshReadFailuresPreserveCredentials 验证消费前读取失败不会改变原凭据生命周期。
func TestMemoryRefreshReadFailuresPreserveCredentials(t *testing.T) {
	for _, mode := range []string{"atomic", "basic"} {
		for _, stage := range []string{"access", "activity", "expired session"} {
			t.Run(mode+"/"+stage, func(t *testing.T) {
				ctx := context.Background()
				mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60; cfg.IsShare = false })
				storage := &memoryReadFailureStorage{Storage: memory.NewStorage()}
				mgr.storage = storage
				if mode == "basic" {
					mgr.storage = nonAtomicManagerStorage{Storage: storage}
				}
				pair, err := mgr.LoginWithRefreshToken(ctx, "refresh-read-failure", "web", "browser")
				if err != nil {
					t.Fatal(err)
				}
				key := mgr.getTokenKey(pair.AccessToken)
				switch stage {
				case "activity":
					key = mgr.getActiveKey(pair.AccessToken)
				case "expired session":
					key = mgr.getSessionKey(pair.LoginID)
					if err = storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), time.Hour); err != nil {
						t.Fatal(err)
					}
				}
				before := captureTerminalLifecycle(t, mgr, ctx, terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken, mgr.getSessionKey(pair.LoginID)))
				events := 0
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(*listener.EventData) { events++ }, listener.ListenerConfig{Async: false})
				storage.failKey = key
				result, err := mgr.RefreshToken(ctx, pair.RefreshToken)
				storage.failKey = ""
				if !errors.Is(err, derror.ErrStorageUnavailable) || result != nil || storage.failures != 1 || events != 0 {
					t.Fatalf("refresh=%+v,%v failures=%d events=%d", result, err, storage.failures, events)
				}
				assertTerminalLifecycleUnchanged(t, mgr, ctx, before)

				// Recovery retries normal credentials, while inactivity still retires the expired lifecycle. 恢复后正常凭据可重试，不活跃超时仍须使原生命周期失效。
				result, err = mgr.RefreshToken(ctx, pair.RefreshToken)
				if stage == "expired session" {
					if result != nil || !errors.Is(err, derror.ErrActiveTimeout) {
						t.Fatalf("inactive retry=%+v,%v", result, err)
					}
					if storage.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) {
						t.Fatal("inactive retry retained refresh credential")
					}
				} else if err != nil || result == nil || !mgr.IsLogin(ctx, result.AccessToken) {
					t.Fatalf("recovered refresh=%+v,%v", result, err)
				}
			})
		}
	}
}

// TestMemoryMalformedRefreshPreservesAccess checks malformed or foreign refresh records cannot revoke access credentials. TestMemoryMalformedRefreshPreservesAccess 验证损坏或跨体系刷新记录不会撤销访问凭据。
func TestMemoryMalformedRefreshPreservesAccess(t *testing.T) {
	for _, operation := range []string{"refresh", "revoke"} {
		for _, kind := range []string{"unsupported value", "invalid json", "foreign namespace"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				ctx := context.Background()
				mgr := newTestManager(t, nil)
				mgr.storage = memory.NewStorage()
				pair, err := mgr.LoginWithRefreshToken(ctx, "corrupt-refresh")
				if err != nil {
					t.Fatal(err)
				}
				var corrupt any = struct{}{}
				want := derror.ErrTypeConvert
				switch kind {
				case "invalid json":
					corrupt = "{"
					want = derror.ErrSerializeFailed
				case "foreign namespace":
					info, err := mgr.getRefreshTokenInfo(ctx, pair.RefreshToken)
					if err != nil {
						t.Fatal(err)
					}
					info.AuthType = "foreign:"
					corrupt, err = mgr.serializer.Encode(info)
					if err != nil {
						t.Fatal(err)
					}
					want = derror.ErrInvalidRefreshToken
				}
				if err = mgr.storage.Set(ctx, mgr.getRefreshTokenKey(pair.RefreshToken), corrupt, time.Hour); err != nil {
					t.Fatal(err)
				}
				before := captureTerminalLifecycle(t, mgr, ctx, terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken, mgr.getSessionKey(pair.LoginID)))
				if operation == "refresh" {
					result, refreshErr := mgr.RefreshToken(ctx, pair.RefreshToken)
					err = refreshErr
					if result != nil {
						t.Fatal("malformed refresh returned a pair")
					}
				} else {
					err = mgr.RevokeRefreshToken(ctx, pair.RefreshToken)
				}
				if operation == "revoke" && kind == "foreign namespace" {
					if err != nil {
						t.Fatalf("invalid revoke should be idempotent: %v", err)
					}
				} else if !errors.Is(err, want) {
					t.Fatalf("error=%v, want %v", err, want)
				}
				assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
				if !mgr.IsLogin(ctx, pair.AccessToken) {
					t.Fatal("malformed refresh affected independent access validity")
				}
			})
		}
	}
}

// TestMemoryPruningReadFailurePreservesSession checks failed login pruning keeps the stored index available for retry. TestMemoryPruningReadFailurePreservesSession 验证登录清理读取失败时保留会话索引以便重试。
func TestMemoryPruningReadFailurePreservesSession(t *testing.T) {
	for _, stage := range []string{"access", "activity", "refresh owner"} {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.ActiveTimeout = 60
				cfg.IsShare = false
				cfg.MaxLoginCount = 1
				cfg.ConcurrencyScope = config.ConcurrencyScopeAccount
			})
			storage := &memoryReadFailureStorage{Storage: memory.NewStorage()}
			mgr.storage = storage
			pair, err := mgr.LoginWithRefreshToken(ctx, "prune-read-failure", "web")
			if err != nil {
				t.Fatal(err)
			}
			key := mgr.getTokenKey(pair.AccessToken)
			switch stage {
			case "activity":
				key = mgr.getActiveKey(pair.AccessToken)
			case "refresh owner":
				key = mgr.getRefreshTokenKey(pair.RefreshToken)
				if err = storage.Set(ctx, mgr.getTokenKey(pair.AccessToken), string(TokenStateKickOut), time.Hour); err != nil {
					t.Fatal(err)
				}
			}
			before := captureTerminalLifecycle(t, mgr, ctx, terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken, mgr.getSessionKey(pair.LoginID)))
			storage.failKey = key
			token, err := mgr.Login(ctx, pair.LoginID, "phone")
			storage.failKey = ""
			if !errors.Is(err, derror.ErrStorageUnavailable) || token != "" || storage.failures != 1 {
				t.Fatalf("login=%q,%v failures=%d", token, err, storage.failures)
			}
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			token, err = mgr.Login(ctx, pair.LoginID, "phone")
			if err != nil || !mgr.IsLogin(ctx, token) {
				t.Fatalf("retry login=%q,%v", token, err)
			}
			if count, err := mgr.GetOnlineTerminalCount(ctx, pair.LoginID); err != nil || count != 1 {
				t.Fatalf("retry count=%d,%v", count, err)
			}
		})
	}
}

type memoryCloseFailureStorage struct {
	*memory.Storage
	closes int
}

// Close records the release attempt and reports a component failure. Close 记录释放尝试并返回组件错误。
func (s *memoryCloseFailureStorage) Close() error {
	s.closes++
	return errors.New("injected close failure")
}

type closeFailureLogger struct {
	managerLifecycleTestLogger
	errors []string
}

// Errorf captures shutdown diagnostics. Errorf 捕获关闭过程的诊断信息。
func (l *closeFailureLogger) Errorf(format string, args ...any) {
	l.errors = append(l.errors, fmt.Sprintf(format, args...))
}

// TestMemoryCloseFailureStillReleasesOwnedResources checks a storage close error cannot skip logger release or reopen admission. TestMemoryCloseFailureStillReleasesOwnedResources 验证存储关闭失败不影响日志器释放和拒绝新任务。
func TestMemoryCloseFailureStillReleasesOwnedResources(t *testing.T) {
	mgr := newTestManager(t, nil)
	storage := &memoryCloseFailureStorage{Storage: memory.NewStorage()}
	logger := &closeFailureLogger{}
	pool := &managerLifecycleTestPool{}
	mgr.storage, mgr.logger, mgr.pool = storage, logger, pool
	mgr.CloseManager()
	mgr.CloseManager()
	if !mgr.IsClosed() || storage.closes != 1 || pool.stopCount() != 1 || logger.flushCount() != 1 || logger.closeCount() != 1 {
		t.Fatalf("release counts storage=%d pool=%d logger=%d/%d", storage.closes, pool.stopCount(), logger.flushCount(), logger.closeCount())
	}
	if len(logger.errors) != 1 || !strings.Contains(logger.errors[0], "injected close failure") {
		t.Fatalf("close diagnostics=%v", logger.errors)
	}
	if _, err := mgr.Login(context.Background(), "closed"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("closed login=%v", err)
	}
	if mgr.submitAsync("closed", func() { t.Error("closed task executed") }, func() { t.Error("closed fallback executed") }) {
		t.Fatal("closed manager accepted task")
	}
}
