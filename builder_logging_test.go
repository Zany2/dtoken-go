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
		LoggerPrefix("[AUTH] ").LoggerLevel(dlog.LevelWarn).
		LoggerMaxSize(1).LoggerMaxBackups(2).LoggerMaxAge(3).
		LoggerCompress(true).LoggerLocalTime(true).Build()
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
	if !strings.Contains(string(data), "[WARN] [AUTH] visible-warning") || strings.Contains(string(data), "filtered-info") {
		t.Fatalf("builder logger output = %q", data)
	}
}

// TestBuilderRejectsInvalidZapLogging verifies invalid native rotation settings fail before assembly. TestBuilderRejectsInvalidZapLogging 验证非法原生轮转配置在装配前被拒绝。
func TestBuilderRejectsInvalidZapLogging(t *testing.T) {
	if _, err := NewBuilder().LoggerMaxSize(-1).Build(); err == nil {
		t.Fatal("invalid logging configuration was accepted")
	}
}
