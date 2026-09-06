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
	"github.com/Zany2/dtoken-go/core/utils"
)

// TestManagerRefreshTokenPreservesTerminalExtra verifies terminal data survives rotation independently of access storage. TestManagerRefreshTokenPreservesTerminalExtra 验证终端数据可独立于访问令牌存储保留并跨轮换继承。
func TestManagerRefreshTokenPreservesTerminalExtra(t *testing.T) {
	ctx := context.Background()
	for _, storageMode := range []string{"atomic", "basic"} {
		t.Run(storageMode, func(t *testing.T) {
			for _, scenario := range []string{"live access", "expired access and session", "legacy record"} {
				t.Run(scenario, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.Timeout = 60
						cfg.RefreshTokenTimeout = 120
					})
					if storageMode == "basic" {
						mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
					}
					terminalExtra := map[string]any{"kind": "terminal", "nested": map[string]any{"client": "desktop"}}
					wantExtra := map[string]any{"kind": "terminal", "nested": map[string]any{"client": "desktop"}}
					tokenExtra := map[string]any{"kind": "token"}
					pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{
						LoginOptions: LoginOptions{
							LoginID:       "refresh-terminal-extra",
							Device:        "web",
							DeviceID:      "browser",
							Extra:         tokenExtra,
							TerminalExtra: terminalExtra,
						},
					})
					if err != nil {
						t.Fatalf("LoginWithRefreshTokenOptions() error = %v", err)
					}
					otherToken, err := mgr.Login(ctx, "other-refresh-account", "web", "other")
					if err != nil {
						t.Fatalf("Login(other account) error = %v", err)
					}

					// The new record must still decode into the unchanged public metadata structure. 新记录必须仍能解码到未修改的公开元数据结构。
					stored, err := mgr.storage.Get(ctx, mgr.getRefreshTokenKey(pair.RefreshToken))
					if err != nil {
						t.Fatalf("Get(refresh record) error = %v", err)
					}
					raw, err := utils.ToBytes(stored)
					if err != nil {
						t.Fatalf("ToBytes(refresh record) error = %v", err)
					}
					var legacy RefreshTokenInfo
					if err = mgr.serializer.Decode(raw, &legacy); err != nil || legacy.LoginID != pair.LoginID || legacy.AccessToken != pair.AccessToken || !reflect.DeepEqual(legacy.Extra, tokenExtra) {
						t.Fatalf("public metadata decode = %+v, %v, want existing flat fields", legacy, err)
					}
					if scenario == "legacy record" {
						if err = mgr.saveToStorage(ctx, mgr.getRefreshTokenKey(pair.RefreshToken), legacy, 2*time.Minute); err != nil {
							t.Fatalf("save legacy record error = %v", err)
						}
						// Legacy pairs omit lifecycle identity on both sides. 旧令牌对的两侧均不包含生命周期标识。
						access, accessErr := mgr.getTokenInfo(ctx, pair.AccessToken)
						if accessErr != nil {
							t.Fatalf("get legacy access metadata error = %v", accessErr)
						}
						if err = mgr.saveToStorage(ctx, mgr.getTokenKey(pair.AccessToken), *access); err != nil {
							t.Fatalf("save legacy access metadata error = %v", err)
						}
						wantExtra = nil
					}

					// Caller mutations after login must not alter the persisted terminal snapshot. 登录后调用方的修改不能改变已持久化的终端快照。
					terminalExtra["kind"] = "changed"
					terminalExtra["nested"].(map[string]any)["client"] = "changed"
					rotations, revocations := 0, 0
					mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRefreshTokenRotate, func(*listener.EventData) {
						rotations++
					}, listener.ListenerConfig{Async: false})
					mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRefreshTokenRevoke, func(*listener.EventData) {
						revocations++
					}, listener.ListenerConfig{Async: false})

					for i := 0; i < 2; i++ {
						oldPair := pair
						if scenario == "expired access and session" {
							// Simulate storage after access-side expiration without sleeps. 无需等待，模拟访问侧数据过期后的存储状态。
							if err = mgr.storage.Delete(ctx, mgr.getTokenKey(pair.AccessToken), mgr.getSessionKey(pair.LoginID), mgr.getRenewKey(pair.AccessToken), mgr.getActiveKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken)); err != nil {
								t.Fatalf("expire access-side fixture error = %v", err)
							}
						}
						pair, err = mgr.RefreshToken(ctx, oldPair.RefreshToken)
						if err != nil {
							t.Fatalf("RefreshToken(round %d) error = %v", i+1, err)
						}
						terminal, err := mgr.GetTerminalInfoByToken(ctx, pair.AccessToken)
						if err != nil || !reflect.DeepEqual(terminal.Extra, wantExtra) {
							t.Fatalf("terminal after round %d = %+v, %v, want extra %+v", i+1, terminal, err, wantExtra)
						}
						tokenInfo, err := mgr.GetTokenInfo(ctx, pair.AccessToken)
						if err != nil || !reflect.DeepEqual(tokenInfo.Extra, tokenExtra) {
							t.Fatalf("token metadata after round %d = %+v, %v, want separate token extra", i+1, tokenInfo, err)
						}
						record, err := mgr.getRefreshTokenInfo(ctx, pair.RefreshToken)
						if err != nil || record.AccessToken != pair.AccessToken || !reflect.DeepEqual(record.TerminalExtra, wantExtra) {
							t.Fatalf("refresh record after round %d = %+v, %v, want inherited terminal snapshot", i+1, record, err)
						}

						// Cleanup must retire only the old pair and leave the new reverse binding intact. 清理只能撤销旧令牌对，并保留新的反向绑定。
						for _, key := range []string{mgr.getTokenKey(oldPair.AccessToken), mgr.getRefreshTokenKey(oldPair.RefreshToken), mgr.getTokenRefreshKey(oldPair.AccessToken)} {
							if mgr.storage.Exists(ctx, key) {
								t.Fatalf("old pair key %q remains after rotation", key)
							}
						}
						reverse, err := mgr.storage.Get(ctx, mgr.getTokenRefreshKey(pair.AccessToken))
						if err != nil {
							t.Fatalf("Get(new reverse binding) error = %v", err)
						}
						reverseBytes, err := utils.ToBytes(reverse)
						if err != nil || string(reverseBytes) != pair.RefreshToken {
							t.Fatalf("new reverse binding = %v, %v, want %s", reverse, err, pair.RefreshToken)
						}
						if _, err = mgr.RefreshToken(ctx, oldPair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
							t.Fatalf("RefreshToken(replayed) error = %v, want ErrInvalidRefreshToken", err)
						}
					}
					for i := 0; i < 2; i++ {
						if err = mgr.RevokeRefreshToken(ctx, pair.RefreshToken); err != nil {
							t.Fatalf("RevokeRefreshToken() error = %v", err)
						}
					}
					for _, key := range []string{mgr.getTokenKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken)} {
						if mgr.storage.Exists(ctx, key) {
							t.Fatalf("revoked pair key %q remains", key)
						}
					}
					if rotations != 2 || revocations != 1 {
						t.Fatalf("refresh events = %d rotations, %d revocations, want 2 and 1", rotations, revocations)
					}
					if err = mgr.CheckLogin(ctx, otherToken); err != nil {
						t.Fatalf("other account CheckLogin() error = %v", err)
					}
				})
			}
		})
	}
}

// TestRefreshTokenRecordJSONCompatibility verifies the optional field does not nest or rename existing JSON metadata. TestRefreshTokenRecordJSONCompatibility 验证可选字段不会使已有 JSON 元数据嵌套或更名。
func TestRefreshTokenRecordJSONCompatibility(t *testing.T) {
	legacy := RefreshTokenInfo{AuthType: "user", LoginID: "user-1", AccessToken: "access-1", ExpiresIn: 120}
	legacyBytes, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Marshal(legacy) error = %v", err)
	}
	var legacyFields map[string]any
	if err = json.Unmarshal(legacyBytes, &legacyFields); err != nil {
		t.Fatalf("Unmarshal(legacy) error = %v", err)
	}
	for _, extra := range []map[string]any{nil, {"client": "desktop"}} {
		encoded, err := json.Marshal(refreshTokenRecord{RefreshTokenInfo: legacy, TerminalExtra: extra})
		if err != nil {
			t.Fatalf("Marshal(record) error = %v", err)
		}
		var fields map[string]any
		if err = json.Unmarshal(encoded, &fields); err != nil {
			t.Fatalf("Unmarshal(record) error = %v", err)
		}
		storedExtra, hasExtra := fields["terminalExtra"]
		if (extra != nil) != hasExtra || (hasExtra && !reflect.DeepEqual(storedExtra, extra)) {
			t.Fatalf("stored terminalExtra = %v, present=%v, want %+v", storedExtra, hasExtra, extra)
		}
		delete(fields, "terminalExtra")
		if !reflect.DeepEqual(fields, legacyFields) {
			t.Fatalf("existing JSON fields = %+v, want %+v", fields, legacyFields)
		}
	}
}
