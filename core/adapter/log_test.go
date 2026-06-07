package adapter

import "testing"

// TestLogLevelString verifies log level text mapping TestLogLevelString 验证日志级别文本映射
func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{level: LogLevelDebug, want: "DEBUG"},
		{level: LogLevelInfo, want: "INFO"},
		{level: LogLevelWarn, want: "WARN"},
		{level: LogLevelError, want: "ERROR"},
		{level: LogLevel(99), want: "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Fatalf("LogLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

// TestNopLoggerMethodsDoNotPanic verifies no-op logger safely ignores all methods TestNopLoggerMethodsDoNotPanic 验证空日志安全忽略所有方法
func TestNopLoggerMethodsDoNotPanic(t *testing.T) {
	logger := NewNopLogger()
	if logger == nil {
		t.Fatal("NewNopLogger() = nil")
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
