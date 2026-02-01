// @Author daixk 2026/2/1 15:07:00
package banner

import (
	"fmt"
	"github.com/Zany2/dtoken-go/core"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"strings"
	"time"
)

const (
	// BannerText Banner 文本内容
	BannerText = `
 ____  _____     _
|  _ \|_   _|__ | | _____ _ __
| | | | | |/ _ \| |/ / _ \ '_ \
| |_| | | | (_) |   <  __/ | | |
|____/  |_|\___/|_|\_\___|_| |_|

`
)

// PrintBanner 打印启动 Banner 和关键配置信息
func PrintBanner(cfg *config.Config) {
	if cfg == nil || !cfg.IsPrintBanner {
		return
	}

	// 打印 Banner
	fmt.Print(BannerText)
	fmt.Printf(":: DToken-Go ::        (version %s)\n\n", core.Version)

	// 打印关键配置信息
	fmt.Println("========================================")
	fmt.Println("         Configuration Summary          ")
	fmt.Println("========================================")

	// 认证配置
	fmt.Printf("AuthType         : %s\n", strings.TrimSuffix(cfg.AuthType, ":"))
	fmt.Printf("TokenName        : %s\n", cfg.TokenName)
	fmt.Printf("TokenStyle       : %s\n", getTokenStyleName(cfg.TokenStyle))

	// 超时配置
	fmt.Printf("Timeout          : %s\n", formatDuration(cfg.Timeout))
	if cfg.AutoRenew {
		fmt.Printf("AutoRenew        : Enabled\n")
		fmt.Printf("  ├─ MaxRefresh  : %s\n", formatDuration(cfg.RenewMaxRefresh))
		fmt.Printf("  └─ Interval    : %s\n", formatDuration(cfg.RenewInterval))
	} else {
		fmt.Printf("AutoRenew        : Disabled\n")
	}
	fmt.Printf("ActiveTimeout    : %s\n", formatDuration(cfg.ActiveTimeout))

	// 并发配置
	fmt.Printf("Concurrency      : %s\n", formatConcurrency(cfg))

	// Token 读取配置
	fmt.Printf("Token Source     : %s\n", formatTokenSource(cfg))

	// 日志配置
	if cfg.IsLog {
		fmt.Printf("Logging          : Enabled\n")
	} else {
		fmt.Printf("Logging          : Disabled\n")
	}

	fmt.Println("========================================")
	fmt.Printf("Started at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("========================================")
	fmt.Println()
}

// getTokenStyleName 获取 Token 风格名称
func getTokenStyleName(style adapter.TokenStyle) string {
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
		return "Unknown"
	}
}

// formatDuration 格式化时长显示
func formatDuration(seconds int64) string {
	if seconds == config.NoLimit {
		return "No Limit"
	}
	if seconds <= 0 {
		return "Disabled"
	}

	d := time.Duration(seconds) * time.Second

	// 大于等于 1 天
	if d >= 24*time.Hour {
		days := d / (24 * time.Hour)
		hours := (d % (24 * time.Hour)) / time.Hour
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}

	// 大于等于 1 小时
	if d >= time.Hour {
		hours := d / time.Hour
		minutes := (d % time.Hour) / time.Minute
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}

	// 大于等于 1 分钟
	if d >= time.Minute {
		minutes := d / time.Minute
		seconds := (d % time.Minute) / time.Second
		if seconds > 0 {
			return fmt.Sprintf("%dm %ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	// 小于 1 分钟
	return fmt.Sprintf("%ds", seconds)
}

// formatConcurrency 格式化并发配置显示
func formatConcurrency(cfg *config.Config) string {
	if !cfg.IsConcurrent {
		return fmt.Sprintf("Disabled (Scope: %s)", cfg.ConcurrencyScope)
	}

	var parts []string
	parts = append(parts, "Enabled")
	parts = append(parts, fmt.Sprintf("Scope: %s", cfg.ConcurrencyScope))

	if cfg.IsShare {
		parts = append(parts, "Share: Yes")
	} else {
		parts = append(parts, "Share: No")
	}

	if cfg.MaxLoginCount == config.NoLimit {
		parts = append(parts, "Max: Unlimited")
	} else {
		parts = append(parts, fmt.Sprintf("Max: %d", cfg.MaxLoginCount))
	}

	return strings.Join(parts, ", ")
}

// formatTokenSource 格式化 Token 读取来源显示
func formatTokenSource(cfg *config.Config) string {
	var sources []string
	if cfg.IsReadHeader {
		sources = append(sources, "Header")
	}
	if cfg.IsReadCookie {
		sources = append(sources, "Cookie")
	}
	if cfg.IsReadBody {
		sources = append(sources, "Body")
	}

	if len(sources) == 0 {
		return "None"
	}

	return strings.Join(sources, ", ")
}
