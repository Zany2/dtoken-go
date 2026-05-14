package dlog

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewLoggerWithConfigWritesFile verifies file output and close flushing 测试文件输出与关闭刷新
func TestNewLoggerWithConfigWritesFile(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{
		Path:       dir,
		FileFormat: "test.log",
		Prefix:     "[TEST] ",
		Level:      LevelDebug,
		Stdout:     false,
		QueueSize:  4,
	})
	if err != nil {
		t.Fatalf("NewLoggerWithConfig() error = %v", err)
	}

	logger.Debug("debug message")
	logger.Infof("hello %s", "world")
	logger.Close()

	data, err := os.ReadFile(filepath.Join(dir, "test.log"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "[DEBUG] [TEST] debug message") {
		t.Fatalf("log file missing debug line: %q", text)
	}
	if !strings.Contains(text, "[INFO] [TEST] hello world") {
		t.Fatalf("log file missing info line: %q", text)
	}
}

// TestPrepareConfigDefaultsAndStdoutOnly verifies default normalization 测试默认配置归一化
func TestPrepareConfigDefaultsAndStdoutOnly(t *testing.T) {
	cfg, err := prepareConfig(&LoggerConfig{StdoutOnly: true})
	if err != nil {
		t.Fatalf("prepareConfig() error = %v", err)
	}
	if !cfg.Stdout || !cfg.StdoutOnly {
		t.Fatalf("stdout only config = %+v", cfg)
	}
	if cfg.FileFormat != "" {
		t.Fatalf("StdoutOnly should not force file format, got %q", cfg.FileFormat)
	}

	dir := t.TempDir()
	cfg, err = prepareConfig(&LoggerConfig{Path: dir, RotateExpire: -time.Second})
	if err != nil {
		t.Fatalf("prepareConfig(file) error = %v", err)
	}
	if cfg.FileFormat == "" || cfg.RotateSize <= 0 || cfg.RotateBackupLimit <= 0 {
		t.Fatalf("file defaults not applied: %+v", cfg)
	}
	if cfg.RotateExpire != 0 {
		t.Fatalf("negative RotateExpire should normalize to 0, got %v", cfg.RotateExpire)
	}
}

// TestLoggerHelpers verifies formatting helpers 测试日志格式化辅助函数
func TestLoggerHelpers(t *testing.T) {
	logger, err := NewLoggerWithConfig(&LoggerConfig{StdoutOnly: true, Prefix: "[X] "})
	if err != nil {
		t.Fatalf("NewLoggerWithConfig() error = %v", err)
	}
	defer logger.Close()

	line := string(logger.buildLine(LevelError, logger.currentCfg(), "err", errors.New("boom"), 7, true))
	if !strings.Contains(line, "[ERROR] [X] err boom 7 true") {
		t.Fatalf("buildLine() = %q", line)
	}

	var buf bytes.Buffer
	appendValue(&buf, []byte("bytes"))
	appendValue(&buf, nil)
	if buf.String() != "bytes<nil>" {
		t.Fatalf("appendValue() = %q", buf.String())
	}

	if levelString(LevelWarn) != "WARN" {
		t.Fatalf("levelString(LevelWarn) = %q", levelString(LevelWarn))
	}
	if normalizeBaseName("APP_{Y}-{m}-{d}.log") != "APP" {
		t.Fatalf("normalizeBaseName() = %q", normalizeBaseName("APP_{Y}-{m}-{d}.log"))
	}
	if secureRandomInt(10) < 0 || secureRandomInt(10) >= 10 {
		t.Fatal("secureRandomInt() should stay in range")
	}
}

// TestLoggerRuntimeControls verifies mutable logger controls 测试运行时控制接口
func TestLoggerRuntimeControls(t *testing.T) {
	logger, err := NewLoggerWithConfig(&LoggerConfig{StdoutOnly: true})
	if err != nil {
		t.Fatalf("NewLoggerWithConfig() error = %v", err)
	}
	defer logger.Close()

	logger.SetLevel(LevelDebug)
	logger.SetPrefix("[RUNTIME] ")
	logger.SetStdout(false)
	cfg := logger.currentCfg()
	if cfg.Level != LevelDebug || cfg.Prefix != "[RUNTIME] " || cfg.Stdout {
		t.Fatalf("runtime config = %+v", cfg)
	}

	logger.SetConfig(&LoggerConfig{StdoutOnly: true, Prefix: "[NEW] "})
	if logger.currentCfg().Prefix != "[NEW] " {
		t.Fatalf("SetConfig() prefix = %q", logger.currentCfg().Prefix)
	}
	logger.Flush()
	logger.Close()
	logger.Close()
}
