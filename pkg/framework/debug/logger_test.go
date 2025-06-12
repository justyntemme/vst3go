package debug

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	t.Run("BasicLogging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(&buf, "TEST", FlagLevel|FlagPrefix)
		
		logger.Info("Hello %s", "World")
		
		output := buf.String()
		if !strings.Contains(output, "[INFO]") {
			t.Error("Missing log level")
		}
		if !strings.Contains(output, "[TEST]") {
			t.Error("Missing prefix")
		}
		if !strings.Contains(output, "Hello World") {
			t.Error("Missing message")
		}
	})
	
	t.Run("LogLevels", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(&buf, "", FlagLevel)
		logger.SetLevel(LogLevelWarn)
		
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")
		
		output := buf.String()
		if strings.Contains(output, "debug message") {
			t.Error("Debug message should not be logged")
		}
		if strings.Contains(output, "info message") {
			t.Error("Info message should not be logged")
		}
		if !strings.Contains(output, "warn message") {
			t.Error("Warn message should be logged")
		}
		if !strings.Contains(output, "error message") {
			t.Error("Error message should be logged")
		}
	})
	
	t.Run("Disabled", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(&buf, "", DefaultFlags)
		logger.SetEnabled(false)
		
		logger.Info("should not appear")
		
		if buf.Len() > 0 {
			t.Error("Disabled logger should not write")
		}
	})
	
	t.Run("FileInfo", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(&buf, "", FlagShortFile|FlagLevel)
		
		logger.Info("test")
		
		output := buf.String()
		if !strings.Contains(output, ".go:") {
			t.Errorf("Missing file info in output: %s", output)
		}
	})
	
	t.Run("ConditionalLogging", func(t *testing.T) {
		var buf bytes.Buffer
		SetOutput(&buf)
		SetLevel(LogLevelDebug)
		
		DebugIf(true, "should appear")
		DebugIf(false, "should not appear")
		
		output := buf.String()
		if !strings.Contains(output, "should appear") {
			t.Error("Conditional true message missing")
		}
		if strings.Contains(output, "should not appear") {
			t.Error("Conditional false message should not appear")
		}
	})
}

func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelFatal, "FATAL"},
		{LogLevel(99), "UNKNOWN"},
	}
	
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("LogLevel.String() = %v, want %v", got, tt.expected)
		}
	}
}

func BenchmarkLogger(b *testing.B) {
	logger := New(bytes.NewBuffer(nil), "BENCH", DefaultFlags)
	
	b.Run("Enabled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("Benchmark message %d", i)
		}
	})
	
	b.Run("Disabled", func(b *testing.B) {
		logger.SetEnabled(false)
		for i := 0; i < b.N; i++ {
			logger.Info("Benchmark message %d", i)
		}
	})
	
	b.Run("BelowLevel", func(b *testing.B) {
		logger.SetEnabled(true)
		logger.SetLevel(LogLevelError)
		for i := 0; i < b.N; i++ {
			logger.Info("Benchmark message %d", i)
		}
	})
}