package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check function
type HealthCheck func(ctx context.Context) HealthCheckResult

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status      HealthStatus            `json:"status"`
	Message     string                  `json:"message"`
	Details     map[string]interface{}  `json:"details,omitempty"`
	Duration    time.Duration           `json:"duration"`
	Timestamp   time.Time               `json:"timestamp"`
	CheckType   string                  `json:"check_type"`
	Component   string                  `json:"component"`
	Metadata    map[string]string       `json:"metadata,omitempty"`
}

// HealthCheckConfig configures a health check
type HealthCheckConfig struct {
	Name        string        `json:"name"`
	Check       HealthCheck   `json:"-"`
	Timeout     time.Duration `json:"timeout"`
	Interval    time.Duration `json:"interval"`
	Critical    bool          `json:"critical"`
	Enabled     bool          `json:"enabled"`
	Tags        []string      `json:"tags"`
	Description string        `json:"description"`
}

// HealthReport represents a comprehensive health report
type HealthReport struct {
	Status       HealthStatus                   `json:"status"`
	Timestamp    time.Time                      `json:"timestamp"`
	Version      string                         `json:"version"`
	BuildInfo    BuildInfo                      `json:"build_info"`
	SystemInfo   SystemInfo                     `json:"system_info"`
	Checks       map[string]HealthCheckResult   `json:"checks"`
	Summary      HealthSummary                  `json:"summary"`
	Uptime       time.Duration                  `json:"uptime"`
	Dependencies map[string]DependencyStatus   `json:"dependencies"`
}

// BuildInfo contains build information
type BuildInfo struct {
	Version   string    `json:"version"`
	BuildTime time.Time `json:"build_time"`
	Commit    string    `json:"commit"`
	GoVersion string    `json:"go_version"`
}

// SystemInfo contains system information
type SystemInfo struct {
	Hostname     string    `json:"hostname"`
	PID          int       `json:"pid"`
	GoRoutines   int       `json:"goroutines"`
	MemoryUsage  MemoryInfo `json:"memory_usage"`
	CPUUsage     float64   `json:"cpu_usage"`
	NumCPU       int       `json:"num_cpu"`
	NumGoroutine int       `json:"num_goroutine"`
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// HealthSummary contains a summary of health checks
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded"`
	Unhealthy int `json:"unhealthy"`
	Critical  int `json:"critical"`
}

// DependencyStatus represents the status of external dependencies
type DependencyStatus struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// HealthChecker manages health checks
type HealthChecker struct {
	checks       map[string]*HealthCheckConfig
	checksMutex  sync.RWMutex
	results      map[string]HealthCheckResult
	resultsMutex sync.RWMutex
	startTime    time.Time
	version      string
	buildInfo    BuildInfo
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(version, buildTime, commit string) *HealthChecker {
	return &HealthChecker{
		checks:    make(map[string]*HealthCheckConfig),
		results:   make(map[string]HealthCheckResult),
		startTime: time.Now(),
		version:   version,
		buildInfo: BuildInfo{
			Version:   version,
			BuildTime: parseBuildTime(buildTime),
			Commit:    commit,
			GoVersion: runtime.Version(),
		},
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(config HealthCheckConfig) {
	hc.checksMutex.Lock()
	defer hc.checksMutex.Unlock()
	hc.checks[config.Name] = &config
}

// RemoveCheck removes a health check
func (hc *HealthChecker) RemoveCheck(name string) {
	hc.checksMutex.Lock()
	defer hc.checksMutex.Unlock()
	delete(hc.checks, name)
}

// RunCheck runs a specific health check
func (hc *HealthChecker) RunCheck(ctx context.Context, name string) (HealthCheckResult, error) {
	hc.checksMutex.RLock()
	config, exists := hc.checks[name]
	hc.checksMutex.RUnlock()

	if !exists {
		return HealthCheckResult{}, fmt.Errorf("health check '%s' not found", name)
	}

	if !config.Enabled {
		return HealthCheckResult{
			Status:    HealthStatusHealthy,
			Message:   "Check disabled",
			CheckType: config.Name,
			Component: "health_checker",
			Timestamp: time.Now(),
		}, nil
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Run the check
	startTime := time.Now()
	result := config.Check(timeoutCtx)
	duration := time.Since(startTime)

	// Enhance result with metadata
	result.Duration = duration
	result.Timestamp = time.Now()
	result.CheckType = config.Name
	result.Component = "health_checker"

	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["timeout"] = config.Timeout.String()
	result.Details["interval"] = config.Interval.String()
	result.Details["critical"] = config.Critical
	result.Details["enabled"] = config.Enabled
	result.Details["tags"] = config.Tags

	// Store result
	hc.resultsMutex.Lock()
	hc.results[name] = result
	hc.resultsMutex.Unlock()

	return result, nil
}

// RunAllChecks runs all health checks
func (hc *HealthChecker) RunAllChecks(ctx context.Context) map[string]HealthCheckResult {
	hc.checksMutex.RLock()
	checks := make(map[string]*HealthCheckConfig)
	for name, config := range hc.checks {
		checks[name] = config
	}
	hc.checksMutex.RUnlock()

	results := make(map[string]HealthCheckResult)
	var wg sync.WaitGroup
	resultsChan := make(chan struct {
		name   string
		result HealthCheckResult
	}, len(checks))

	// Run checks concurrently
	for name, config := range checks {
		wg.Add(1)
		go func(checkName string, checkConfig *HealthCheckConfig) {
			defer wg.Done()
			result, err := hc.RunCheck(ctx, checkName)
			if err != nil {
				result = HealthCheckResult{
					Status:    HealthStatusUnhealthy,
					Message:   fmt.Sprintf("Check failed: %v", err),
					CheckType: checkName,
					Component: "health_checker",
					Timestamp: time.Now(),
				}
			}
			resultsChan <- struct {
				name   string
				result HealthCheckResult
			}{name: checkName, result: result}
		}(name, config)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results[result.name] = result.result
	}

	return results
}

// GetHealthReport generates a comprehensive health report
func (hc *HealthChecker) GetHealthReport(ctx context.Context) HealthReport {
	checks := hc.RunAllChecks(ctx)

	// Calculate summary
	summary := HealthSummary{}
	overallStatus := HealthStatusHealthy

	criticalFailures := 0
	for _, result := range checks {
		summary.Total++
		switch result.Status {
		case HealthStatusHealthy:
			summary.Healthy++
		case HealthStatusDegraded:
			summary.Degraded++
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		case HealthStatusUnhealthy:
			summary.Unhealthy++
			overallStatus = HealthStatusUnhealthy

			// Check if this is a critical failure
			hc.checksMutex.RLock()
			if config, exists := hc.checks[result.CheckType]; exists && config.Critical {
				criticalFailures++
			}
			hc.checksMutex.RUnlock()
		}
	}

	summary.Critical = criticalFailures

	// If there are critical failures, mark as unhealthy regardless of other checks
	if criticalFailures > 0 {
		overallStatus = HealthStatusUnhealthy
	}

	// Get system information
	systemInfo := hc.getSystemInfo()

	// Get dependency status
	dependencies := hc.getDependencyStatus(ctx)

	return HealthReport{
		Status:       overallStatus,
		Timestamp:    time.Now(),
		Version:      hc.version,
		BuildInfo:    hc.buildInfo,
		SystemInfo:   systemInfo,
		Checks:       checks,
		Summary:      summary,
		Uptime:       time.Since(hc.startTime),
		Dependencies: dependencies,
	}
}

// getSystemInfo collects system information
func (hc *HealthChecker) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		Hostname:     getHealthHostname(),
		PID:          getHealthPID(),
		GoRoutines:   runtime.NumGoroutine(),
		MemoryUsage: MemoryInfo{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}
}

// getDependencyStatus gets the status of external dependencies
func (hc *HealthChecker) getDependencyStatus(ctx context.Context) map[string]DependencyStatus {
	dependencies := make(map[string]DependencyStatus)

	// Add database dependency status
	if dbResult, exists := hc.results["database"]; exists {
		dependencies["database"] = DependencyStatus{
			Status:    dbResult.Status,
			Message:   dbResult.Message,
			Details:   dbResult.Details,
			UpdatedAt: dbResult.Timestamp,
		}
	}

	// Add cache dependency status
	if cacheResult, exists := hc.results["cache"]; exists {
		dependencies["cache"] = DependencyStatus{
			Status:    cacheResult.Status,
			Message:   cacheResult.Message,
			Details:   cacheResult.Details,
			UpdatedAt: cacheResult.Timestamp,
		}
	}

	// Add external services dependency status
	if externalResult, exists := hc.results["external_services"]; exists {
		dependencies["external_services"] = DependencyStatus{
			Status:    externalResult.Status,
			Message:   externalResult.Message,
			Details:   externalResult.Details,
			UpdatedAt: externalResult.Timestamp,
		}
	}

	return dependencies
}

// SetupDefaultChecks sets up default health checks
func (hc *HealthChecker) SetupDefaultChecks(databaseCheck, cacheCheck HealthCheck) {
	// Database health check
	hc.AddCheck(HealthCheckConfig{
		Name:        "database",
		Check:       databaseCheck,
		Timeout:     5 * time.Second,
		Interval:    30 * time.Second,
		Critical:    true,
		Enabled:     true,
		Tags:        []string{"critical", "database"},
		Description: "Database connectivity and performance check",
	})

	// Cache health check
	hc.AddCheck(HealthCheckConfig{
		Name:        "cache",
		Check:       cacheCheck,
		Timeout:     3 * time.Second,
		Interval:    30 * time.Second,
		Critical:    false,
		Enabled:     true,
		Tags:        []string{"cache"},
		Description: "Cache connectivity and performance check",
	})

	// Memory health check
	hc.AddCheck(HealthCheckConfig{
		Name: "memory",
		Check: func(ctx context.Context) HealthCheckResult {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Check memory usage (alert if > 1GB)
			maxMemory := uint64(1024 * 1024 * 1024) // 1GB
			if m.Alloc > maxMemory {
				return HealthCheckResult{
					Status:  HealthStatusDegraded,
					Message: fmt.Sprintf("High memory usage: %.2f MB", float64(m.Alloc)/1024/1024),
					Details: map[string]interface{}{
						"alloc_mb":      float64(m.Alloc) / 1024 / 1024,
						"sys_mb":        float64(m.Sys) / 1024 / 1024,
						"num_gc":        m.NumGC,
						"max_memory_mb": float64(maxMemory) / 1024 / 1024,
					},
				}
			}

			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("Memory usage: %.2f MB", float64(m.Alloc)/1024/1024),
				Details: map[string]interface{}{
					"alloc_mb": float64(m.Alloc) / 1024 / 1024,
					"sys_mb":   float64(m.Sys) / 1024 / 1024,
					"num_gc":   m.NumGC,
				},
			}
		},
		Timeout:     1 * time.Second,
		Interval:    60 * time.Second,
		Critical:    false,
		Enabled:     true,
		Tags:        []string{"system"},
		Description: "Memory usage check",
	})

	// Goroutine health check
	hc.AddCheck(HealthCheckConfig{
		Name: "goroutines",
		Check: func(ctx context.Context) HealthCheckResult {
			count := runtime.NumGoroutine()
			maxGoroutines := 1000

			if count > maxGoroutines {
				return HealthCheckResult{
					Status:  HealthStatusDegraded,
					Message: fmt.Sprintf("High goroutine count: %d", count),
					Details: map[string]interface{}{
						"count":          count,
						"max_goroutines": maxGoroutines,
					},
				}
			}

			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("Goroutine count: %d", count),
				Details: map[string]interface{}{
					"count": count,
				},
			}
		},
		Timeout:     1 * time.Second,
		Interval:    60 * time.Second,
		Critical:    false,
		Enabled:     true,
		Tags:        []string{"system"},
		Description: "Goroutine count check",
	})
}

// HealthHandler provides HTTP handlers for health checks
type HealthHandler struct {
	healthChecker *HealthChecker
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthChecker *HealthChecker) *HealthHandler {
	return &HealthHandler{
		healthChecker: healthChecker,
	}
}

// BasicHealth returns a basic health check endpoint
func (hh *HealthHandler) BasicHealth(c *gin.Context) {
	ctx := c.Request.Context()

	// Run only critical checks for basic health
	criticalChecks := []string{"database", "cache"}
	overallStatus := HealthStatusHealthy

	for _, checkName := range criticalChecks {
		result, err := hh.healthChecker.RunCheck(ctx, checkName)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    HealthStatusUnhealthy,
				"timestamp": time.Now(),
				"message":   fmt.Sprintf("Health check failed: %v", err),
			})
			return
		}

		if result.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if result.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	statusCode := http.StatusOK
	if overallStatus == HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    overallStatus,
		"timestamp": time.Now(),
		"version":   hh.healthChecker.version,
	})
}

// DetailedHealth returns a detailed health check endpoint
func (hh *HealthHandler) DetailedHealth(c *gin.Context) {
	ctx := c.Request.Context()
	report := hh.healthChecker.GetHealthReport(ctx)

	statusCode := http.StatusOK
	if report.Status == HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, report)
}

// HealthCheckByName returns a specific health check
func (hh *HealthHandler) HealthCheckByName(c *gin.Context) {
	checkName := c.Param("check")
	ctx := c.Request.Context()

	result, err := hh.healthChecker.RunCheck(ctx, checkName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Health check '%s' not found", checkName),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Livez returns a liveness probe endpoint
func (hh *HealthHandler) Livez(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
		"message":   "Service is alive",
	})
}

// Readyz returns a readiness probe endpoint
func (hh *HealthHandler) Readyz(c *gin.Context) {
	ctx := c.Request.Context()

	// Check only critical dependencies
	criticalChecks := []string{"database"}
	for _, checkName := range criticalChecks {
		result, err := hh.healthChecker.RunCheck(ctx, checkName)
		if err != nil || result.Status == HealthStatusUnhealthy {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "not ready",
				"timestamp": time.Now(),
				"message":   fmt.Sprintf("Critical check '%s' failed", checkName),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now(),
		"message":   "Service is ready",
	})
}

// Helper functions

func parseBuildTime(buildTime string) time.Time {
	if buildTime == "" || buildTime == "unknown" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		return time.Now()
	}
	return t
}

func getHealthHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getHealthPID() int {
	return os.Getpid()
}

// Global health checker instance
var GlobalHealthChecker *HealthChecker

// InitializeHealthChecker initializes the global health checker
func InitializeHealthChecker(version, buildTime, commit string) {
	GlobalHealthChecker = NewHealthChecker(version, buildTime, commit)
}

// GetHealthChecker returns the global health checker
func GetHealthChecker() *HealthChecker {
	if GlobalHealthChecker == nil {
		GlobalHealthChecker = NewHealthChecker("unknown", "", "")
	}
	return GlobalHealthChecker
}