package shutdown

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestShutdownManager_RequestCompletion tests Property 14: Graceful Shutdown Request Completion
// **Feature: production-readiness, Property 14: Graceful Shutdown Request Completion**
// **Validates: Requirements 9.2**
//
// Property: For any in-flight request when shutdown is initiated, the request must be
// allowed to complete (up to 30 second timeout) before connections are closed
func TestShutdownManager_RequestCompletion(t *testing.T) {
	tests := []struct {
		name              string
		requestDuration   time.Duration
		shutdownTimeout   time.Duration
		expectCompletion  bool
		expectError       bool
	}{
		{
			name:             "short request completes before timeout",
			requestDuration:  100 * time.Millisecond,
			shutdownTimeout:  1 * time.Second,
			expectCompletion: true,
			expectError:      false,
		},
		{
			name:             "request completes just before timeout",
			requestDuration:  900 * time.Millisecond,
			shutdownTimeout:  1 * time.Second,
			expectCompletion: true,
			expectError:      false,
		},
		{
			name:             "request exceeds timeout",
			requestDuration:  2 * time.Second,
			shutdownTimeout:  500 * time.Millisecond,
			expectCompletion: false,
			expectError:      true,
		},
		{
			name:             "multiple requests complete before timeout",
			requestDuration:  200 * time.Millisecond,
			shutdownTimeout:  1 * time.Second,
			expectCompletion: true,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.shutdownTimeout)

			// Track request completion
			requestCompleted := false
			var requestMu sync.Mutex

			// Register a hook that simulates an in-flight request
			requestHook := &testHook{
				name:     "request-handler",
				priority: 1,
				onShutdown: func(ctx context.Context) error {
					// Simulate request processing
					select {
					case <-time.After(tt.requestDuration):
						requestMu.Lock()
						requestCompleted = true
						requestMu.Unlock()
						return nil
					case <-ctx.Done():
						// Context cancelled before request completed
						return ctx.Err()
					}
				},
			}

			if err := manager.RegisterHook(requestHook); err != nil {
				t.Fatalf("Failed to register hook: %v", err)
			}

			// Initiate shutdown
			ctx := context.Background()
			shutdownErr := manager.Shutdown(ctx)

			requestMu.Lock()
			completed := requestCompleted
			requestMu.Unlock()

			if tt.expectCompletion && !completed {
				t.Errorf("Expected request to complete, but it didn't")
			}

			if tt.expectError && shutdownErr == nil {
				t.Errorf("Expected shutdown error due to timeout, but got nil")
			}

			if !tt.expectError && shutdownErr != nil {
				t.Errorf("Expected no shutdown error, but got: %v", shutdownErr)
			}
		})
	}
}

// TestShutdownManager_HookPriority tests that shutdown hooks are executed in priority order
func TestShutdownManager_HookPriority(t *testing.T) {
	manager := NewManager(5 * time.Second)

	executionOrder := []string{}
	var mu sync.Mutex

	// Register hooks with different priorities
	hook1 := &testHook{
		name:     "high-priority",
		priority: 1,
		onShutdown: func(ctx context.Context) error {
			mu.Lock()
			executionOrder = append(executionOrder, "high-priority")
			mu.Unlock()
			return nil
		},
	}

	hook2 := &testHook{
		name:     "medium-priority",
		priority: 5,
		onShutdown: func(ctx context.Context) error {
			mu.Lock()
			executionOrder = append(executionOrder, "medium-priority")
			mu.Unlock()
			return nil
		},
	}

	hook3 := &testHook{
		name:     "low-priority",
		priority: 10,
		onShutdown: func(ctx context.Context) error {
			mu.Lock()
			executionOrder = append(executionOrder, "low-priority")
			mu.Unlock()
			return nil
		},
	}

	if err := manager.RegisterHook(hook1); err != nil {
		t.Fatalf("Failed to register hook1: %v", err)
	}
	if err := manager.RegisterHook(hook2); err != nil {
		t.Fatalf("Failed to register hook2: %v", err)
	}
	if err := manager.RegisterHook(hook3); err != nil {
		t.Fatalf("Failed to register hook3: %v", err)
	}

	ctx := context.Background()
	if err := manager.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(executionOrder) != 3 {
		t.Fatalf("Expected 3 hooks to execute, got %d", len(executionOrder))
	}

	// Lower priority number should execute first
	if executionOrder[0] != "high-priority" {
		t.Errorf("Expected high-priority to execute first, got %s", executionOrder[0])
	}
	if executionOrder[1] != "medium-priority" {
		t.Errorf("Expected medium-priority to execute second, got %s", executionOrder[1])
	}
	if executionOrder[2] != "low-priority" {
		t.Errorf("Expected low-priority to execute third, got %s", executionOrder[2])
	}
}

// TestShutdownManager_HookError tests that shutdown continues even if a hook fails
func TestShutdownManager_HookError(t *testing.T) {
	manager := NewManager(5 * time.Second)

	hook1Executed := false
	hook2Executed := false
	hook3Executed := false

	hook1 := &testHook{
		name:     "hook1",
		priority: 1,
		onShutdown: func(ctx context.Context) error {
			hook1Executed = true
			return nil
		},
	}

	hook2 := &testHook{
		name:     "hook2",
		priority: 2,
		onShutdown: func(ctx context.Context) error {
			hook2Executed = true
			return errors.New("hook2 failed")
		},
	}

	hook3 := &testHook{
		name:     "hook3",
		priority: 3,
		onShutdown: func(ctx context.Context) error {
			hook3Executed = true
			return nil
		},
	}

	manager.RegisterHook(hook1)
	manager.RegisterHook(hook2)
	manager.RegisterHook(hook3)

	ctx := context.Background()
	err := manager.Shutdown(ctx)

	if err == nil {
		t.Error("Expected shutdown to return error when hook fails")
	}

	if !hook1Executed {
		t.Error("Expected hook1 to execute")
	}
	if !hook2Executed {
		t.Error("Expected hook2 to execute")
	}
	if !hook3Executed {
		t.Error("Expected hook3 to execute even after hook2 failed")
	}
}

// TestShutdownManager_ContextCancellation tests that shutdown respects context cancellation
func TestShutdownManager_ContextCancellation(t *testing.T) {
	manager := NewManager(30 * time.Second)

	slowHook := &testHook{
		name:     "slow-hook",
		priority: 1,
		onShutdown: func(ctx context.Context) error {
			select {
			case <-time.After(5 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	manager.RegisterHook(slowHook)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := manager.Shutdown(ctx)

	if err == nil {
		t.Error("Expected shutdown to return error due to context cancellation")
	}

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

// TestShutdownManager_NotifyShutdown tests that shutdown notification channel works
func TestShutdownManager_NotifyShutdown(t *testing.T) {
	manager := NewManager(5 * time.Second)

	notifyChan := manager.NotifyShutdown()

	// Start a goroutine that waits for shutdown notification
	notified := make(chan bool, 1)
	go func() {
		<-notifyChan
		notified <- true
	}()

	// Initiate shutdown
	ctx := context.Background()
	go manager.Shutdown(ctx)

	// Wait for notification
	select {
	case <-notified:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Expected to receive shutdown notification")
	}
}

// TestShutdownManager_MultipleShutdownCalls tests that multiple shutdown calls are handled safely
func TestShutdownManager_MultipleShutdownCalls(t *testing.T) {
	manager := NewManager(5 * time.Second)

	hook := &testHook{
		name:     "test-hook",
		priority: 1,
		onShutdown: func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	manager.RegisterHook(hook)

	ctx := context.Background()

	// Call shutdown multiple times concurrently
	var wg sync.WaitGroup
	errors := make([]error, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errors[idx] = manager.Shutdown(ctx)
		}(i)
	}

	wg.Wait()

	// At least one should succeed, others should either succeed or indicate already shut down
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	if successCount == 0 {
		t.Error("Expected at least one shutdown call to succeed")
	}
}

// testHook is a test implementation of ShutdownHook
type testHook struct {
	name       string
	priority   int
	onShutdown func(ctx context.Context) error
}

func (h *testHook) Name() string {
	return h.name
}

func (h *testHook) Priority() int {
	return h.priority
}

func (h *testHook) Shutdown(ctx context.Context) error {
	if h.onShutdown != nil {
		return h.onShutdown(ctx)
	}
	return nil
}
