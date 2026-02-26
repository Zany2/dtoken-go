// @Author daixk 2026/2/1 15:30:00
package banner

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"testing"
)

// TestPrintBanner_Full 测试完整配置的 Banner 打印
func TestPrintBanner_Full(t *testing.T) {
	t.Log("========== 测试完整配置的 Banner ==========")
	cfg := &config.Config{
		IsPrintBanner:    true,
		AuthType:         "login",
		TokenName:        "token",
		TokenStyle:       adapter.TokenStyleUUID,
		Timeout:          86400, // 1 天
		AutoRenew:        true,
		RenewMaxRefresh:  604800, // 7 天
		RenewInterval:    3600,   // 1 小时
		ActiveTimeout:    1800,   // 30 分钟
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

// TestPrintBanner_Simple 测试简单配置的 Banner 打印
func TestPrintBanner_Simple(t *testing.T) {
	t.Log("========== 测试简单配置的 Banner ==========")
	cfg := &config.Config{
		IsPrintBanner:    true,
		AuthType:         "admin:",
		TokenName:        "admin-token",
		TokenStyle:       adapter.TokenStyleSimple,
		Timeout:          7200, // 2 小时
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

// TestPrintBanner_JWT 测试 JWT 风格的 Banner 打印
func TestPrintBanner_JWT(t *testing.T) {
	t.Log("========== 测试 JWT 风格的 Banner ==========")
	cfg := &config.Config{
		IsPrintBanner:    true,
		AuthType:         "api:",
		TokenName:        "jwt-token",
		TokenStyle:       adapter.TokenStyleJWT,
		Timeout:          3600, // 1 小时
		AutoRenew:        true,
		RenewMaxRefresh:  86400, // 1 天
		RenewInterval:    1800,  // 30 分钟
		ActiveTimeout:    600,   // 10 分钟
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

// TestPrintBanner_Disabled 测试禁用 Banner 打印
func TestPrintBanner_Disabled(t *testing.T) {
	t.Log("========== 测试禁用 Banner（不应该有输出）==========")
	cfg := &config.Config{
		IsPrintBanner: false,
		AuthType:      "login:",
		TokenName:     "token",
	}
	PrintBanner(cfg)
	t.Log("========== 禁用 Banner 测试完成 ==========")
}

// TestPrintBanner_Nil 测试 nil 配置
func TestPrintBanner_Nil(t *testing.T) {
	t.Log("========== 测试 nil 配置（不应该有输出）==========")
	PrintBanner(nil)
	t.Log("========== nil 配置测试完成 ==========")
}
