package engine

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestManagerTokenValidationDefersAccountSession checks that only session-dependent operations load invalid account data. TestManagerTokenValidationDefersAccountSession 验证仅依赖会话的操作读取无效账号数据。
func TestManagerTokenValidationDefersAccountSession(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"missing", "null", "foreign namespace", "malformed", "read failure"} {
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
			case "null":
				err = mgr.saveToStorage(ctx, key, nil)
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
			before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getTokenKey(token)})
			if err := mgr.CheckLogin(ctx, token); err != nil {
				t.Fatalf("CheckLogin unexpectedly loaded session: %v", err)
			}
			if !mgr.IsLogin(ctx, token) {
				t.Fatal("IsLogin rejected valid token mapping")
			}
			for _, query := range []struct {
				name string
				call func(context.Context, string) (string, error)
				want string
			}{
				{"login ID", mgr.GetLoginID, "session-validation"}, {"device", mgr.GetDevice, "web"}, {"device ID", mgr.GetDeviceID, "browser"},
			} {
				if value, err := query.call(ctx, token); value != query.want || err != nil {
					t.Fatalf("%s = %q, %v, want %q and nil", query.name, value, err, query.want)
				}
			}
			if device, id, err := mgr.GetDeviceAndDeviceID(ctx, token); device != "web" || id != "browser" || err != nil {
				t.Fatalf("device identity = %q/%q, %v", device, id, err)
			}
			if created, err := mgr.GetTokenCreateTime(ctx, token); created <= 0 || err != nil {
				t.Fatalf("creation time = %d, %v", created, err)
			}
			if _, err := mgr.GetSessionByToken(ctx, token); !errors.Is(err, wantErr) {
				t.Fatalf("GetSessionByToken = %v, want %v", err, wantErr)
			}
			if _, err := mgr.GetPermissionsByToken(ctx, token); !errors.Is(err, wantErr) {
				t.Fatalf("GetPermissionsByToken = %v, want %v", err, wantErr)
			}
			if _, err := mgr.GetRolesByToken(ctx, token); !errors.Is(err, wantErr) {
				t.Fatalf("GetRolesByToken = %v, want %v", err, wantErr)
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
			if pool.taskCount() != 1 {
				t.Fatalf("coalesced maintenance tasks = %d, want 1", pool.taskCount())
			}
			pool.runAll()
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			// Activity belongs to the valid credential and no longer requires readable Session data. 活跃时间属于有效凭据，不再要求 Session 数据可读。
			if value, err := mgr.storage.Get(ctx, mgr.getActiveKey(token)); err != nil || value == nil {
				t.Fatalf("activity maintenance = %v, %v", value, err)
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
					// Candidate rejection need not eagerly delete its stored lifecycle. 拒绝共享候选不要求立即删除其存储生命周期。
					if checkErr := mgr.CheckLogin(ctx, token); !errors.Is(checkErr, derror.ErrInvalidToken) {
						t.Fatalf("malformed candidate remained usable: %v", checkErr)
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
