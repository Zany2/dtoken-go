// @Author daixk 2025/12/22 15:56:00
package nop

import "testing"

// TestNopLoggerMethodsDoNothing verifies all no-op logger methods are callable 测试空日志器方法可安全调用
func TestNopLoggerMethodsDoNothing(t *testing.T) {
	logger := NewNopLogger()
	if logger == nil {
		t.Fatal("NewNopLogger() returned nil")
	}
	logger.Print("plain")
	logger.Printf("%s", "plain")
	logger.Debug("debug")
	logger.Debugf("%s", "debug")
	logger.Info("info")
	logger.Infof("%s", "info")
	logger.Warn("warn")
	logger.Warnf("%s", "warn")
	logger.Error("error")
	logger.Errorf("%s", "error")
}

// TestNilNopLoggerMethodsDoNothing verifies nil alias receivers remain safe. TestNilNopLoggerMethodsDoNothing 验证 nil 别名接收者仍可安全调用。
func TestNilNopLoggerMethodsDoNothing(t *testing.T) {
	var logger *NopLogger
	logger.Print("plain")
	logger.Printf("%s", "plain")
	logger.Debug("debug")
	logger.Debugf("%s", "debug")
	logger.Info("info")
	logger.Infof("%s", "info")
	logger.Warn("warn")
	logger.Warnf("%s", "warn")
	logger.Error("error")
	logger.Errorf("%s", "error")
}
