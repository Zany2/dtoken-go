// @Author daixk 2025/12/22 15:56:00
package dlog

import (
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
)

// LogLevel aliases adapter.LogLevel 日志级别别名
type LogLevel = adapter.LogLevel

// Log level aliases 日志级别别名
const (
	// Debug level 调试级别
	LevelDebug = adapter.LogLevelDebug
	// Info level 信息级别
	LevelInfo = adapter.LogLevelInfo
	// Warn level 警告级别
	LevelWarn = adapter.LogLevelWarn
	// Error level 错误级别（最高）
	LevelError = adapter.LogLevelError
)

// Default logger configuration values 默认日志配置值
const (
	// Default log prefix 默认日志前缀
	DefaultPrefix = "[DTOKEN] "
	// Default log filename format 默认文件命名格式
	DefaultFileFormat = "DTOKEN_{Y}-{m}-{d}.log"
	// Default time format 默认时间格式
	DefaultTimeFormat = "2006-01-02 15:04:05"
	// Default log directory name 默认日志目录名
	DefaultDirName = "dtoken_logs"
	// Default log filename prefix 默认日志文件基础前缀
	DefaultBaseName = "DTOKEN"
	// Default async queue size 默认异步队列大小
	DefaultQueueSize = 4096
	// Rotate threshold (10MB) 文件滚动大小阈值
	DefaultRotateSize = 10 * 1024 * 1024
	// Rotate by time interval (1 day) 时间滚动间隔
	DefaultRotateExpire = 24 * time.Hour
	// Max number of backups 最大备份数量
	DefaultRotateBackupLimit = 10
	// Retain logs for 7 days 备份保留天数
	DefaultRotateBackupDays = 7
)
