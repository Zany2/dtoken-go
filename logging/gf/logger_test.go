// @Author daixk 2025/12/22 15:56:00
package gf

import (
	"bytes"
	"context"
	"reflect"
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

// TestGFLoggerLevelAndContextContract verifies actual levels, filtering, context, and argument forwarding. TestGFLoggerLevelAndContextContract 验证实际级别、过滤、上下文及参数传递。
func TestGFLoggerLevelAndContextContract(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "trace-value")
	for _, mask := range []int{glog.LEVEL_ALL, glog.LEVEL_WARN | glog.LEVEL_ERRO, glog.LEVEL_NONE} {
		raw := glog.New()
		raw.SetStdoutPrint(false)
		raw.SetAsync(false)
		raw.SetLevel(mask)
		logger := NewGFLogger(ctx, raw)
		methods := []struct {
			name      string
			level     int
			plain     func(...any)
			formatted func(string, ...any)
		}{
			{"Print", glog.LEVEL_NONE, logger.Print, logger.Printf},
			{"Debug", glog.LEVEL_DEBU, logger.Debug, logger.Debugf},
			{"Info", glog.LEVEL_INFO, logger.Info, logger.Infof},
			{"Warn", glog.LEVEL_WARN, logger.Warn, logger.Warnf},
			{"Error", glog.LEVEL_ERRO, logger.Error, logger.Errorf},
		}
		for _, method := range methods {
			var levels []int
			var values [][]any
			raw.SetHandlers(func(gotCtx context.Context, input *glog.HandlerInput) {
				if gotCtx != ctx || gotCtx.Value(contextKey{}) != "trace-value" {
					t.Errorf("%s changed the supplied context", method.name)
				}
				levels = append(levels, input.Level)
				values = append(values, append([]any(nil), input.Values...))
			})
			method.plain("value", 7, true)
			method.formatted("value=%d enabled=%t literal=%%", 7, true)
			if method.level != glog.LEVEL_NONE && mask&method.level == 0 {
				if len(levels) != 0 {
					t.Fatalf("%s bypassed level mask %d", method.name, mask)
				}
				continue
			}
			if !reflect.DeepEqual(levels, []int{method.level, method.level}) {
				t.Fatalf("%s levels = %v for mask %d", method.name, levels, mask)
			}
			want := [][]any{{"value", 7, true}, {"value=7 enabled=true literal=%"}}
			if !reflect.DeepEqual(values, want) {
				t.Fatalf("%s arguments = %#v, want %#v", method.name, values, want)
			}
		}
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
