// @Author daixk 2025/11/27 21:08:00
package nop

import "github.com/Zany2/dtoken-go/core/adapter"

// NopLogger reuses the core no-op logger implementation NopLogger 复用 core 层的空日志实现
type NopLogger = adapter.NopLogger

// Interface assertion keeps log contract checked at compile time 接口断言在编译期检查日志契约
var _ adapter.Log = (*NopLogger)(nil)

// NewNopLogger creates a no-op logger NewNopLogger 创建空日志器
func NewNopLogger() *NopLogger {
	return adapter.NewNopLogger()
}
