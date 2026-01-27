// @Author daixk 2025/12/12 10:45:00
package adapter

// Log 日志行为抽象接口
type Log interface {
	Print(v ...any)                 // 无级别日志输出
	Printf(format string, v ...any) // 无级别格式化日志输出

	Debug(v ...any)                 // 输出调试级别日志
	Debugf(format string, v ...any) // 输出调试级别格式化日志

	Info(v ...any)                 // 输出信息级别日志
	Infof(format string, v ...any) // 输出信息级别格式化日志

	Warn(v ...any)                 // 输出警告级别日志
	Warnf(format string, v ...any) // 输出警告级别格式化日志

	Error(v ...any)                 // 输出错误级别日志
	Errorf(format string, v ...any) // 输出错误级别格式化日志
}

// LogLevel 日志级别定义
type LogLevel int

const (
	LogLevelDebug LogLevel = iota + 1 // 调试级别
	LogLevelInfo                      // 信息级别
	LogLevelWarn                      // 警告级别
	LogLevelError                     // 错误级别（最高）
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogControl 日志运行时控制接口
type LogControl interface {
	Log

	// ---- 生命周期 ----
	Close() // 关闭日志并释放资源
	Flush() // 刷新缓冲区

	// ---- 运行时配置 ----
	SetLevel(level LogLevel) // 动态更新日志级别
	SetPrefix(prefix string) // 动态更新日志前缀
	SetStdout(enable bool)   // 开关控制台输出

	// ---- 状态查询 ----
	LogPath() string   // 获取日志目录
	DropCount() uint64 // 获取丢弃的日志数量
}
