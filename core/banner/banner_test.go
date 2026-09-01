// @Author daixk 2025/12/22 15:56:00
package banner

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
)

// TestGetTokenStyleName verifies all built-in and unknown token styles. TestGetTokenStyleName 验证所有内置及未知 Token 风格。
func TestGetTokenStyleName(t *testing.T) {
	tests := []struct {
		name  string
		style adapter.TokenStyle
		want  string
	}{
		{name: "uuid", style: adapter.TokenStyleUUID, want: "UUID"},
		{name: "simple", style: adapter.TokenStyleSimple, want: "Simple"},
		{name: "random32", style: adapter.TokenStyleRandom32, want: "Random-32"},
		{name: "random64", style: adapter.TokenStyleRandom64, want: "Random-64"},
		{name: "random128", style: adapter.TokenStyleRandom128, want: "Random-128"},
		{name: "jwt", style: adapter.TokenStyleJWT, want: "JWT"},
		{name: "hash", style: adapter.TokenStyleHash, want: "Hash"},
		{name: "timestamp", style: adapter.TokenStyleTimestamp, want: "Timestamp"},
		{name: "tik", style: adapter.TokenStyleTik, want: "Tik"},
		{name: "unknown", style: adapter.TokenStyle("custom"), want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTokenStyleName(tt.style); got != tt.want {
				t.Fatalf("getTokenStyleName(%q) = %q, want %q", tt.style, got, tt.want)
			}
		})
	}
}

// TestFormatDuration verifies sentinel, disabled, and human-readable duration formatting. TestFormatDuration 验证哨兵值、禁用值及可读时长格式化。
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds int64
		want    string
	}{
		{name: "unlimited", seconds: config.NoLimit, want: "No Limit"},
		{name: "negative", seconds: -2, want: "Disabled"},
		{name: "zero", seconds: 0, want: "Disabled"},
		{name: "seconds", seconds: 59, want: "59s"},
		{name: "exact minute", seconds: 60, want: "1m"},
		{name: "minute and seconds", seconds: 61, want: "1m 1s"},
		{name: "exact hour", seconds: 3600, want: "1h"},
		{name: "hour and minutes", seconds: 3660, want: "1h 1m"},
		{name: "exact day", seconds: 86400, want: "1d"},
		{name: "day and hours", seconds: 90000, want: "1d 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.seconds); got != tt.want {
				t.Fatalf("formatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

// TestFormatConcurrency verifies disabled, finite, and unlimited concurrency summaries. TestFormatConcurrency 验证禁用、有限及无限并发摘要。
func TestFormatConcurrency(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{
			name: "disabled",
			cfg: &config.Config{
				ConcurrencyScope:      config.ConcurrencyScopeDevice,
				ReplacedLoginExitMode: config.ReplacedLoginExitModeNewDevice,
			},
			want: "Disabled (Scope: device, Exit: new_device)",
		},
		{
			name: "unlimited",
			cfg: &config.Config{
				IsConcurrent:          true,
				ConcurrencyScope:      config.ConcurrencyScopeAccount,
				ReplacedLoginExitMode: config.ReplacedLoginExitModeOldDevice,
				IsShare:               true,
				MaxLoginCount:         config.NoLimit,
				OverflowLogoutMode:    config.LogoutModeKickout,
			},
			want: "Enabled, Scope: account, Exit: old_device, Share: Yes, Max: Unlimited, Overflow: kickout",
		},
		{
			name: "finite",
			cfg: &config.Config{
				IsConcurrent:          true,
				ConcurrencyScope:      config.ConcurrencyScopeDevice,
				ReplacedLoginExitMode: config.ReplacedLoginExitModeNewDevice,
				IsShare:               false,
				MaxLoginCount:         3,
				OverflowLogoutMode:    config.LogoutModeReplaced,
			},
			want: "Enabled, Scope: device, Exit: new_device, Share: No, Max: 3, Overflow: replaced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatConcurrency(tt.cfg); got != tt.want {
				t.Fatalf("formatConcurrency() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFormatTokenSource verifies source ordering and the no-source fallback. TestFormatTokenSource 验证来源顺序及无来源兜底文本。
func TestFormatTokenSource(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{name: "none", cfg: &config.Config{}, want: "None"},
		{name: "header", cfg: &config.Config{IsReadHeader: true}, want: "Header, Authorization Bearer"},
		{
			name: "all except header",
			cfg:  &config.Config{IsReadCookie: true, IsReadQuery: true, IsReadBody: true},
			want: "Cookie, Query, Body",
		},
		{
			name: "all",
			cfg:  &config.Config{IsReadHeader: true, IsReadCookie: true, IsReadQuery: true, IsReadBody: true},
			want: "Header, Authorization Bearer, Cookie, Query, Body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTokenSource(tt.cfg); got != tt.want {
				t.Fatalf("formatTokenSource() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFormatCookieConfig verifies nil, normalized domain, and complete cookie summaries. TestFormatCookieConfig 验证 nil、域名归一化及完整 Cookie 摘要。
func TestFormatCookieConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.CookieConfig
		want string
	}{
		{name: "nil", cfg: nil, want: "nil"},
		{
			name: "empty domain",
			cfg:  &config.CookieConfig{Path: "/", SameSite: config.SameSiteLax},
			want: "Path: /, Domain: <current-host>, Secure: No, HttpOnly: No, SameSite: Lax, MaxAge: Disabled",
		},
		{
			name: "complete",
			cfg: &config.CookieConfig{
				Domain:   " example.com ",
				Path:     "/api",
				Secure:   true,
				HttpOnly: true,
				SameSite: config.SameSiteNone,
				MaxAge:   3661,
			},
			want: "Path: /api, Domain: example.com, Secure: Yes, HttpOnly: Yes, SameSite: None, MaxAge: 1h 1m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCookieConfig(tt.cfg); got != tt.want {
				t.Fatalf("formatCookieConfig() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFormatBooleanAndSameSite verifies the small scalar formatters. TestFormatBooleanAndSameSite 验证布尔值和 SameSite 格式化函数。
func TestFormatBooleanAndSameSite(t *testing.T) {
	if got := formatEnabled(true); got != "Enabled" {
		t.Fatalf("formatEnabled(true) = %q, want %q", got, "Enabled")
	}
	if got := formatEnabled(false); got != "Disabled" {
		t.Fatalf("formatEnabled(false) = %q, want %q", got, "Disabled")
	}
	if got := formatYesNo(true); got != "Yes" {
		t.Fatalf("formatYesNo(true) = %q, want %q", got, "Yes")
	}
	if got := formatYesNo(false); got != "No" {
		t.Fatalf("formatYesNo(false) = %q, want %q", got, "No")
	}

	for _, tt := range []struct {
		mode config.SameSiteMode
		want string
	}{
		{mode: "", want: "Default"},
		{mode: config.SameSiteStrict, want: "Strict"},
		{mode: config.SameSiteLax, want: "Lax"},
		{mode: config.SameSiteNone, want: "None"},
		{mode: config.SameSiteMode("custom"), want: "custom"},
	} {
		if got := formatSameSite(tt.mode); got != tt.want {
			t.Fatalf("formatSameSite(%q) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

// TestPrintBannerOutput verifies enabled banner output contains key configuration fields. TestPrintBannerOutput 验证启用时 Banner 输出包含关键配置字段。
func TestPrintBannerOutput(t *testing.T) {
	cfg := &config.Config{
		IsPrintBanner:         true,
		AuthType:              "login:",
		KeyPrefix:             "dtoken:",
		TokenName:             "access-token",
		TokenStyle:            adapter.TokenStyleJWT,
		Timeout:               3600,
		RefreshTokenTimeout:   config.NoLimit,
		AutoRenew:             true,
		RenewMaxRefresh:       1800,
		RenewInterval:         600,
		ActiveTimeout:         0,
		IsConcurrent:          true,
		ConcurrencyScope:      config.ConcurrencyScopeAccount,
		IsShare:               true,
		MaxLoginCount:         config.NoLimit,
		ReplacedLoginExitMode: config.ReplacedLoginExitModeOldDevice,
		OverflowLogoutMode:    config.LogoutModeKickout,
		IsReadHeader:          true,
		IsReadCookie:          true,
		IsReadQuery:           true,
		IsReadBody:            false,
		IsLog:                 true,
		AsyncEvent:            false,
		CookieConfig: &config.CookieConfig{
			Path:     "/",
			SameSite: config.SameSiteLax,
		},
	}

	output := captureStdout(t, func() { PrintBanner(cfg) })
	for _, want := range []string{
		BannerText,
		":: DToken-Go ::        (version " + core.Version + ")",
		"AuthType         : login",
		"KeyPrefix        : dtoken",
		"TokenName        : access-token",
		"TokenStyle       : JWT",
		"Timeout          : 1h",
		"RefreshTokenTTL  : No Limit",
		"AutoRenew        : Enabled",
		"  - MaxRefresh   : 30m",
		"  - Interval     : 10m",
		"ActiveTimeout    : Disabled",
		"Concurrency      : Enabled, Scope: account, Exit: old_device, Share: Yes, Max: Unlimited, Overflow: kickout",
		"Token Source     : Header, Authorization Bearer, Cookie, Query",
		"Cookie           : Path: /, Domain: <current-host>, Secure: No, HttpOnly: No, SameSite: Lax, MaxAge: Disabled",
		"AsyncEvent       : Disabled",
		"Logging          : Enabled",
		"Started at: ",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("PrintBanner output missing %q\noutput:\n%s", want, output)
		}
	}
}

// TestPrintBannerDisabledFeatures verifies disabled feature branches in enabled banner output. TestPrintBannerDisabledFeatures 验证 Banner 启用但功能关闭时的分支输出。
func TestPrintBannerDisabledFeatures(t *testing.T) {
	cfg := &config.Config{
		IsPrintBanner:         true,
		AuthType:              "auth:",
		KeyPrefix:             "prefix:",
		TokenName:             "token",
		TokenStyle:            adapter.TokenStyleSimple,
		Timeout:               60,
		RefreshTokenTimeout:   60,
		ActiveTimeout:         config.NoLimit,
		ConcurrencyScope:      config.ConcurrencyScopeDevice,
		ReplacedLoginExitMode: config.ReplacedLoginExitModeNewDevice,
		IsReadHeader:          false,
		IsReadCookie:          false,
		IsReadQuery:           false,
		IsReadBody:            false,
		IsConcurrent:          false,
		IsLog:                 false,
		AsyncEvent:            false,
	}

	output := captureStdout(t, func() { PrintBanner(cfg) })
	for _, want := range []string{
		"TokenStyle       : Simple",
		"AutoRenew        : Disabled",
		"ActiveTimeout    : No Limit",
		"Concurrency      : Disabled (Scope: device, Exit: new_device)",
		"Token Source     : None",
		"AsyncEvent       : Disabled",
		"Logging          : Disabled",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("PrintBanner output missing %q\noutput:\n%s", want, output)
		}
	}
	if strings.Contains(output, "Cookie           :") {
		t.Fatalf("PrintBanner output contains cookie summary when cookie reading is disabled:\n%s", output)
	}
}

// TestPrintBannerSkipsNilAndDisabled verifies no output for skipped banner cases. TestPrintBannerSkipsNilAndDisabled 验证 nil 和禁用配置不会输出内容。
func TestPrintBannerSkipsNilAndDisabled(t *testing.T) {
	for _, tt := range []struct {
		name string
		cfg  *config.Config
	}{
		{name: "nil", cfg: nil},
		{name: "disabled", cfg: &config.Config{IsPrintBanner: false}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if output := captureStdout(t, func() { PrintBanner(tt.cfg) }); output != "" {
				t.Fatalf("PrintBanner() output = %q, want empty output", output)
			}
		})
	}
}

// captureStdout captures synchronous stdout writes for a single test. captureStdout 捕获单个测试中的同步标准输出。
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	previous := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = previous
		_ = reader.Close()
	}()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(output)
}
