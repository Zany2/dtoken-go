package manager

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

func TestManagerAutoRenewHonorsThresholdAndInterval(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 20
		cfg.AutoRenew = true
		cfg.RenewMaxRefresh = 5
		cfg.RenewInterval = 4
	})
	storage := requireManagerTestStorage(t, mgr)

	token, err := mgr.Login(ctx, "auto-renew", "web")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = storage.Delete(ctx, mgr.getRenewKey(token)); err != nil {
		t.Fatalf("Delete(renew marker) error = %v", err)
	}
	if err = storage.Expire(ctx, mgr.getTokenKey(token), 10*time.Second); err != nil {
		t.Fatalf("Expire(token, 10s) error = %v", err)
	}
	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatalf("CheckLogin(above threshold) error = %v", err)
	}
	if storage.Exists(ctx, mgr.getRenewKey(token)) {
		t.Fatal("renew marker exists above threshold, want no auto renew")
	}

	if err = storage.Expire(ctx, mgr.getTokenKey(token), 2*time.Second); err != nil {
		t.Fatalf("Expire(token, 2s) error = %v", err)
	}
	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatalf("CheckLogin(below threshold) error = %v", err)
	}
	waitForManagerTest(t, 500*time.Millisecond, func() bool {
		return storage.Exists(ctx, mgr.getRenewKey(token))
	})
	ttlAfterRenew, err := mgr.GetTokenTTL(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenTTL(after renew) error = %v", err)
	}
	if ttlAfterRenew < 15 || ttlAfterRenew > 20 {
		t.Fatalf("token ttl after renew = %d, want close to configured timeout", ttlAfterRenew)
	}

	if err = storage.Expire(ctx, mgr.getTokenKey(token), 1*time.Second); err != nil {
		t.Fatalf("Expire(token, 1s) error = %v", err)
	}
	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatalf("CheckLogin(renew interval blocked) error = %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	blockedTTL, err := mgr.GetTokenTTL(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenTTL(blocked) error = %v", err)
	}
	if blockedTTL > 2 {
		t.Fatalf("token ttl with renew marker = %d, want still near forced 1s ttl", blockedTTL)
	}
}

func TestManagerLoginByTokenRenewsTokenAndSessionTTL(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 30
		cfg.AutoRenew = false
		cfg.RenewInterval = 5
	})
	storage := requireManagerTestStorage(t, mgr)

	token, err := mgr.Login(ctx, "login-by-token", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = storage.Expire(ctx, mgr.getTokenKey(token), time.Second); err != nil {
		t.Fatalf("Expire(token) error = %v", err)
	}
	if err = storage.Expire(ctx, mgr.getSessionKey("login-by-token"), time.Second); err != nil {
		t.Fatalf("Expire(session) error = %v", err)
	}

	if err = mgr.LoginByToken(ctx, token); err != nil {
		t.Fatalf("LoginByToken() error = %v", err)
	}
	waitForManagerTest(t, 500*time.Millisecond, func() bool {
		tokenTTL, _ := mgr.GetTokenTTL(ctx, token)
		sessionTTL, _ := storage.TTL(ctx, mgr.getSessionKey("login-by-token"))
		return tokenTTL >= 20 && sessionTTL >= 20*time.Second
	})

	if !storage.Exists(ctx, mgr.getRenewKey(token)) {
		t.Fatal("renew marker missing after LoginByToken, want refreshed renew interval marker")
	}
}

func TestManagerLoginWithTimeoutEntryPoint(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
	})

	token, err := mgr.LoginWithTimeout(ctx, "timeout-entry", 2*time.Minute, "app", "device-1")
	if err != nil {
		t.Fatalf("LoginWithTimeout() error = %v", err)
	}
	info, err := mgr.GetTokenInfo(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenInfo() error = %v", err)
	}
	if info.LoginID != "timeout-entry" || info.Device != "app" || info.DeviceId != "device-1" || info.Timeout != 120 {
		t.Fatalf("TokenInfo = %+v, want login/device/deviceId/custom timeout", info)
	}
	ttl, err := mgr.GetTokenTTL(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 120 {
		t.Fatalf("GetTokenTTL() = %d, want 1..120", ttl)
	}
	if _, err = mgr.LoginWithTimeout(ctx, "", time.Minute); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("LoginWithTimeout(empty id) error = %v, want ErrIDIsEmpty", err)
	}
}

func TestManagerOverflowLogoutModesPreserveExpectedTokenState(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		mode    config.LogoutMode
		wantErr error
	}{
		{name: "logout", mode: config.LogoutModeLogout, wantErr: derror.ErrInvalidToken},
		{name: "kickout", mode: config.LogoutModeKickout, wantErr: derror.ErrTokenKickout},
		{name: "replaced", mode: config.LogoutModeReplaced, wantErr: derror.ErrTokenReplaced},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsConcurrent = true
				cfg.IsShare = false
				cfg.MaxLoginCount = 1
				cfg.OverflowLogoutMode = tt.mode
			})

			first, err := mgr.Login(ctx, "overflow-"+tt.name, "web", "old")
			if err != nil {
				t.Fatalf("first Login() error = %v", err)
			}
			second, err := mgr.Login(ctx, "overflow-"+tt.name, "web", "new")
			if err != nil {
				t.Fatalf("second Login() error = %v", err)
			}
			if first == second {
				t.Fatal("overflow login reused token, want a new token")
			}
			if err = mgr.CheckLogin(ctx, first); !errors.Is(err, tt.wantErr) {
				t.Fatalf("CheckLogin(old token) error = %v, want %v", err, tt.wantErr)
			}
			if err = mgr.CheckLogin(ctx, second); err != nil {
				t.Fatalf("CheckLogin(new token) error = %v", err)
			}
		})
	}
}

func TestManagerConcurrencyPolicyFullMatrix(t *testing.T) {
	ctx := context.Background()

	t.Run("non-concurrent replacement modes", func(t *testing.T) {
		tests := []struct {
			name       string
			scope      config.ConcurrencyScope
			mode       config.ReplacedLoginExitMode
			setup      func(*testing.T, *Manager) (oldToken string, keptToken string)
			login      func(*Manager) (string, error)
			assertions func(*testing.T, *Manager, string, string, string, error)
		}{
			{
				name:  "account old device replaces every old terminal",
				scope: config.ConcurrencyScopeAccount,
				mode:  config.ReplacedLoginExitModeOldDevice,
				setup: func(t *testing.T, mgr *Manager) (string, string) {
					t.Helper()
					token, err := mgr.Login(ctx, "matrix-account-old", "web", "a")
					if err != nil {
						t.Fatalf("Login(old) error = %v", err)
					}
					return token, ""
				},
				login: func(mgr *Manager) (string, error) {
					return mgr.Login(ctx, "matrix-account-old", "mobile", "b")
				},
				assertions: func(t *testing.T, mgr *Manager, oldToken, _ string, newToken string, err error) {
					t.Helper()
					if err != nil {
						t.Fatalf("new Login() error = %v", err)
					}
					if err = mgr.CheckLogin(ctx, oldToken); !errors.Is(err, derror.ErrTokenReplaced) {
						t.Fatalf("old token CheckLogin() error = %v, want ErrTokenReplaced", err)
					}
					if err = mgr.CheckLogin(ctx, newToken); err != nil {
						t.Fatalf("new token CheckLogin() error = %v", err)
					}
				},
			},
			{
				name:  "account new device rejects new login",
				scope: config.ConcurrencyScopeAccount,
				mode:  config.ReplacedLoginExitModeNewDevice,
				setup: func(t *testing.T, mgr *Manager) (string, string) {
					t.Helper()
					token, err := mgr.Login(ctx, "matrix-account-new", "web", "a")
					if err != nil {
						t.Fatalf("Login(old) error = %v", err)
					}
					return token, ""
				},
				login: func(mgr *Manager) (string, error) {
					return mgr.Login(ctx, "matrix-account-new", "mobile", "b")
				},
				assertions: func(t *testing.T, mgr *Manager, oldToken, _ string, newToken string, err error) {
					t.Helper()
					if !errors.Is(err, derror.ErrLoginLimitExceeded) {
						t.Fatalf("new Login() error = %v, want ErrLoginLimitExceeded", err)
					}
					if newToken != "" {
						t.Fatalf("new token = %q, want empty on rejected login", newToken)
					}
					if err = mgr.CheckLogin(ctx, oldToken); err != nil {
						t.Fatalf("old token CheckLogin() error = %v", err)
					}
				},
			},
			{
				name:  "device old device replaces only same device type",
				scope: config.ConcurrencyScopeDevice,
				mode:  config.ReplacedLoginExitModeOldDevice,
				setup: func(t *testing.T, mgr *Manager) (string, string) {
					t.Helper()
					web, err := mgr.Login(ctx, "matrix-device-old", "web", "a")
					if err != nil {
						t.Fatalf("Login(web) error = %v", err)
					}
					mobile, err := mgr.Login(ctx, "matrix-device-old", "mobile", "a")
					if err != nil {
						t.Fatalf("Login(mobile) error = %v", err)
					}
					return web, mobile
				},
				login: func(mgr *Manager) (string, error) {
					return mgr.Login(ctx, "matrix-device-old", "web", "b")
				},
				assertions: func(t *testing.T, mgr *Manager, oldToken, keptToken, newToken string, err error) {
					t.Helper()
					if err != nil {
						t.Fatalf("new Login() error = %v", err)
					}
					if err = mgr.CheckLogin(ctx, oldToken); !errors.Is(err, derror.ErrTokenReplaced) {
						t.Fatalf("old same-device token CheckLogin() error = %v, want ErrTokenReplaced", err)
					}
					if err = mgr.CheckLogin(ctx, keptToken); err != nil {
						t.Fatalf("other-device token CheckLogin() error = %v", err)
					}
					if err = mgr.CheckLogin(ctx, newToken); err != nil {
						t.Fatalf("new token CheckLogin() error = %v", err)
					}
				},
			},
			{
				name:  "device new device rejects only same device type",
				scope: config.ConcurrencyScopeDevice,
				mode:  config.ReplacedLoginExitModeNewDevice,
				setup: func(t *testing.T, mgr *Manager) (string, string) {
					t.Helper()
					web, err := mgr.Login(ctx, "matrix-device-new", "web", "a")
					if err != nil {
						t.Fatalf("Login(web) error = %v", err)
					}
					mobile, err := mgr.Login(ctx, "matrix-device-new", "mobile", "a")
					if err != nil {
						t.Fatalf("Login(mobile) error = %v", err)
					}
					return web, mobile
				},
				login: func(mgr *Manager) (string, error) {
					return mgr.Login(ctx, "matrix-device-new", "web", "b")
				},
				assertions: func(t *testing.T, mgr *Manager, oldToken, keptToken, newToken string, err error) {
					t.Helper()
					if !errors.Is(err, derror.ErrLoginLimitExceeded) {
						t.Fatalf("new Login() error = %v, want ErrLoginLimitExceeded", err)
					}
					if newToken != "" {
						t.Fatalf("new token = %q, want empty on rejected login", newToken)
					}
					for name, token := range map[string]string{"same-device": oldToken, "other-device": keptToken} {
						if err = mgr.CheckLogin(ctx, token); err != nil {
							t.Fatalf("%s token CheckLogin() error = %v", name, err)
						}
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsConcurrent = false
					cfg.ConcurrencyScope = tt.scope
					cfg.ReplacedLoginExitMode = tt.mode
				})
				oldToken, keptToken := tt.setup(t, mgr)
				newToken, err := tt.login(mgr)
				tt.assertions(t, mgr, oldToken, keptToken, newToken, err)
			})
		}
	})

	t.Run("share modes", func(t *testing.T) {
		tests := []struct {
			name      string
			scope     config.ConcurrencyScope
			isShare   bool
			wantReuse bool
		}{
			{name: "account share reuses same concrete device", scope: config.ConcurrencyScopeAccount, isShare: true, wantReuse: true},
			{name: "account no share creates new token", scope: config.ConcurrencyScopeAccount, isShare: false, wantReuse: false},
			{name: "device share reuses same concrete device", scope: config.ConcurrencyScopeDevice, isShare: true, wantReuse: true},
			{name: "device no share creates new token", scope: config.ConcurrencyScopeDevice, isShare: false, wantReuse: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsConcurrent = true
					cfg.IsShare = tt.isShare
					cfg.ConcurrencyScope = tt.scope
					cfg.MaxLoginCount = 3
				})
				first, err := mgr.Login(ctx, "matrix-share-"+tt.name, "web", "same")
				if err != nil {
					t.Fatalf("first Login() error = %v", err)
				}
				second, err := mgr.Login(ctx, "matrix-share-"+tt.name, "web", "same")
				if err != nil {
					t.Fatalf("second Login() error = %v", err)
				}
				reused := first == second
				if reused != tt.wantReuse {
					t.Fatalf("token reused = %v, want %v, first=%q second=%q", reused, tt.wantReuse, first, second)
				}
				wantCount := 2
				if tt.wantReuse {
					wantCount = 1
				}
				tokens, err := mgr.GetTokenValueListByLoginID(ctx, "matrix-share-"+tt.name, true)
				if err != nil {
					t.Fatalf("GetTokenValueListByLoginID() error = %v", err)
				}
				if len(tokens) != wantCount {
					t.Fatalf("alive token count = %d, want %d, tokens=%v", len(tokens), wantCount, tokens)
				}
			})
		}
	})

	t.Run("overflow modes by scope and share switch", func(t *testing.T) {
		tests := []struct {
			name        string
			scope       config.ConcurrencyScope
			isShare     bool
			mode        config.LogoutMode
			wantOldErr  error
			expectedSet func(second, third string) []string
		}{
			{name: "account share logout", scope: config.ConcurrencyScopeAccount, isShare: true, mode: config.LogoutModeLogout, wantOldErr: derror.ErrInvalidToken, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "account share kickout", scope: config.ConcurrencyScopeAccount, isShare: true, mode: config.LogoutModeKickout, wantOldErr: derror.ErrTokenKickout, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "account share replaced", scope: config.ConcurrencyScopeAccount, isShare: true, mode: config.LogoutModeReplaced, wantOldErr: derror.ErrTokenReplaced, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "account no share logout", scope: config.ConcurrencyScopeAccount, isShare: false, mode: config.LogoutModeLogout, wantOldErr: derror.ErrInvalidToken, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "account no share kickout", scope: config.ConcurrencyScopeAccount, isShare: false, mode: config.LogoutModeKickout, wantOldErr: derror.ErrTokenKickout, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "account no share replaced", scope: config.ConcurrencyScopeAccount, isShare: false, mode: config.LogoutModeReplaced, wantOldErr: derror.ErrTokenReplaced, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "device share logout", scope: config.ConcurrencyScopeDevice, isShare: true, mode: config.LogoutModeLogout, wantOldErr: derror.ErrInvalidToken, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "device share kickout", scope: config.ConcurrencyScopeDevice, isShare: true, mode: config.LogoutModeKickout, wantOldErr: derror.ErrTokenKickout, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "device share replaced", scope: config.ConcurrencyScopeDevice, isShare: true, mode: config.LogoutModeReplaced, wantOldErr: derror.ErrTokenReplaced, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "device no share logout", scope: config.ConcurrencyScopeDevice, isShare: false, mode: config.LogoutModeLogout, wantOldErr: derror.ErrInvalidToken, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "device no share kickout", scope: config.ConcurrencyScopeDevice, isShare: false, mode: config.LogoutModeKickout, wantOldErr: derror.ErrTokenKickout, expectedSet: func(second, third string) []string { return []string{second, third} }},
			{name: "device no share replaced", scope: config.ConcurrencyScopeDevice, isShare: false, mode: config.LogoutModeReplaced, wantOldErr: derror.ErrTokenReplaced, expectedSet: func(second, third string) []string { return []string{second, third} }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				loginID := "matrix-overflow-" + tt.name
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsConcurrent = true
					cfg.IsShare = tt.isShare
					cfg.ConcurrencyScope = tt.scope
					cfg.MaxLoginCount = 2
					cfg.OverflowLogoutMode = tt.mode
				})
				first, err := mgr.Login(ctx, loginID, "web", "a")
				if err != nil {
					t.Fatalf("first Login() error = %v", err)
				}
				second, err := mgr.Login(ctx, loginID, "web", "b")
				if err != nil {
					t.Fatalf("second Login() error = %v", err)
				}
				third, err := mgr.Login(ctx, loginID, "web", "c")
				if err != nil {
					t.Fatalf("third Login() error = %v", err)
				}
				if err = mgr.CheckLogin(ctx, first); !errors.Is(err, tt.wantOldErr) {
					t.Fatalf("oldest token CheckLogin() error = %v, want %v", err, tt.wantOldErr)
				}
				for name, token := range map[string]string{"second": second, "third": third} {
					if err = mgr.CheckLogin(ctx, token); err != nil {
						t.Fatalf("%s token CheckLogin() error = %v", name, err)
					}
				}
				alive, err := mgr.GetTokenValueListByDevice(ctx, loginID, "web", true)
				if err != nil {
					t.Fatalf("GetTokenValueListByDevice() error = %v", err)
				}
				if !sameStrings(alive, tt.expectedSet(second, third)) {
					t.Fatalf("alive tokens = %v, want [%s %s]", alive, second, third)
				}
			})
		}
	})

	t.Run("device scope overflow ignores other device types", func(t *testing.T) {
		tests := []struct {
			name    string
			isShare bool
			mode    config.LogoutMode
		}{
			{name: "share logout", isShare: true, mode: config.LogoutModeLogout},
			{name: "share kickout", isShare: true, mode: config.LogoutModeKickout},
			{name: "share replaced", isShare: true, mode: config.LogoutModeReplaced},
			{name: "no share logout", isShare: false, mode: config.LogoutModeLogout},
			{name: "no share kickout", isShare: false, mode: config.LogoutModeKickout},
			{name: "no share replaced", isShare: false, mode: config.LogoutModeReplaced},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				loginID := "matrix-device-ignore-" + tt.name
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsConcurrent = true
					cfg.IsShare = tt.isShare
					cfg.ConcurrencyScope = config.ConcurrencyScopeDevice
					cfg.MaxLoginCount = 2
					cfg.OverflowLogoutMode = tt.mode
				})
				webA, err := mgr.Login(ctx, loginID, "web", "a")
				if err != nil {
					t.Fatalf("Login(web/a) error = %v", err)
				}
				webB, err := mgr.Login(ctx, loginID, "web", "b")
				if err != nil {
					t.Fatalf("Login(web/b) error = %v", err)
				}
				mobile, err := mgr.Login(ctx, loginID, "mobile", "a")
				if err != nil {
					t.Fatalf("Login(mobile/a) error = %v", err)
				}
				webC, err := mgr.Login(ctx, loginID, "web", "c")
				if err != nil {
					t.Fatalf("Login(web/c) error = %v", err)
				}
				if webA == webB || webB == webC || webA == webC {
					t.Fatalf("web tokens should be distinct, got %q %q %q", webA, webB, webC)
				}
				if err = mgr.CheckLogin(ctx, mobile); err != nil {
					t.Fatalf("mobile token CheckLogin() error = %v, want other device alive", err)
				}
				webAlive, err := mgr.GetTokenValueListByDevice(ctx, loginID, "web", true)
				if err != nil {
					t.Fatalf("GetTokenValueListByDevice(web) error = %v", err)
				}
				if !sameStrings(webAlive, []string{webB, webC}) {
					t.Fatalf("alive web tokens = %v, want web/b and web/c", webAlive)
				}
			})
		}
	})

	t.Run("non-concurrent full option matrix", func(t *testing.T) {
		scopes := []config.ConcurrencyScope{
			config.ConcurrencyScopeAccount,
			config.ConcurrencyScopeDevice,
		}
		replacedModes := []config.ReplacedLoginExitMode{
			config.ReplacedLoginExitModeOldDevice,
			config.ReplacedLoginExitModeNewDevice,
		}
		shareModes := []bool{false, true}
		maxLoginCounts := []int64{config.NoLimit, 1, 2}
		overflowModes := []config.LogoutMode{
			config.LogoutModeLogout,
			config.LogoutModeKickout,
			config.LogoutModeReplaced,
		}

		for _, scope := range scopes {
			for _, replacedMode := range replacedModes {
				for _, isShare := range shareModes {
					for _, maxLoginCount := range maxLoginCounts {
						for _, overflowMode := range overflowModes {
							name := fmt.Sprintf("scope=%s/replaced=%s/share=%t/max=%d/overflow=%s",
								scope, replacedMode, isShare, maxLoginCount, overflowMode)
							t.Run(name, func(t *testing.T) {
								loginID := "matrix-non-concurrent-" + name
								mgr := newTestManager(t, func(cfg *config.Config) {
									cfg.IsConcurrent = false
									cfg.IsShare = isShare
									cfg.ConcurrencyScope = scope
									cfg.MaxLoginCount = maxLoginCount
									cfg.ReplacedLoginExitMode = replacedMode
									cfg.OverflowLogoutMode = overflowMode
								})

								oldToken, err := mgr.Login(ctx, loginID, "web", "a")
								if err != nil {
									t.Fatalf("old Login() error = %v", err)
								}
								var keptToken string
								if scope == config.ConcurrencyScopeDevice {
									keptToken, err = mgr.Login(ctx, loginID, "mobile", "a")
									if err != nil {
										t.Fatalf("other device Login() error = %v", err)
									}
								}

								newToken, err := mgr.Login(ctx, loginID, "web", "b")
								switch replacedMode {
								case config.ReplacedLoginExitModeOldDevice:
									if err != nil {
										t.Fatalf("new Login() error = %v", err)
									}
									if err = mgr.CheckLogin(ctx, oldToken); !errors.Is(err, derror.ErrTokenReplaced) {
										t.Fatalf("old token CheckLogin() error = %v, want ErrTokenReplaced", err)
									}
									if err = mgr.CheckLogin(ctx, newToken); err != nil {
										t.Fatalf("new token CheckLogin() error = %v", err)
									}
								case config.ReplacedLoginExitModeNewDevice:
									if !errors.Is(err, derror.ErrLoginLimitExceeded) {
										t.Fatalf("new Login() error = %v, want ErrLoginLimitExceeded", err)
									}
									if newToken != "" {
										t.Fatalf("new token = %q, want empty when login is rejected", newToken)
									}
									if err = mgr.CheckLogin(ctx, oldToken); err != nil {
										t.Fatalf("old token CheckLogin() error = %v", err)
									}
								}
								if keptToken != "" {
									if err = mgr.CheckLogin(ctx, keptToken); err != nil {
										t.Fatalf("other device token CheckLogin() error = %v", err)
									}
								}
							})
						}
					}
				}
			}
		}
	})

	t.Run("concurrent full option matrix", func(t *testing.T) {
		scopes := []config.ConcurrencyScope{
			config.ConcurrencyScopeAccount,
			config.ConcurrencyScopeDevice,
		}
		replacedModes := []config.ReplacedLoginExitMode{
			config.ReplacedLoginExitModeOldDevice,
			config.ReplacedLoginExitModeNewDevice,
		}
		shareModes := []bool{false, true}
		maxLoginCounts := []int64{config.NoLimit, 1, 2}
		overflowModes := []struct {
			mode    config.LogoutMode
			wantErr error
		}{
			{mode: config.LogoutModeLogout, wantErr: derror.ErrInvalidToken},
			{mode: config.LogoutModeKickout, wantErr: derror.ErrTokenKickout},
			{mode: config.LogoutModeReplaced, wantErr: derror.ErrTokenReplaced},
		}

		for _, scope := range scopes {
			for _, replacedMode := range replacedModes {
				for _, isShare := range shareModes {
					for _, maxLoginCount := range maxLoginCounts {
						for _, overflowMode := range overflowModes {
							name := fmt.Sprintf("scope=%s/replaced=%s/share=%t/max=%d/overflow=%s",
								scope, replacedMode, isShare, maxLoginCount, overflowMode.mode)
							t.Run(name, func(t *testing.T) {
								loginID := "matrix-concurrent-" + name
								mgr := newTestManager(t, func(cfg *config.Config) {
									cfg.IsConcurrent = true
									cfg.IsShare = isShare
									cfg.ConcurrencyScope = scope
									cfg.MaxLoginCount = maxLoginCount
									cfg.ReplacedLoginExitMode = replacedMode
									cfg.OverflowLogoutMode = overflowMode.mode
								})

								first, err := mgr.Login(ctx, loginID, "web", "a")
								if err != nil {
									t.Fatalf("first Login() error = %v", err)
								}
								second, err := mgr.Login(ctx, loginID, "web", "b")
								if err != nil {
									t.Fatalf("second Login() error = %v", err)
								}
								tokens := []string{first, second}
								if maxLoginCount == config.NoLimit || maxLoginCount == 2 {
									third, loginErr := mgr.Login(ctx, loginID, "web", "c")
									if loginErr != nil {
										t.Fatalf("third Login() error = %v", loginErr)
									}
									tokens = append(tokens, third)
								}

								if maxLoginCount == config.NoLimit {
									for _, token := range tokens {
										if err = mgr.CheckLogin(ctx, token); err != nil {
											t.Fatalf("unlimited token CheckLogin() error = %v", err)
										}
									}
									alive, err := mgr.GetTokenValueListByDevice(ctx, loginID, "web", true)
									if err != nil {
										t.Fatalf("GetTokenValueListByDevice() error = %v", err)
									}
									if !sameStrings(alive, tokens) {
										t.Fatalf("alive tokens = %v, want all tokens %v", alive, tokens)
									}
									return
								}

								if err = mgr.CheckLogin(ctx, first); !errors.Is(err, overflowMode.wantErr) {
									t.Fatalf("oldest token CheckLogin() error = %v, want %v", err, overflowMode.wantErr)
								}
								wantAlive := tokens[len(tokens)-int(maxLoginCount):]
								alive, err := mgr.GetTokenValueListByDevice(ctx, loginID, "web", true)
								if err != nil {
									t.Fatalf("GetTokenValueListByDevice() error = %v", err)
								}
								if !sameStrings(alive, wantAlive) {
									t.Fatalf("alive tokens = %v, want %v", alive, wantAlive)
								}
								for _, token := range wantAlive {
									if err = mgr.CheckLogin(ctx, token); err != nil {
										t.Fatalf("alive token CheckLogin() error = %v", err)
									}
								}
							})
						}
					}
				}
			}
		}
	})

	t.Run("login options override global concurrency matrix", func(t *testing.T) {
		scopes := []config.ConcurrencyScope{
			config.ConcurrencyScopeAccount,
			config.ConcurrencyScopeDevice,
		}
		shareModes := []bool{false, true}
		overflowModes := []struct {
			mode    config.LogoutMode
			wantErr error
		}{
			{mode: config.LogoutModeLogout, wantErr: derror.ErrInvalidToken},
			{mode: config.LogoutModeKickout, wantErr: derror.ErrTokenKickout},
			{mode: config.LogoutModeReplaced, wantErr: derror.ErrTokenReplaced},
		}

		for _, scope := range scopes {
			for _, isShare := range shareModes {
				for _, overflowMode := range overflowModes {
					name := fmt.Sprintf("scope=%s/share=%t/overflow=%s", scope, isShare, overflowMode.mode)
					t.Run(name, func(t *testing.T) {
						loginID := "matrix-options-" + name
						maxLoginCount := int64(1)
						mgr := newTestManager(t, func(cfg *config.Config) {
							cfg.IsConcurrent = false
							cfg.IsShare = !isShare
							cfg.ConcurrencyScope = scope
							cfg.MaxLoginCount = config.NoLimit
							cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeNewDevice
							cfg.OverflowLogoutMode = config.LogoutModeLogout
						})

						first, err := mgr.LoginWithOptions(ctx, LoginOptions{
							LoginID:            loginID,
							Device:             "web",
							DeviceID:           "a",
							IsConcurrent:       boolPtr(true),
							IsShare:            boolPtr(isShare),
							MaxLoginCount:      &maxLoginCount,
							OverflowLogoutMode: &overflowMode.mode,
						})
						if err != nil {
							t.Fatalf("first LoginWithOptions() error = %v", err)
						}
						second, err := mgr.LoginWithOptions(ctx, LoginOptions{
							LoginID:            loginID,
							Device:             "web",
							DeviceID:           "b",
							IsConcurrent:       boolPtr(true),
							IsShare:            boolPtr(isShare),
							MaxLoginCount:      &maxLoginCount,
							OverflowLogoutMode: &overflowMode.mode,
						})
						if err != nil {
							t.Fatalf("second LoginWithOptions() error = %v", err)
						}
						if err = mgr.CheckLogin(ctx, first); !errors.Is(err, overflowMode.wantErr) {
							t.Fatalf("first CheckLogin() error = %v, want %v", err, overflowMode.wantErr)
						}
						if err = mgr.CheckLogin(ctx, second); err != nil {
							t.Fatalf("second CheckLogin() error = %v", err)
						}
					})
				}
			}
		}
	})
}

func TestManagerAccountAndDeviceScopedExitOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("logout by login id invalidates all terminals", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		web, err := mgr.Login(ctx, "account-logout", "web")
		if err != nil {
			t.Fatalf("Login(web) error = %v", err)
		}
		mobile, err := mgr.Login(ctx, "account-logout", "mobile")
		if err != nil {
			t.Fatalf("Login(mobile) error = %v", err)
		}
		if err = mgr.LogoutByLoginID(ctx, "account-logout"); err != nil {
			t.Fatalf("LogoutByLoginID() error = %v", err)
		}
		for _, token := range []string{web, mobile} {
			if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
				t.Fatalf("CheckLogin(%s) error = %v, want ErrInvalidToken", token, err)
			}
		}
		if _, err = mgr.GetSession(ctx, "account-logout"); !errors.Is(err, derror.ErrSessionNotFound) {
			t.Fatalf("GetSession(after logout all) error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("kickout by device type keeps other devices alive", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		webA, err := mgr.Login(ctx, "device-kickout", "web", "a")
		if err != nil {
			t.Fatalf("Login(web/a) error = %v", err)
		}
		webB, err := mgr.Login(ctx, "device-kickout", "web", "b")
		if err != nil {
			t.Fatalf("Login(web/b) error = %v", err)
		}
		mobile, err := mgr.Login(ctx, "device-kickout", "mobile", "a")
		if err != nil {
			t.Fatalf("Login(mobile/a) error = %v", err)
		}
		if err = mgr.KickoutByDevice(ctx, "device-kickout", "web"); err != nil {
			t.Fatalf("KickoutByDevice() error = %v", err)
		}
		for _, token := range []string{webA, webB} {
			if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrTokenKickout) {
				t.Fatalf("web token CheckLogin() error = %v, want ErrTokenKickout", err)
			}
		}
		if err = mgr.CheckLogin(ctx, mobile); err != nil {
			t.Fatalf("mobile CheckLogin() error = %v, want alive", err)
		}
	})

	t.Run("replace concrete device keeps sibling terminals alive", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		webA, err := mgr.Login(ctx, "concrete-replace", "web", "a")
		if err != nil {
			t.Fatalf("Login(web/a) error = %v", err)
		}
		webB, err := mgr.Login(ctx, "concrete-replace", "web", "b")
		if err != nil {
			t.Fatalf("Login(web/b) error = %v", err)
		}
		if err = mgr.ReplaceByDeviceAndDeviceId(ctx, "concrete-replace", "web", "a"); err != nil {
			t.Fatalf("ReplaceByDeviceAndDeviceId() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, webA); !errors.Is(err, derror.ErrTokenReplaced) {
			t.Fatalf("web/a CheckLogin() error = %v, want ErrTokenReplaced", err)
		}
		if err = mgr.CheckLogin(ctx, webB); err != nil {
			t.Fatalf("web/b CheckLogin() error = %v, want alive", err)
		}
	})
}

func TestManagerRejectsNewLoginWhenReplacementModeIsNewDevice(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = false
		cfg.ConcurrencyScope = config.ConcurrencyScopeAccount
		cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeNewDevice
	})

	first, err := mgr.Login(ctx, "reject-new-device", "web")
	if err != nil {
		t.Fatalf("first Login() error = %v", err)
	}
	if _, err = mgr.Login(ctx, "reject-new-device", "mobile"); !errors.Is(err, derror.ErrLoginLimitExceeded) {
		t.Fatalf("second Login() error = %v, want ErrLoginLimitExceeded", err)
	}
	if err = mgr.CheckLogin(ctx, first); err != nil {
		t.Fatalf("old token CheckLogin() error = %v, want old token alive", err)
	}
	tokens, err := mgr.GetTokenValueListByLoginID(ctx, "reject-new-device", true)
	if err != nil {
		t.Fatalf("GetTokenValueListByLoginID() error = %v", err)
	}
	if !sameStrings(tokens, []string{first}) {
		t.Fatalf("alive tokens = %v, want only first token", tokens)
	}
}

func TestManagerDeviceAndConcreteExitOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("kickout concrete device keeps sibling device alive", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		webA, err := mgr.Login(ctx, "concrete-kickout", "web", "a")
		if err != nil {
			t.Fatalf("Login(web/a) error = %v", err)
		}
		webB, err := mgr.Login(ctx, "concrete-kickout", "web", "b")
		if err != nil {
			t.Fatalf("Login(web/b) error = %v", err)
		}
		mobile, err := mgr.Login(ctx, "concrete-kickout", "mobile", "a")
		if err != nil {
			t.Fatalf("Login(mobile/a) error = %v", err)
		}

		if err = mgr.KickoutByDeviceAndDeviceId(ctx, "concrete-kickout", "web", "a"); err != nil {
			t.Fatalf("KickoutByDeviceAndDeviceId() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, webA); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("web/a CheckLogin() error = %v, want ErrTokenKickout", err)
		}
		for name, token := range map[string]string{"web/b": webB, "mobile/a": mobile} {
			if err = mgr.CheckLogin(ctx, token); err != nil {
				t.Fatalf("%s CheckLogin() error = %v, want alive", name, err)
			}
		}
	})

	t.Run("replace device type replaces only that device", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		webA, err := mgr.Login(ctx, "device-replace", "web", "a")
		if err != nil {
			t.Fatalf("Login(web/a) error = %v", err)
		}
		webB, err := mgr.Login(ctx, "device-replace", "web", "b")
		if err != nil {
			t.Fatalf("Login(web/b) error = %v", err)
		}
		mobile, err := mgr.Login(ctx, "device-replace", "mobile", "a")
		if err != nil {
			t.Fatalf("Login(mobile/a) error = %v", err)
		}

		if err = mgr.ReplaceByDevice(ctx, "device-replace", "web"); err != nil {
			t.Fatalf("ReplaceByDevice() error = %v", err)
		}
		for _, token := range []string{webA, webB} {
			if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrTokenReplaced) {
				t.Fatalf("web token CheckLogin() error = %v, want ErrTokenReplaced", err)
			}
		}
		if err = mgr.CheckLogin(ctx, mobile); err != nil {
			t.Fatalf("mobile CheckLogin() error = %v, want alive", err)
		}
	})

	t.Run("logout concrete device removes only exact terminal", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		webA, err := mgr.Login(ctx, "concrete-logout", "web", "a")
		if err != nil {
			t.Fatalf("Login(web/a) error = %v", err)
		}
		webB, err := mgr.Login(ctx, "concrete-logout", "web", "b")
		if err != nil {
			t.Fatalf("Login(web/b) error = %v", err)
		}
		if err = mgr.LogoutByDeviceAndDeviceId(ctx, "concrete-logout", "web", "a"); err != nil {
			t.Fatalf("LogoutByDeviceAndDeviceId() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, webA); !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("web/a CheckLogin() error = %v, want ErrInvalidToken", err)
		}
		if latest, err := mgr.GetTokenValueByLoginID(ctx, "concrete-logout", "web"); err != nil || latest != webB {
			t.Fatalf("GetTokenValueByLoginID(web) = %q, %v, want %q", latest, err, webB)
		}
	})
}

func TestManagerForEachTerminalByDeviceStopsEarly(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
	})
	if _, err := mgr.Login(ctx, "foreach-device", "web", "a"); err != nil {
		t.Fatalf("Login(web/a) error = %v", err)
	}
	if _, err := mgr.Login(ctx, "foreach-device", "web", "b"); err != nil {
		t.Fatalf("Login(web/b) error = %v", err)
	}
	if _, err := mgr.Login(ctx, "foreach-device", "mobile", "a"); err != nil {
		t.Fatalf("Login(mobile/a) error = %v", err)
	}

	visited := 0
	err := mgr.ForEachTerminalByDevice(ctx, "foreach-device", "web", func(terminal TerminalInfo) bool {
		if terminal.Device != "web" {
			t.Fatalf("visited terminal device = %q, want web", terminal.Device)
		}
		visited++
		return false
	})
	if err != nil {
		t.Fatalf("ForEachTerminalByDevice() error = %v", err)
	}
	if visited != 1 {
		t.Fatalf("visited = %d, want early stop after one web terminal", visited)
	}
	if err = mgr.ForEachTerminalByDevice(ctx, "foreach-device", " ", func(TerminalInfo) bool { return true }); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("ForEachTerminalByDevice(empty device) error = %v, want ErrInvalidParam", err)
	}
}

func TestManagerPermissionAndRoleTokenCheckMatrix(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	token, err := mgr.Login(ctx, "access-matrix", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissionsByToken(ctx, token, []string{"order:read", "order:write", "user:*"}); err != nil {
		t.Fatalf("AddPermissionsByToken() error = %v", err)
	}
	if err = mgr.CheckPermissionAndByToken(ctx, token, []string{"order:read", "user:create"}); err != nil {
		t.Fatalf("CheckPermissionAndByToken() error = %v", err)
	}
	if err = mgr.CheckPermissionOrByToken(ctx, token, []string{"missing", "order:write"}); err != nil {
		t.Fatalf("CheckPermissionOrByToken() error = %v", err)
	}
	if !mgr.HasPermissionsOrByToken(ctx, token, []string{"missing", "user:update"}) {
		t.Fatal("HasPermissionsOrByToken() = false, want wildcard permission match")
	}
	if err = mgr.CheckPermissionAndByToken(ctx, token, []string{"order:read", "missing"}); !errors.Is(err, derror.ErrPermissionDenied) {
		t.Fatalf("CheckPermissionAndByToken(denied) error = %v, want ErrPermissionDenied", err)
	}
	if err = mgr.RemovePermissionsByToken(ctx, token, []string{"order:write"}); err != nil {
		t.Fatalf("RemovePermissionsByToken() error = %v", err)
	}
	if mgr.HasPermissionByToken(ctx, token, "order:write") {
		t.Fatal("HasPermissionByToken(order:write) = true after removal, want false")
	}

	if err = mgr.AddRolesByToken(ctx, token, []string{"admin", "auditor"}); err != nil {
		t.Fatalf("AddRolesByToken() error = %v", err)
	}
	if err = mgr.CheckRoleAndByToken(ctx, token, []string{"admin", "auditor"}); err != nil {
		t.Fatalf("CheckRoleAndByToken() error = %v", err)
	}
	if err = mgr.CheckRoleOrByToken(ctx, token, []string{"missing", "admin"}); err != nil {
		t.Fatalf("CheckRoleOrByToken() error = %v", err)
	}
	if !mgr.HasRolesOrByToken(ctx, token, []string{"missing", "auditor"}) {
		t.Fatal("HasRolesOrByToken() = false, want true")
	}
	if err = mgr.CheckRoleAndByToken(ctx, token, []string{"admin", "missing"}); !errors.Is(err, derror.ErrRoleDenied) {
		t.Fatalf("CheckRoleAndByToken(denied) error = %v, want ErrRoleDenied", err)
	}
	if err = mgr.RemoveRolesByToken(ctx, token, []string{"auditor"}); err != nil {
		t.Fatalf("RemoveRolesByToken() error = %v", err)
	}
	if mgr.HasRoleByToken(ctx, token, "auditor") {
		t.Fatal("HasRoleByToken(auditor) = true after removal, want false")
	}
}

func TestManagerTerminateDispatchesAccountAndDeviceScopes(t *testing.T) {
	ctx := context.Background()

	t.Run("logout by login id", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		first, err := mgr.Login(ctx, "terminate-logout-all", "web")
		if err != nil {
			t.Fatalf("Login(web) error = %v", err)
		}
		second, err := mgr.Login(ctx, "terminate-logout-all", "mobile")
		if err != nil {
			t.Fatalf("Login(mobile) error = %v", err)
		}
		if err = mgr.Terminate(ctx, TerminateOptions{LoginID: "terminate-logout-all", Action: TerminateActionLogout}); err != nil {
			t.Fatalf("Terminate(logout all) error = %v", err)
		}
		for _, token := range []string{first, second} {
			if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
				t.Fatalf("CheckLogin(%s) error = %v, want ErrInvalidToken", token, err)
			}
		}
	})

	t.Run("kickout by device type", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		web, err := mgr.Login(ctx, "terminate-kick-device", "web")
		if err != nil {
			t.Fatalf("Login(web) error = %v", err)
		}
		mobile, err := mgr.Login(ctx, "terminate-kick-device", "mobile")
		if err != nil {
			t.Fatalf("Login(mobile) error = %v", err)
		}
		if err = mgr.Terminate(ctx, TerminateOptions{LoginID: "terminate-kick-device", Device: "web", Action: TerminateActionKickout}); err != nil {
			t.Fatalf("Terminate(kickout device) error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, web); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("web CheckLogin() error = %v, want ErrTokenKickout", err)
		}
		if err = mgr.CheckLogin(ctx, mobile); err != nil {
			t.Fatalf("mobile CheckLogin() error = %v, want alive", err)
		}
	})

	t.Run("replace by concrete device", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		webA, err := mgr.Login(ctx, "terminate-replace-device-id", "web", "a")
		if err != nil {
			t.Fatalf("Login(web/a) error = %v", err)
		}
		webB, err := mgr.Login(ctx, "terminate-replace-device-id", "web", "b")
		if err != nil {
			t.Fatalf("Login(web/b) error = %v", err)
		}
		if err = mgr.Terminate(ctx, TerminateOptions{LoginID: "terminate-replace-device-id", Device: "web", DeviceID: "a", Action: TerminateActionReplace}); err != nil {
			t.Fatalf("Terminate(replace concrete device) error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, webA); !errors.Is(err, derror.ErrTokenReplaced) {
			t.Fatalf("web/a CheckLogin() error = %v, want ErrTokenReplaced", err)
		}
		if err = mgr.CheckLogin(ctx, webB); err != nil {
			t.Fatalf("web/b CheckLogin() error = %v, want alive", err)
		}
	})
}

func TestManagerIntrospectionInactiveBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	info, err := mgr.IntrospectToken(ctx, "")
	if err != nil {
		t.Fatalf("IntrospectToken(empty) error = %v", err)
	}
	if info.Active || info.Error != "invalid_token" {
		t.Fatalf("IntrospectToken(empty) = %+v, want inactive invalid_token", info)
	}

	token, err := mgr.Login(ctx, "inspect-inactive", "web")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.Kickout(ctx, token); err != nil {
		t.Fatalf("Kickout() error = %v", err)
	}
	info, err = mgr.IntrospectToken(ctx, token)
	if err != nil {
		t.Fatalf("IntrospectToken(kickout) error = %v", err)
	}
	if info.Active || info.Error == "" {
		t.Fatalf("IntrospectToken(kickout) = %+v, want inactive with error", info)
	}
}

func TestManagerGetTokenInfoStateSemantics(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		action  func(context.Context, *Manager, string) error
		wantErr error
	}{
		{name: "logout", action: func(ctx context.Context, mgr *Manager, token string) error { return mgr.Logout(ctx, token) }, wantErr: derror.ErrInvalidToken},
		{name: "kickout", action: func(ctx context.Context, mgr *Manager, token string) error { return mgr.Kickout(ctx, token) }, wantErr: derror.ErrTokenKickout},
		{name: "replace", action: func(ctx context.Context, mgr *Manager, token string) error { return mgr.Replace(ctx, token) }, wantErr: derror.ErrTokenReplaced},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			token, err := mgr.Login(ctx, "state-"+tt.name, "web")
			if err != nil {
				t.Fatalf("Login() error = %v", err)
			}
			if err = tt.action(ctx, mgr, token); err != nil {
				t.Fatalf("state action error = %v", err)
			}
			if _, err = mgr.GetTokenInfo(ctx, token); !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetTokenInfo() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerRefreshTokenRejectsDisabledAccountAndDevice(t *testing.T) {
	ctx := context.Background()

	t.Run("account disabled", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.RefreshTokenTimeout = 60
		})
		pair, err := mgr.LoginWithRefreshToken(ctx, "refresh-disabled-account", "web")
		if err != nil {
			t.Fatalf("LoginWithRefreshToken() error = %v", err)
		}
		if err = mgr.saveToStorage(ctx, mgr.getDisableKey("refresh-disabled-account"), DisableInfo{DisableTime: time.Now().Unix()}, time.Minute); err != nil {
			t.Fatalf("save disable marker error = %v", err)
		}
		if _, err = mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrAccountDisabled) {
			t.Fatalf("RefreshToken(disabled account) error = %v, want ErrAccountDisabled", err)
		}
	})

	t.Run("device disabled", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.RefreshTokenTimeout = 60
		})
		pair, err := mgr.LoginWithRefreshToken(ctx, "refresh-disabled-device", "web", "browser")
		if err != nil {
			t.Fatalf("LoginWithRefreshToken() error = %v", err)
		}
		info := DeviceDisableInfo{Device: "web", DeviceId: "browser", DisableTime: time.Now().Unix()}
		if err = mgr.saveToStorage(ctx, mgr.getDisableDeviceAndDeviceIdKey("refresh-disabled-device", "web", "browser"), info, time.Minute); err != nil {
			t.Fatalf("save device disable marker error = %v", err)
		}
		if _, err = mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrDeviceDisabled) {
			t.Fatalf("RefreshToken(disabled device) error = %v, want ErrDeviceDisabled", err)
		}
	})
}

func TestManagerSaveSessionWithMinTTLKeepsLongerTTL(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	key := mgr.getSessionKey("ttl-session")
	session := Session{AuthType: mgr.GetConfig().AuthType, LoginID: "ttl-session", CreateTime: time.Now().Unix()}

	if err := mgr.saveToStorage(ctx, key, session, time.Minute); err != nil {
		t.Fatalf("saveToStorage(initial) error = %v", err)
	}
	if err := mgr.saveSessionWithMinTTL(ctx, key, session, time.Second); err != nil {
		t.Fatalf("saveSessionWithMinTTL(shorter) error = %v", err)
	}
	ttl, err := requireManagerTestStorage(t, mgr).TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL() error = %v", err)
	}
	if ttl < 50*time.Second {
		t.Fatalf("session ttl after shorter save = %v, want longer existing ttl preserved", ttl)
	}

	if err = mgr.saveSessionWithMinTTL(ctx, key, session, 0); err != nil {
		t.Fatalf("saveSessionWithMinTTL(no expire) error = %v", err)
	}
	ttl, err = requireManagerTestStorage(t, mgr).TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL(no expire) error = %v", err)
	}
	if ttl != adapter.TTLNoExpire {
		t.Fatalf("session ttl after no-expire save = %v, want TTLNoExpire", ttl)
	}
}

func TestManagerEventPayloadsForCoreFlows(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.RefreshTokenTimeout = 60
	})

	var events []*listener.EventData
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
		copyData := *data
		events = append(events, &copyData)
	}, listener.ListenerConfig{Async: false})

	token, err := mgr.Login(ctx, "event-user", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissionsByToken(ctx, token, []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissionsByToken() error = %v", err)
	}
	if !mgr.HasPermissionByToken(ctx, token, "article:read") {
		t.Fatal("HasPermissionByToken() = false, want true")
	}
	if err = mgr.AddRolesByToken(ctx, token, []string{"editor"}); err != nil {
		t.Fatalf("AddRolesByToken() error = %v", err)
	}
	if !mgr.HasRoleByToken(ctx, token, "editor") {
		t.Fatal("HasRoleByToken() = false, want true")
	}
	pair, err := mgr.LoginWithRefreshToken(ctx, "event-refresh", "mobile", "phone")
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	if err = mgr.RevokeRefreshToken(ctx, pair.RefreshToken); err != nil {
		t.Fatalf("RevokeRefreshToken() error = %v", err)
	}
	if err = mgr.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	assertManagerEvent(t, events, listener.EventLogin, "event-user", "web", "browser", token, nil)
	assertManagerEvent(t, events, listener.EventPermissionChange, "event-user", "web", "browser", token, map[string]any{
		listener.ExtraKeyAction: listener.ActionAdd,
	})
	assertManagerEvent(t, events, listener.EventPermissionCheck, "event-user", "web", "browser", token, map[string]any{
		listener.ExtraKeyPermission: "article:read",
		listener.ExtraKeyResult:     true,
	})
	assertManagerEvent(t, events, listener.EventRoleChange, "event-user", "web", "browser", token, map[string]any{
		listener.ExtraKeyAction: listener.ActionAdd,
	})
	assertManagerEvent(t, events, listener.EventRoleCheck, "event-user", "web", "browser", token, map[string]any{
		listener.ExtraKeyRole:   "editor",
		listener.ExtraKeyResult: true,
	})
	assertManagerEvent(t, events, listener.EventRefreshTokenCreate, "event-refresh", "mobile", "phone", pair.AccessToken, map[string]any{
		listener.ExtraKeyAction:       listener.ActionCreate,
		listener.ExtraKeyRefreshToken: pair.RefreshToken,
	})
	assertManagerEvent(t, events, listener.EventRefreshTokenRevoke, "event-refresh", "mobile", "phone", pair.AccessToken, map[string]any{
		listener.ExtraKeyAction:       listener.ActionRevoke,
		listener.ExtraKeyRefreshToken: pair.RefreshToken,
	})
	assertManagerEvent(t, events, listener.EventLogout, "event-user", "web", "browser", token, nil)
}

func TestManagerSearchPaginationBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
	})

	for _, loginID := range []string{"page-a", "page-b", "page-c"} {
		if _, err := mgr.Login(ctx, loginID, "web"); err != nil {
			t.Fatalf("Login(%s) error = %v", loginID, err)
		}
	}

	sessions, err := mgr.SearchSessionId(ctx, "page-", -10, 2)
	if err != nil {
		t.Fatalf("SearchSessionId(negative start) error = %v", err)
	}
	if !sameStrings(sessions, []string{"page-a", "page-b"}) {
		t.Fatalf("sessions with negative start = %v, want first two sorted sessions", sessions)
	}
	sessions, err = mgr.SearchSessionId(ctx, "page-", 99, 2)
	if err != nil {
		t.Fatalf("SearchSessionId(out of range) error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions out of range = %v, want empty", sessions)
	}
	tokens, err := mgr.SearchTokenValue(ctx, "page-", 1, 1)
	if err != nil {
		t.Fatalf("SearchTokenValue(page) error = %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("paged token count = %d, want 1", len(tokens))
	}
}

func waitForManagerTest(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !condition() {
		t.Fatalf("condition was not satisfied within %s", timeout)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func assertManagerEvent(t *testing.T, events []*listener.EventData, event listener.Event, loginID, device, deviceID, token string, extra map[string]any) {
	t.Helper()
	for _, data := range events {
		if data.Event != event || data.LoginID != loginID || data.Device != device || data.DeviceId != deviceID || data.Token != token {
			continue
		}
		matched := true
		for key, want := range extra {
			if got := data.Extra[key]; got != want {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("event %s loginID=%s device=%s deviceID=%s token=%s extra=%v not found in %+v", event, loginID, device, deviceID, token, extra, events)
}
