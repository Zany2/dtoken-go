// Package dlog adapts Zap and Lumberjack to the framework logging contracts. Package dlog 将 Zap 与 Lumberjack 适配到框架日志契约。
package dlog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/contract"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger writes synchronously through Zap and delegates file rotation to Lumberjack. Logger 通过 Zap 同步输出日志，并将文件轮转交给 Lumberjack。
type Logger struct {
	mu      sync.RWMutex       // mu coordinates writes, reconfiguration, and shutdown. mu 协调写入、配置变更与关闭。
	cfg     LoggerConfig       // cfg stores the active configuration. cfg 保存当前配置。
	backend *zap.Logger        // backend writes level-filtered messages. backend 输出经过级别过滤的日志。
	plain   *zap.Logger        // plain writes messages without a level label or threshold. plain 输出不带级别标签且不受阈值过滤的日志。
	level   zap.AtomicLevel    // level controls the Zap level threshold. level 控制 Zap 日志级别阈值。
	file    *lumberjack.Logger // file owns the rolling file writer. file 持有滚动文件写入器。
	closed  bool               // closed prevents writes after shutdown. closed 阻止关闭后继续写入。
}

// The logger satisfies the framework's output and lifecycle contracts. 日志器实现框架的输出与生命周期契约。
var _ contract.LogControl = (*Logger)(nil)

// NewLoggerWithConfig creates a synchronous logger; nil selects the default configuration. NewLoggerWithConfig 创建同步日志器，nil 表示使用默认配置。
func NewLoggerWithConfig(cfg *LoggerConfig) (*Logger, error) {
	logger := &Logger{}
	if err := logger.configure(cfg); err != nil {
		return nil, err
	}
	return logger, nil
}

// SetConfig replaces valid configuration and leaves the current outputs unchanged on failure. SetConfig 替换有效配置，失败时保留当前输出。
func (l *Logger) SetConfig(cfg *LoggerConfig) {
	_ = l.configure(cfg)
}

// configure validates and installs a complete output configuration. configure 校验并安装完整输出配置。
func (l *Logger) configure(cfg *LoggerConfig) error {
	if l == nil {
		return fmt.Errorf("dlog: nil logger")
	}
	config := normalizeConfig(cfg)
	if err := config.Validate(); err != nil {
		return err
	}

	// Serialize replacement with active writes so a closed writer cannot reopen. 串行化输出替换与正在进行的写入，避免已关闭的写入器被重新打开。
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return fmt.Errorf("dlog: logger is closed")
	}
	if !config.StdoutOnly {
		if err := os.MkdirAll(config.Path, 0755); err != nil {
			return fmt.Errorf("dlog: create log directory: %w", err)
		}

		// Check the file without truncating it before releasing the active output. 在释放当前输出前检查文件可写性，不截断已有内容。
		probe, err := os.OpenFile(filepath.Join(config.Path, config.Filename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("dlog: open log file: %w", err)
		}
		if err := probe.Close(); err != nil {
			return fmt.Errorf("dlog: close log file probe: %w", err)
		}
	}
	l.closeOutputs()
	l.cfg = *config
	l.level = zap.NewAtomicLevelAt(zapLevel(config.Level))
	if !config.StdoutOnly {
		l.file = &lumberjack.Logger{
			Filename:   filepath.Join(config.Path, config.Filename),
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
			LocalTime:  config.LocalTime,
		}
	}
	l.configureOutputs()
	return nil
}

// configureOutputs builds Zap cores while the caller holds the write lock. configureOutputs 在调用方持有写锁时构建 Zap 输出核心。
func (l *Logger) configureOutputs() {
	outputs := make([]zapcore.WriteSyncer, 0, 2)
	if l.file != nil {
		outputs = append(outputs, zapcore.AddSync(l.file))
	}
	if l.cfg.Stdout {
		// Do not fsync stdout pipes: Windows waits for readers to drain them. 不对标准输出管道执行 fsync，避免 Windows 等待读取端消费数据而阻塞。
		outputs = append(outputs, zapcore.Lock(zapcore.AddSync(struct{ io.Writer }{os.Stdout})))
	}
	if len(outputs) == 0 {
		l.backend, l.plain = zap.NewNop(), zap.NewNop()
		return
	}
	sink := zapcore.NewMultiWriteSyncer(outputs...)
	encoder := zapcore.EncoderConfig{
		TimeKey: "time", LevelKey: "level", MessageKey: "message",
		LineEnding:       zapcore.DefaultLineEnding,
		ConsoleSeparator: " ",
		EncodeTime: func(value time.Time, out zapcore.PrimitiveArrayEncoder) {
			out.AppendString(value.Format(l.cfg.TimeFormat))
		},
		EncodeLevel: func(value zapcore.Level, out zapcore.PrimitiveArrayEncoder) {
			out.AppendString("[" + value.CapitalString() + "]")
		},
	}
	l.backend = zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(encoder), sink, l.level))
	encoder.LevelKey = ""
	l.plain = zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(encoder), sink, zapcore.DebugLevel))
}

// enabled checks the threshold before evaluating user-supplied formatting methods. enabled 在执行用户提供的格式化方法前检查日志级别阈值。
func (l *Logger) enabled(level LogLevel) bool {
	if l == nil {
		return false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return !l.closed && l.backend != nil && (level == 0 || l.level.Enabled(zapLevel(level)))
}

// write formats ordinary arguments outside the lifecycle lock. write 在生命周期锁外格式化普通参数。
func (l *Logger) write(level LogLevel, args ...any) {
	if l.enabled(level) {
		l.log(level, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
	}
}

// writef formats a message outside the lifecycle lock. writef 在生命周期锁外格式化消息。
func (l *Logger) writef(level LogLevel, format string, args ...any) {
	if l.enabled(level) {
		l.log(level, fmt.Sprintf(format, args...))
	}
}

// log holds the read lock until synchronous output completes. log 持有读锁直到同步输出完成。
func (l *Logger) log(level LogLevel, message string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.closed {
		return
	}
	if level == 0 {
		l.plain.Info(l.cfg.Prefix + message)
		return
	}
	l.backend.Log(zapLevel(level), l.cfg.Prefix+message)
}

// SetLevel changes the Zap level threshold without rebuilding outputs. SetLevel 调整 Zap 日志级别阈值，不重建输出目标。
func (l *Logger) SetLevel(level LogLevel) {
	if l == nil || !validLevel(level) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed && l.backend != nil {
		l.cfg.Level = level
		l.level.SetLevel(zapLevel(level))
	}
}

// SetPrefix changes the prefix used by subsequent messages. SetPrefix 修改后续日志使用的前缀。
func (l *Logger) SetPrefix(prefix string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.cfg.Prefix = prefix
	}
}

// SetStdout toggles console output while retaining the current rolling file. SetStdout 切换控制台输出，并保留当前滚动文件。
func (l *Logger) SetStdout(enable bool) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed && l.backend != nil {
		l.cfg.Stdout = enable
		l.configureOutputs()
	}
}

// Flush synchronizes Zap outputs; Lumberjack writes synchronously and exposes no fsync operation. Flush 同步 Zap 输出；Lumberjack 同步写入且不提供 fsync 操作。
func (l *Logger) Flush() {
	if l == nil {
		return
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if !l.closed && l.backend != nil {
		_ = l.backend.Sync()
	}
}

// closeOutputs syncs Zap and closes the owned file while the caller holds the write lock. closeOutputs 在调用方持有写锁时同步 Zap 并关闭自有文件。
func (l *Logger) closeOutputs() {
	if l.backend != nil {
		_ = l.backend.Sync()
	}
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}

// Close waits for active writes and closes the owned file once, without closing stdout. Close 等待正在进行的写入并仅关闭一次自有文件，不关闭标准输出。
func (l *Logger) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		l.closeOutputs()
	}
}

// LogPath returns the configured log directory. LogPath 返回配置的日志目录。
func (l *Logger) LogPath() string {
	if l == nil {
		return ""
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cfg.Path
}

// DropCount always returns zero because synchronous output has no queue-overflow drops. DropCount 始终返回零，同步输出不存在队列溢出丢弃。
func (l *Logger) DropCount() uint64 { return 0 }

// zapLevel maps framework levels to Zap levels. zapLevel 将框架日志级别映射到 Zap 级别。
func zapLevel(level LogLevel) zapcore.Level {
	switch level {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Print writes an unfiltered message without a level label. Print 输出不受级别过滤且不带级别标签的消息。
func (l *Logger) Print(v ...any) { l.write(0, v...) }

// Printf formats an unfiltered message without a level label. Printf 格式化输出不受级别过滤且不带级别标签的消息。
func (l *Logger) Printf(format string, v ...any) { l.writef(0, format, v...) }

// Debug writes a debug message. Debug 输出调试消息。
func (l *Logger) Debug(v ...any) { l.write(LevelDebug, v...) }

// Debugf writes a formatted debug message. Debugf 输出格式化调试消息。
func (l *Logger) Debugf(format string, v ...any) { l.writef(LevelDebug, format, v...) }

// Info writes an informational message. Info 输出信息消息。
func (l *Logger) Info(v ...any) { l.write(LevelInfo, v...) }

// Infof writes a formatted informational message. Infof 输出格式化信息消息。
func (l *Logger) Infof(format string, v ...any) { l.writef(LevelInfo, format, v...) }

// Warn writes a warning message. Warn 输出警告消息。
func (l *Logger) Warn(v ...any) { l.write(LevelWarn, v...) }

// Warnf writes a formatted warning message. Warnf 输出格式化警告消息。
func (l *Logger) Warnf(format string, v ...any) { l.writef(LevelWarn, format, v...) }

// Error writes an error message. Error 输出错误消息。
func (l *Logger) Error(v ...any) { l.write(LevelError, v...) }

// Errorf writes a formatted error message. Errorf 输出格式化错误消息。
func (l *Logger) Errorf(format string, v ...any) { l.writef(LevelError, format, v...) }
