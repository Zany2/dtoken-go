package manager

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestManagerValidationRejectsInvalidAccountSession checks identity APIs against the same session failures. TestManagerValidationRejectsInvalidAccountSession 验证各身份接口一致拒绝无效账号 Session。
func TestManagerValidationRejectsInvalidAccountSession(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"missing", "foreign namespace", "malformed", "read failure"} {
		t.Run(state, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 30 })
			token, err := mgr.Login(ctx, "session-validation", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			key := mgr.getSessionKey("session-validation")
			wantErr := derror.ErrInvalidToken
			switch state {
			case "missing":
				err = mgr.storage.Delete(ctx, key)
			case "foreign namespace":
				err = mgr.saveToStorage(ctx, key, Session{AuthType: "foreign:", LoginID: "session-validation"})
			case "malformed":
				err = mgr.storage.Set(ctx, key, []byte("not-json"), time.Minute)
				wantErr = derror.ErrSerializeFailed
			case "read failure":
				mgr.storage = &managerSessionReadErrorStorage{Storage: mgr.storage, key: key}
				wantErr = derror.ErrStorageUnavailable
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.CheckLogin(ctx, token); !errors.Is(err, wantErr) {
				t.Fatalf("CheckLogin = %v, want %v", err, wantErr)
			}
			if mgr.IsLogin(ctx, token) {
				t.Fatal("IsLogin accepted invalid session")
			}
			for _, query := range []struct {
				name string
				call func(context.Context, string) (string, error)
			}{
				{"login ID", mgr.GetLoginID}, {"device", mgr.GetDevice}, {"device ID", mgr.GetDeviceID},
			} {
				if value, err := query.call(ctx, token); value != "" || !errors.Is(err, wantErr) {
					t.Fatalf("%s = %q, %v, want empty and %v", query.name, value, err, wantErr)
				}
			}
			if device, id, err := mgr.GetDeviceAndDeviceID(ctx, token); device != "" || id != "" || !errors.Is(err, wantErr) {
				t.Fatalf("device identity = %q/%q, %v", device, id, err)
			}
			if created, err := mgr.GetTokenCreateTime(ctx, token); created != 0 || !errors.Is(err, wantErr) {
				t.Fatalf("creation time = %d, %v", created, err)
			}
			if err := mgr.LoginByToken(ctx, token); !errors.Is(err, wantErr) {
				t.Fatalf("LoginByToken = %v, want %v", err, wantErr)
			}
			result, err := mgr.IntrospectToken(ctx, token)
			if errors.Is(wantErr, derror.ErrInvalidToken) {
				if err != nil || result == nil || result.Active || result.LoginID != "" {
					t.Fatalf("introspection = %+v, %v, want inactive", result, err)
				}
			} else if result != nil || !errors.Is(err, wantErr) {
				t.Fatalf("introspection = %+v, %v, want infrastructure error %v", result, err, wantErr)
			}
			if pool.taskCount() != 0 {
				t.Fatal("invalid session scheduled token maintenance")
			}

			// Metadata reads intentionally remain available for diagnostics. 元数据读取仍用于诊断，不代表登录态有效。
			if info, err := mgr.GetTokenInfo(ctx, token); err != nil || info.LoginID != "session-validation" {
				t.Fatalf("metadata read = %+v, %v", info, err)
			}
		})
	}
}

// managerSessionReadErrorStorage fails only the session read while retaining a positive existence check. managerSessionReadErrorStorage 仅让 Session 读取失败，同时保留键存在结果。
type managerSessionReadErrorStorage struct {
	adapter.Storage
	key string
}

func (s *managerSessionReadErrorStorage) Get(ctx context.Context, key string) (any, error) {
	if key == s.key {
		return nil, errors.New("session read unavailable")
	}
	return s.Storage.Get(ctx, key)
}

// TestManagerRejectsNegativeActivityTimestamp covers overflow in validation, sharing and cleanup. TestManagerRejectsNegativeActivityTimestamp 覆盖校验、共享与清理中的时间差溢出。
func TestManagerRejectsNegativeActivityTimestamp(t *testing.T) {
	ctx := context.Background()
	for _, marker := range []int64{-1, math.MinInt64} {
		for _, path := range []string{"check", "login by token", "introspection", "sharing"} {
			t.Run(path+"/"+time.Duration(marker).String(), func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.ActiveTimeout = 30
					cfg.IsShare = true
				})
				token, err := mgr.Login(ctx, "invalid-active-timestamp", "web", "browser")
				if err != nil {
					t.Fatal(err)
				}
				if err := mgr.storage.Set(ctx, mgr.getActiveKey(token), marker, time.Minute); err != nil {
					t.Fatal(err)
				}
				pool := &managerQueuedMaintenancePool{}
				mgr.pool = pool
				t.Cleanup(pool.runAll)
				switch path {
				case "check":
					err = mgr.CheckLogin(ctx, token)
				case "login by token":
					err = mgr.LoginByToken(ctx, token)
				case "introspection":
					result, inspectErr := mgr.IntrospectToken(ctx, token)
					if inspectErr != nil || result == nil || result.Active {
						t.Fatalf("introspection accepted negative timestamp: %+v, %v", result, inspectErr)
					}
				case "sharing":
					fresh, loginErr := mgr.Login(ctx, "invalid-active-timestamp", "web", "browser")
					if loginErr != nil || fresh == "" || fresh == token {
						t.Fatalf("shared malformed token: %q, %v", fresh, loginErr)
					}
					if mgr.storage.Exists(ctx, mgr.getTokenKey(token)) || mgr.storage.Exists(ctx, mgr.getActiveKey(token)) {
						t.Fatal("cleanup retained malformed token or activity marker")
					}
				}
				if (path == "check" || path == "login by token") && !errors.Is(err, derror.ErrInvalidToken) {
					t.Fatalf("negative timestamp validation = %v, want ErrInvalidToken", err)
				}
				if pool.taskCount() != 0 {
					t.Fatal("malformed activity scheduled maintenance")
				}
			})
		}
	}
}
