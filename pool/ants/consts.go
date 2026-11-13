// @Author daixk 2025/12/22 15:56:00
package ants

import "time"

// Renew pool default configuration values 续期池默认配置值
const (
	// DefaultMinSize defines minimum pool size DefaultMinSize 定义最小协程数
	DefaultMinSize = 20

	// DefaultMaxSize defines maximum pool size DefaultMaxSize 定义最大协程数
	DefaultMaxSize = 200

	// DefaultScaleUpRate defines scale-up threshold DefaultScaleUpRate 定义扩容阈值
	DefaultScaleUpRate = 0.8

	// DefaultScaleDownRate defines scale-down threshold DefaultScaleDownRate 定义缩容阈值
	DefaultScaleDownRate = 0.3

	// DefaultCheckInterval defines auto-scaling check interval DefaultCheckInterval 定义自动扩缩容检查间隔
	DefaultCheckInterval = 30 * time.Second

	// DefaultExpiry defines idle worker expiry duration DefaultExpiry 定义空闲协程过期时间
	DefaultExpiry = time.Minute

	// DefaultStopTimeout defines stop wait timeout DefaultStopTimeout 定义停止时等待运行任务的超时时间
	DefaultStopTimeout = 3 * time.Second
)
