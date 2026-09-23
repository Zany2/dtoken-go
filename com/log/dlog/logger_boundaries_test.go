package dlog

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLoggerPlainOutputAndInvalidLevel verifies plain output bypasses filtering and invalid levels preserve the threshold. TestLoggerPlainOutputAndInvalidLevel 验证普通输出绕过级别过滤，非法级别不会改变阈值。
func TestLoggerPlainOutputAndInvalidLevel(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, FileFormat: "plain.log", Level: LevelError})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	for _, level := range []LogLevel{0, -1, 99} {
		l.SetLevel(level)
		if l.currentCfg().Level != LevelError {
			t.Fatalf("SetLevel(%d) changed the threshold", level)
		}
	}
	l.Print("plain", 7)
	l.Printf("formatted %s=%d", "value", 8)
	l.Info("filtered")
	l.Error("visible error")
	l.Flush()
	data, err := os.ReadFile(filepath.Join(dir, "plain.log"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"plain 7", "formatted value=8", "[ERROR]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %q", want, text)
		}
	}
	if strings.Contains(text, "filtered") || strings.Contains(text, "[INFO]") || strings.Contains(text, "UNKNOWN") {
		t.Fatalf("unexpected filtering or level label: %q", text)
	}
}

// TestLoggerSubsecondTimestamp verifies fractional timestamps are not frozen by the second cache. TestLoggerSubsecondTimestamp 验证亚秒时间戳不会被秒级缓存冻结。
func TestLoggerSubsecondTimestamp(t *testing.T) {
	l := &Logger{}
	first := time.Date(2026, time.September, 23, 10, 20, 30, 123000000, time.UTC)
	second := first.Add(500 * time.Millisecond)
	for _, format := range []string{time.RFC3339Nano, "15:04:05.000", "15:04:05,999", DefaultTimeFormat} {
		for _, now := range []time.Time{first, second} {
			if got := l.getTimeString(now, now.Unix(), format); got != now.Format(format) {
				t.Fatalf("timestamp(%q) = %q, want %q", format, got, now.Format(format))
			}
		}
	}
}

// panicLogError exercises fmt's protection against unsafe Error methods. panicLogError 覆盖 fmt 对不安全 Error 方法的保护。
type panicLogError struct{}

// Error intentionally panics for both nil and non-nil receivers. Error 对 nil 和非 nil 接收者均故意触发 panic。
func (*panicLogError) Error() string { panic("broken error") }

// TestLoggerErrorFormattingDoesNotPanic verifies error arguments retain standard fmt safety. TestLoggerErrorFormattingDoesNotPanic 验证错误参数保留标准 fmt 的安全处理。
func TestLoggerErrorFormattingDoesNotPanic(t *testing.T) {
	var nilError *panicLogError
	for _, value := range []error{nilError, &panicLogError{}} {
		var buf bytes.Buffer
		appendValue(&buf, value)
		if got, want := buf.String(), fmt.Sprint(value); got != want {
			t.Fatalf("error formatting = %q, want %q", got, want)
		}
	}
}

// TestLoggerCleanupMatchesFileFamily verifies retention cannot delete files that merely share a prefix. TestLoggerCleanupMatchesFileFamily 验证保留策略不会删除仅前缀相似的文件。
func TestLoggerCleanupMatchesFileFamily(t *testing.T) {
	for _, format := range []string{"APP_{Y}-{m}-{d}.log", "{Y}-{m}-{d}_APP.log", "APP.log"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			cfg := LoggerConfig{Path: dir, FileFormat: format, RotateBackupDays: 1, RotateBackupLimit: 10}
			l := &Logger{}
			current := l.formatFileName(time.Now(), cfg)
			l.curName = current
			backup := strings.TrimSuffix(current, ".log") + "_20260923_102030_123_123456789.log"
			unrelated := []string{"APPLICATION.log", "APP_audit.log", "APP_2026-09-23_audit.log", "OTHER.log"}
			old := time.Now().Add(-72 * time.Hour)
			for _, name := range append([]string{current, backup}, unrelated...) {
				path := filepath.Join(dir, name)
				if err := os.WriteFile(path, []byte(name), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(path, old, old); err != nil {
					t.Fatal(err)
				}
			}
			l.cleanup(cfg)
			if _, err := os.Stat(filepath.Join(dir, backup)); !os.IsNotExist(err) {
				t.Fatalf("expired backup remains: %v", err)
			}
			for _, name := range append(unrelated, current) {
				if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
					t.Fatalf("cleanup removed %q: %v", name, err)
				}
			}
		})
	}
}

// TestLoggerRapidRotationPreservesEveryLine verifies rotations and Close retain accepted logs without backup collisions. TestLoggerRapidRotationPreservesEveryLine 验证快速轮转和关闭会保留所有已接收日志，不发生备份名称碰撞。
func TestLoggerRapidRotationPreservesEveryLine(t *testing.T) {
	const count = 64
	dir := t.TempDir()
	l, err := NewLoggerWithConfig(&LoggerConfig{
		Path: dir, FileFormat: "rapid.log", QueueSize: count,
		RotateSize: 1, RotateBackupLimit: count + 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	for i := 0; i < count; i++ {
		l.Infof("entry-%03d", i)
	}
	l.Close()
	var all strings.Builder
	for _, name := range listLogFiles(t, dir) {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		all.Write(data)
	}
	for i := 0; i < count; i++ {
		if n := strings.Count(all.String(), fmt.Sprintf("entry-%03d", i)); n != 1 {
			t.Fatalf("entry %d appears %d times, want 1", i, n)
		}
	}
}

// TestLoggerConcurrentCloseAndFlush verifies close drains admitted writes and leaves no stranded queue entries. TestLoggerConcurrentCloseAndFlush 验证并发关闭会清空已入队日志，不遗留队列条目。
func TestLoggerConcurrentCloseAndFlush(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, FileFormat: "close.log", QueueSize: 512})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	l.Info("accepted before close")
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for j := 0; j < 32; j++ {
				l.Info("concurrent")
				l.Flush()
			}
		}()
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-start
		l.Close()
	}()
	close(start)
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent Close/Flush blocked")
	}
	if len(l.queue) != 0 {
		t.Fatalf("closed logger has %d stranded entries", len(l.queue))
	}
	data, err := os.ReadFile(filepath.Join(dir, "close.log"))
	if err != nil || !strings.Contains(string(data), "accepted before close") {
		t.Fatalf("Close lost an accepted log: %q, %v", data, err)
	}
}

// TestLoggerConfigSwitchKeepsRoutingAndQueueConsistent verifies runtime changes reopen the correct file without pretending to resize the queue. TestLoggerConfigSwitchKeepsRoutingAndQueueConsistent 验证运行时变更重新打开正确文件，并保留实际队列容量。
func TestLoggerConfigSwitchKeepsRoutingAndQueueConsistent(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	l, err := NewLoggerWithConfig(&LoggerConfig{Path: first, FileFormat: "first.log", QueueSize: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	l.Info("first output")
	l.Flush()
	l.SetConfig(&LoggerConfig{Path: second, FileFormat: "second.log", QueueSize: 64})
	l.Info("second output")
	l.Flush()
	if l.currentCfg().QueueSize != cap(l.queue) || cap(l.queue) != 8 {
		t.Fatal("runtime config misrepresents queue capacity")
	}
	for path, want := range map[string]string{
		filepath.Join(first, "first.log"):   "first output",
		filepath.Join(second, "second.log"): "second output",
	} {
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), want) {
			t.Fatalf("output at %q = %q, %v", path, data, err)
		}
	}
}
