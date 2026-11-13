package dlog

import "github.com/Zany2/dtoken-go/contract"

// LogLevel uses the framework's shared severity type. LogLevel 使用框架共享日志级别类型。
type LogLevel = contract.LogLevel

// Supported logging levels 日志支持的级别。
const (
	// LevelDebug enables diagnostic messages. LevelDebug 启用调试消息。
	LevelDebug = contract.LogLevelDebug
	// LevelInfo enables informational messages. LevelInfo 启用信息消息。
	LevelInfo = contract.LogLevelInfo
	// LevelWarn enables warning messages. LevelWarn 启用警告消息。
	LevelWarn = contract.LogLevelWarn
	// LevelError enables error messages. LevelError 启用错误消息。
	LevelError = contract.LogLevelError
)

// Bundled logging defaults 内置日志默认值。
const (
	// DefaultPrefix identifies framework log messages. DefaultPrefix 标识框架日志消息。
	DefaultPrefix = "[DTOKEN] "
	// DefaultFilename is the active filename managed by Lumberjack. DefaultFilename 是 Lumberjack 管理的活动文件名。
	DefaultFilename = "DTOKEN.log"
	// DefaultDirName is the default output directory. DefaultDirName 是默认输出目录。
	DefaultDirName = "dtoken_logs"
	// DefaultTimeFormat is the default timestamp layout. DefaultTimeFormat 是默认时间布局。
	DefaultTimeFormat = "2006-01-02 15:04:05"
	// DefaultMaxSize limits each file to ten MiB. DefaultMaxSize 将单个文件限制为十 MiB。
	DefaultMaxSize = 10
	// DefaultMaxBackups retains ten rotated files. DefaultMaxBackups 保留十个轮转文件。
	DefaultMaxBackups = 10
	// DefaultMaxAge retains backups for seven days. DefaultMaxAge 将备份保留七天。
	DefaultMaxAge = 7
)
