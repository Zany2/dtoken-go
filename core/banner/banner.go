// @Author daixk 2025/12/22 15:56:00
package banner

import (
	"fmt"
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
	if cfg == nil || !cfg.IsPrintBanner {
		return
	}

	// Print banner 打印 Banner
	fmt.Print(BannerText)
	fmt.Printf(":: DToken-Go ::        (version %s)\n\n", core.Version)

	// Print config summary 打印关键配置信息
	fmt.Println("========================================")
	fmt.Println("         Configuration Summary          ")
	fmt.Println("========================================")

	// Print auth config 打印认证配置
	fmt.Printf("AuthType         : %s\n", strings.TrimSuffix(cfg.AuthType, ":"))
	fmt.Printf("KeyPrefix        : %s\n", strings.TrimSuffix(cfg.KeyPrefix, ":"))
	fmt.Printf("TokenName        : %s\n", cfg.TokenName)
	fmt.Printf("TokenStyle       : %s\n", getTokenStyleName(cfg.TokenStyle))

	// Print timeout config 打印超时配置
	fmt.Printf("Timeout          : %s\n", formatDuration(cfg.Timeout))
	if cfg.AutoRenew {
		fmt.Printf("AutoRenew        : Enabled\n")
		fmt.Printf("  ├─ MaxRefresh  : %s\n", formatDuration(cfg.RenewMaxRefresh))
		fmt.Printf("  └─ Interval    : %s\n", formatDuration(cfg.RenewInterval))
	} else {
		fmt.Printf("AutoRenew        : Disabled\n")
	}
	fmt.Printf("ActiveTimeout    : %s\n", formatDuration(cfg.ActiveTimeout))

	// Print concurrency config 打印并发配置
	fmt.Printf("Concurrency      : %s\n", formatConcurrency(cfg))

	// Print token source config 打印 Token 读取配置
	fmt.Printf("Token Source     : %s\n", formatTokenSource(cfg))

	// Print log config 打印日志配置
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

// getTokenStyleName gets token style name getTokenStyleName 获取 Token 风格名称
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

// formatDuration formats duration display formatDuration 格式化时长显示
func formatDuration(seconds int64) string {
	if seconds == config.NoLimit {
		return "No Limit"
	}
	if seconds <= 0 {
		return "Disabled"
	}

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

// formatTokenSource formats token source display formatTokenSource 格式化 Token 读取来源
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
