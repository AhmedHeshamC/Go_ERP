package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// New creates a new zerolog logger with the specified level and development mode
func New(level string, development bool) *zerolog.Logger {
	// Parse log level
	zerologLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(zerologLevel)

	// Configure console writer for development mode
	var output = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}

	if development {
		output.FormatLevel = func(i interface{}) string {
			return strings.ToUpper(i.(string))
		}
		output.FormatMessage = func(i interface{}) string {
			return fmt.Sprintf("| %s", i)
		}
		output.FormatFieldName = func(i interface{}) string {
			return fmt.Sprintf("%s:", i)
		}
		output.FormatFieldValue = func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("%s", i))
		}
	}

	// Create logger
	logger := zerolog.New(output).With().Timestamp().Logger()

	if development {
		logger = logger.With().Caller().Logger()
	}

	return &logger
}

// parseLogLevel converts a string log level to zerolog level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithContext adds context fields to the logger
func WithContext(logger *zerolog.Logger, fields map[string]interface{}) zerolog.Logger {
	ctx := logger.With()
	for key, value := range fields {
		ctx = ctx.Interface(key, value)
	}
	return ctx.Logger()
}

// WithRequestID adds a request ID to the logger
func WithRequestID(logger *zerolog.Logger, requestID string) zerolog.Logger {
	return logger.With().Str("request_id", requestID).Logger()
}

// WithUserID adds a user ID to the logger
func WithUserID(logger *zerolog.Logger, userID string) zerolog.Logger {
	return logger.With().Str("user_id", userID).Logger()
}

// WithError adds an error to the logger
func WithError(logger *zerolog.Logger, err error) zerolog.Logger {
	return logger.With().Err(err).Logger()
}

// WithModule adds a module name to the logger
func WithModule(logger *zerolog.Logger, module string) zerolog.Logger {
	return logger.With().Str("module", module).Logger()
}

// Default returns a default logger configuration
func Default() *zerolog.Logger {
	return New("info", true)
}

// Production returns a production-ready logger configuration
func Production() *zerolog.Logger {
	return New("info", false)
}

// Development returns a development logger configuration
func Development() *zerolog.Logger {
	return New("debug", true)
}

// Test returns a test logger configuration (silent output)
func Test() zerolog.Logger {
	return zerolog.Nop()
}