// @Author daixk 2025/12/22 15:56:00
package adapter

// NopLogger is a no-op logger used when logging is disabled NopLogger 是日志关闭时使用的空日志实现
type NopLogger struct{}

// Interface assertion keeps logger contract checked at compile time 接口断言在编译期检查日志契约
var _ Log = (*NopLogger)(nil)

// NewNopLogger creates a no-op logger NewNopLogger 创建空日志实现
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

// Print ignores plain log output Print 忽略普通日志输出
func (l *NopLogger) Print(v ...any) {}

// Printf ignores formatted plain log output Printf 忽略格式化普通日志输出
func (l *NopLogger) Printf(format string, v ...any) {}

// Debug ignores debug log output Debug 忽略调试日志输出
func (l *NopLogger) Debug(v ...any) {}

// Debugf ignores formatted debug log output Debugf 忽略格式化调试日志输出
func (l *NopLogger) Debugf(format string, v ...any) {}

// Info ignores info log output Info 忽略信息日志输出
func (l *NopLogger) Info(v ...any) {}

// Infof ignores formatted info log output Infof 忽略格式化信息日志输出
func (l *NopLogger) Infof(format string, v ...any) {}

// Warn ignores warn log output Warn 忽略警告日志输出
func (l *NopLogger) Warn(v ...any) {}

// Warnf ignores formatted warn log output Warnf 忽略格式化警告日志输出
func (l *NopLogger) Warnf(format string, v ...any) {}

// Error ignores error log output Error 忽略错误日志输出
func (l *NopLogger) Error(v ...any) {}

// Errorf ignores formatted error log output Errorf 忽略格式化错误日志输出
func (l *NopLogger) Errorf(format string, v ...any) {}
