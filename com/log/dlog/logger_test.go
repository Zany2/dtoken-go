// @Author daixk 2025/12/22 15:56:00
package dlog

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
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

// TestLoggerFlushWritesQueuedLogs verifies Flush waits for queued writes TestLoggerFlushWritesQueuedLogs 验证 Flush 会等待已入队日志落盘。
func TestLoggerFlushWritesQueuedLogs(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{
		Path:       dir,
		FileFormat: "flush.log",
		Prefix:     "[FLUSH] ",
		Level:      LevelDebug,
		Stdout:     false,
		QueueSize:  16,
	})
	if err != nil {
		t.Fatalf("NewLoggerWithConfig() error = %v", err)
	}
	defer logger.Close()

	logger.Info("flush message")
	logger.Flush()

	data, err := os.ReadFile(filepath.Join(dir, "flush.log"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "[INFO] [FLUSH] flush message") {
		t.Fatalf("Flush() did not write queued log, got %q", string(data))
	}
}

// TestLoggerRotateAndCleanupKeepsCurrentFile verifies rotation cleanup preserves current file TestLoggerRotateAndCleanupKeepsCurrentFile 验证滚动清理不会删除当前文件。
func TestLoggerRotateAndCleanupKeepsCurrentFile(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{
		Path:              dir,
		FileFormat:        "rotate.log",
		Prefix:            "[ROTATE] ",
		Level:             LevelDebug,
		Stdout:            false,
		QueueSize:         64,
		RotateSize:        80,
		RotateBackupLimit: 1,
	})
	if err != nil {
		t.Fatalf("NewLoggerWithConfig() error = %v", err)
	}
	defer logger.Close()

	for i := 0; i < 12; i++ {
		logger.Infof("message-%02d %s", i, strings.Repeat("x", 64))
		logger.Flush()
	}

	if !waitForLogFileCount(dir, 2, time.Second) {
		files := listLogFiles(t, dir)
		t.Fatalf("log files = %v, want current file plus one backup", files)
	}

	files := listLogFiles(t, dir)
	if len(files) != 2 {
		t.Fatalf("log files = %v, want 2", files)
	}
	if _, err = os.Stat(filepath.Join(dir, "rotate.log")); err != nil {
		t.Fatalf("current log file should be kept: %v", err)
	}
}

// TestLoggerDropCountIncrementsWhenQueueFull verifies queue overflow accounting TestLoggerDropCountIncrementsWhenQueueFull 验证队列满时丢弃计数递增。
func TestLoggerDropCountIncrementsWhenQueueFull(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{
		Path:       dir,
		FileFormat: "drop.log",
		Stdout:     false,
		QueueSize:  1,
	})
	if err != nil {
		t.Fatalf("NewLoggerWithConfig() error = %v", err)
	}
	defer logger.Close()

	for i := 0; i < 10_000 && logger.DropCount() == 0; i++ {
		logger.Info("drop candidate")
	}

	if logger.DropCount() == 0 {
		t.Fatal("DropCount() = 0, want dropped logs after queue overflow")
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
	if cfg.Path != "" {
		t.Fatalf("StdoutOnly should not force path, got %q", cfg.Path)
	}
	stdoutOnlyDir := filepath.Join(t.TempDir(), "stdout-only")
	cfg, err = prepareConfig(&LoggerConfig{StdoutOnly: true, Path: stdoutOnlyDir})
	if err != nil {
		t.Fatalf("prepareConfig(stdout only path) error = %v", err)
	}
	if _, err = os.Stat(stdoutOnlyDir); !os.IsNotExist(err) {
		t.Fatalf("StdoutOnly should not create log directory, stat error = %v", err)
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

// TestLoggerTimeCacheTracksFormat verifies time cache respects format changes TestLoggerTimeCacheTracksFormat 验证时间缓存会感知时间格式变化。
func TestLoggerTimeCacheTracksFormat(t *testing.T) {
	logger, err := NewLoggerWithConfig(&LoggerConfig{StdoutOnly: true})
	if err != nil {
		t.Fatalf("NewLoggerWithConfig() error = %v", err)
	}
	defer logger.Close()

	now := time.Now()
	sec := now.Unix()
	first := logger.getTimeString(now, sec, "2006")
	second := logger.getTimeString(now, sec, "15")
	if first == second {
		t.Fatalf("getTimeString() should respect format changes, got %q and %q", first, second)
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

// TestNilLoggerMethodsAreSafe verifies nil logger calls do not panic TestNilLoggerMethodsAreSafe 验证空日志器调用安全。
func TestNilLoggerMethodsAreSafe(t *testing.T) {
	var logger *Logger

	logger.Print("print")
	logger.Printf("%s", "print")
	logger.Debug("debug")
	logger.Debugf("%s", "debug")
	logger.Info("info")
	logger.Infof("%s", "info")
	logger.Warn("warn")
	logger.Warnf("%s", "warn")
	logger.Error("error")
	logger.Errorf("%s", "error")

	logger.SetLevel(LevelDebug)
	logger.SetPrefix("[NIL] ")
	logger.SetStdout(false)
	logger.SetConfig(DefaultLoggerConfig())
	logger.Flush()
	logger.Close()

	if path := logger.LogPath(); path != "" {
		t.Fatalf("LogPath(nil) = %q, want empty", path)
	}
	if drops := logger.DropCount(); drops != 0 {
		t.Fatalf("DropCount(nil) = %d, want 0", drops)
	}
}

func listLogFiles(t *testing.T, dir string) []string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}

	files := make([]string, 0, len(matches))
	for _, match := range matches {
		files = append(files, filepath.Base(match))
	}
	sort.Strings(files)
	return files
}

func waitForLogFileCount(dir string, count int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.log"))
		if len(matches) == count {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
