// @Author daixk 2025/12/22 15:56:00
package ants

import "time"

const (
	DefaultMinSize       = 20               // Minimum pool size 最小协程数
	DefaultMaxSize       = 200              // Maximum pool size 最大协程数
	DefaultScaleUpRate   = 0.8              // Scale-up threshold 扩容阈值
	DefaultScaleDownRate = 0.3              // Scale-down threshold 缩容阈值
	DefaultCheckInterval = 30 * time.Second // Interval for auto-scaling checks 检查间隔
	DefaultExpiry        = time.Minute      // Idle worker expiry duration 空闲协程过期时间
	DefaultStopTimeout   = 3 * time.Second  // Stop timeout for waiting running tasks 停止时等待运行任务的超时时间
)
