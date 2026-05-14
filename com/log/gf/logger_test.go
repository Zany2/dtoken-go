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
