package dlog

import (
	"reflect"
	"testing"
	"time"
)

// TestDefaultLoggerConfig verifies default values 测试默认日志配置
func TestDefaultLoggerConfig(t *testing.T) {
	cfg := DefaultLoggerConfig()
	if cfg.TimeFormat != DefaultTimeFormat {
		t.Fatalf("TimeFormat = %q, want %q", cfg.TimeFormat, DefaultTimeFormat)
	}
	if cfg.FileFormat != DefaultFileFormat {
		t.Fatalf("FileFormat = %q, want %q", cfg.FileFormat, DefaultFileFormat)
	}
	if cfg.Level != LevelInfo {
		t.Fatalf("Level = %v, want %v", cfg.Level, LevelInfo)
	}
	if !cfg.Stdout {
		t.Fatal("Stdout should be enabled by default")
	}
}

// TestLoggerConfigSettersAndClone verifies fluent setters and clone behavior 测试链式配置与克隆行为
func TestLoggerConfigSettersAndClone(t *testing.T) {
	cfg := DefaultLoggerConfig().
		SetPath("logs").
		SetFileFormat("app.log").
		SetPrefix("[APP] ").
		SetLevel(LevelDebug).
		SetTimeFormat(time.RFC3339).
		SetStdout(false).
		SetStdoutOnly(true).
		SetQueueSize(8).
		SetRotateSize(1024).
		SetRotateExpire(time.Hour).
		SetRotateBackupLimit(3).
		SetRotateBackupDays(2)

	want := &LoggerConfig{
		Path:              "logs",
		FileFormat:        "app.log",
		Prefix:            "[APP] ",
		Level:             LevelDebug,
		TimeFormat:        time.RFC3339,
		Stdout:            true,
		StdoutOnly:        true,
		QueueSize:         8,
		RotateSize:        1024,
		RotateExpire:      time.Hour,
		RotateBackupLimit: 3,
		RotateBackupDays:  2,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("config = %+v, want %+v", cfg, want)
	}

	clone := cfg.Clone()
	clone.SetPrefix("[CLONE] ")
	if cfg.Prefix == clone.Prefix {
		t.Fatal("Clone() should return an independent copy")
	}
	if nilClone := (*LoggerConfig)(nil).Clone(); nilClone == nil {
		t.Fatal("Clone() on nil should return empty config")
	}
}
