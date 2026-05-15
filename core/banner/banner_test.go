// @Author daixk 2025/12/22 15:56:00
package banner

import (
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
)

// TestPrintBanner_Full tests full banner config TestPrintBanner_Full 测试完整配置的 Banner 打印
func TestPrintBanner_Full(t *testing.T) {
	cfg := &config.Config{
		IsPrintBanner:    true,
		AuthType:         "login",
		KeyPrefix:        "dtoken:",
		TokenName:        "token",
		TokenStyle:       adapter.TokenStyleUUID,
		Timeout:          86400,
		AutoRenew:        true,
		RenewMaxRefresh:  604800,
		RenewInterval:    3600,
		ActiveTimeout:    1800,
		IsConcurrent:     true,
		ConcurrencyScope: "user",
		IsShare:          false,
		MaxLoginCount:    5,
		IsReadHeader:     true,
		IsReadCookie:     false,
		IsReadBody:       false,
		IsLog:            true,
	}
	PrintBanner(cfg)
}

// TestPrintBanner_Simple tests simple banner config TestPrintBanner_Simple 测试简单配置的 Banner 打印
func TestPrintBanner_Simple(t *testing.T) {
	cfg := &config.Config{
		IsPrintBanner:    true,
		AuthType:         "admin:",
		KeyPrefix:        "admin:",
		TokenName:        "admin-token",
		TokenStyle:       adapter.TokenStyleSimple,
		Timeout:          7200,
		AutoRenew:        false,
		ActiveTimeout:    config.NoLimit,
		IsConcurrent:     false,
		ConcurrencyScope: "device",
		IsReadHeader:     true,
		IsReadCookie:     false,
		IsReadBody:       false,
		IsLog:            false,
	}
	PrintBanner(cfg)
}

// TestPrintBanner_JWT tests JWT banner config TestPrintBanner_JWT 测试 JWT 风格的 Banner 打印
func TestPrintBanner_JWT(t *testing.T) {
	cfg := &config.Config{
		IsPrintBanner:    true,
		AuthType:         "api:",
		KeyPrefix:        "api:",
		TokenName:        "jwt-token",
		TokenStyle:       adapter.TokenStyleJWT,
		Timeout:          3600,
		AutoRenew:        true,
		RenewMaxRefresh:  86400,
		RenewInterval:    1800,
		ActiveTimeout:    600,
		IsConcurrent:     true,
		ConcurrencyScope: "user",
		IsShare:          true,
		MaxLoginCount:    config.NoLimit,
		IsReadHeader:     true,
		IsReadCookie:     true,
		IsReadBody:       true,
		IsLog:            true,
	}
	PrintBanner(cfg)
}

// TestPrintBanner_Disabled tests disabled banner TestPrintBanner_Disabled 测试禁用 Banner 打印
func TestPrintBanner_Disabled(t *testing.T) {
	cfg := &config.Config{
		IsPrintBanner: false,
		AuthType:      "login:",
		TokenName:     "token",
	}
	PrintBanner(cfg)
}

// TestPrintBanner_Nil tests nil config TestPrintBanner_Nil 测试 nil 配置
func TestPrintBanner_Nil(t *testing.T) {
	PrintBanner(nil)
}
