package config

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// TestValidateAcceptsSupportedTokenStyleMatrix verifies every supported token style. TestValidateAcceptsSupportedTokenStyleMatrix 验证所有支持的 Token 风格。
func TestValidateAcceptsSupportedTokenStyleMatrix(t *testing.T) {
	styles := []adapter.TokenStyle{
		adapter.TokenStyleUUID,
		adapter.TokenStyleSimple,
		adapter.TokenStyleRandom32,
		adapter.TokenStyleRandom64,
		adapter.TokenStyleRandom128,
		adapter.TokenStyleJWT,
		adapter.TokenStyleHash,
		adapter.TokenStyleTimestamp,
		adapter.TokenStyleTik,
	}

	for _, style := range styles {
		t.Run(string(style), func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.TokenStyle = style
			if style == adapter.TokenStyleJWT {
				cfg.JwtSecretKey = "jwt-secret"
			}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

// TestValidateTokenStyleAndJWTSecretMatrix verifies JWT secret is required only for JWT style. TestValidateTokenStyleAndJWTSecretMatrix 验证仅 JWT 风格需要密钥。
func TestValidateTokenStyleAndJWTSecretMatrix(t *testing.T) {
	tests := []struct {
		name    string
		style   adapter.TokenStyle
		secret  string
		wantErr bool
	}{
		{name: "jwt with secret", style: adapter.TokenStyleJWT, secret: "secret"},
		{name: "jwt with blank secret", style: adapter.TokenStyleJWT, secret: "   ", wantErr: true},
		{name: "uuid with blank secret", style: adapter.TokenStyleUUID, secret: "   "},
		{name: "hash with blank secret", style: adapter.TokenStyleHash, secret: "   "},
		{name: "invalid style", style: adapter.TokenStyle("bad-style"), secret: "secret", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.TokenStyle = tt.style
			cfg.JwtSecretKey = tt.secret

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

// TestValidateAcceptsConcurrencyPolicyMatrix verifies all concurrency policy enum combinations. TestValidateAcceptsConcurrencyPolicyMatrix 验证并发策略枚举组合。
func TestValidateAcceptsConcurrencyPolicyMatrix(t *testing.T) {
	scopes := []ConcurrencyScope{ConcurrencyScopeAccount, ConcurrencyScopeDevice}
	concurrentModes := []bool{false, true}
	shareModes := []bool{false, true}
	maxLoginCounts := []int64{NoLimit, 1, DefaultMaxLoginCount}
	replacedModes := []ReplacedLoginExitMode{ReplacedLoginExitModeOldDevice, ReplacedLoginExitModeNewDevice}
	overflowModes := []LogoutMode{LogoutModeLogout, LogoutModeKickout, LogoutModeReplaced}

	for _, scope := range scopes {
		for _, isConcurrent := range concurrentModes {
			for _, isShare := range shareModes {
				for _, maxLoginCount := range maxLoginCounts {
					for _, replacedMode := range replacedModes {
						for _, overflowMode := range overflowModes {
							name := strings.Join([]string{
								"scope=" + string(scope),
								"concurrent=" + boolName(isConcurrent),
								"share=" + boolName(isShare),
								"max=" + int64Name(maxLoginCount),
								"replaced=" + string(replacedMode),
								"overflow=" + string(overflowMode),
							}, "/")
							t.Run(name, func(t *testing.T) {
								cfg := DefaultConfig()
								cfg.ConcurrencyScope = scope
								cfg.IsConcurrent = isConcurrent
								cfg.IsShare = isShare
								cfg.MaxLoginCount = maxLoginCount
								cfg.ReplacedLoginExitMode = replacedMode
								cfg.OverflowLogoutMode = overflowMode

								if err := cfg.Validate(); err != nil {
									t.Fatalf("Validate() error = %v", err)
								}
							})
						}
					}
				}
			}
		}
	}
}

// TestValidateRejectsInvalidConcurrencyPolicyMatrix verifies invalid concurrency enum and count values. TestValidateRejectsInvalidConcurrencyPolicyMatrix 验证非法并发策略配置。
func TestValidateRejectsInvalidConcurrencyPolicyMatrix(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(cfg *Config)
	}{
		{name: "invalid concurrency scope", mutate: func(cfg *Config) { cfg.ConcurrencyScope = ConcurrencyScope("bad") }},
		{name: "invalid replaced mode", mutate: func(cfg *Config) { cfg.ReplacedLoginExitMode = ReplacedLoginExitMode("bad") }},
		{name: "invalid overflow mode", mutate: func(cfg *Config) { cfg.OverflowLogoutMode = LogoutMode("bad") }},
		{name: "zero max login count", mutate: func(cfg *Config) { cfg.MaxLoginCount = 0 }},
		{name: "less than no limit max login count", mutate: func(cfg *Config) { cfg.MaxLoginCount = -2 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)

			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

// TestValidateAcceptsTokenSourceMatrix verifies any enabled token source combination is valid. TestValidateAcceptsTokenSourceMatrix 验证任意非空 Token 来源组合。
func TestValidateAcceptsTokenSourceMatrix(t *testing.T) {
	for _, readBody := range []bool{false, true} {
		for _, readQuery := range []bool{false, true} {
			for _, readHeader := range []bool{false, true} {
				for _, readCookie := range []bool{false, true} {
					name := strings.Join([]string{
						"body=" + boolName(readBody),
						"query=" + boolName(readQuery),
						"header=" + boolName(readHeader),
						"cookie=" + boolName(readCookie),
					}, "/")
					t.Run(name, func(t *testing.T) {
						cfg := DefaultConfig()
						cfg.IsReadBody = readBody
						cfg.IsReadQuery = readQuery
						cfg.IsReadHeader = readHeader
						cfg.IsReadCookie = readCookie

						err := cfg.Validate()
						if !readBody && !readQuery && !readHeader && !readCookie {
							if err == nil {
								t.Fatal("Validate() error = nil, want no token source error")
							}
							return
						}
						if err != nil {
							t.Fatalf("Validate() error = %v", err)
						}
					})
				}
			}
		}
	}
}

// TestValidateCookieConfigMatrix verifies cookie read and SameSite/Secure combinations. TestValidateCookieConfigMatrix 验证 Cookie 读取和 SameSite/Secure 组合。
func TestValidateCookieConfigMatrix(t *testing.T) {
	tests := []struct {
		name       string
		readCookie bool
		cookie     *CookieConfig
		wantErr    bool
	}{
		{name: "cookie disabled nil config", cookie: nil},
		{name: "cookie enabled nil config", readCookie: true, cookie: nil, wantErr: true},
		{name: "lax insecure", readCookie: true, cookie: &CookieConfig{Path: "/", SameSite: SameSiteLax}},
		{name: "strict insecure", readCookie: true, cookie: &CookieConfig{Path: "/", SameSite: SameSiteStrict}},
		{name: "none secure", readCookie: true, cookie: &CookieConfig{Path: "/", SameSite: SameSiteNone, Secure: true}},
		{name: "none insecure", readCookie: true, cookie: &CookieConfig{Path: "/", SameSite: SameSiteNone}, wantErr: true},
		{name: "empty same site", readCookie: true, cookie: &CookieConfig{Path: "/"}},
		{name: "valid max age", readCookie: true, cookie: &CookieConfig{Path: "/", SameSite: SameSiteLax, MaxAge: 3600}},
		{name: "negative max age", readCookie: true, cookie: &CookieConfig{Path: "/", SameSite: SameSiteLax, MaxAge: -1}, wantErr: true},
		{name: "path without slash", readCookie: true, cookie: &CookieConfig{Path: "api", SameSite: SameSiteLax}, wantErr: true},
		{name: "blank path", readCookie: true, cookie: &CookieConfig{Path: "   ", SameSite: SameSiteLax}, wantErr: true},
		{name: "domain with whitespace", readCookie: true, cookie: &CookieConfig{Domain: "example .com", Path: "/", SameSite: SameSiteLax}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.IsReadCookie = tt.readCookie
			cfg.CookieConfig = tt.cookie

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

// TestValidateNumericBoundaryMatrix verifies no-limit and positive numeric boundaries. TestValidateNumericBoundaryMatrix 验证无限制和正数边界。
func TestValidateNumericBoundaryMatrix(t *testing.T) {
	validValues := []int64{NoLimit, 1, 60}
	fields := []struct {
		name string
		set  func(cfg *Config, value int64)
	}{
		{name: "Timeout", set: func(cfg *Config, value int64) { cfg.Timeout = value }},
		{name: "RefreshTokenTimeout", set: func(cfg *Config, value int64) { cfg.RefreshTokenTimeout = value }},
		{name: "RenewMaxRefresh", set: func(cfg *Config, value int64) { cfg.RenewMaxRefresh = value }},
		{name: "RenewInterval", set: func(cfg *Config, value int64) { cfg.RenewInterval = value }},
		{name: "ActiveTimeout", set: func(cfg *Config, value int64) { cfg.ActiveTimeout = value }},
		{name: "MaxLoginCount", set: func(cfg *Config, value int64) { cfg.MaxLoginCount = value }},
	}

	for _, field := range fields {
		for _, value := range validValues {
			t.Run(field.name+"/valid/"+int64Name(value), func(t *testing.T) {
				cfg := DefaultConfig()
				cfg.AutoRenew = false
				field.set(cfg, value)

				if err := cfg.Validate(); err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
			})
		}

		for _, value := range []int64{-2, 0} {
			t.Run(field.name+"/invalid/"+int64Name(value), func(t *testing.T) {
				cfg := DefaultConfig()
				cfg.AutoRenew = false
				field.set(cfg, value)

				if err := cfg.Validate(); err == nil {
					t.Fatal("Validate() error = nil, want numeric boundary error")
				}
			})
		}
	}
}

// TestValidateAutoRenewRelationMatrix verifies auto-renew timing relations. TestValidateAutoRenewRelationMatrix 验证自动续期时间关系组合。
func TestValidateAutoRenewRelationMatrix(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(cfg *Config)
		wantErr bool
	}{
		{name: "disabled allows unlimited timeout", mutate: func(cfg *Config) {
			cfg.AutoRenew = false
			cfg.Timeout = NoLimit
		}},
		{name: "enabled rejects unlimited timeout", mutate: func(cfg *Config) {
			cfg.AutoRenew = true
			cfg.Timeout = NoLimit
		}, wantErr: true},
		{name: "enabled accepts unlimited refresh threshold", mutate: func(cfg *Config) {
			cfg.AutoRenew = true
			cfg.Timeout = 60
			cfg.RenewMaxRefresh = NoLimit
		}},
		{name: "enabled rejects refresh threshold above timeout", mutate: func(cfg *Config) {
			cfg.AutoRenew = true
			cfg.Timeout = 60
			cfg.RenewMaxRefresh = 61
		}, wantErr: true},
		{name: "enabled accepts renew interval below timeout", mutate: func(cfg *Config) {
			cfg.AutoRenew = true
			cfg.Timeout = 60
			cfg.RenewMaxRefresh = 30
			cfg.RenewInterval = 30
		}},
		{name: "enabled rejects renew interval equal timeout", mutate: func(cfg *Config) {
			cfg.AutoRenew = true
			cfg.Timeout = 60
			cfg.RenewInterval = 60
		}, wantErr: true},
		{name: "enabled accepts unlimited renew interval", mutate: func(cfg *Config) {
			cfg.AutoRenew = true
			cfg.Timeout = 60
			cfg.RenewMaxRefresh = 30
			cfg.RenewInterval = NoLimit
		}},
		{name: "enabled accepts active timeout no limit", mutate: func(cfg *Config) {
			cfg.AutoRenew = true
			cfg.Timeout = 60
			cfg.RenewMaxRefresh = 30
			cfg.RenewInterval = 30
			cfg.ActiveTimeout = NoLimit
		}},
		{name: "enabled rejects renew interval equal active timeout", mutate: func(cfg *Config) {
			cfg.AutoRenew = true
			cfg.Timeout = 60
			cfg.RenewInterval = 30
			cfg.ActiveTimeout = 30
		}, wantErr: true},
		{name: "disabled allows relation otherwise invalid", mutate: func(cfg *Config) {
			cfg.AutoRenew = false
			cfg.Timeout = 60
			cfg.RenewMaxRefresh = 61
			cfg.RenewInterval = 60
			cfg.ActiveTimeout = 30
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("Validate() error = nil, want auto-renew relation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

// TestValidateNameAndNamespaceBoundaryMatrix verifies name and namespace boundaries. TestValidateNameAndNamespaceBoundaryMatrix 验证名称和命名空间边界。
func TestValidateNameAndNamespaceBoundaryMatrix(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(cfg *Config)
		afterFunc func(t *testing.T, cfg *Config)
		wantErr   bool
	}{
		{name: "auth type trims and appends separator", mutate: func(cfg *Config) { cfg.AuthType = " user " }, afterFunc: func(t *testing.T, cfg *Config) {
			t.Helper()
			if cfg.AuthType != "user:" {
				t.Fatalf("AuthType = %q, want user:", cfg.AuthType)
			}
		}},
		{name: "key prefix trims and appends separator", mutate: func(cfg *Config) { cfg.KeyPrefix = " app " }, afterFunc: func(t *testing.T, cfg *Config) {
			t.Helper()
			if cfg.KeyPrefix != "app:" {
				t.Fatalf("KeyPrefix = %q, want app:", cfg.KeyPrefix)
			}
		}},
		{name: "token name max length", mutate: func(cfg *Config) { cfg.TokenName = strings.Repeat("a", 64) }},
		{name: "auth type max length with separator", mutate: func(cfg *Config) { cfg.AuthType = strings.Repeat("a", 63) }},
		{name: "key prefix max length with separator", mutate: func(cfg *Config) { cfg.KeyPrefix = strings.Repeat("a", 63) }},
		{name: "token name too long", mutate: func(cfg *Config) { cfg.TokenName = strings.Repeat("a", 65) }, wantErr: true},
		{name: "auth type too long after normalization", mutate: func(cfg *Config) { cfg.AuthType = strings.Repeat("a", 64) }, wantErr: true},
		{name: "key prefix too long after normalization", mutate: func(cfg *Config) { cfg.KeyPrefix = strings.Repeat("a", 64) }, wantErr: true},
		{name: "token name whitespace", mutate: func(cfg *Config) { cfg.TokenName = "dt token" }, wantErr: true},
		{name: "auth type whitespace", mutate: func(cfg *Config) { cfg.AuthType = "auth type" }, wantErr: true},
		{name: "key prefix whitespace", mutate: func(cfg *Config) { cfg.KeyPrefix = "key prefix" }, wantErr: true},
		{name: "auth type separator only", mutate: func(cfg *Config) { cfg.AuthType = "::" }, wantErr: true},
		{name: "key prefix separator only", mutate: func(cfg *Config) { cfg.KeyPrefix = "::" }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("Validate() error = nil, want name or namespace error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if !tt.wantErr && tt.afterFunc != nil {
				tt.afterFunc(t, cfg)
			}
		})
	}
}

// TestConfigSetterChainMatrix verifies setters can be chained and preserve values. TestConfigSetterChainMatrix 验证 setter 可链式设置并保留配置。
func TestConfigSetterChainMatrix(t *testing.T) {
	cookie := &CookieConfig{
		Domain:   "example.com",
		Path:     "/api",
		Secure:   true,
		HttpOnly: true,
		SameSite: SameSiteNone,
		MaxAge:   3600,
	}
	cfg := DefaultConfig().
		SetAuthType("admin").
		SetKeyPrefix("iam").
		SetTokenName("access_token").
		SetTimeout(120).
		SetRefreshTokenTimeout(240).
		SetRenewMaxRefresh(60).
		SetRenewInterval(30).
		SetActiveTimeout(90).
		SetIsConcurrent(false).
		SetIsShare(false).
		SetMaxLoginCount(3).
		SetReplacedLoginExitMode(ReplacedLoginExitModeNewDevice).
		SetOverflowLogoutMode(LogoutModeReplaced).
		SetIsReadBody(true).
		SetIsReadQuery(true).
		SetIsReadHeader(true).
		SetIsReadCookie(true).
		SetTokenStyle(adapter.TokenStyleJWT).
		SetJwtSecretKey("secret").
		SetAutoRenew(true).
		SetIsLog(true).
		SetIsPrintBanner(false).
		SetAsyncEvent(false).
		SetCookieConfig(cookie)

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.AuthType != "admin:" || cfg.KeyPrefix != "iam:" {
		t.Fatalf("namespaces = %q %q, want admin: iam:", cfg.AuthType, cfg.KeyPrefix)
	}
	if cfg.TokenName != "access_token" ||
		cfg.Timeout != 120 ||
		cfg.RefreshTokenTimeout != 240 ||
		cfg.RenewMaxRefresh != 60 ||
		cfg.RenewInterval != 30 ||
		cfg.ActiveTimeout != 90 ||
		cfg.IsConcurrent ||
		cfg.IsShare ||
		cfg.MaxLoginCount != 3 ||
		cfg.ReplacedLoginExitMode != ReplacedLoginExitModeNewDevice ||
		cfg.OverflowLogoutMode != LogoutModeReplaced ||
		!cfg.IsReadBody ||
		!cfg.IsReadQuery ||
		!cfg.IsReadHeader ||
		!cfg.IsReadCookie ||
		cfg.TokenStyle != adapter.TokenStyleJWT ||
		cfg.JwtSecretKey != "secret" ||
		!cfg.AutoRenew ||
		!cfg.IsLog ||
		cfg.IsPrintBanner ||
		cfg.AsyncEvent ||
		cfg.CookieConfig != cookie {
		t.Fatalf("setter chain config = %+v", cfg)
	}
}

func boolName(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func int64Name(v int64) string {
	if v == NoLimit {
		return "nolimit"
	}
	return strconv.FormatInt(v, 10)
}
