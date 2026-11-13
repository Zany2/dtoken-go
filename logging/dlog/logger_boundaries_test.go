package dlog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLoggerConcurrentReconfigurationPreservesWrites verifies every completed call survives concurrent output replacement. TestLoggerConcurrentReconfigurationPreservesWrites 验证并发替换输出时每次已完成的日志调用均被保留。
func TestLoggerConcurrentReconfigurationPreservesWrites(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: first, Filename: "switch.log"})
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 6; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			for j := 0; j < 32; j++ {
				logger.Infof("entry-%d-%d;", index, j)
			}
		}(i)
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 16; i++ {
			path := first
			if i%2 == 0 {
				path = second
			}
			logger.SetConfig(&LoggerConfig{Path: path, Filename: "switch.log"})
			logger.Flush()
		}
	}()
	close(start)
	workers.Wait()
	logger.Close()
	text := readLog(t, filepath.Join(first, "switch.log")) + readLog(t, filepath.Join(second, "switch.log"))
	for i := 0; i < 6; i++ {
		for j := 0; j < 32; j++ {
			entry := fmt.Sprintf("entry-%d-%d;", i, j)
			if count := strings.Count(text, entry); count != 1 {
				t.Fatalf("%s occurred %d times, want once", entry, count)
			}
		}
	}
}

// TestLoggerConcurrentLifecycle verifies writes, reconfiguration, and shutdown cannot reopen a closed file. TestLoggerConcurrentLifecycle 验证写入、配置变更与关闭并发时不会重新打开已关闭文件。
func TestLoggerConcurrentLifecycle(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: first, Filename: "race.log"})
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("accepted-before-close")
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 6; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			for j := 0; j < 32; j++ {
				logger.Infof("worker-%d-%d", index, j)
				logger.Flush()
			}
		}(i)
	}
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 8; i++ {
			path := first
			if i%2 == 0 {
				path = second
			}
			logger.SetConfig(&LoggerConfig{Path: path, Filename: "race.log"})
			logger.SetLevel(LevelDebug)
			logger.SetPrefix("[LIVE] ")
			logger.SetStdout(false)
		}
	}()
	go func() { defer workers.Done(); <-start; logger.Close() }()
	close(start)
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent lifecycle blocked")
	}
	logger.Info("ignored-after-close")
	logger.Close()
	logger.Flush()
	unused := filepath.Join(t.TempDir(), "closed-config")
	logger.SetConfig(&LoggerConfig{Path: unused})
	if _, err := os.Stat(unused); !os.IsNotExist(err) {
		t.Fatal("closed logger created a new directory")
	}
	text := readLog(t, filepath.Join(first, "race.log"))
	if data, err := os.ReadFile(filepath.Join(second, "race.log")); err == nil {
		text += string(data)
	}
	if !strings.Contains(text, "accepted-before-close") || strings.Contains(text, "ignored-after-close") || logger.DropCount() != 0 {
		t.Fatal("shutdown lost completed output or accepted new output")
	}
	for _, dir := range []string{first, second} {
		path := filepath.Join(dir, "race.log")
		if _, err := os.Stat(path); err == nil {
			if err := os.Rename(path, filepath.Join(dir, "released.log")); err != nil {
				t.Fatalf("closed file is still held: %v", err)
			}
		}
	}
}

// reconfiguringMessage changes runtime controls while fmt evaluates a user argument. reconfiguringMessage 在 fmt 处理用户参数时修改运行时控制。
type reconfiguringMessage struct{ logger *Logger }

// String exercises reentrant formatting outside logger locks. String 验证格式化发生在日志锁外，允许重入控制方法。
func (m reconfiguringMessage) String() string { m.logger.SetPrefix("[REENTRANT] "); return "message" }

// TestLoggerFormattingAllowsRuntimeControls verifies user formatting cannot deadlock the logger. TestLoggerFormattingAllowsRuntimeControls 验证用户格式化不会导致日志器死锁。
func TestLoggerFormattingAllowsRuntimeControls(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, Filename: "format.log"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		logger.Info(reconfiguringMessage{logger})
		logger.Infof("%s", reconfiguringMessage{logger})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("formatting deadlocked")
	}
	defer logger.Close()
	if strings.Count(readLog(t, filepath.Join(dir, "format.log")), "[REENTRANT] message") != 2 {
		t.Fatal("reentrant formatting did not reach both outputs")
	}
}

// panickingLogError exercises fmt's protection against unsafe Error methods. panickingLogError 验证 fmt 对不安全 Error 方法的保护。
type panickingLogError struct{}

// Error deliberately panics to simulate a broken application error. Error 故意触发 panic，模拟应用错误对象异常。
func (*panickingLogError) Error() string { panic("broken error") }

// TestLoggerErrorFormatting verifies standard fmt behavior for nil and panicking error values. TestLoggerErrorFormatting 验证空值与触发 panic 的错误对象使用标准 fmt 行为。
func TestLoggerErrorFormatting(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLoggerWithConfig(&LoggerConfig{Path: dir, Filename: "errors.log"})
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	var nilError *panickingLogError
	for _, value := range []error{nilError, &panickingLogError{}} {
		logger.Error(value)
	}
	text := readLog(t, filepath.Join(dir, "errors.log"))
	for _, value := range []error{nilError, &panickingLogError{}} {
		if !strings.Contains(text, fmt.Sprint(value)) {
			t.Fatal("error formatting differs from fmt")
		}
	}
}

// TestNilAndZeroLoggerMethods verifies harmless calls before initialization or after shutdown. TestNilAndZeroLoggerMethods 验证初始化前或关闭后的调用安全。
func TestNilAndZeroLoggerMethods(t *testing.T) {
	for _, logger := range []*Logger{nil, {}} {
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
		logger.SetLevel(LevelDebug)
		logger.SetPrefix("prefix")
		logger.SetStdout(false)
		logger.Flush()
		logger.Close()
		logger.Close()
		logger.SetConfig(&LoggerConfig{StdoutOnly: true})
		if logger.DropCount() != 0 || logger.LogPath() != "" {
			t.Fatal("uninitialized logger reported resources")
		}
	}
}
