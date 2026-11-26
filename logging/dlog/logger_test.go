package dlog

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoggerSynchronousOutput verifies every logging method is visible before Flush or Close. TestLoggerSynchronousOutput 验证所有日志方法在 Flush 或 Close 之前即可读取输出。
func TestLoggerSynchronousOutput(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, Filename: "app.log", Prefix: "[TEST] ", Level: LevelDebug, TimeFormat: time.RFC3339Nano})
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	logger.Print("plain", 7)
	logger.Printf("plain-%d", 8)
	logger.Debug("debug")
	logger.Debugf("debug-%d", 1)
	logger.Info("info")
	logger.Infof("info-%d", 2)
	logger.Warn("warn")
	logger.Warnf("warn-%d", 3)
	logger.Error("error")
	logger.Errorf("error-%d", 4)
	text := readLog(t, filepath.Join(dir, "app.log"))
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != 10 {
		t.Fatalf("got %d lines, want 10", len(lines))
	}
	for _, line := range lines {
		timestamp := strings.SplitN(line, " ", 2)[0]
		if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
			t.Fatalf("invalid timestamp in %q: %v", line, err)
		}
	}
	for _, want := range []string{"[TEST] plain 7", "[TEST] plain-8", "[DEBUG] [TEST] debug", "[DEBUG] [TEST] debug-1", "[INFO] [TEST] info", "[INFO] [TEST] info-2", "[WARN] [TEST] warn", "[WARN] [TEST] warn-3", "[ERROR] [TEST] error", "[ERROR] [TEST] error-4"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %q", want, text)
		}
	}
	if strings.Contains(lines[0], "[INFO]") || logger.DropCount() != 0 {
		t.Fatal("plain logging or synchronous drop accounting changed")
	}
}

// TestLoggerLevelAndPrefix verifies dynamic controls and unfiltered plain output. TestLoggerLevelAndPrefix 验证动态控制与不受级别过滤的普通输出。
func TestLoggerLevelAndPrefix(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, Filename: "level.log", Level: LevelError})
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	logger.Info("filtered-info")
	logger.Print("unfiltered-plain")
	logger.Error("visible-error")
	logger.SetLevel(LevelDebug)
	logger.SetPrefix("[NEW] ")
	logger.Debug("visible-debug")
	for _, level := range []LogLevel{0, -1, 99} {
		logger.SetLevel(level)
	}
	logger.Debug("still-debug")
	logger.Flush()
	text := readLog(t, filepath.Join(dir, "level.log"))
	for _, want := range []string{"unfiltered-plain", "visible-error", "[DEBUG] [NEW] visible-debug", "still-debug"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %q", want, text)
		}
	}
	if strings.Contains(text, "filtered-info") {
		t.Fatal("level filtering failed")
	}
}

// TestLoggerStdoutRouting verifies console toggles retain file output and Close does not close stdout. TestLoggerStdoutRouting 验证控制台切换保留文件输出，Close 不关闭标准输出。
func TestLoggerStdoutRouting(t *testing.T) {
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = original; _ = writer.Close(); _ = reader.Close() }()
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, Filename: "routing.log"})
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	logger.Info("file-only")
	logger.SetStdout(true)
	logger.Info("both-targets")
	logger.SetStdout(false)
	logger.Info("file-after-toggle")
	logger.Close()
	if _, err := writer.WriteString("stdout-still-open\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	console, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(console), "both-targets") || !strings.Contains(string(console), "stdout-still-open") || strings.Contains(string(console), "file-only") || strings.Contains(string(console), "file-after-toggle") {
		t.Fatalf("unexpected console routing: %q", console)
	}
	file := readLog(t, filepath.Join(dir, "routing.log"))
	for _, want := range []string{"file-only", "both-targets", "file-after-toggle"} {
		if !strings.Contains(file, want) {
			t.Fatalf("file output lost %q", want)
		}
	}
}

// TestLoggerStdoutOnlySkipsFiles verifies console-only configuration creates no log directory. TestLoggerStdoutOnlySkipsFiles 验证仅控制台配置不创建日志目录。
func TestLoggerStdoutOnlySkipsFiles(t *testing.T) {
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = original; _ = writer.Close(); _ = reader.Close() }()
	path := filepath.Join(t.TempDir(), "unused")
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: path, StdoutOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	logger.SetStdout(false)
	logger.Info("muted")
	logger.Flush()
	logger.SetStdout(true)
	logger.Info("resumed-console")
	logger.Flush()
	logger.Close()
	_ = writer.Close()
	console, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(console), "muted") || !strings.Contains(string(console), "resumed-console") {
		t.Fatalf("unexpected console-only routing: %q", console)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("console-only mode created a directory: %v", err)
	}
}

// TestLoggerFileFailureKeepsConsole verifies rejected file entries reach stdout and write errors reach stderr. TestLoggerFileFailureKeepsConsole 验证文件拒绝的日志仍输出到标准输出，写入错误报告到标准错误输出。
func TestLoggerFileFailureKeepsConsole(t *testing.T) {
	dir := t.TempDir()
	console, err := os.CreateTemp(t.TempDir(), "stdout-")
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()
	diagnostics, err := os.CreateTemp(t.TempDir(), "stderr-")
	if err != nil {
		t.Fatal(err)
	}
	defer diagnostics.Close()
	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = console, diagnostics
	defer func() { os.Stdout, os.Stderr = originalOut, originalErr }()
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, Filename: "size.log", MaxSize: 1, Stdout: true})
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	oversized := "oversized-entry-" + strings.Repeat("x", 1024*1024)
	logger.Info(oversized)
	logger.Info("file-recovered")
	logger.Close()
	file := readLog(t, filepath.Join(dir, "size.log"))
	if strings.Contains(file, "oversized-entry-") || !strings.Contains(file, "file-recovered") {
		t.Fatal("rejected entry corrupted subsequent file output")
	}
	if !strings.Contains(readLog(t, console.Name()), oversized) {
		t.Fatal("file failure suppressed console output")
	}
	if !strings.Contains(readLog(t, diagnostics.Name()), "exceeds maximum file size") {
		t.Fatal("file write error was not reported to stderr")
	}
}

// TestLoggerReconfiguration verifies configuration snapshots, rejected changes, and output replacement. TestLoggerReconfiguration 验证配置快照、非法变更拒绝及输出替换。
func TestLoggerReconfiguration(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	cfg := &LoggerConfig{Path: first, Filename: "first.log"}
	logger, err := NewLoggerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	cfg.Filename = "mutated.log"
	logger.Info("first-output")
	logger.SetConfig(&LoggerConfig{Path: second, Filename: "bad.log", MaxSize: -1})
	logger.Info("after-invalid-config")
	badPath := filepath.Join(second, "not-a-directory")
	if err := os.WriteFile(badPath, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	logger.SetConfig(&LoggerConfig{Path: badPath})
	logger.Info("after-invalid-directory")
	logger.SetConfig(&LoggerConfig{Path: second, Filename: "second.log", Prefix: "[SECOND] "})
	logger.Info("second-output")
	logger.Close()
	firstText := readLog(t, filepath.Join(first, "first.log"))
	for _, want := range []string{"first-output", "after-invalid-config", "after-invalid-directory"} {
		if !strings.Contains(firstText, want) {
			t.Fatalf("original output lost %q", want)
		}
	}
	secondText := readLog(t, filepath.Join(second, "second.log"))
	if !strings.Contains(secondText, "[SECOND] second-output") || strings.Contains(firstText, "second-output") || logger.LogPath() != second {
		t.Fatal("reconfiguration used the wrong output")
	}
	if err := os.Rename(filepath.Join(first, "first.log"), filepath.Join(first, "closed.log")); err != nil {
		t.Fatalf("old output was not released: %v", err)
	}
}

// TestLoggerRotationAndCompression verifies real size-based rotation and readable gzip backups. TestLoggerRotationAndCompression 验证真实大小轮转与可读取的 gzip 备份。
func TestLoggerRotationAndCompression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rotate.log")
	original := "previous-file\n" + strings.Repeat("x", 1024*1024-64)
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, Filename: "rotate.log", MaxSize: 1, MaxBackups: 1, Compress: true})
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	logger.Info("new-file-" + strings.Repeat("y", 128))
	if !strings.Contains(readLog(t, path), "new-file-") {
		t.Fatal("active file did not receive the new entry")
	}
	var backup string
	waitFor(t, func() bool {
		files, _ := filepath.Glob(filepath.Join(dir, "rotate-*.log.gz"))
		if len(files) != 1 {
			return false
		}
		backup = files[0]
		_, err := os.Stat(strings.TrimSuffix(backup, ".gz"))
		return os.IsNotExist(err)
	})
	file, err := os.Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer compressed.Close()
	data, err := io.ReadAll(compressed)
	if err != nil || string(data) != original {
		t.Fatalf("rotation lost backup content: length=%d, error=%v", len(data), err)
	}
}

// TestLoggerRetention verifies native count and age limits leave unrelated files untouched. TestLoggerRetention 验证原生数量与时间保留策略不会影响无关文件。
func TestLoggerRetention(t *testing.T) {
	tests := []struct {
		name              string
		count, days, want int
	}{
		{"count", 1, 0, 1}, {"age", 0, 1, 2}, {"unlimited", 0, 0, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, age := range []time.Duration{72 * time.Hour, 12 * time.Hour, time.Hour} {
				name := "retain-" + time.Now().UTC().Add(-age).Format("2006-01-02T15-04-05.000") + ".log"
				if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
					t.Fatal(err)
				}
			}
			unrelated := filepath.Join(dir, "retain_audit.log")
			if err := os.WriteFile(unrelated, []byte("audit"), 0600); err != nil {
				t.Fatal(err)
			}
			logger, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, Filename: "retain.log", MaxBackups: tt.count, MaxAge: tt.days})
			if err != nil {
				t.Fatal(err)
			}
			defer logger.Close()
			logger.Info("active")
			waitFor(t, func() bool {
				files, _ := filepath.Glob(filepath.Join(dir, "retain-*.log"))
				return len(files) == tt.want
			})
			if readLog(t, unrelated) != "audit" {
				t.Fatal("unrelated file changed")
			}
			if !strings.Contains(readLog(t, filepath.Join(dir, "retain.log")), "active") {
				t.Fatal("cleanup removed the active file")
			}
		})
	}
}

// readLog reads file output without requiring a Flush call. readLog 无需调用 Flush 即可读取文件输出。
func readLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// waitFor waits for Lumberjack's asynchronous housekeeping to finish. waitFor 等待 Lumberjack 的异步备份维护完成。
func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("Lumberjack housekeeping did not finish")
}

// TestLoggerRejectsInvalidConstruction verifies invalid configuration is returned to callers. TestLoggerRejectsInvalidConstruction 验证创建时将非法配置错误返回给调用方。
func TestLoggerRejectsInvalidConstruction(t *testing.T) {
	if logger, err := NewLoggerWithConfig(&LoggerConfig{MaxSize: -1}); err == nil || logger != nil {
		t.Fatal("invalid construction succeeded")
	}
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLoggerWithConfig(&LoggerConfig{Path: path}); err == nil {
		t.Fatal("file was accepted as a directory")
	}
}

// TestLoggerRejectsInvalidFileTarget verifies unusable files are rejected before replacing active output. TestLoggerRejectsInvalidFileTarget 验证不可用的文件目标在替换当前输出前被拒绝。
func TestLoggerRejectsInvalidFileTarget(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "occupied.log"), 0755); err != nil {
		t.Fatal(err)
	}
	bad := &LoggerConfig{Path: dir, Filename: "occupied.log"}
	t.Run("construction", func(t *testing.T) {
		logger, err := NewLoggerWithConfig(bad)
		if logger != nil {
			logger.Close()
		}
		if err == nil {
			t.Fatal("directory was accepted as a log file")
		}
	})
	t.Run("reconfiguration", func(t *testing.T) {
		original := t.TempDir()
		logger, err := NewLoggerWithConfig(&LoggerConfig{Path: original, Filename: "active.log"})
		if err != nil {
			t.Fatal(err)
		}
		defer logger.Close()
		logger.Info("before-rejected-target")
		logger.SetConfig(bad)
		logger.Info("after-rejected-target")
		if logger.LogPath() != original || !strings.Contains(readLog(t, filepath.Join(original, "active.log")), "after-rejected-target") {
			t.Fatal("invalid file target replaced the active output")
		}
		if info, err := os.Stat(filepath.Join(dir, "occupied.log")); err != nil || !info.IsDir() {
			t.Fatal("rejected target directory was changed")
		}
	})
}
