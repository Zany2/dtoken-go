// @Author daixk 2025/11/27 22:58:00
package gf

import (
	"context"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/gogf/gf/v2/os/glog"
)

// GFLogger implements the GoFrame logger adapter GoFrame 日志适配器实现
type GFLogger struct {
	ctx context.Context
	l   *glog.Logger
}

// Interface assertion keeps log contract checked at compile time 接口断言在编译期检查日志契约
var _ adapter.Log = (*GFLogger)(nil)

// NewGFLogger creates a GoFrame logger adapter 创建新的 GoFrame 日志适配器
func NewGFLogger(ctx context.Context, l *glog.Logger) *GFLogger {
	if ctx == nil {
		ctx = context.Background()
	}
	return &GFLogger{
		ctx: ctx,
		l:   l,
	}
}

// Print writes plain logs through GoFrame 通过 GoFrame 输出普通日志
func (g *GFLogger) Print(v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Print(g.ctx, v...)
}

// Printf writes formatted logs through GoFrame 通过 GoFrame 输出格式化日志
func (g *GFLogger) Printf(format string, v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Printf(g.ctx, format, v...)
}

// Debug writes debug logs through GoFrame 通过 GoFrame 输出调试日志
func (g *GFLogger) Debug(v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Debug(g.ctx, v...)
}

// Debugf writes formatted debug logs through GoFrame 通过 GoFrame 输出格式化调试日志
func (g *GFLogger) Debugf(format string, v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Debugf(g.ctx, format, v...)
}

// Info writes info logs through GoFrame 通过 GoFrame 输出信息日志
func (g *GFLogger) Info(v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Info(g.ctx, v...)
}

// Infof writes formatted info logs through GoFrame 通过 GoFrame 输出格式化信息日志
func (g *GFLogger) Infof(format string, v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Infof(g.ctx, format, v...)
}

// Warn writes warning logs through GoFrame 通过 GoFrame 输出警告日志
func (g *GFLogger) Warn(v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Warning(g.ctx, v...)
}

// Warnf writes formatted warning logs through GoFrame 通过 GoFrame 输出格式化警告日志
func (g *GFLogger) Warnf(format string, v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Warningf(g.ctx, format, v...)
}

// Error writes error logs through GoFrame 通过 GoFrame 输出错误日志
func (g *GFLogger) Error(v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Error(g.ctx, v...)
}

// Errorf writes formatted error logs through GoFrame 通过 GoFrame 输出格式化错误日志
func (g *GFLogger) Errorf(format string, v ...any) {
	if g == nil || g.l == nil {
		return
	}
	g.l.Errorf(g.ctx, format, v...)
}
