package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerSharedLoginEnforcesLimit covers shared-token retention, scope, overflow modes and boundary counts. TestManagerSharedLoginEnforcesLimit 覆盖共享 Token 保留、作用域、超限模式与数量边界。
func TestManagerSharedLoginEnforcesLimit(t *testing.T) {
	ctx := context.Background()
	for _, scope := range []config.ConcurrencyScope{config.ConcurrencyScopeAccount, config.ConcurrencyScopeDevice} {
		for _, mode := range []struct {
			mode  config.LogoutMode
			err   error
			event listener.Event
		}{
			{config.LogoutModeLogout, derror.ErrInvalidToken, listener.EventLogout},
			{config.LogoutModeKickout, derror.ErrTokenKickout, listener.EventKickout},
			{config.LogoutModeReplaced, derror.ErrTokenReplaced, listener.EventReplace},
		} {
			for _, tc := range []struct {
				name       string
				device     string
				deviceID   string
				shared     int
				limit      int64
				global     bool
				account    []int
				deviceOnly []int
			}{
				{"newest-limit-one", "web", "c", 3, 1, false, []int{3}, []int{0, 3}},
				{"oldest-limit-one", "web", "a", 1, 1, false, []int{1}, []int{0, 1}},
				{"device-id-only", "", "a", 1, 2, false, []int{1, 3}, []int{0, 1, 3}},
				{"no-device-filter", "", "", 3, 2, false, []int{2, 3}, []int{0, 2, 3}},
				{"global-limit", "web", "a", 1, 1, true, []int{1}, []int{0, 1}},
				{"at-limit", "web", "c", 3, 4, false, []int{0, 1, 2, 3}, []int{0, 1, 2, 3}},
				{"unlimited", "web", "c", 3, config.NoLimit, false, []int{0, 1, 2, 3}, []int{0, 1, 2, 3}},
			} {
				t.Run(string(scope)+"/"+string(mode.mode)+"/"+tc.name, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.IsConcurrent = true
						cfg.IsShare = true
						cfg.ConcurrencyScope = scope
						cfg.MaxLoginCount = config.NoLimit
						cfg.OverflowLogoutMode = config.LogoutModeLogout
						if tc.global {
							cfg.MaxLoginCount = tc.limit
							cfg.OverflowLogoutMode = mode.mode
						}
					})
					if tc.global {
						mgr.storage = &managerStorageOnly{inner: newManagerTestStorage()}
					}
					id := "shared-login-limit"
					unlimited := int64(config.NoLimit)
					var pairs []*RefreshTokenPair
					for i, deviceID := range []string{"phone", "a", "b", "c"} {
						device := "web"
						if i == 0 {
							device = "mobile"
						}
						pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{
							LoginOptions: LoginOptions{LoginID: id, Device: device, DeviceID: deviceID, MaxLoginCount: &unlimited},
						})
						if err != nil {
							t.Fatal(err)
						}
						pairs = append(pairs, pair)
					}
					if err := mgr.SetSessionValue(ctx, id, "theme", "dark"); err != nil {
						t.Fatal(err)
					}
					before, err := mgr.getSession(ctx, id)
					if err != nil {
						t.Fatal(err)
					}

					// A shared login must remain independent of token generation even when eviction is required. 即使需要淘汰，共享登录也不能依赖 Token 生成器。
					mgr.generator = managerErrorGenerator{}
					var events []*listener.EventData
					mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(event *listener.EventData) {
						events = append(events, event)
					}, listener.ListenerConfig{Async: false})
					limit := tc.limit
					if tc.name == "at-limit" && scope == config.ConcurrencyScopeDevice {
						limit = 3
					}
					opts := LoginOptions{LoginID: id, Device: tc.device, DeviceID: tc.deviceID}
					if !tc.global {
						opts.MaxLoginCount = &limit
						opts.OverflowLogoutMode = &mode.mode
					}
					token, err := mgr.LoginWithOptions(ctx, opts)
					if err != nil || token != pairs[tc.shared].AccessToken {
						t.Fatalf("shared login = %q, %v, want %q", token, err, pairs[tc.shared].AccessToken)
					}
					after, err := mgr.getSession(ctx, id)
					if err != nil {
						t.Fatal(err)
					}
					wantIndices := tc.account
					if scope == config.ConcurrencyScopeDevice {
						wantIndices = tc.deviceOnly
					}
					var wantTokens, gotTokens []string
					kept := make(map[int]bool)
					for _, i := range wantIndices {
						kept[i] = true
						wantTokens = append(wantTokens, pairs[i].AccessToken)
					}
					for _, terminal := range after.TerminalInfos {
						gotTokens = append(gotTokens, terminal.Token)
					}
					if !reflect.DeepEqual(gotTokens, wantTokens) || after.Data["theme"] != "dark" || after.CreateTime != before.CreateTime || after.HistoryTerminalCount != before.HistoryTerminalCount {
						t.Fatalf("session = %+v, want tokens %v and unchanged account data/history", after, wantTokens)
					}
					if len(events) != len(pairs)-len(kept)+1 {
						t.Fatalf("events = %+v, want eviction events followed by one shared login", events)
					}
					eventIndex := 0
					for i, pair := range pairs {
						if kept[i] {
							if err := mgr.CheckLogin(ctx, pair.AccessToken); err != nil {
								t.Fatalf("retained token %d is invalid: %v", i, err)
							}
						} else {
							if _, err := mgr.getTokenRecord(ctx, pair.AccessToken); !errors.Is(err, mode.err) {
								t.Fatalf("retired token %d state = %v, want %v", i, err, mode.err)
							}
							event := events[eventIndex]
							if event.Event != mode.event || event.Token != pair.AccessToken || event.Device != pair.Device || event.DeviceID != pair.DeviceID {
								t.Fatalf("eviction event %d = %+v", eventIndex, event)
							}
							eventIndex++
						}
						for _, key := range []string{mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken)} {
							if exists := mgr.storage.Exists(ctx, key); exists != kept[i] {
								t.Fatalf("refresh binding %q exists = %v, want %v", key, exists, kept[i])
							}
						}
					}
					sharedEvent := events[eventIndex]
					if sharedEvent.Event != listener.EventLogin || sharedEvent.Token != token || sharedEvent.Device != "web" || sharedEvent.DeviceID != pairs[tc.shared].DeviceID || sharedEvent.Extra[listener.ExtraKeyShared] != true {
						t.Fatalf("shared login event = %+v", sharedEvent)
					}
				})
			}
		}
	}
}
