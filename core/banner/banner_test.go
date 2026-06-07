// @Author daixk 2025/12/22 15:56:00
package banner

import (
	"fmt"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
)

// TestPrintBannerFull prints full banner config. TestPrintBannerFull 打印完整配置的 Banner。
func TestPrintBannerFull(t *testing.T) {
	fmt.Println("===== Full Banner =====")
	cfg := &config.Config{
		IsPrintBanner:         true,
		AuthType:              "login",
		KeyPrefix:             "dtoken:",
		TokenName:             "token",
		TokenStyle:            adapter.TokenStyleUUID,
		Timeout:               86400,
		RefreshTokenTimeout:   604800,
		AutoRenew:             true,
		RenewMaxRefresh:       43200,
		RenewInterval:         3600,
		ActiveTimeout:         1800,
		IsConcurrent:          true,
		ConcurrencyScope:      config.ConcurrencyScopeAccount,
		IsShare:               false,
		MaxLoginCount:         5,
		ReplacedLoginExitMode: config.ReplacedLoginExitModeOldDevice,
		OverflowLogoutMode:    config.LogoutModeKickout,
		IsReadHeader:          true,
		IsReadCookie:          true,
		IsReadQuery:           true,
		IsReadBody:            true,
		IsLog:                 true,
		AsyncEvent:            true,
		CookieConfig:          config.DefaultCookieConfig(),
	}
	PrintBanner(cfg)
}

// TestPrintBannerSimple prints compact disabled state config. TestPrintBannerSimple 打印精简禁用状态配置。
func TestPrintBannerSimple(t *testing.T) {
	fmt.Println("===== Simple Banner =====")
	cfg := &config.Config{
		IsPrintBanner:         true,
		AuthType:              "admin:",
		KeyPrefix:             "admin:",
		TokenName:             "admin-token",
		TokenStyle:            adapter.TokenStyleSimple,
		Timeout:               7200,
		RefreshTokenTimeout:   config.NoLimit,
		AutoRenew:             false,
		ActiveTimeout:         config.NoLimit,
		IsConcurrent:          false,
		ConcurrencyScope:      config.ConcurrencyScopeDevice,
		ReplacedLoginExitMode: config.ReplacedLoginExitModeNewDevice,
		IsReadHeader:          true,
		IsReadCookie:          false,
		IsReadBody:            false,
		IsLog:                 false,
		AsyncEvent:            false,
	}
	PrintBanner(cfg)
}

// TestPrintBannerJWT prints JWT banner config. TestPrintBannerJWT 打印 JWT 风格的 Banner。
func TestPrintBannerJWT(t *testing.T) {
	fmt.Println("===== JWT Banner =====")
	cfg := &config.Config{
		IsPrintBanner:         true,
		AuthType:              "api:",
		KeyPrefix:             "api:",
		TokenName:             "jwt-token",
		TokenStyle:            adapter.TokenStyleJWT,
		Timeout:               3600,
		RefreshTokenTimeout:   86400,
		AutoRenew:             true,
		RenewMaxRefresh:       1800,
		RenewInterval:         600,
		ActiveTimeout:         600,
		IsConcurrent:          true,
		ConcurrencyScope:      config.ConcurrencyScopeAccount,
		IsShare:               true,
		MaxLoginCount:         config.NoLimit,
		ReplacedLoginExitMode: config.ReplacedLoginExitModeOldDevice,
		OverflowLogoutMode:    config.LogoutModeReplaced,
		IsReadHeader:          true,
		IsReadCookie:          false,
		IsReadBody:            true,
		IsLog:                 true,
		AsyncEvent:            true,
	}
	PrintBanner(cfg)
}

// TestPrintBannerDisabled prints disabled banner config. TestPrintBannerDisabled 打印禁用 Banner 场景。
func TestPrintBannerDisabled(t *testing.T) {
	fmt.Println("===== Disabled Banner =====")
	cfg := &config.Config{
		IsPrintBanner: false,
		AuthType:      "login:",
		TokenName:     "token",
	}
	PrintBanner(cfg)
	fmt.Println("disabled banner should print nothing above this line except the title")
}

// TestPrintBannerNil prints nil config scenario. TestPrintBannerNil 打印 nil 配置场景。
func TestPrintBannerNil(t *testing.T) {
	fmt.Println("===== Nil Banner =====")
	PrintBanner(nil)
	fmt.Println("nil banner should print nothing above this line except the title")
}

// TestFormatCookieConfigNil prints nil cookie config formatting. TestFormatCookieConfigNil 打印空 Cookie 配置格式化结果。
func TestFormatCookieConfigNil(t *testing.T) {
	fmt.Printf("nil cookie config: %s\n", formatCookieConfig(nil))
}
