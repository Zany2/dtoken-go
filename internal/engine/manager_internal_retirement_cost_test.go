package engine

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerLoginRetirementRecordReuse checks complete replacement, overflow and shared-login cleanup costs. TestManagerLoginRetirementRecordReuse 检查完整顶替、超限及共享登录清理中的记录复用成本。
func TestManagerLoginRetirementRecordReuse(t *testing.T) {
	ctx := context.Background()
	for _, scope := range []config.ConcurrencyScope{config.ConcurrencyScopeAccount, config.ConcurrencyScopeDevice} {
		for _, policy := range []string{"replace", "overflow", "share"} {
			for _, mode := range []config.LogoutMode{config.LogoutModeLogout, config.LogoutModeKickout, config.LogoutModeReplaced} {
				if policy == "replace" && mode != config.LogoutModeReplaced {
					continue
				}
				t.Run(string(scope)+"/"+policy+"/"+string(mode), func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.IsShare, cfg.IsConcurrent, cfg.MaxLoginCount = false, true, config.NoLimit
						cfg.ConcurrencyScope, cfg.ActiveTimeout = scope, 60
					})
					const id = "internal-retirement"
					var pairs []*RefreshTokenPair
					var extraKeys []string
					for i, device := range []string{"web", "web", "mobile"} {
						pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
							LoginID: id, Device: device, DeviceID: fmt.Sprint(i), Extra: map[string]any{"tenant": "test"},
						}})
						if err != nil {
							t.Fatal(err)
						}
						record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
						if err != nil {
							t.Fatal(err)
						}
						pairs = append(pairs, pair)
						extraKeys = append(extraKeys, mgr.getTokenExtraKey(record.AccessID))
					}
					mgr.config.IsConcurrent = policy != "replace"
					mgr.config.IsShare = policy == "share"
					mgr.config.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
					mgr.config.OverflowLogoutMode, mgr.config.MaxLoginCount = mode, 2
					if policy == "share" {
						mgr.config.MaxLoginCount = 1
					}
					storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int), sessionKey: mgr.getSessionKey(id)}
					mgr.storage = storage
					events := registerTerminalLifecycleEvents(mgr)
					token, err := mgr.Login(ctx, id, "web", "1")
					if err != nil {
						t.Fatal(err)
					}
					if storage.sessionWrites != 1 || policy == "share" && token != pairs[1].AccessToken {
						t.Fatalf("login = %q, session writes=%d", token, storage.sessionWrites)
					}
					retiredCount := 0
					for i, pair := range pairs {
						inScope := scope == config.ConcurrencyScopeAccount || pair.Device == "web"
						retired := inScope && (policy == "replace" || policy == "share" && i != 1 || policy == "overflow" && (i == 0 || scope == config.ConcurrencyScopeAccount && i == 1))
						wantGets := 0
						// The limit check first verifies occupancy. 数量限制先核实占位。
						if inScope {
							wantGets++
						}

						// Retirement or share selection performs one additional read. 下线或共享候选选择再读取一次。
						if retired || policy == "share" && i == 1 {
							wantGets++
						}
						if storage.gets[mgr.getTokenKey(pair.AccessToken)] != wantGets || storage.gets[extraKeys[i]] != 0 {
							t.Fatalf("terminal %d primary/Extra reads = %d/%d, want %d/0", i, storage.gets[mgr.getTokenKey(pair.AccessToken)], storage.gets[extraKeys[i]], wantGets)
						}
						if !retired {
							if !storage.Exists(ctx, extraKeys[i]) || !storage.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) {
								t.Fatal("retained terminal lost its metadata")
							}
							continue
						}
						retiredCount++
						if _, err := mgr.getTokenRecord(ctx, pair.AccessToken); !errors.Is(err, tokenStateError(tokenStateFromLogoutMode(mode))) {
							t.Fatalf("retired terminal state = %v", err)
						}
						for _, key := range []string{extraKeys[i], mgr.getActiveKey(pair.AccessToken), mgr.getRenewKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken)} {
							if storage.Exists(ctx, key) {
								t.Fatalf("retirement retained %s", key)
							}
						}
					}
					count := 0
					for _, event := range events() {
						if event.Event == listener.EventLogout || event.Event == listener.EventKickout || event.Event == listener.EventReplace {
							count++
						}
					}
					if count != retiredCount {
						t.Fatalf("retirement events = %d, want %d", count, retiredCount)
					}
				})
			}
		}
	}
}

// TestManagerDisableRetirementRecordReuse checks primary reads, preserved details and foreign ownership. TestManagerDisableRetirementRecordReuse 检查封禁清理的主记录读取、详情保留与外部归属保护。
func TestManagerDisableRetirementRecordReuse(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"active", "permanent", "expired", "kickout", "foreign account", "foreign namespace", "foreign refresh", "malformed"} {
		t.Run(state, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 600
				if state == "permanent" {
					cfg.Timeout = config.NoLimit
				}
			})
			pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
				LoginID: "disable-retirement", Device: "web", Extra: map[string]any{"tenant": "test"},
			}})
			if err != nil {
				t.Fatal(err)
			}
			record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
			if err != nil {
				t.Fatal(err)
			}
			primary, extra := mgr.getTokenKey(pair.AccessToken), mgr.getTokenExtraKey(record.AccessID)
			switch state {
			case "expired":
				err = mgr.storage.Delete(ctx, primary)
			case "kickout":
				err = mgr.storage.Set(ctx, primary, string(TokenStateKickOut), 10*time.Minute)
			case "foreign account":
				record.LoginID = "other"
				err = mgr.storage.Set(ctx, primary, encodeTokenRecord(record), 10*time.Minute)
			case "foreign namespace":
				record.AuthType += "-other"
				err = mgr.storage.Set(ctx, primary, encodeTokenRecord(record), 10*time.Minute)
			case "foreign refresh":
				info, readErr := mgr.getRefreshTokenInfo(ctx, pair.RefreshToken)
				if readErr != nil {
					t.Fatal(readErr)
				}
				info.LoginID = "other"
				if err = mgr.saveToStorage(ctx, mgr.getRefreshTokenKey(pair.RefreshToken), info); err == nil {
					err = mgr.storage.Delete(ctx, primary)
				}
			case "malformed":
				err = mgr.storage.Set(ctx, primary, "{", 10*time.Minute)
			}
			if err != nil {
				t.Fatal(err)
			}
			foreign := state == "foreign account" || state == "foreign namespace" || state == "foreign refresh" || state == "malformed"
			before := captureTerminalLifecycle(t, mgr, ctx, []string{primary, extra, mgr.getTokenRefreshKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken)})
			storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int)}
			codec := &retirementCostCodec{Codec: mgr.serializer}
			mgr.storage, mgr.serializer = storage, codec
			if err := mgr.Disable(ctx, pair.LoginID, time.Hour); err != nil {
				t.Fatal(err)
			}
			// Session and refresh metadata are decoded without loading Extra. 仅解码 Session 与刷新数据，不读取 Extra。
			wantDecodes := 2
			if foreign && state != "foreign refresh" {
				wantDecodes = 1
			} else if state == "expired" || state == "kickout" {
				// Refresh ownership check and revocation still read separately. 刷新归属核对及撤销仍各读取一次。
				wantDecodes = 3
			}
			if storage.gets[primary] != 1 || storage.gets[extra] != 0 || codec.decodes != wantDecodes {
				t.Fatalf("primary/Extra reads, decodes = %d/%d/%d", storage.gets[primary], storage.gets[extra], codec.decodes)
			}
			if foreign {
				assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
				return
			}
			if storage.Exists(ctx, mgr.getTokenRefreshKey(pair.AccessToken)) || storage.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) {
				t.Fatal("disable retained refresh credential")
			}
			if state == "active" || state == "permanent" {
				current, err := mgr.getTokenRecord(ctx, pair.AccessToken)
				if err != nil || !current.Revoked || current.AccessID != record.AccessID || !storage.Exists(ctx, extra) {
					t.Fatalf("revoked record = %+v, %v", current, err)
				}
				ttl, err := storage.TTL(ctx, primary)
				if err != nil || state == "permanent" && ttl != adapter.TTLNoExpire || state == "active" && (ttl <= 0 || ttl > 10*time.Minute) {
					t.Fatalf("revoked TTL = %v, %v", ttl, err)
				}
			}
			if err := mgr.Untie(ctx, pair.LoginID); err != nil {
				t.Fatal(err)
			}
			if err := mgr.CheckLogin(ctx, pair.AccessToken); err == nil {
				t.Fatal("untie revived retired credential")
			}
		})
	}
}

// TestManagerKnownInactiveStateTTL avoids another primary read when fallback has already been resolved. TestManagerKnownInactiveStateTTL 验证已确认失效的凭据在 TTL 回退时不再次读取主键。
func TestManagerKnownInactiveStateTTL(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) { cfg.Timeout = 600 })
	storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int), failTTL: true}
	mgr.storage = storage
	if ttl := mgr.tokenStateExpiration(ctx, "missing", nil); ttl != 10*time.Minute || storage.gets[mgr.getTokenKey("missing")] != 0 {
		t.Fatalf("known inactive TTL = %v, reads=%d", ttl, storage.gets[mgr.getTokenKey("missing")])
	}
	if err := mgr.applyLogoutModeToToken(ctx, "missing", nil, config.LogoutModeKickout); err != nil {
		t.Fatal(err)
	}
	if storage.gets[mgr.getTokenKey("missing")] != 0 {
		t.Fatal("known inactive state write fetched the primary again")
	}
	if _, err := mgr.getTokenRecord(ctx, "missing"); !errors.Is(err, derror.ErrTokenKickout) {
		t.Fatalf("state = %v", err)
	}
}
