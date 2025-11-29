package testutil

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

// NewTestLogger creates a new logger for testing
func NewTestLogger(t *testing.T) zerolog.Logger {
	// Create a logger that writes to stdout for testing
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	}

	logger := zerolog.New(output).
		With().
		Timestamp().
		Str("test", t.Name()).
		Logger()

	// Set log level based on environment
	if os.Getenv("TEST_LOG_LEVEL") == "debug" {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	return logger
}

// NewSilentTestLogger creates a silent logger for testing
func NewSilentTestLogger(t *testing.T) zerolog.Logger {
	logger := zerolog.New(zerolog.New(nil)).
		With().
		Timestamp().
		Str("test", t.Name()).
		Logger()

	return logger
}
