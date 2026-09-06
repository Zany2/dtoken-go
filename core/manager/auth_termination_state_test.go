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

// TestManagerConcreteDeviceTerminationValidatesArguments verifies invalid device arguments cannot trigger terminal writes. TestManagerConcreteDeviceTerminationValidatesArguments 验证非法设备参数不会触发终端写入。
func TestManagerConcreteDeviceTerminationValidatesArguments(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		action  func(*Manager, context.Context, string, ...string) error
		wantErr error
		event   listener.Event
	}{
		{"logout", (*Manager).LogoutByDeviceAndDeviceID, derror.ErrInvalidToken, listener.EventLogout},
		{"kickout", (*Manager).KickoutByDeviceAndDeviceID, derror.ErrTokenKickout, listener.EventKickout},
		{"replace", (*Manager).ReplaceByDeviceAndDeviceID, derror.ErrTokenReplaced, listener.EventReplace},
	}
	invalid := []struct {
		name string
		args []string
	}{
		{"missing arguments", nil},
		{"missing device ID", []string{"web"}},
		{"blank device", []string{" \t", "phone"}},
		{"blank device ID", []string{"web", " \t"}},
		{"extra argument", []string{"web", "phone", "unexpected"}},
		{"extra empty argument", []string{"web", "phone", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsConcurrent = true
				cfg.IsShare = false
			})
			loginID := "device-argument-validation"
			var tokens []string
			for _, device := range [][2]string{{"web", "phone"}, {"web", "tablet"}, {"mobile", "phone"}} {
				token, err := mgr.Login(ctx, loginID, device[0], device[1])
				if err != nil {
					t.Fatalf("Login(%v) error = %v", device, err)
				}
				tokens = append(tokens, token)
			}

			// Snapshot persisted state to detect unintended writes from rejected requests. 保存持久化状态快照，检查被拒绝的请求是否意外写入。
			keys := []string{mgr.getSessionKey(loginID)}
			for _, token := range tokens {
				keys = append(keys, mgr.getTokenKey(token))
			}
			before := make(map[string]any, len(keys))
			for _, key := range keys {
				value, err := mgr.storage.Get(ctx, key)
				if err != nil || value == nil {
					t.Fatalf("snapshot(%q) = %v, %v, want existing state", key, value, err)
				}
				before[key] = value
			}
			var events []listener.EventData
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
				events = append(events, *data)
			}, listener.ListenerConfig{Async: false})

			for _, input := range invalid {
				if err := tt.action(mgr, ctx, loginID, input.args...); !errors.Is(err, derror.ErrInvalidParam) {
					t.Fatalf("%s error = %v, want ErrInvalidParam", input.name, err)
				}
				for _, key := range keys {
					if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, before[key]) {
						t.Fatalf("%s changed %q: value=%v, error=%v", input.name, key, got, err)
					}
				}
				if len(events) != 0 {
					t.Fatalf("%s emitted events: %+v", input.name, events)
				}
			}

			// Parameter errors take precedence over storage access, while empty account errors remain unchanged. 参数错误先于存储访问返回，同时保留空账号的既有错误。
			storage := mgr.storage
			mgr.storage = &managerFailingStorage{Storage: storage, getErr: errors.New("unexpected storage read")}
			if err := tt.action(mgr, ctx, loginID, "web", "phone", "extra"); !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("invalid arguments with failing storage error = %v, want ErrInvalidParam", err)
			}
			if err := tt.action(mgr, ctx, "", "web", "phone", "extra"); !errors.Is(err, derror.ErrIDIsEmpty) {
				t.Fatalf("empty login ID error = %v, want ErrIDIsEmpty", err)
			}
			mgr.storage = storage

			// Normalized valid arguments must target both device dimensions and retain idempotency. 合法参数规范化后必须同时匹配两个设备维度，并保持幂等。
			for i := 0; i < 2; i++ {
				if err := tt.action(mgr, ctx, loginID, " web ", " phone\t"); err != nil {
					t.Fatalf("valid termination attempt %d error = %v", i+1, err)
				}
			}
			if err := mgr.CheckLogin(ctx, tokens[0]); !errors.Is(err, tt.wantErr) {
				t.Fatalf("target token state error = %v, want %v", err, tt.wantErr)
			}
			for _, token := range tokens[1:] {
				if err := mgr.CheckLogin(ctx, token); err != nil {
					t.Fatalf("unmatched token %q error = %v", token, err)
				}
			}
			remaining, err := mgr.GetTokenValueListByLoginID(ctx, loginID)
			if err != nil || !reflect.DeepEqual(remaining, tokens[1:]) {
				t.Fatalf("remaining tokens = %v, %v, want %v", remaining, err, tokens[1:])
			}
			if len(events) != 1 || events[0].Event != tt.event || events[0].Token != tokens[0] ||
				events[0].LoginID != loginID || events[0].Device != "web" || events[0].DeviceID != "phone" {
				t.Fatalf("termination events = %+v, want one %s event for the target device", events, tt.event)
			}
		})
	}
}

// TestManagerScopedTerminationPersistsStateDuringTemporaryDisable verifies reversible disable rules cannot prevent an explicit terminal state transition. TestManagerScopedTerminationPersistsStateDuringTemporaryDisable 验证可逆封禁规则不会阻止显式终端状态迁移。
func TestManagerScopedTerminationPersistsStateDuringTemporaryDisable(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		action  func(*Manager, string) error
		wantErr error
	}{
		{
			name: "kickout",
			action: func(mgr *Manager, loginID string) error {
				return mgr.KickoutByDevice(ctx, loginID, "web")
			},
			wantErr: derror.ErrTokenKickout,
		},
		{
			name: "replace",
			action: func(mgr *Manager, loginID string) error {
				return mgr.ReplaceByDevice(ctx, loginID, "web")
			},
			wantErr: derror.ErrTokenReplaced,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsConcurrent = true
				cfg.IsShare = false
			})
			loginID := "disabled-termination-" + tt.name
			token, err := mgr.Login(ctx, loginID, "web", "browser")
			if err != nil {
				t.Fatalf("Login(web) error = %v", err)
			}
			keptToken, err := mgr.Login(ctx, loginID, "mobile", "phone")
			if err != nil {
				t.Fatalf("Login(mobile) error = %v", err)
			}
			if err = mgr.DisableDevice(ctx, loginID, "web", time.Minute); err != nil {
				t.Fatalf("DisableDevice() error = %v", err)
			}

			if err = tt.action(mgr, loginID); err != nil {
				t.Fatalf("termination action error = %v", err)
			}
			if err = mgr.UntieDevice(ctx, loginID, "web"); err != nil {
				t.Fatalf("UntieDevice() error = %v", err)
			}
			if err = mgr.CheckLogin(ctx, token); !errors.Is(err, tt.wantErr) {
				t.Fatalf("terminated token CheckLogin() error = %v, want %v", err, tt.wantErr)
			}
			if err = mgr.CheckLogin(ctx, keptToken); err != nil {
				t.Fatalf("kept token CheckLogin() error = %v", err)
			}
		})
	}
}

// TestManagerScopedTerminationSkipsEventsForStaleTokens verifies cleanup does not report state changes for already inactive tokens. TestManagerScopedTerminationSkipsEventsForStaleTokens 验证清理过程不会为已失效 Token 上报状态变更事件。
func TestManagerScopedTerminationSkipsEventsForStaleTokens(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
	})

	staleToken, err := mgr.Login(ctx, "termination-events", "web", "stale")
	if err != nil {
		t.Fatalf("Login(stale) error = %v", err)
	}
	activeToken, err := mgr.Login(ctx, "termination-events", "mobile", "active")
	if err != nil {
		t.Fatalf("Login(active) error = %v", err)
	}
	if err = mgr.storage.Delete(ctx, mgr.getTokenKey(staleToken)); err != nil {
		t.Fatalf("Delete(stale token) error = %v", err)
	}

	var tokens []string
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventKickout, func(data *listener.EventData) {
		tokens = append(tokens, data.Token)
	}, listener.ListenerConfig{Async: false})

	if err = mgr.KickoutByLoginID(ctx, "termination-events"); err != nil {
		t.Fatalf("KickoutByLoginID() error = %v", err)
	}
	if len(tokens) != 1 || tokens[0] != activeToken {
		t.Fatalf("kickout event tokens = %v, want [%s]", tokens, activeToken)
	}
}
