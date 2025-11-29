package shutdown

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ShutdownHook represents a component that needs to perform cleanup during shutdown
type ShutdownHook interface {
	// Name returns the name of the hook for logging
	Name() string

	// Shutdown performs the cleanup operation
	Shutdown(ctx context.Context) error

	// Priority returns the priority of the hook (lower numbers run first)
	Priority() int
}

// ShutdownManager coordinates graceful shutdown of the application
type ShutdownManager interface {
	// RegisterHook registers a shutdown hook
	RegisterHook(hook ShutdownHook) error

	// Shutdown initiates graceful shutdown, executing all registered hooks
	Shutdown(ctx context.Context) error

	// NotifyShutdown returns a channel that is closed when shutdown is initiated
	NotifyShutdown() <-chan struct{}
}

// manager implements ShutdownManager
type manager struct {
	hooks           []ShutdownHook
	hooksMu         sync.RWMutex
	shutdownTimeout time.Duration
	shutdownOnce    sync.Once
	shutdownChan    chan struct{}
	isShuttingDown  bool
	shutdownMu      sync.RWMutex
}

// NewManager creates a new ShutdownManager with the specified timeout
func NewManager(shutdownTimeout time.Duration) ShutdownManager {
	return &manager{
		hooks:           make([]ShutdownHook, 0),
		shutdownTimeout: shutdownTimeout,
		shutdownChan:    make(chan struct{}),
		isShuttingDown:  false,
	}
}

// RegisterHook registers a shutdown hook
func (m *manager) RegisterHook(hook ShutdownHook) error {
	if hook == nil {
		return fmt.Errorf("hook cannot be nil")
	}

	m.hooksMu.Lock()
	defer m.hooksMu.Unlock()

	// Check if hook with same name already exists
	for _, h := range m.hooks {
		if h.Name() == hook.Name() {
			return fmt.Errorf("hook with name %s already registered", hook.Name())
		}
	}

	m.hooks = append(m.hooks, hook)

	// Sort hooks by priority (lower priority number runs first)
	sort.Slice(m.hooks, func(i, j int) bool {
		return m.hooks[i].Priority() < m.hooks[j].Priority()
	})

	return nil
}

// Shutdown initiates graceful shutdown
func (m *manager) Shutdown(ctx context.Context) error {
	// Check if already shutting down
	m.shutdownMu.Lock()
	if m.isShuttingDown {
		m.shutdownMu.Unlock()
		return nil // Already shutting down, return success
	}
	m.isShuttingDown = true
	m.shutdownMu.Unlock()

	// Close shutdown notification channel once
	m.shutdownOnce.Do(func() {
		close(m.shutdownChan)
	})

	// Create context with timeout if not already set
	shutdownCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(ctx, m.shutdownTimeout)
		defer cancel()
	}

	// Execute hooks in priority order
	m.hooksMu.RLock()
	hooks := make([]ShutdownHook, len(m.hooks))
	copy(hooks, m.hooks)
	m.hooksMu.RUnlock()

	var shutdownErrors []error

	for _, hook := range hooks {
		// Check if context is cancelled
		select {
		case <-shutdownCtx.Done():
			shutdownErrors = append(shutdownErrors, fmt.Errorf("shutdown timeout exceeded"))
			return combineErrors(shutdownErrors)
		default:
		}

		// Execute hook with timeout protection
		hookErr := m.executeHook(shutdownCtx, hook)
		if hookErr != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("hook %s failed: %w", hook.Name(), hookErr))
			// Continue with other hooks even if one fails
		}
	}

	if len(shutdownErrors) > 0 {
		return combineErrors(shutdownErrors)
	}

	return nil
}

// executeHook executes a single hook with error handling
func (m *manager) executeHook(ctx context.Context, hook ShutdownHook) error {
	// Create a channel to receive the result
	done := make(chan error, 1)

	go func() {
		done <- hook.Shutdown(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NotifyShutdown returns a channel that is closed when shutdown is initiated
func (m *manager) NotifyShutdown() <-chan struct{} {
	return m.shutdownChan
}

// combineErrors combines multiple errors into a single error
func combineErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	if len(errors) == 1 {
		return errors[0]
	}

	errMsg := fmt.Sprintf("%d errors occurred during shutdown:", len(errors))
	for i, err := range errors {
		errMsg += fmt.Sprintf("\n  %d. %v", i+1, err)
	}

	return fmt.Errorf("%s", errMsg)
}
