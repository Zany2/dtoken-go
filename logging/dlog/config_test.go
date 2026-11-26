package dlog

import (
	"math"
	"testing"
	"time"
)

// TestLoggerConfigDefaults verifies native units, zero retention limits, and independent configuration copies. TestLoggerConfigDefaults 验证原生单位、零值保留策略及独立配置副本。
func TestLoggerConfigDefaults(t *testing.T) {
	cfg := DefaultLoggerConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Filename != "DTOKEN.log" || cfg.MaxSize != 10 || cfg.MaxBackups != 10 || cfg.MaxAge != 7 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	clone := cfg.Clone()
	clone.Filename = "other.log"
	if cfg.Filename != "DTOKEN.log" {
		t.Fatal("Clone changed the source")
	}
	partial := normalizeConfig(&LoggerConfig{Path: t.TempDir(), MaxBackups: 0, MaxAge: 0})
	if partial.MaxSize != DefaultMaxSize || partial.MaxBackups != 0 || partial.MaxAge != 0 {
		t.Fatalf("zero retention limits were overridden: %+v", partial)
	}
	if err := (*LoggerConfig)(nil).Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestLoggerConfigValidation rejects invalid output paths, legacy templates, and unsupported rotation values. TestLoggerConfigValidation 拒绝非法输出路径、旧日期模板和不受支持的轮转值。
func TestLoggerConfigValidation(t *testing.T) {
	tests := []LoggerConfig{
		{Path: " "}, {Filename: " "}, {Filename: "."}, {Filename: ".."},
		{Filename: "nested/app.log"}, {Filename: `nested\app.log`}, {Filename: "APP_{Y}.log"},
		{Level: LogLevel(99)}, {TimeFormat: " "}, {MaxSize: -1}, {MaxBackups: -1}, {MaxAge: -1},
		{MaxAge: int(math.MaxInt64/int64(24*time.Hour)) + 1},
	}
	for _, cfg := range tests {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("Validate(%+v) accepted an invalid value", cfg)
		}
	}
}

// TestLoggerConfigRetentionBoundary verifies the largest duration-safe retention remains valid. TestLoggerConfigRetentionBoundary 验证不会导致时长溢出的最大保留天数仍然有效。
func TestLoggerConfigRetentionBoundary(t *testing.T) {
	cfg := &LoggerConfig{MaxAge: int(math.MaxInt64 / int64(24*time.Hour))}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
