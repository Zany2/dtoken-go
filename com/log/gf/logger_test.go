// @Author daixk 2025/12/22 15:56:00
package gf

import (
	"bytes"
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

// TestGFLoggerMethodsDelegateToGoFrame verifies all log levels reach the configured writer. TestGFLoggerMethodsDelegateToGoFrame 验证所有日志级别都会写入配置的 Writer。
func TestGFLoggerMethodsDelegateToGoFrame(t *testing.T) {
	var output bytes.Buffer
	raw := glog.NewWithWriter(&output)
	raw.SetStdoutPrint(false)
	raw.SetHeaderPrint(false)
	raw.SetLevelPrint(false)
	raw.SetLevel(glog.LEVEL_ALL)
	logger := NewGFLogger(context.Background(), raw)

	logger.Print("print-message")
	logger.Printf("printf-%s", "message")
	logger.Debug("debug-message")
	logger.Debugf("debugf-%s", "message")
	logger.Info("info-message")
	logger.Infof("infof-%s", "message")
	logger.Warn("warn-message")
	logger.Warnf("warnf-%s", "message")
	logger.Error("error-message")
	logger.Errorf("errorf-%s", "message")

	text := output.String()
	for _, want := range []string{
		"print-message",
		"printf-message",
		"debug-message",
		"debugf-message",
		"info-message",
		"infof-message",
		"warn-message",
		"warnf-message",
		"error-message",
		"errorf-message",
	} {
		if !bytes.Contains(output.Bytes(), []byte(want)) {
			t.Fatalf("GoFrame output missing %q: %q", want, text)
		}
	}
}
