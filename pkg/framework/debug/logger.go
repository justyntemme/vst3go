// Package debug provides debugging utilities for VST3 plugin development.
package debug

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	// LogLevelDebug is for detailed debugging information.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is for general informational messages.
	LogLevelInfo
	// LogLevelWarn is for warning messages.
	LogLevelWarn
	// LogLevelError is for error messages.
	LogLevelError
	// LogLevelFatal is for fatal errors that should terminate the plugin.
	LogLevelFatal
	// LogLevelOff disables all logging.
	LogLevelOff
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging for VST3 plugins.
type Logger struct {
	mu          sync.Mutex
	output      io.Writer
	level       LogLevel
	prefix      string
	flags       int
	enabled     bool
	includeTime bool
	includeLine bool
}

// Flags for logger output formatting.
const (
	FlagTime     = 1 << iota // Include timestamp
	FlagShortFile            // Include short file name and line number
	FlagLongFile             // Include full file path and line number
	FlagLevel                // Include log level
	FlagPrefix               // Include prefix
)

// DefaultFlags are the default formatting flags.
const DefaultFlags = FlagTime | FlagShortFile | FlagLevel | FlagPrefix

var (
	// defaultLogger is the global logger instance.
	defaultLogger *Logger
	once          sync.Once
)

// init initializes the default logger.
func init() {
	defaultLogger = New(os.Stderr, "", DefaultFlags)
	defaultLogger.SetLevel(LogLevelInfo)
}

// New creates a new logger instance.
func New(output io.Writer, prefix string, flags int) *Logger {
	return &Logger{
		output:      output,
		prefix:      prefix,
		flags:       flags,
		level:       LogLevelInfo,
		enabled:     true,
		includeTime: flags&FlagTime != 0,
		includeLine: flags&(FlagShortFile|FlagLongFile) != 0,
	}
}

// NewFileLogger creates a logger that writes to a file.
func NewFileLogger(filename, prefix string, flags int) (*Logger, error) {
	// Create log directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Open log file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	
	return New(file, prefix, flags), nil
}

// SetOutput sets the output destination for the logger.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetPrefix sets the logger prefix.
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// SetFlags sets the output formatting flags.
func (l *Logger) SetFlags(flags int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flags = flags
	l.includeTime = flags&FlagTime != 0
	l.includeLine = flags&(FlagShortFile|FlagLongFile) != 0
}

// SetEnabled enables or disables the logger.
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// IsEnabled returns whether the logger is enabled.
func (l *Logger) IsEnabled() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.enabled
}

// log writes a log message at the specified level.
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if !l.enabled || level < l.level {
		return
	}
	
	// Build the log message
	var sb strings.Builder
	
	// Add timestamp
	if l.flags&FlagTime != 0 {
		sb.WriteString(time.Now().Format("2006-01-02 15:04:05.000 "))
	}
	
	// Add log level
	if l.flags&FlagLevel != 0 {
		sb.WriteString(fmt.Sprintf("[%s] ", level.String()))
	}
	
	// Add prefix
	if l.flags&FlagPrefix != 0 && l.prefix != "" {
		sb.WriteString(fmt.Sprintf("[%s] ", l.prefix))
	}
	
	// Add file and line number
	if l.flags&(FlagShortFile|FlagLongFile) != 0 {
		_, file, line, ok := runtime.Caller(2) // Skip 2 frames: log() and Debug/Info/etc
		if ok {
			if l.flags&FlagShortFile != 0 {
				file = filepath.Base(file)
			}
			sb.WriteString(fmt.Sprintf("%s:%d: ", file, line))
		}
	}
	
	// Add the actual message
	msg := fmt.Sprintf(format, args...)
	sb.WriteString(msg)
	
	// Ensure newline
	if !strings.HasSuffix(msg, "\n") {
		sb.WriteString("\n")
	}
	
	// Write to output
	l.output.Write([]byte(sb.String()))
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogLevelDebug, format, args...)
}

// Info logs an informational message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogLevelInfo, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogLevelWarn, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogLevelError, format, args...)
}

// Fatal logs a fatal error message and panics.
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LogLevelFatal, format, args...)
	panic(fmt.Sprintf(format, args...))
}

// Global logger functions

// Default returns the default logger instance.
func Default() *Logger {
	return defaultLogger
}

// SetOutput sets the output destination for the default logger.
func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// SetLevel sets the minimum log level for the default logger.
func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// SetPrefix sets the prefix for the default logger.
func SetPrefix(prefix string) {
	defaultLogger.SetPrefix(prefix)
}

// SetFlags sets the output formatting flags for the default logger.
func SetFlags(flags int) {
	defaultLogger.SetFlags(flags)
}

// SetEnabled enables or disables the default logger.
func SetEnabled(enabled bool) {
	defaultLogger.SetEnabled(enabled)
}

// Debug logs a debug message using the default logger.
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs an informational message using the default logger.
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs a warning message using the default logger.
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs an error message using the default logger.
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs a fatal error message using the default logger and panics.
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// Conditional logging helpers

// DebugIf logs a debug message if the condition is true.
func DebugIf(condition bool, format string, args ...interface{}) {
	if condition {
		defaultLogger.Debug(format, args...)
	}
}

// WarnIf logs a warning message if the condition is true.
func WarnIf(condition bool, format string, args ...interface{}) {
	if condition {
		defaultLogger.Warn(format, args...)
	}
}

// ErrorIf logs an error message if the condition is true.
func ErrorIf(condition bool, format string, args ...interface{}) {
	if condition {
		defaultLogger.Error(format, args...)
	}
}