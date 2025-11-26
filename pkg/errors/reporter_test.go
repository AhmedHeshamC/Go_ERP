package errors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReporter(t *testing.T) {
	t.Run("creates reporter with default config", func(t *testing.T) {
		logger := zerolog.Nop()
		reporter, err := NewReporter(nil, &logger)
		require.NoError(t, err)
		assert.NotNil(t, reporter)
		assert.NotNil(t, reporter.config)
	})

	t.Run("creates reporter with custom config", func(t *testing.T) {
		logger := zerolog.Nop()
		config := DefaultConfig()
		config.Environment = "test"
		reporter, err := NewReporter(config, &logger)
		require.NoError(t, err)
		assert.Equal(t, "test", reporter.config.Environment)
	})
}

func TestReporter_Report(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.AsyncReporting = false
	reporter, _ := NewReporter(config, &logger)

	t.Run("reports error successfully", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("test error")
		reporter.Report(ctx, err, SeverityError, ErrorTypeSystem, "test message", nil, nil)
		
		stats := reporter.GetStats()
		assert.Greater(t, stats["buffer_size"].(int), 0)
	})

	t.Run("respects enabled flag", func(t *testing.T) {
		config := DefaultConfig()
		config.Enabled = false
		config.AsyncReporting = false
		disabledReporter, _ := NewReporter(config, &logger)
		
		ctx := context.Background()
		err := errors.New("test error")
		disabledReporter.Report(ctx, err, SeverityError, ErrorTypeSystem, "test", nil, nil)
		
		stats := disabledReporter.GetStats()
		assert.Equal(t, 0, stats["buffer_size"].(int))
	})
}

func TestReporter_ReportHTTPError(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.AsyncReporting = false
	reporter, _ := NewReporter(config, &logger)

	t.Run("reports HTTP error", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("http error")
		reporter.ReportHTTPError(ctx, err, 500, "internal server error", nil)
		
		stats := reporter.GetStats()
		assert.Greater(t, stats["buffer_size"].(int), 0)
	})

	t.Run("ignores configured status codes", func(t *testing.T) {
		config := DefaultConfig()
		config.IgnoreStatusCodes = []int{404}
		config.AsyncReporting = false
		reporter, _ := NewReporter(config, &logger)
		
		ctx := context.Background()
		err := errors.New("not found")
		reporter.ReportHTTPError(ctx, err, 404, "not found", nil)
		
		stats := reporter.GetStats()
		assert.Equal(t, 0, stats["buffer_size"].(int))
	})
}

func TestReporter_ReportPanic(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.AsyncReporting = false
	reporter, _ := NewReporter(config, &logger)

	t.Run("reports panic", func(t *testing.T) {
		ctx := context.Background()
		recovered := "panic: something went wrong"
		stack := []byte("goroutine 1 [running]:\nmain.main()\n\t/path/to/file.go:10 +0x123")
		
		reporter.ReportPanic(ctx, recovered, stack, nil)
		
		stats := reporter.GetStats()
		assert.Greater(t, stats["buffer_size"].(int), 0)
	})
}

func TestReporter_GetStats(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.AsyncReporting = false
	reporter, _ := NewReporter(config, &logger)

	stats := reporter.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "enabled")
	assert.Contains(t, stats, "environment")
	assert.Contains(t, stats, "buffer_size")
	assert.Contains(t, stats, "unique_errors")
}

func TestMapStatusCodeToSeverity(t *testing.T) {
	logger := zerolog.Nop()
	reporter, _ := NewReporter(DefaultConfig(), &logger)

	tests := []struct {
		statusCode int
		expected   Severity
	}{
		{200, SeverityDebug},
		{300, SeverityInfo},
		{400, SeverityWarning},
		{500, SeverityError},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			severity := reporter.mapStatusCodeToSeverity(tt.statusCode)
			assert.Equal(t, tt.expected, severity)
		})
	}
}

func TestMapStatusCodeToErrorType(t *testing.T) {
	logger := zerolog.Nop()
	reporter, _ := NewReporter(DefaultConfig(), &logger)

	tests := []struct {
		statusCode int
		expected   ErrorType
	}{
		{400, ErrorTypeValidation},
		{401, ErrorTypeAuthentication},
		{403, ErrorTypeAuthorization},
		{429, ErrorTypeRateLimit},
		{500, ErrorTypeSystem},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			errorType := reporter.mapStatusCodeToErrorType(tt.statusCode)
			assert.Equal(t, tt.expected, errorType)
		})
	}
}

func TestSanitizer(t *testing.T) {
	fields := []string{"password", "token", "secret"}
	sanitizer := newSanitizer(fields)

	t.Run("identifies sensitive fields", func(t *testing.T) {
		assert.True(t, sanitizer.shouldSanitize("password"))
		assert.True(t, sanitizer.shouldSanitize("user_password"))
		assert.True(t, sanitizer.shouldSanitize("api_token"))
		assert.False(t, sanitizer.shouldSanitize("username"))
	})
}

func TestSanitizeData(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.EnableSanitization = true
	reporter, _ := NewReporter(config, &logger)

	t.Run("sanitizes map data", func(t *testing.T) {
		data := map[string]interface{}{
			"username": "john",
			"password": "secret123",
			"email":    "john@example.com",
		}
		
		sanitized := reporter.sanitizeData(data)
		sanitizedMap := sanitized.(map[string]interface{})
		
		assert.Equal(t, "john", sanitizedMap["username"])
		assert.Equal(t, "[REDACTED]", sanitizedMap["password"])
		assert.Equal(t, "john@example.com", sanitizedMap["email"])
	})

	t.Run("sanitizes nested data", func(t *testing.T) {
		data := map[string]interface{}{
			"user": map[string]interface{}{
				"name":     "john",
				"password": "secret123",
			},
		}
		
		sanitized := reporter.sanitizeData(data)
		sanitizedMap := sanitized.(map[string]interface{})
		userMap := sanitizedMap["user"].(map[string]interface{})
		
		assert.Equal(t, "john", userMap["name"])
		assert.Equal(t, "[REDACTED]", userMap["password"])
	})
}

func TestGenerateFingerprint(t *testing.T) {
	logger := zerolog.Nop()
	reporter, _ := NewReporter(DefaultConfig(), &logger)

	t.Run("generates consistent fingerprint", func(t *testing.T) {
		report1 := &ErrorReport{
			Type:  ErrorTypeSystem,
			Error: "test error",
			StackFrames: []StackFrame{
				{Function: "main", File: "main.go", Line: 10},
			},
		}
		
		report2 := &ErrorReport{
			Type:  ErrorTypeSystem,
			Error: "test error",
			StackFrames: []StackFrame{
				{Function: "main", File: "main.go", Line: 10},
			},
		}
		
		fp1 := reporter.generateFingerprint(report1)
		fp2 := reporter.generateFingerprint(report2)
		
		assert.Equal(t, fp1, fp2)
	})

	t.Run("generates different fingerprints for different errors", func(t *testing.T) {
		report1 := &ErrorReport{
			Type:  ErrorTypeSystem,
			Error: "error 1",
		}
		
		report2 := &ErrorReport{
			Type:  ErrorTypeSystem,
			Error: "error 2",
		}
		
		fp1 := reporter.generateFingerprint(report1)
		fp2 := reporter.generateFingerprint(report2)
		
		assert.NotEqual(t, fp1, fp2)
	})
}

func TestUpdateErrorCount(t *testing.T) {
	logger := zerolog.Nop()
	reporter, _ := NewReporter(DefaultConfig(), &logger)

	t.Run("tracks error occurrences", func(t *testing.T) {
		fingerprint := "test-fingerprint"
		
		count1 := reporter.updateErrorCount(fingerprint)
		assert.Equal(t, 1, count1)
		
		count2 := reporter.updateErrorCount(fingerprint)
		assert.Equal(t, 2, count2)
		
		count3 := reporter.updateErrorCount(fingerprint)
		assert.Equal(t, 3, count3)
	})
}

func TestGetPlatformInfo(t *testing.T) {
	logger := zerolog.Nop()
	reporter, _ := NewReporter(DefaultConfig(), &logger)

	info := reporter.getPlatformInfo()
	assert.NotNil(t, info)
	assert.NotEmpty(t, info.OS)
	assert.NotEmpty(t, info.Architecture)
	assert.NotEmpty(t, info.Runtime)
	assert.NotEmpty(t, info.Version)
}

func TestCaptureStackFrames(t *testing.T) {
	logger := zerolog.Nop()
	reporter, _ := NewReporter(DefaultConfig(), &logger)

	frames := reporter.captureStackFrames()
	assert.NotEmpty(t, frames)
	
	for _, frame := range frames {
		assert.NotEmpty(t, frame.Function)
		assert.NotEmpty(t, frame.File)
		assert.Greater(t, frame.Line, 0)
	}
}

func TestGlobalReporter(t *testing.T) {
	t.Run("initializes global reporter", func(t *testing.T) {
		logger := zerolog.Nop()
		config := DefaultConfig()
		err := InitGlobalReporter(config, &logger)
		require.NoError(t, err)
		
		reporter := GetGlobalReporter()
		assert.NotNil(t, reporter)
	})

	t.Run("creates default reporter if not initialized", func(t *testing.T) {
		globalReporter = nil
		reporter := GetGlobalReporter()
		assert.NotNil(t, reporter)
	})
}

func TestGlobalConvenienceFunctions(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.AsyncReporting = false
	InitGlobalReporter(config, &logger)

	t.Run("Report function works", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("test error")
		Report(ctx, err, SeverityError, ErrorTypeSystem, "test", nil, nil)
		
		reporter := GetGlobalReporter()
		stats := reporter.GetStats()
		assert.Greater(t, stats["buffer_size"].(int), 0)
	})

	t.Run("ReportHTTPError function works", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("http error")
		ReportHTTPError(ctx, err, 500, "error", nil)
		// Should not panic
	})

	t.Run("ReportPanic function works", func(t *testing.T) {
		ctx := context.Background()
		ReportPanic(ctx, "panic", []byte("stack"), nil)
		// Should not panic
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.True(t, config.Enabled)
	assert.Equal(t, "development", config.Environment)
	assert.Equal(t, 1.0, config.SampleRate)
	assert.True(t, config.AsyncReporting)
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()
	assert.False(t, config.Debug)
	assert.Equal(t, 0.1, config.SampleRate)
	assert.Equal(t, SeverityError, config.MinSeverity)
	assert.False(t, config.IncludeUserData)
	assert.True(t, config.EnableSanitization)
}

func TestAsyncReporting(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.AsyncReporting = true
	reporter, _ := NewReporter(config, &logger)

	ctx := context.Background()
	err := errors.New("async test error")
	reporter.Report(ctx, err, SeverityError, ErrorTypeSystem, "test", nil, nil)

	// Give worker time to process
	time.Sleep(100 * time.Millisecond)

	stats := reporter.GetStats()
	assert.GreaterOrEqual(t, stats["buffer_size"].(int), 0)
}

func TestSanitizeContext(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.EnableSanitization = true
	reporter, _ := NewReporter(config, &logger)

	ctx := &Context{
		Tags: map[string]string{
			"username": "john",
			"password": "secret",
		},
		Extra: map[string]interface{}{
			"api_key": "12345",
			"data":    "normal",
		},
	}

	reporter.sanitizeContext(ctx)

	assert.Equal(t, "[REDACTED]", ctx.Tags["password"])
	assert.Equal(t, "john", ctx.Tags["username"])
	assert.Equal(t, "[REDACTED]", ctx.Extra["api_key"])
	assert.Equal(t, "normal", ctx.Extra["data"])
}
