// @Author daixk 2025/12/22 15:56:00
package banner

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/core"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
)

const (
	// BannerText stores banner text content BannerText 存储 Banner 文本内容
	BannerText = `
 ____  _____     _
|  _ \|_   _|__ | | _____ _ __
| | | | | |/ _ \| |/ / _ \ '_ \
| |_| | | | (_) |   <  __/ | | |
|____/  |_|\___/|_|\_\___|_| |_|

`
)

// PrintBanner prints startup banner and key config info PrintBanner 打印启动 Banner 和关键配置信息
func PrintBanner(cfg *config.Config) {
	// Skip when config is missing or banner is disabled 配置缺失或禁用 Banner 时跳过
	if cfg == nil || !cfg.IsPrintBanner {
		return
	}

	// Print banner title 打印 Banner 标题
	fmt.Print(BannerText)
	fmt.Printf(":: DToken-Go ::        (version %s)\n", core.Version)
	fmt.Printf(":: Go Version ::       (%s)\n\n", runtime.Version())

	// Print section header 打印配置摘要标题
	fmt.Println("========================================")
	fmt.Println("         Configuration Summary          ")
	fmt.Println("========================================")

	// Print identity and token settings 打印认证与 Token 配置
	fmt.Printf("AuthType         : %s\n", strings.TrimSuffix(cfg.AuthType, ":"))
	fmt.Printf("KeyPrefix        : %s\n", strings.TrimSuffix(cfg.KeyPrefix, ":"))
	fmt.Printf("TokenName        : %s\n", cfg.TokenName)
	fmt.Printf("TokenStyle       : %s\n", getTokenStyleName(cfg.TokenStyle))

	// Print timeout settings 打印超时配置
	fmt.Printf("Timeout          : %s\n", formatDuration(cfg.Timeout))
	fmt.Printf("RefreshTokenTTL  : %s\n", formatDuration(cfg.RefreshTokenTimeout))
	if cfg.AutoRenew {
		fmt.Printf("AutoRenew        : Enabled\n")
		fmt.Printf("  - MaxRefresh   : %s\n", formatDuration(cfg.RenewMaxRefresh))
		fmt.Printf("  - Interval     : %s\n", formatDuration(cfg.RenewInterval))
	} else {
		fmt.Printf("AutoRenew        : Disabled\n")
	}
	fmt.Printf("ActiveTimeout    : %s\n", formatDuration(cfg.ActiveTimeout))

	// Print policy and source settings 打印策略与 Token 来源配置
	fmt.Printf("Concurrency      : %s\n", formatConcurrency(cfg))
	fmt.Printf("Token Source     : %s\n", formatTokenSource(cfg))
	if cfg.IsReadCookie {
		fmt.Printf("Cookie           : %s\n", formatCookieConfig(cfg.CookieConfig))
	}
	fmt.Printf("AsyncEvent       : %s\n", formatEnabled(cfg.AsyncEvent))
	fmt.Printf("Logging          : %s\n", formatEnabled(cfg.IsLog))

	// Print startup time 打印启动时间
	fmt.Println("========================================")
	fmt.Printf("Started at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("========================================")
	fmt.Println()
}

// getTokenStyleName gets token style name getTokenStyleName 获取 Token 风格名称
func getTokenStyleName(style adapter.TokenStyle) string {
	// Match known token styles 匹配已知 Token 风格
	switch style {
	case adapter.TokenStyleUUID:
		return "UUID"
	case adapter.TokenStyleSimple:
		return "Simple"
	case adapter.TokenStyleRandom32:
		return "Random-32"
	case adapter.TokenStyleRandom64:
		return "Random-64"
	case adapter.TokenStyleRandom128:
		return "Random-128"
	case adapter.TokenStyleJWT:
		return "JWT"
	case adapter.TokenStyleHash:
		return "Hash"
	case adapter.TokenStyleTimestamp:
		return "Timestamp"
	case adapter.TokenStyleTik:
		return "Tik"
	default:
		// Fallback for unsupported styles 不支持的风格使用兜底展示
		return "Unknown"
	}
}

// formatDuration formats duration display formatDuration 格式化时长显示
func formatDuration(seconds int64) string {
	// Format unlimited duration 格式化无限制时长
	if seconds == config.NoLimit {
		return "No Limit"
	}

	// Format disabled duration 格式化禁用时长
	if seconds <= 0 {
		return "Disabled"
	}

	// Convert seconds to duration 转换秒数为时长
	d := time.Duration(seconds) * time.Second

	// Format day-level duration 格式化天级时长
	if d >= 24*time.Hour {
		days := d / (24 * time.Hour)
		hours := (d % (24 * time.Hour)) / time.Hour
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}

	// Format hour-level duration 格式化小时级时长
	if d >= time.Hour {
		hours := d / time.Hour
		minutes := (d % time.Hour) / time.Minute
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}

	// Format minute-level duration 格式化分钟级时长
	if d >= time.Minute {
		minutes := d / time.Minute
		seconds := (d % time.Minute) / time.Second
		if seconds > 0 {
			return fmt.Sprintf("%dm %ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	// Format second-level duration 格式化秒级时长
	return fmt.Sprintf("%ds", seconds)
}

// formatConcurrency formats concurrency config formatConcurrency 格式化并发配置
func formatConcurrency(cfg *config.Config) string {
	// Format disabled concurrency 格式化禁用并发
	if !cfg.IsConcurrent {
		return fmt.Sprintf("Disabled (Scope: %s, Exit: %s)", cfg.ConcurrencyScope, cfg.ReplacedLoginExitMode)
	}

	// Collect enabled concurrency parts 收集启用并发时的展示片段
	var parts []string
	parts = append(parts, "Enabled")
	parts = append(parts, fmt.Sprintf("Scope: %s", cfg.ConcurrencyScope))
	parts = append(parts, fmt.Sprintf("Exit: %s", cfg.ReplacedLoginExitMode))
	parts = append(parts, fmt.Sprintf("Share: %s", formatYesNo(cfg.IsShare)))

	// Format max login count 格式化最大登录数量
	if cfg.MaxLoginCount == config.NoLimit {
		parts = append(parts, "Max: Unlimited")
	} else {
		parts = append(parts, fmt.Sprintf("Max: %d", cfg.MaxLoginCount))
	}
	parts = append(parts, fmt.Sprintf("Overflow: %s", cfg.OverflowLogoutMode))

	// Join concurrency summary 拼接并发摘要
	return strings.Join(parts, ", ")
}

// formatTokenSource formats token source display formatTokenSource 格式化 Token 读取来源
func formatTokenSource(cfg *config.Config) string {
	// Collect enabled token sources 收集启用的 Token 来源
	var sources []string
	if cfg.IsReadHeader {
		sources = append(sources, "Header")
		sources = append(sources, "Authorization Bearer")
	}
	if cfg.IsReadCookie {
		sources = append(sources, "Cookie")
	}
	if cfg.IsReadQuery {
		sources = append(sources, "Query")
	}
	if cfg.IsReadBody {
		sources = append(sources, "Body")
	}

	// Return none when all sources are disabled 全部来源禁用时返回 None
	if len(sources) == 0 {
		return "None"
	}

	// Join token source summary 拼接 Token 来源摘要
	return strings.Join(sources, ", ")
}

// formatCookieConfig formats cookie options formatCookieConfig 格式化 Cookie 配置
func formatCookieConfig(cfg *config.CookieConfig) string {
	// Format missing cookie config 格式化缺失的 Cookie 配置
	if cfg == nil {
		return "nil"
	}

	// Normalize empty domain display 规范化空域名展示
	domain := strings.TrimSpace(cfg.Domain)
	if domain == "" {
		domain = "<current-host>"
	}

	// Collect cookie option parts 收集 Cookie 配置片段
	parts := []string{
		fmt.Sprintf("Path: %s", cfg.Path),
		fmt.Sprintf("Domain: %s", domain),
		fmt.Sprintf("Secure: %s", formatYesNo(cfg.Secure)),
		fmt.Sprintf("HttpOnly: %s", formatYesNo(cfg.HttpOnly)),
		fmt.Sprintf("SameSite: %s", formatSameSite(cfg.SameSite)),
		fmt.Sprintf("MaxAge: %s", formatDuration(cfg.MaxAge)),
	}

	// Join cookie option summary 拼接 Cookie 配置摘要
	return strings.Join(parts, ", ")
}

// formatEnabled formats bool as enabled text formatEnabled 格式化启用状态
func formatEnabled(value bool) string {
	// Convert bool to enabled text 转换布尔值为启用文本
	if value {
		return "Enabled"
	}
	return "Disabled"
}

// formatYesNo formats bool as yes/no text formatYesNo 格式化布尔值
func formatYesNo(value bool) string {
	// Convert bool to yes/no text 转换布尔值为 Yes/No 文本
	if value {
		return "Yes"
	}
	return "No"
}

// formatSameSite formats cookie sameSite mode formatSameSite 格式化 Cookie SameSite 模式
func formatSameSite(mode config.SameSiteMode) string {
	// Use default label when mode is empty 模式为空时展示默认值
	if mode == "" {
		return "Default"
	}

	// Return configured SameSite value 返回配置的 SameSite 值
	return string(mode)
}
