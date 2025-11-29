package shutdown

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// HTTPServerHook handles graceful shutdown of HTTP server
type HTTPServerHook struct {
	server   *http.Server
	logger   *zerolog.Logger
	priority int
}

// NewHTTPServerHook creates a new HTTP server shutdown hook
func NewHTTPServerHook(server *http.Server, logger *zerolog.Logger, priority int) *HTTPServerHook {
	return &HTTPServerHook{
		server:   server,
		logger:   logger,
		priority: priority,
	}
}

func (h *HTTPServerHook) Name() string {
	return "http-server"
}

func (h *HTTPServerHook) Priority() int {
	return h.priority
}

func (h *HTTPServerHook) Shutdown(ctx context.Context) error {
	h.logger.Info().Msg("Shutting down HTTP server...")

	if err := h.server.Shutdown(ctx); err != nil {
		h.logger.Error().Err(err).Msg("HTTP server shutdown failed")
		return fmt.Errorf("http server shutdown failed: %w", err)
	}

	h.logger.Info().Msg("HTTP server shut down successfully")
	return nil
}

// DatabaseHook handles graceful shutdown of database connections
type DatabaseHook struct {
	closeFunc func() error
	logger    *zerolog.Logger
	priority  int
}

// NewDatabaseHook creates a new database shutdown hook
func NewDatabaseHook(closeFunc func() error, logger *zerolog.Logger, priority int) *DatabaseHook {
	return &DatabaseHook{
		closeFunc: closeFunc,
		logger:    logger,
		priority:  priority,
	}
}

func (h *DatabaseHook) Name() string {
	return "database"
}

func (h *DatabaseHook) Priority() int {
	return h.priority
}

func (h *DatabaseHook) Shutdown(ctx context.Context) error {
	h.logger.Info().Msg("Closing database connections...")

	// Create a channel to handle the close operation
	done := make(chan error, 1)
	go func() {
		done <- h.closeFunc()
	}()

	select {
	case err := <-done:
		if err != nil {
			h.logger.Error().Err(err).Msg("Database shutdown failed")
			return fmt.Errorf("database shutdown failed: %w", err)
		}
		h.logger.Info().Msg("Database connections closed successfully")
		return nil
	case <-ctx.Done():
		h.logger.Warn().Msg("Database shutdown timed out")
		return ctx.Err()
	}
}

// CacheHook handles graceful shutdown of cache connections
type CacheHook struct {
	closeFunc func() error
	logger    *zerolog.Logger
	priority  int
}

// NewCacheHook creates a new cache shutdown hook
func NewCacheHook(closeFunc func() error, logger *zerolog.Logger, priority int) *CacheHook {
	return &CacheHook{
		closeFunc: closeFunc,
		logger:    logger,
		priority:  priority,
	}
}

func (h *CacheHook) Name() string {
	return "cache"
}

func (h *CacheHook) Priority() int {
	return h.priority
}

func (h *CacheHook) Shutdown(ctx context.Context) error {
	h.logger.Info().Msg("Closing cache connections...")

	// Create a channel to handle the close operation
	done := make(chan error, 1)
	go func() {
		done <- h.closeFunc()
	}()

	select {
	case err := <-done:
		if err != nil {
			h.logger.Error().Err(err).Msg("Cache shutdown failed")
			return fmt.Errorf("cache shutdown failed: %w", err)
		}
		h.logger.Info().Msg("Cache connections closed successfully")
		return nil
	case <-ctx.Done():
		h.logger.Warn().Msg("Cache shutdown timed out")
		return ctx.Err()
	}
}

// GenericHook is a generic shutdown hook for custom cleanup operations
type GenericHook struct {
	name         string
	priority     int
	shutdownFunc func(ctx context.Context) error
	logger       *zerolog.Logger
}

// NewGenericHook creates a new generic shutdown hook
func NewGenericHook(name string, priority int, shutdownFunc func(ctx context.Context) error, logger *zerolog.Logger) *GenericHook {
	return &GenericHook{
		name:         name,
		priority:     priority,
		shutdownFunc: shutdownFunc,
		logger:       logger,
	}
}

func (h *GenericHook) Name() string {
	return h.name
}

func (h *GenericHook) Priority() int {
	return h.priority
}

func (h *GenericHook) Shutdown(ctx context.Context) error {
	h.logger.Info().Str("hook", h.name).Msg("Executing shutdown hook...")

	start := time.Now()
	err := h.shutdownFunc(ctx)
	duration := time.Since(start)

	if err != nil {
		h.logger.Error().
			Err(err).
			Str("hook", h.name).
			Dur("duration", duration).
			Msg("Shutdown hook failed")
		return err
	}

	h.logger.Info().
		Str("hook", h.name).
		Dur("duration", duration).
		Msg("Shutdown hook completed successfully")
	return nil
}
