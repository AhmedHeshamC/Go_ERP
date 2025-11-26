package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus represents the health status of the system
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck interface defines a health check
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
	Timeout() time.Duration
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

// HealthStatus represents the overall health status
type HealthReport struct {
	Status    HealthStatus            `json:"status"`
	Checks    map[string]CheckResult  `json:"checks"`
	Timestamp time.Time               `json:"timestamp"`
}

// HTTPStatusCode returns the appropriate HTTP status code for the health status
func (h *HealthReport) HTTPStatusCode() int {
	switch h.Status {
	case StatusHealthy:
		return 200
	case StatusDegraded:
		return 200 // Degraded still returns 200 but with warning
	case StatusUnhealthy:
		return 503
	default:
		return 503
	}
}

// HealthChecker manages health checks
type HealthChecker struct {
	checks        map[string]HealthCheck
	checksMutex   sync.RWMutex
	shuttingDown  bool
	shutdownMutex sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks:       make(map[string]HealthCheck),
		shuttingDown: false,
	}
}

// RegisterCheck registers a health check
func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	hc.checksMutex.Lock()
	defer hc.checksMutex.Unlock()
	hc.checks[name] = check
}

// SetShuttingDown sets the shutting down flag
func (hc *HealthChecker) SetShuttingDown(shuttingDown bool) {
	hc.shutdownMutex.Lock()
	defer hc.shutdownMutex.Unlock()
	hc.shuttingDown = shuttingDown
}

// IsShuttingDown returns whether the system is shutting down
func (hc *HealthChecker) IsShuttingDown() bool {
	hc.shutdownMutex.RLock()
	defer hc.shutdownMutex.RUnlock()
	return hc.shuttingDown
}

// CheckLiveness performs a liveness check
// Liveness checks only verify that the application is running
// Validates: Requirements 8.1
func (hc *HealthChecker) CheckLiveness(ctx context.Context) *HealthReport {
	return &HealthReport{
		Status:    StatusHealthy,
		Checks:    make(map[string]CheckResult),
		Timestamp: time.Now(),
	}
}

// CheckReadiness performs a readiness check
// Readiness checks verify that the application is ready to serve traffic
// This includes checking database and Redis connectivity
// Validates: Requirements 8.2, 8.3, 8.4, 8.5
func (hc *HealthChecker) CheckReadiness(ctx context.Context) *HealthReport {
	// If shutting down, immediately return unhealthy
	// Validates: Requirements 8.4
	if hc.IsShuttingDown() {
		return &HealthReport{
			Status: StatusUnhealthy,
			Checks: map[string]CheckResult{
				"shutdown": {
					Status:    StatusUnhealthy,
					Message:   "System is shutting down",
					Timestamp: time.Now(),
				},
			},
			Timestamp: time.Now(),
		}
	}

	hc.checksMutex.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.checksMutex.RUnlock()

	results := make(map[string]CheckResult)
	overallStatus := StatusHealthy

	// Run all checks with timeout
	// Validates: Requirements 8.5 (1 second max timeout)
	for name, check := range checks {
		result := hc.runSingleCheck(ctx, name, check)
		results[name] = result

		// Update overall status based on check results
		// Only database is critical - if it fails, system is unhealthy
		// Other dependencies (like Redis) are not critical for readiness
		// Validates: Requirements 8.2
		if result.Status == StatusUnhealthy && name == "database" {
			overallStatus = StatusUnhealthy
		}
	}

	return &HealthReport{
		Status:    overallStatus,
		Checks:    results,
		Timestamp: time.Now(),
	}
}

// runSingleCheck runs a single health check with timeout
func (hc *HealthChecker) runSingleCheck(ctx context.Context, name string, check HealthCheck) CheckResult {
	// Create timeout context (1 second max)
	// Validates: Requirements 8.5
	timeout := check.Timeout()
	if timeout > 1*time.Second {
		timeout = 1 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := check.Check(checkCtx)
	duration := time.Since(start)

	result := CheckResult{
		Timestamp: time.Now(),
		Duration:  duration,
		Details:   make(map[string]interface{}),
	}

	if err != nil {
		// Check if it was a timeout
		if checkCtx.Err() == context.DeadlineExceeded {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("Health check timed out after %v", timeout)
			result.Details["error"] = "timeout"
			result.Details["timeout"] = timeout.String()
		} else {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("Health check failed: %v", err)
			result.Details["error"] = err.Error()
		}
	} else {
		result.Status = StatusHealthy
		result.Message = "Health check passed"
	}

	result.Details["duration"] = duration.String()
	result.Details["check_name"] = name

	return result
}
