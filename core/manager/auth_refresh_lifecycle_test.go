package manager

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerOldRefreshTokenPreservesReusedAccess verifies stale refresh credentials cannot retire a new login. TestManagerOldRefreshTokenPreservesReusedAccess 验证旧刷新凭证不能下线复用 Token 的新登录。
func TestManagerOldRefreshTokenPreservesReusedAccess(t *testing.T) {
	ctx := context.Background()
	for _, storageMode := range []string{"atomic", "basic"} {
		for _, operation := range []string{"revoke", "rotate"} {
			for _, loginMode := range []string{"plain", "pair", "foreign pair"} {
				for _, recordMode := range []string{"current", "legacy"} {
					t.Run(storageMode+"/"+operation+"/"+loginMode+"/"+recordMode, func(t *testing.T) {
						mgr := newTestManager(t, func(cfg *config.Config) {
							cfg.Timeout = 60
							cfg.RefreshTokenTimeout = 120
							cfg.ActiveTimeout = 60
							cfg.RenewInterval = 30
						})
						if storageMode == "basic" {
							mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
						}
						opts := LoginOptions{LoginID: "refresh-reuse", Token: "fixed-access", Device: "web", DeviceID: "browser"}
						oldPair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: opts})
						if err != nil {
							t.Fatalf("old login error = %v", err)
						}
						oldAccess, err := mgr.getTokenRecord(ctx, oldPair.AccessToken)
						if err != nil {
							t.Fatalf("get old access record error = %v", err)
						}
						oldRefresh, err := mgr.getRefreshTokenInfo(ctx, oldPair.RefreshToken)
						if err != nil || oldRefresh.AccessID == "" || oldRefresh.AccessID != oldAccess.AccessID {
							t.Fatalf("old refresh binding = %+v, %v, want matching nonempty identity", oldRefresh, err)
						}
						if recordMode == "legacy" {
							if err = mgr.saveToStorage(ctx, mgr.getRefreshTokenKey(oldPair.RefreshToken), oldRefresh.RefreshTokenInfo, 2*time.Minute); err != nil {
								t.Fatalf("save legacy refresh error = %v", err)
							}
						}

						// Expire only the access side; the old refresh credential remains usable. 仅模拟访问侧过期，旧刷新凭证保持可用。
						if err = mgr.storage.Delete(ctx, mgr.getTokenKey(opts.Token), mgr.getSessionKey(opts.LoginID), mgr.getTokenRefreshKey(opts.Token), mgr.getActiveKey(opts.Token), mgr.getRenewKey(opts.Token)); err != nil {
							t.Fatalf("expire access fixture error = %v", err)
						}
						if loginMode == "foreign pair" {
							opts.LoginID = "refresh-reuse-other"
						}
						var newPair *RefreshTokenPair
						if loginMode == "plain" {
							_, err = mgr.LoginWithOptions(ctx, opts)
						} else {
							newPair, err = mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: opts})
						}
						if err != nil {
							t.Fatalf("new login error = %v", err)
						}
						current, err := mgr.getTokenRecord(ctx, opts.Token)
						if err != nil || current.AccessID == "" || current.AccessID == oldAccess.AccessID {
							t.Fatalf("new access binding = %+v, %v, want distinct nonempty identity", current, err)
						}

						// Force equal timestamps and terminal indexes so only lifecycle identity can disambiguate same-account reuse. 强制时间戳和终端序号相同，使同账号复用只能通过生命周期标识区分。
						current.CreateTime = oldAccess.CreateTime
						if err = mgr.saveToStorage(ctx, mgr.getTokenKey(opts.Token), *current); err != nil {
							t.Fatalf("save current record error = %v", err)
						}
						sess, err := mgr.getSession(ctx, opts.LoginID)
						if err != nil || len(sess.TerminalInfos) != 1 {
							t.Fatalf("new session = %+v, %v, want one terminal", sess, err)
						}
						sess.TerminalInfos[0].CreateTime = oldAccess.CreateTime
						if err = mgr.saveToStorage(ctx, mgr.getSessionKey(opts.LoginID), *sess); err != nil {
							t.Fatalf("save current session error = %v", err)
						}
						wantTerminal := sess.TerminalInfos[0]
						keys := []string{mgr.getTokenKey(opts.Token), mgr.getTokenRefreshKey(opts.Token), mgr.getActiveKey(opts.Token), mgr.getRenewKey(opts.Token)}
						if newPair != nil {
							keys = append(keys, mgr.getRefreshTokenKey(newPair.RefreshToken))
						}
						before := make(map[string]any, len(keys))
						for _, key := range keys {
							before[key], err = mgr.storage.Get(ctx, key)
							if err != nil {
								t.Fatalf("snapshot %q error = %v", key, err)
							}
						}
						logouts, destroys, revokes, rotations := 0, 0, 0, 0
						mgr.GetEventManager().RegisterFuncWithConfig(listener.EventLogout, func(*listener.EventData) { logouts++ }, listener.ListenerConfig{Async: false})
						mgr.GetEventManager().RegisterFuncWithConfig(listener.EventDestroySession, func(*listener.EventData) { destroys++ }, listener.ListenerConfig{Async: false})
						mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRefreshTokenRevoke, func(*listener.EventData) { revokes++ }, listener.ListenerConfig{Async: false})
						mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRefreshTokenRotate, func(*listener.EventData) { rotations++ }, listener.ListenerConfig{Async: false})
						if operation == "revoke" {
							err = mgr.RevokeRefreshToken(ctx, oldPair.RefreshToken)
						} else {
							var rotated *RefreshTokenPair
							rotated, err = mgr.RefreshToken(ctx, oldPair.RefreshToken)
							if err == nil && (rotated.AccessToken == opts.Token || rotated.RefreshToken == oldPair.RefreshToken) {
								t.Fatal("rotation did not issue a distinct pair")
							}
						}
						if err != nil {
							t.Fatalf("%s old refresh error = %v", operation, err)
						}
						for _, key := range keys {
							after, getErr := mgr.storage.Get(ctx, key)
							if getErr != nil || !reflect.DeepEqual(after, before[key]) {
								t.Fatalf("new lifecycle key %q changed: before=%v after=%v error=%v", key, before[key], after, getErr)
							}
						}
						sess, err = mgr.getSession(ctx, opts.LoginID)
						if err != nil {
							t.Fatalf("get new session after cleanup error = %v", err)
						}
						matches := sess.filterTerminals(func(ti TerminalInfo) bool { return ti.Token == opts.Token })
						if len(matches) != 1 || !reflect.DeepEqual(matches[0], wantTerminal) {
							t.Fatalf("new terminal changed: got %+v, want %+v", matches, wantTerminal)
						}
						if mgr.storage.Exists(ctx, mgr.getRefreshTokenKey(oldPair.RefreshToken)) {
							t.Fatal("old refresh credential was not consumed")
						}
						if _, err = mgr.RefreshToken(ctx, oldPair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
							t.Fatalf("replayed refresh error = %v, want ErrInvalidRefreshToken", err)
						}
						if err = mgr.RevokeRefreshToken(ctx, oldPair.RefreshToken); err != nil {
							t.Fatalf("repeated revoke error = %v", err)
						}
						if logouts != 0 || destroys != 0 || (operation == "revoke" && (revokes != 1 || rotations != 0)) || (operation == "rotate" && (rotations != 1 || revokes != 0)) {
							t.Fatalf("unexpected events: logout=%d destroy=%d revoke=%d rotate=%d", logouts, destroys, revokes, rotations)
						}
					})
				}
			}
		}
	}
}

// TestManagerRefreshBindingSurvivesRenewal verifies metadata rewrites preserve the original access binding. TestManagerRefreshBindingSurvivesRenewal 验证元数据重写保留原访问绑定。
func TestManagerRefreshBindingSurvivesRenewal(t *testing.T) {
	ctx := context.Background()
	for _, storageMode := range []string{"atomic", "basic"} {
		for _, scenario := range []string{"manual renewal", "shared login", "detached access", "missing session", "legacy pair"} {
			t.Run(storageMode+"/"+scenario, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.Timeout = 60
					cfg.RefreshTokenTimeout = 120
				})
				if storageMode == "basic" {
					mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
				}
				pair, err := mgr.LoginWithRefreshToken(ctx, "refresh-renewal", "web", "browser")
				if err != nil {
					t.Fatalf("login error = %v", err)
				}
				before, err := mgr.getTokenRecord(ctx, pair.AccessToken)
				if err != nil || before.AccessID == "" {
					t.Fatalf("initial binding = %+v, %v", before, err)
				}
				switch scenario {
				case "manual renewal":
					err = mgr.RenewTimeout(ctx, pair.AccessToken, 2*time.Minute)
				case "shared login":
					var shared string
					shared, err = mgr.Login(ctx, pair.LoginID, "web", "browser")
					if err == nil && shared != pair.AccessToken {
						t.Fatalf("shared token = %q, want %q", shared, pair.AccessToken)
					}
				case "detached access":
					var sess *Session
					sess, err = mgr.getSession(ctx, pair.LoginID)
					if err == nil {
						sess.removeTerminalByToken(pair.AccessToken)
						err = mgr.saveToStorage(ctx, mgr.getSessionKey(pair.LoginID), *sess)
					}
				case "missing session":
					err = mgr.storage.Delete(ctx, mgr.getSessionKey(pair.LoginID))
				case "legacy pair":
					var refresh *refreshTokenRecord
					refresh, err = mgr.getRefreshTokenInfo(ctx, pair.RefreshToken)
					if err == nil {
						err = mgr.saveToStorage(ctx, mgr.getRefreshTokenKey(pair.RefreshToken), refresh.RefreshTokenInfo)
					}
					if err == nil {
						err = mgr.saveToStorage(ctx, mgr.getTokenKey(pair.AccessToken), before.TokenInfo)
					}
				}
				if err != nil {
					t.Fatalf("prepare %s error = %v", scenario, err)
				}
				if scenario != "legacy pair" {
					after, getErr := mgr.getTokenRecord(ctx, pair.AccessToken)
					if getErr != nil || after.AccessID != before.AccessID {
						t.Fatalf("binding after %s = %+v, %v, want %q", scenario, after, getErr, before.AccessID)
					}
				}

				// A renewed access token can outlive its reverse lookup; revocation must still find its lifecycle. 续期后的访问 Token 可比反向索引存活更久，撤销仍须识别其生命周期。
				if err = mgr.storage.Delete(ctx, mgr.getTokenRefreshKey(pair.AccessToken)); err != nil {
					t.Fatalf("expire reverse lookup error = %v", err)
				}
				if err = mgr.RevokeRefreshToken(ctx, pair.RefreshToken); err != nil {
					t.Fatalf("revoke error = %v", err)
				}
				if mgr.storage.Exists(ctx, mgr.getTokenKey(pair.AccessToken)) || mgr.storage.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) {
					t.Fatal("revocation left the original access or refresh record")
				}
			})
		}
	}
}

// TestManagerRefreshRotationCanReuseExpiredAccessValue verifies cleanup also protects the replacement pair itself. TestManagerRefreshRotationCanReuseExpiredAccessValue 验证清理也能保护复用旧访问值的替代令牌对本身。
func TestManagerRefreshRotationCanReuseExpiredAccessValue(t *testing.T) {
	ctx := context.Background()
	for _, storageMode := range []string{"atomic", "basic"} {
		t.Run(storageMode, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 60
				cfg.RefreshTokenTimeout = 120
			})
			if storageMode == "basic" {
				mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
			}

			// Explicit login leaves the test generator at sequence zero; rotation will generate this same value. 显式登录保留测试生成器的零序号，轮换将生成相同值。
			pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
				LoginID: "refresh-same-value", Device: "web", DeviceID: "browser", Token: "token-refresh-same-value-web-browser-1",
			}})
			if err != nil {
				t.Fatalf("login error = %v", err)
			}
			old, err := mgr.getRefreshTokenInfo(ctx, pair.RefreshToken)
			if err != nil {
				t.Fatalf("get old refresh record error = %v", err)
			}
			if err = mgr.storage.Delete(ctx, mgr.getTokenKey(pair.AccessToken), mgr.getSessionKey(pair.LoginID), mgr.getTokenRefreshKey(pair.AccessToken)); err != nil {
				t.Fatalf("expire access side error = %v", err)
			}
			next, err := mgr.RefreshToken(ctx, pair.RefreshToken)
			if err != nil {
				t.Fatalf("rotate error = %v", err)
			}
			if next.AccessToken != pair.AccessToken || next.RefreshToken == pair.RefreshToken {
				t.Fatalf("replacement = %+v, want same access value and new refresh", next)
			}
			current, err := mgr.getRefreshTokenInfo(ctx, next.RefreshToken)
			if err != nil || current.AccessID == "" || current.AccessID == old.AccessID {
				t.Fatalf("replacement binding = %+v, %v, want a fresh identity", current, err)
			}
			if err = mgr.CheckLogin(ctx, next.AccessToken); err != nil {
				t.Fatalf("replacement login was retired: %v", err)
			}
			if err = mgr.RevokeRefreshToken(ctx, next.RefreshToken); err != nil {
				t.Fatalf("revoke replacement error = %v", err)
			}
			if mgr.storage.Exists(ctx, mgr.getTokenKey(next.AccessToken)) || mgr.storage.Exists(ctx, mgr.getTokenRefreshKey(next.AccessToken)) {
				t.Fatal("replacement revocation did not clean its own access binding")
			}
		})
	}
}

// TestTokenRecordJSONCompatibility verifies lifecycle identity does not change public token fields. TestTokenRecordJSONCompatibility 验证生命周期标识不改变公开 Token 字段。
func TestTokenRecordJSONCompatibility(t *testing.T) {
	legacy := TokenInfo{AuthType: "user", LoginID: "token-record", CreateTime: 123, Timeout: 60}
	encoded, err := json.Marshal(tokenRecord{TokenInfo: legacy, AccessID: "unique-login"})
	if err != nil {
		t.Fatalf("encode token record error = %v", err)
	}
	var public TokenInfo
	if err = json.Unmarshal(encoded, &public); err != nil || !reflect.DeepEqual(public, legacy) {
		t.Fatalf("public token metadata = %+v, %v, want %+v", public, err, legacy)
	}
	legacyBytes, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("encode legacy token error = %v", err)
	}
	var decoded tokenRecord
	if err = json.Unmarshal(legacyBytes, &decoded); err != nil || decoded.AccessID != "" || !reflect.DeepEqual(decoded.TokenInfo, legacy) {
		t.Fatalf("legacy token decode = %+v, %v", decoded, err)
	}
}
