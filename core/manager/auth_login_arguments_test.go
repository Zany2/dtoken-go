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

// TestManagerLoginDeviceArguments verifies login argument limits without changing optional device semantics. TestManagerLoginDeviceArguments 验证登录参数数量限制，同时保留可选设备参数的语义。
func TestManagerLoginDeviceArguments(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		login       func(*Manager, string, ...string) (any, error)
		emptyResult any
		timeout     int64
	}{
		{
			name: "login",
			login: func(mgr *Manager, loginID string, args ...string) (any, error) {
				return mgr.Login(ctx, loginID, args...)
			},
			emptyResult: "",
			timeout:     60,
		},
		{
			name: "login with timeout",
			login: func(mgr *Manager, loginID string, args ...string) (any, error) {
				return mgr.LoginWithTimeout(ctx, loginID, 2*time.Minute, args...)
			},
			emptyResult: "",
			timeout:     120,
		},
		{
			name: "login with refresh token",
			login: func(mgr *Manager, loginID string, args ...string) (any, error) {
				return mgr.LoginWithRefreshToken(ctx, loginID, args...)
			},
			emptyResult: (*RefreshTokenPair)(nil),
			timeout:     60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("reject before side effects", func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsConcurrent = false
					cfg.IsShare = false
					cfg.ConcurrencyScope = config.ConcurrencyScopeAccount
					cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
				})
				loginID := "login-argument-validation"
				pair, err := mgr.LoginWithRefreshToken(ctx, loginID, "web", "phone")
				if err != nil {
					t.Fatalf("initial login error = %v", err)
				}

				// Rejected requests must preserve the existing session and both sides of its refresh binding. 被拒绝的请求必须保留已有 Session 及刷新令牌的双向绑定。
				keys := []string{
					mgr.getSessionKey(loginID), mgr.getTokenKey(pair.AccessToken),
					mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken),
				}
				before := make(map[string]any, len(keys))
				for _, key := range keys {
					value, err := mgr.storage.Get(ctx, key)
					if err != nil || value == nil {
						t.Fatalf("snapshot(%q) = %v, %v, want existing state", key, value, err)
					}
					before[key] = value
				}
				events := 0
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(*listener.EventData) {
					events++
				}, listener.ListenerConfig{Async: false})

				for _, args := range [][]string{
					{"web", "phone", "unexpected"},
					{"web", "phone", ""},
					{"", "", ""},
					{"web", "phone", "extra", "another"},
				} {
					result, err := tt.login(mgr, loginID, args...)
					if !errors.Is(err, derror.ErrInvalidParam) || !reflect.DeepEqual(result, tt.emptyResult) {
						t.Fatalf("login(%q) = %v, %v, want empty result and ErrInvalidParam", args, result, err)
					}
					for _, key := range keys {
						if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, before[key]) {
							t.Fatalf("login(%q) changed %q: value=%v, error=%v", args, key, got, err)
						}
					}
					if events != 0 {
						t.Fatalf("rejected login emitted %d events, want 0", events)
					}
				}

				// Argument validation must precede generation and storage access, including for new accounts. 参数校验必须先于生成器和存储访问，对新账号同样生效。
				mgr.generator = managerErrorGenerator{}
				if result, err := tt.login(mgr, "new-account", "web", "phone", "extra"); !errors.Is(err, derror.ErrInvalidParam) || !reflect.DeepEqual(result, tt.emptyResult) {
					t.Fatalf("login with failing generator = %v, %v, want empty result and ErrInvalidParam", result, err)
				}
				mgr.storage = &managerFailingStorage{Storage: mgr.storage, getErr: errors.New("unexpected storage read")}
				if result, err := tt.login(mgr, loginID, "web", "phone", "extra"); !errors.Is(err, derror.ErrInvalidParam) || !reflect.DeepEqual(result, tt.emptyResult) {
					t.Fatalf("login with failing storage = %v, %v, want empty result and ErrInvalidParam", result, err)
				}
				if result, err := tt.login(mgr, "", "web", "phone", "extra"); !errors.Is(err, derror.ErrIDIsEmpty) || !reflect.DeepEqual(result, tt.emptyResult) {
					t.Fatalf("empty login ID = %v, %v, want empty result and ErrIDIsEmpty", result, err)
				}
				if events != 0 {
					t.Fatalf("invalid login emitted %d events, want 0", events)
				}
			})

			for _, input := range []struct {
				name     string
				args     []string
				device   string
				deviceID string
			}{
				{"no device arguments", nil, "", ""},
				{"device only", []string{" web\t"}, "web", ""},
				{"concrete device", []string{" web ", " phone\t"}, "web", "phone"},
				{"device ID only", []string{" \t", " phone "}, "", "phone"},
				{"blank device", []string{" \t"}, "", ""},
				{"blank device fields", []string{" \t", "\n"}, "", ""},
			} {
				t.Run(input.name, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) { cfg.Timeout = 60 })
					result, err := tt.login(mgr, "valid-login-arguments", input.args...)
					if err != nil {
						t.Fatalf("login(%q) error = %v", input.args, err)
					}
					var token string
					switch value := result.(type) {
					case string:
						token = value
					case *RefreshTokenPair:
						if value == nil || value.RefreshToken == "" {
							t.Fatalf("refresh token pair = %+v, want a non-empty refresh token", value)
						}
						token = value.AccessToken
						refreshInfo, err := mgr.getRefreshTokenInfo(ctx, value.RefreshToken)
						if err != nil || refreshInfo.AccessToken != token || refreshInfo.Device != input.device || refreshInfo.DeviceID != input.deviceID {
							t.Fatalf("refresh metadata = %+v, %v, want normalized device binding", refreshInfo, err)
						}
					default:
						t.Fatalf("unexpected login result type %T", result)
					}
					info, err := mgr.GetTokenInfo(ctx, token)
					if err != nil || info.LoginID != "valid-login-arguments" || info.Device != input.device || info.DeviceID != input.deviceID || info.Timeout != tt.timeout {
						t.Fatalf("token metadata = %+v, %v, want normalized device fields and timeout %d", info, err, tt.timeout)
					}
				})
			}
		})
	}
}
