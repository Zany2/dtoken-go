package dtoken

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/logging/dlog"
)

// TestBuilderConfiguresZapLogging verifies public builder settings reach the default logging pipeline. TestBuilderConfiguresZapLogging 验证公开构建器配置传递到默认日志输出链路。
func TestBuilderConfiguresZapLogging(t *testing.T) {
	dir := t.TempDir()
	auth, err := NewBuilder().IsLog(true).IsPrintBanner(false).AutoRenew(false).
		LoggerPath(dir).LoggerFilename("auth.log").LoggerStdout(false).
		LoggerLevel(dlog.LevelWarn).
		LoggerMaxSize(1).LoggerMaxBackups(2).LoggerMaxAge(3).
		LoggerCompress(true).LoggerLocalTime(true).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	logger := auth.Manager().GetLogger()
	logger.Info("filtered-info")
	logger.Warn("visible-warning")
	auth.Close()
	data, err := os.ReadFile(filepath.Join(dir, "auth.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[DTOKEN] [WARN] visible-warning") || strings.Contains(string(data), "filtered-info") {
		t.Fatalf("builder logger output = %q", data)
	}
}

// TestBuilderRejectsInvalidZapLogging verifies enabled default logging rejects invalid rotation settings. TestBuilderRejectsInvalidZapLogging 验证启用默认日志时拒绝非法轮转配置。
func TestBuilderRejectsInvalidZapLogging(t *testing.T) {
	if _, err := NewBuilder().IsLog(true).LoggerMaxSize(-1).BuildInstance(); err == nil {
		t.Fatal("invalid logging configuration was accepted")
	}
}
