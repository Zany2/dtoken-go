// @Author daixk 2025/11/27 21:08:00
package nop

// NopLogger 用于禁用所有日志输出的空日志器
type NopLogger struct{}

// NewNopLogger 创建一个新的空日志器
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

func (n *NopLogger) Print(v ...any)                 {}
func (n *NopLogger) Printf(format string, v ...any) {}

func (n *NopLogger) Debug(v ...any)                 {}
func (n *NopLogger) Debugf(format string, v ...any) {}

func (n *NopLogger) Info(v ...any)                 {}
func (n *NopLogger) Infof(format string, v ...any) {}

func (n *NopLogger) Warn(v ...any)                 {}
func (n *NopLogger) Warnf(format string, v ...any) {}

func (n *NopLogger) Error(v ...any)                 {}
func (n *NopLogger) Errorf(format string, v ...any) {}
