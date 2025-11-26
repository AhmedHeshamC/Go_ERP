package shutdown

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestHTTPServerHook(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("SuccessfulShutdown", func(t *testing.T) {
		server := &http.Server{Addr: ":0"}
		hook := NewHTTPServerHook(server, &logger, 1)

		if hook.Name() != "http-server" {
			t.Errorf("Expected name 'http-server', got %s", hook.Name())
		}

		if hook.Priority() != 1 {
			t.Errorf("Expected priority 1, got %d", hook.Priority())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Shutdown should succeed even if server wasn't started
		if err := hook.Shutdown(ctx); err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("ShutdownWithTimeout", func(t *testing.T) {
		server := &http.Server{Addr: ":0"}
		hook := NewHTTPServerHook(server, &logger, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Wait for context to timeout
		time.Sleep(2 * time.Millisecond)

		// Shutdown may fail due to timeout
		_ = hook.Shutdown(ctx)
	})
}

func TestDatabaseHook(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("SuccessfulShutdown", func(t *testing.T) {
		closed := false
		closeFunc := func() error {
			closed = true
			return nil
		}

		hook := NewDatabaseHook(closeFunc, &logger, 2)

		if hook.Name() != "database" {
			t.Errorf("Expected name 'database', got %s", hook.Name())
		}

		if hook.Priority() != 2 {
			t.Errorf("Expected priority 2, got %d", hook.Priority())
		}

		ctx := context.Background()
		if err := hook.Shutdown(ctx); err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !closed {
			t.Error("Expected database to be closed")
		}
	})

	t.Run("ShutdownWithError", func(t *testing.T) {
		expectedErr := errors.New("database close error")
		closeFunc := func() error {
			return expectedErr
		}

		hook := NewDatabaseHook(closeFunc, &logger, 2)

		ctx := context.Background()
		err := hook.Shutdown(ctx)

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("ShutdownWithTimeout", func(t *testing.T) {
		closeFunc := func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		}

		hook := NewDatabaseHook(closeFunc, &logger, 2)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := hook.Shutdown(ctx)

		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	})
}

func TestCacheHook(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("SuccessfulShutdown", func(t *testing.T) {
		closed := false
		closeFunc := func() error {
			closed = true
			return nil
		}

		hook := NewCacheHook(closeFunc, &logger, 3)

		if hook.Name() != "cache" {
			t.Errorf("Expected name 'cache', got %s", hook.Name())
		}

		if hook.Priority() != 3 {
			t.Errorf("Expected priority 3, got %d", hook.Priority())
		}

		ctx := context.Background()
		if err := hook.Shutdown(ctx); err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !closed {
			t.Error("Expected cache to be closed")
		}
	})

	t.Run("ShutdownWithError", func(t *testing.T) {
		expectedErr := errors.New("cache close error")
		closeFunc := func() error {
			return expectedErr
		}

		hook := NewCacheHook(closeFunc, &logger, 3)

		ctx := context.Background()
		err := hook.Shutdown(ctx)

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("ShutdownWithTimeout", func(t *testing.T) {
		closeFunc := func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		}

		hook := NewCacheHook(closeFunc, &logger, 3)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := hook.Shutdown(ctx)

		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	})
}

func TestGenericHook(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("SuccessfulShutdown", func(t *testing.T) {
		executed := false
		shutdownFunc := func(ctx context.Context) error {
			executed = true
			return nil
		}

		hook := NewGenericHook("custom-hook", 5, shutdownFunc, &logger)

		if hook.Name() != "custom-hook" {
			t.Errorf("Expected name 'custom-hook', got %s", hook.Name())
		}

		if hook.Priority() != 5 {
			t.Errorf("Expected priority 5, got %d", hook.Priority())
		}

		ctx := context.Background()
		if err := hook.Shutdown(ctx); err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !executed {
			t.Error("Expected shutdown function to be executed")
		}
	})

	t.Run("ShutdownWithError", func(t *testing.T) {
		expectedErr := errors.New("custom shutdown error")
		shutdownFunc := func(ctx context.Context) error {
			return expectedErr
		}

		hook := NewGenericHook("custom-hook", 5, shutdownFunc, &logger)

		ctx := context.Background()
		err := hook.Shutdown(ctx)

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("ShutdownWithSlowOperation", func(t *testing.T) {
		shutdownFunc := func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		}

		hook := NewGenericHook("slow-hook", 5, shutdownFunc, &logger)

		ctx := context.Background()
		start := time.Now()
		err := hook.Shutdown(ctx)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if duration < 50*time.Millisecond {
			t.Errorf("Expected duration >= 50ms, got %v", duration)
		}
	})
}

func TestCombineErrors(t *testing.T) {
	t.Run("NoErrors", func(t *testing.T) {
		err := combineErrors([]error{})
		if err != nil {
			t.Errorf("Expected nil, got %v", err)
		}
	})

	t.Run("SingleError", func(t *testing.T) {
		singleErr := errors.New("single error")
		err := combineErrors([]error{singleErr})
		if err != singleErr {
			t.Errorf("Expected %v, got %v", singleErr, err)
		}
	})

	t.Run("MultipleErrors", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		err3 := errors.New("error 3")

		err := combineErrors([]error{err1, err2, err3})
		if err == nil {
			t.Fatal("Expected combined error, got nil")
		}

		errMsg := err.Error()
		if errMsg == "" {
			t.Error("Expected non-empty error message")
		}
	})
}

func TestShutdownHooksIntegration(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewManager(5 * time.Second)

	// Register multiple hooks
	dbClosed := false
	dbHook := NewDatabaseHook(func() error {
		dbClosed = true
		return nil
	}, &logger, 1)

	cacheClosed := false
	cacheHook := NewCacheHook(func() error {
		cacheClosed = true
		return nil
	}, &logger, 2)

	customExecuted := false
	customHook := NewGenericHook("custom", 3, func(ctx context.Context) error {
		customExecuted = true
		return nil
	}, &logger)

	if err := manager.RegisterHook(dbHook); err != nil {
		t.Fatalf("Failed to register database hook: %v", err)
	}

	if err := manager.RegisterHook(cacheHook); err != nil {
		t.Fatalf("Failed to register cache hook: %v", err)
	}

	if err := manager.RegisterHook(customHook); err != nil {
		t.Fatalf("Failed to register custom hook: %v", err)
	}

	// Execute shutdown
	ctx := context.Background()
	if err := manager.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify all hooks executed
	if !dbClosed {
		t.Error("Expected database to be closed")
	}

	if !cacheClosed {
		t.Error("Expected cache to be closed")
	}

	if !customExecuted {
		t.Error("Expected custom hook to be executed")
	}
}
