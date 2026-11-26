package dlog

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// LoggerConfig configures Zap output and Lumberjack's native rotation policy. LoggerConfig 配置 Zap 输出与 Lumberjack 原生轮转策略。
type LoggerConfig struct {
	// Path selects the directory for file output. Path 指定文件输出目录。
	Path string
	// Filename selects a fixed basename without directory separators or date templates. Filename 指定不含目录分隔符和日期模板的固定文件名。
	Filename string
	// Prefix is prepended to each message. Prefix 添加在每条消息之前。
	Prefix string
	// Level selects the minimum severity. Level 指定最低日志级别。
	Level LogLevel
	// TimeFormat uses Go's timestamp layout syntax. TimeFormat 使用 Go 时间布局语法。
	TimeFormat string
	// Stdout also writes to standard output. Stdout 同时写入标准输出。
	Stdout bool
	// StdoutOnly disables file output and initially enables stdout. StdoutOnly 禁用文件输出并在初始化时启用标准输出。
	StdoutOnly bool
	// MaxSize limits each file in MiB; zero selects the framework default. MaxSize 以 MiB 限制单个文件大小，零值使用框架默认值。
	MaxSize int
	// MaxBackups limits retained backups; zero disables the count limit. MaxBackups 限制保留的备份数量，零值表示不按数量限制。
	MaxBackups int
	// MaxAge limits backup age to 0–106751 days; zero disables age-based deletion. MaxAge 以天限制备份保留时间，范围为 0–106751，零值表示不按时间删除。
	MaxAge int
	// Compress enables gzip compression of rotated files. Compress 启用轮转文件的 gzip 压缩。
	Compress bool
	// LocalTime uses local time in backup filenames; otherwise Lumberjack uses UTC. LocalTime 在备份文件名中使用本地时间，否则 Lumberjack 使用 UTC。
	LocalTime bool
}

// DefaultLoggerConfig returns the bundled synchronous logging defaults. DefaultLoggerConfig 返回内置同步日志默认配置。
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Path: DefaultDirName, Filename: DefaultFilename,
		Prefix: DefaultPrefix, Level: LevelInfo,
		TimeFormat: DefaultTimeFormat, Stdout: true,
		MaxSize: DefaultMaxSize, MaxBackups: DefaultMaxBackups, MaxAge: DefaultMaxAge,
	}
}

// Clone returns an independent configuration copy. Clone 返回独立配置副本。
func (c *LoggerConfig) Clone() *LoggerConfig {
	if c == nil {
		return DefaultLoggerConfig()
	}
	copy := *c
	return &copy
}

// normalizeConfig fills unspecified scalar values without overriding explicit retention limits. normalizeConfig 补齐未指定的标量值，不覆盖显式保留策略。
func normalizeConfig(cfg *LoggerConfig) *LoggerConfig {
	c := cfg.Clone()
	if c.Path == "" {
		c.Path = DefaultDirName
	}
	if c.Filename == "" {
		c.Filename = DefaultFilename
	}
	if c.Prefix == "" {
		c.Prefix = DefaultPrefix
	}
	if c.Level == 0 {
		c.Level = LevelInfo
	}
	if c.TimeFormat == "" {
		c.TimeFormat = DefaultTimeFormat
	}
	if c.MaxSize == 0 {
		c.MaxSize = DefaultMaxSize
	}
	if c.StdoutOnly {
		c.Stdout = true
	}
	return c
}

// Validate checks configuration without creating directories or opening files. Validate 校验配置，不创建目录或打开文件。
func (c *LoggerConfig) Validate() error {
	cfg := normalizeConfig(c)
	if strings.TrimSpace(cfg.Path) == "" {
		return fmt.Errorf("dlog: Path must not be whitespace")
	}
	if strings.TrimSpace(cfg.Filename) == "" || cfg.Filename == "." || cfg.Filename == ".." || strings.ContainsAny(cfg.Filename, `/\{}`) {
		return fmt.Errorf("dlog: Filename must be a fixed basename without date templates")
	}
	if !validLevel(cfg.Level) {
		return fmt.Errorf("dlog: invalid Level %v", cfg.Level)
	}
	if strings.TrimSpace(cfg.TimeFormat) == "" {
		return fmt.Errorf("dlog: TimeFormat must not be whitespace")
	}
	if cfg.MaxSize < 0 || int64(cfg.MaxSize) > math.MaxInt64/(1024*1024) {
		return fmt.Errorf("dlog: MaxSize is outside the supported MiB range")
	}
	if cfg.MaxBackups < 0 || cfg.MaxAge < 0 {
		return fmt.Errorf("dlog: MaxBackups and MaxAge must not be negative")
	}
	if int64(cfg.MaxAge) > math.MaxInt64/int64(24*time.Hour) {
		return fmt.Errorf("dlog: MaxAge exceeds the supported retention duration")
	}
	return nil
}

// validLevel reports whether a framework severity is supported. validLevel 判断框架日志级别是否受支持。
func validLevel(level LogLevel) bool {
	return level >= LevelDebug && level <= LevelError
}
