// @Author daixk 2025/12/22 15:56:00
package gf

import (
	"context"
	"testing"

	"github.com/gogf/gf/v2/os/glog"
)

// TestNewGFLogger verifies adapter construction 测试 GoFrame 日志适配器构造
func TestNewGFLogger(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	raw := glog.New()
	logger := NewGFLogger(ctx, raw)
	if logger == nil {
		t.Fatal("NewGFLogger() returned nil")
	}
	if logger.ctx != ctx {
		t.Fatal("NewGFLogger() did not keep context")
	}
	if logger.l != raw {
		t.Fatal("NewGFLogger() did not keep logger")
	}
}

// TestNilGFLoggerDoesNotPanic verifies nil logger calls are safe TestNilGFLoggerDoesNotPanic 验证空日志器调用安全
func TestNilGFLoggerDoesNotPanic(t *testing.T) {
	logger := NewGFLogger(nil, nil)
	if logger == nil {
		t.Fatal("NewGFLogger(nil, nil) returned nil")
	}
	if logger.ctx == nil {
		t.Fatal("NewGFLogger(nil, nil) should use background context")
	}

	logger.Print("plain")
	logger.Printf("plain %s", "format")
	logger.Debug("debug")
	logger.Debugf("debug %s", "format")
	logger.Info("info")
	logger.Infof("info %s", "format")
	logger.Warn("warn")
	logger.Warnf("warn %s", "format")
	logger.Error("error")
	logger.Errorf("error %s", "format")

	var nilLogger *GFLogger
	nilLogger.Info("drop")
}
