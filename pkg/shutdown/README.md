# Shutdown Manager

The shutdown package provides a graceful shutdown manager for coordinating the orderly shutdown of application components.

## Features

- **Priority-based Execution**: Shutdown hooks execute in priority order (lower numbers first)
- **Timeout Protection**: Configurable timeout for the entire shutdown process
- **Error Handling**: Continues executing remaining hooks even if one fails
- **Context Support**: Respects context cancellation for early termination
- **Thread-safe**: Safe for concurrent use
- **Notification Channel**: Provides a channel that closes when shutdown is initiated

## Usage

### Basic Setup

```go
import (
    "context"
    "time"
    "erpgo/pkg/shutdown"
)

// Create shutdown manager with 30 second timeout
shutdownMgr := shutdown.NewManager(30 * time.Second)

// Register shutdown hooks in priority order
httpHook := shutdown.NewHTTPServerHook(server, logger, 1)
shutdownMgr.RegisterHook(httpHook)

dbHook := shutdown.NewDatabaseHook(db.Close, logger, 2)
shutdownMgr.RegisterHook(dbHook)

cacheHook := shutdown.NewCacheHook(cache.Close, logger, 3)
shutdownMgr.RegisterHook(cacheHook)

// Wait for shutdown signal
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// Initiate graceful shutdown
ctx := context.Background()
if err := shutdownMgr.Shutdown(ctx); err != nil {
    logger.Error().Err(err).Msg("Shutdown completed with errors")
}
```

### Custom Shutdown Hooks

You can create custom shutdown hooks by implementing the `ShutdownHook` interface:

```go
type MyCustomHook struct {
    logger *zerolog.Logger
}

func (h *MyCustomHook) Name() string {
    return "my-custom-hook"
}

func (h *MyCustomHook) Priority() int {
    return 5 // Execute after priority 1-4
}

func (h *MyCustomHook) Shutdown(ctx context.Context) error {
    h.logger.Info().Msg("Performing custom cleanup...")
    
    // Your cleanup logic here
    
    return nil
}
```

Or use the `GenericHook` for simple cases:

```go
hook := shutdown.NewGenericHook("my-hook", 5, func(ctx context.Context) error {
    // Your cleanup logic here
    return nil
}, logger)

shutdownMgr.RegisterHook(hook)
```

### Built-in Hooks

The package provides several built-in hooks:

#### HTTPServerHook
Gracefully shuts down an HTTP server, waiting for in-flight requests to complete.

```go
hook := shutdown.NewHTTPServerHook(server, logger, 1)
```

#### DatabaseHook
Closes database connections cleanly.

```go
hook := shutdown.NewDatabaseHook(db.Close, logger, 2)
```

#### CacheHook
Closes cache connections (e.g., Redis).

```go
hook := shutdown.NewCacheHook(cache.Close, logger, 3)
```

#### GenericHook
A flexible hook for custom cleanup operations.

```go
hook := shutdown.NewGenericHook("custom", 4, cleanupFunc, logger)
```

### Priority Guidelines

Recommended priority ordering:

1. **Priority 1**: HTTP Server (stop accepting new requests)
2. **Priority 2**: Database connections (close after requests complete)
3. **Priority 3**: Cache connections (close after database)
4. **Priority 4+**: Other cleanup operations

Lower priority numbers execute first.

### Shutdown Notification

You can be notified when shutdown is initiated:

```go
shutdownChan := shutdownMgr.NotifyShutdown()

go func() {
    <-shutdownChan
    // Shutdown has been initiated
    // Update health checks to return unhealthy
}()
```

## Testing

The package includes comprehensive tests for:

- Request completion within timeout (Property 14)
- Hook priority ordering
- Error handling
- Context cancellation
- Multiple shutdown calls
- Shutdown notification

Run tests:

```bash
go test ./pkg/shutdown/...
```

## Requirements Validation

This implementation validates:

- **Requirement 9.1**: Stop accepting new requests immediately (via HTTP server hook)
- **Requirement 9.2**: Wait for in-flight requests to complete (up to 30 seconds)
- **Requirement 9.3**: Close database connections cleanly
- **Requirement 9.4**: Force-close remaining connections after timeout
- **Requirement 9.5**: Log shutdown progress and forced terminations

## Property-Based Testing

The implementation includes property-based tests that validate:

**Property 14: Graceful Shutdown Request Completion**
- For any in-flight request when shutdown is initiated, the request must be allowed to complete (up to 30 second timeout) before connections are closed
- Validates: Requirements 9.2

## Error Handling

- If a hook fails, the error is logged and shutdown continues with remaining hooks
- All hook errors are collected and returned as a combined error
- Context cancellation is respected and will terminate shutdown early
- Timeout errors are clearly indicated

## Thread Safety

The shutdown manager is thread-safe:

- Multiple goroutines can call `Shutdown()` concurrently
- Only the first call will execute the shutdown sequence
- Subsequent calls return immediately without error
- Hook registration is protected by mutex

## Logging

All shutdown operations are logged with structured logging:

- Shutdown initiation
- Each hook execution (start and completion)
- Hook execution duration
- Errors and warnings
- Forced terminations

## Best Practices

1. **Register hooks early**: Register all hooks during application initialization
2. **Use appropriate priorities**: Ensure hooks execute in the correct order
3. **Handle errors gracefully**: Don't panic in shutdown hooks
4. **Respect context**: Check context cancellation in long-running hooks
5. **Log progress**: Use structured logging to track shutdown progress
6. **Test shutdown**: Include shutdown testing in your test suite
