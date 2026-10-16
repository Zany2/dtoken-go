package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerRefreshIssuanceRejectsReplacedLogin verifies login callbacks cannot redirect refresh issuance to a later lifecycle. TestManagerRefreshIssuanceRejectsReplacedLogin 验证登录回调不能将刷新签发转向后续生命周期。
func TestManagerRefreshIssuanceRejectsReplacedLogin(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"atomic", "basic"} {
		for _, event := range []listener.Event{listener.EventLogin, listener.EventCreateSession} {
			for _, replacement := range []string{"same plain", "same pair", "foreign pair"} {
				t.Run(mode+"/"+string(event)+"/"+replacement, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.Timeout = 60
						cfg.RefreshTokenTimeout = 120
						cfg.ActiveTimeout = 60
						cfg.RenewInterval = 30
					})
					if mode == "basic" {
						mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
					}
					opts := LoginOptions{LoginID: "issuance-original", Token: "issuance-fixed", Device: "web", DeviceID: "browser"}
					newLoginID := opts.LoginID
					if replacement == "foreign pair" {
						newLoginID = "issuance-replacement"
					}
					var callbackErr error
					var newPair *RefreshTokenPair
					var before map[string]any
					called := false
					creates := 0
					mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRefreshTokenCreate, func(*listener.EventData) { creates++ }, listener.ListenerConfig{Async: false})
					mgr.GetEventManager().RegisterFuncWithConfig(event, func(*listener.EventData) {
						if called {
							return
						}
						called = true
						callbackErr = mgr.Logout(ctx, opts.Token)
						if callbackErr != nil {
							return
						}
						newOpts := opts
						newOpts.LoginID = newLoginID
						if replacement == "same plain" {
							_, callbackErr = mgr.LoginWithOptions(ctx, newOpts)
						} else {
							newPair, callbackErr = mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: newOpts})
						}
						if callbackErr != nil {
							return
						}
						keys := []string{mgr.getTokenKey(opts.Token), mgr.getSessionKey(newLoginID), mgr.getTokenRefreshKey(opts.Token), mgr.getActiveKey(opts.Token), mgr.getRenewKey(opts.Token)}
						if newPair != nil {
							keys = append(keys, mgr.getRefreshTokenKey(newPair.RefreshToken))
						}
						before = make(map[string]any, len(keys))
						for _, key := range keys {
							before[key], callbackErr = mgr.storage.Get(ctx, key)
							if callbackErr != nil {
								return
							}
						}
					}, listener.ListenerConfig{Async: false})
					pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: opts})
					if !called || callbackErr != nil {
						t.Fatalf("replacement callback: called=%v error=%v", called, callbackErr)
					}
					if pair != nil || !errors.Is(err, derror.ErrInvalidToken) {
						t.Fatalf("outer issuance = %+v, %v, want nil and ErrInvalidToken", pair, err)
					}
					for key, want := range before {
						got, getErr := mgr.storage.Get(ctx, key)
						if getErr != nil || !reflect.DeepEqual(got, want) {
							t.Fatalf("replacement key %q changed: got=%v want=%v error=%v", key, got, want, getErr)
						}
					}
					wantCreates := 0
					if newPair != nil {
						wantCreates = 1
					}
					if creates != wantCreates {
						t.Fatalf("refresh creation events = %d, want %d", creates, wantCreates)
					}
				})
			}
		}
	}
}

// TestManagerRefreshIssuanceRechecksSession verifies an access mapping alone cannot authorize a new refresh credential. TestManagerRefreshIssuanceRechecksSession 验证只有访问映射不能授权签发新刷新凭证。
func TestManagerRefreshIssuanceRechecksSession(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	var callbackErr error
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventLogin, func(data *listener.EventData) {
		callbackErr = mgr.storage.Delete(ctx, mgr.getSessionKey(data.LoginID))
	}, listener.ListenerConfig{Async: false})
	pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{LoginID: "issuance-expired-session", Token: "issuance-detached"}})
	if callbackErr != nil {
		t.Fatalf("expire session callback error = %v", callbackErr)
	}
	if pair != nil || !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("issuance with missing session = %+v, %v, want nil and ErrInvalidToken", pair, err)
	}
	if mgr.storage.Exists(ctx, mgr.getTokenKey("issuance-detached")) || mgr.storage.Exists(ctx, mgr.getTokenRefreshKey("issuance-detached")) {
		t.Fatal("failed issuance retained its own detached access token or reverse index")
	}
}

// TestManagerRotationFailureCleanupChecksReplacementIdentity verifies compensation never logs out a later login. TestManagerRotationFailureCleanupChecksReplacementIdentity 验证失败清理不会登出后续登录。
func TestManagerRotationFailureCleanupChecksReplacementIdentity(t *testing.T) {
	ctx := context.Background()
	for _, reuse := range []bool{false, true} {
		name := "original replacement"
		if reuse {
			name = "reused replacement"
		}
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			old, err := mgr.LoginWithRefreshToken(ctx, "cleanup-old", "web")
			if err != nil {
				t.Fatalf("old login error = %v", err)
			}
			oldInfo, err := mgr.getRefreshTokenInfo(ctx, old.RefreshToken)
			if err != nil {
				t.Fatalf("read old binding error = %v", err)
			}
			replacement, err := mgr.LoginWithRefreshToken(ctx, "cleanup-replacement", "web")
			if err != nil {
				t.Fatalf("replacement login error = %v", err)
			}
			expected, err := mgr.getRefreshTokenInfo(ctx, replacement.RefreshToken)
			if err != nil {
				t.Fatalf("read replacement binding error = %v", err)
			}
			current := replacement
			if reuse {
				if err = mgr.Logout(ctx, replacement.AccessToken); err != nil {
					t.Fatalf("logout replacement error = %v", err)
				}
				current, err = mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
					LoginID: replacement.LoginID, Token: replacement.AccessToken, Device: "web", Timeout: time.Minute,
				}})
				if err != nil {
					t.Fatalf("reuse replacement value error = %v", err)
				}
			}
			keys := []string{mgr.getTokenKey(current.AccessToken), mgr.getSessionKey(current.LoginID), mgr.getRefreshTokenKey(current.RefreshToken), mgr.getTokenRefreshKey(current.AccessToken)}
			before := make(map[string]any, len(keys))
			for _, key := range keys {
				before[key], err = mgr.storage.Get(ctx, key)
				if err != nil {
					t.Fatalf("snapshot error = %v", err)
				}
			}
			// Fail only old-access decoding so replacement cleanup remains available. 仅令旧访问记录解码失败，使替代登录的清理仍可执行。
			mgr.storage = &managerFailingStorage{Storage: mgr.storage, getValues: map[string]any{mgr.getTokenKey(old.AccessToken): []byte("{")}}
			err = mgr.logoutRotatedAccessToken(ctx, oldInfo, replacement, expected.AccessID)
			if !errors.Is(err, derror.ErrSerializeFailed) {
				t.Fatalf("old cleanup error = %v, want ErrSerializeFailed", err)
			}
			for _, key := range keys {
				got, getErr := mgr.storage.Get(ctx, key)
				if getErr != nil || (reuse && !reflect.DeepEqual(got, before[key])) || (!reuse && got != nil) {
					t.Fatalf("replacement cleanup key %q: reuse=%v value=%v error=%v", key, reuse, got, getErr)
				}
			}
		})
	}
}
