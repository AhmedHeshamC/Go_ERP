package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector collects and exports application metrics
type MetricsCollector struct {
	// Order metrics
	ordersCreated           prometheus.Counter
	ordersCompleted         prometheus.Counter
	ordersCancelled         prometheus.Counter
	orderProcessingDuration prometheus.Histogram

	// Performance metrics
	requestDuration    prometheus.Histogram
	concurrentRequests prometheus.Gauge
	errorRate          prometheus.Counter

	// Business metrics
	revenueTotal        prometheus.Counter
	customerAcquisition prometheus.Counter
	notificationSent    prometheus.Counter

	// System metrics
	memoryUsage         prometheus.Gauge
	cpuUsage            prometheus.Gauge
	databaseConnections prometheus.Gauge

	// Custom metrics
	customMetrics      map[string]prometheus.Metric
	customMetricsMutex sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		ordersCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "erp_orders_created_total",
			Help: "Total number of orders created",
		}),
		ordersCompleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "erp_orders_completed_total",
			Help: "Total number of orders completed",
		}),
		ordersCancelled: promauto.NewCounter(prometheus.CounterOpts{
			Name: "erp_orders_cancelled_total",
			Help: "Total number of orders cancelled",
		}),
		orderProcessingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "erp_order_processing_duration_seconds",
			Help:    "Time spent processing orders",
			Buckets: prometheus.DefBuckets,
		}),
		requestDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "erp_request_duration_seconds",
			Help:    "Time spent processing requests",
			Buckets: prometheus.DefBuckets,
		}),
		concurrentRequests: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "erp_concurrent_requests",
			Help: "Number of concurrent requests",
		}),
		errorRate: promauto.NewCounter(prometheus.CounterOpts{
			Name: "erp_errors_total",
			Help: "Total number of errors",
		}),
		revenueTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "erp_revenue_total",
			Help: "Total revenue generated",
		}),
		customerAcquisition: promauto.NewCounter(prometheus.CounterOpts{
			Name: "erp_customer_acquisition_total",
			Help: "Total number of customers acquired",
		}),
		notificationSent: promauto.NewCounter(prometheus.CounterOpts{
			Name: "erp_notifications_sent_total",
			Help: "Total number of notifications sent",
		}),
		memoryUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "erp_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		}),
		cpuUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "erp_cpu_usage_percent",
			Help: "Current CPU usage percentage",
		}),
		databaseConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "erp_database_connections",
			Help: "Number of active database connections",
		}),
		customMetrics: make(map[string]prometheus.Metric),
	}
}

// RecordOrderCreated records an order creation event
func (mc *MetricsCollector) RecordOrderCreated() {
	mc.ordersCreated.Inc()
}

// RecordOrderCompleted records an order completion event
func (mc *MetricsCollector) RecordOrderCompleted() {
	mc.ordersCompleted.Inc()
}

// RecordOrderCancelled records an order cancellation event
func (mc *MetricsCollector) RecordOrderCancelled() {
	mc.ordersCancelled.Inc()
}

// RecordOrderProcessingDuration records order processing duration
func (mc *MetricsCollector) RecordOrderProcessingDuration(duration time.Duration) {
	mc.orderProcessingDuration.Observe(duration.Seconds())
}

// RecordRequestDuration records request processing duration
func (mc *MetricsCollector) RecordRequestDuration(duration time.Duration) {
	mc.requestDuration.Observe(duration.Seconds())
}

// SetConcurrentRequests sets the number of concurrent requests
func (mc *MetricsCollector) SetConcurrentRequests(count int) {
	mc.concurrentRequests.Set(float64(count))
}

// RecordError records an error event
func (mc *MetricsCollector) RecordError() {
	mc.errorRate.Inc()
}

// RecordRevenue records revenue amount
func (mc *MetricsCollector) RecordRevenue(amount float64) {
	mc.revenueTotal.Add(amount)
}

// RecordCustomerAcquisition records a customer acquisition event
func (mc *MetricsCollector) RecordCustomerAcquisition() {
	mc.customerAcquisition.Inc()
}

// RecordNotificationSent records a notification sent event
func (mc *MetricsCollector) RecordNotificationSent() {
	mc.notificationSent.Inc()
}

// SetMemoryUsage sets the current memory usage
func (mc *MetricsCollector) SetMemoryUsage(bytes uint64) {
	mc.memoryUsage.Set(float64(bytes))
}

// SetCPUUsage sets the current CPU usage
func (mc *MetricsCollector) SetCPUUsage(percent float64) {
	mc.cpuUsage.Set(percent)
}

// SetDatabaseConnections sets the number of database connections
func (mc *MetricsCollector) SetDatabaseConnections(count int) {
	mc.databaseConnections.Set(float64(count))
}

// AddCustomMetric adds a custom metric
func (mc *MetricsCollector) AddCustomMetric(name string, metric prometheus.Metric) {
	mc.customMetricsMutex.Lock()
	defer mc.customMetricsMutex.Unlock()
	mc.customMetrics[name] = metric
}

// GetMetricsSummary returns a summary of all metrics
func (mc *MetricsCollector) GetMetricsSummary() map[string]interface{} {
	mc.customMetricsMutex.RLock()
	defer mc.customMetricsMutex.RUnlock()

	return map[string]interface{}{
		"orders_created":       mc.ordersCreated.Desc().String(),
		"orders_completed":     mc.ordersCompleted.Desc().String(),
		"orders_cancelled":     mc.ordersCancelled.Desc().String(),
		"revenue":              mc.revenueTotal.Desc().String(),
		"customer_acquisition": mc.customerAcquisition.Desc().String(),
		"notifications_sent":   mc.notificationSent.Desc().String(),
		"concurrent_requests":  mc.concurrentRequests.Desc().String(),
		"errors":               mc.errorRate.Desc().String(),
		"memory_usage":         mc.memoryUsage.Desc().String(),
		"cpu_usage":            mc.cpuUsage.Desc().String(),
		"database_connections": mc.databaseConnections.Desc().String(),
		"custom_metrics_count": len(mc.customMetrics),
	}
}

// HealthChecker provides health check functionality
type HealthChecker struct {
	checks map[string]HealthCheck
	mutex  sync.RWMutex
}

// HealthCheck represents a health check function
type HealthCheck func(ctx context.Context) HealthStatus

// HealthStatus represents the status of a health check
type HealthStatus struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]HealthCheck),
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(name string, check HealthCheck) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.checks[name] = check
}

// CheckHealth runs all health checks
func (hc *HealthChecker) CheckHealth(ctx context.Context) map[string]HealthStatus {
	hc.mutex.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.mutex.RUnlock()

	results := make(map[string]HealthStatus)
	for name, check := range checks {
		status := check(ctx)
		results[name] = status
	}

	return results
}

// GetOverallHealth returns the overall health status
func (hc *HealthChecker) GetOverallHealth(ctx context.Context) HealthStatus {
	results := hc.CheckHealth(ctx)

	healthyCount := 0
	totalCount := len(results)

	for _, status := range results {
		if status.Status == "healthy" {
			healthyCount++
		}
	}

	overallStatus := "healthy"
	if healthyCount == 0 {
		overallStatus = "unhealthy"
	} else if healthyCount < totalCount {
		overallStatus = "degraded"
	}

	message := fmt.Sprintf("%d/%d checks healthy", healthyCount, totalCount)

	return HealthStatus{
		Status:  overallStatus,
		Message: message,
		Details: map[string]interface{}{
			"healthy_checks": healthyCount,
			"total_checks":   totalCount,
			"check_results":  results,
		},
	}
}

// Logger provides structured logging functionality
type Logger struct {
	level  LogLevel
	output LoggerOutput
}

// LogLevel represents log level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// LoggerOutput represents log output
type LoggerOutput interface {
	Write(level LogLevel, message string, fields map[string]interface{})
}

// ConsoleLoggerOutput implements console logging
type ConsoleLoggerOutput struct{}

// Write writes a log entry to console
func (clo *ConsoleLoggerOutput) Write(level LogLevel, message string, fields map[string]interface{}) {
	levelStr := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}[level]

	logEntry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     levelStr,
		"message":   message,
	}

	for k, v := range fields {
		logEntry[k] = v
	}

	jsonData, _ := json.Marshal(logEntry)
	log.Println(string(jsonData))
}

// NewLogger creates a new logger
func NewLogger(level LogLevel, output LoggerOutput) *Logger {
	return &Logger{
		level:  level,
		output: output,
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	if l.level <= LogLevelDebug {
		l.log(LogLevelDebug, message, fields...)
	}
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	if l.level <= LogLevelInfo {
		l.log(LogLevelInfo, message, fields...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	if l.level <= LogLevelWarn {
		l.log(LogLevelWarn, message, fields...)
	}
}

// Error logs an error message
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	if l.level <= LogLevelError {
		l.log(LogLevelError, message, fields...)
	}
}

// Fatal logs a fatal message
func (l *Logger) Fatal(message string, fields ...map[string]interface{}) {
	if l.level <= LogLevelFatal {
		l.log(LogLevelFatal, message, fields...)
	}
}

// log logs a message at the specified level
func (l *Logger) log(level LogLevel, message string, fields ...map[string]interface{}) {
	mergedFields := make(map[string]interface{})
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			mergedFields[k] = v
		}
	}

	l.output.Write(level, message, mergedFields)
}

// MonitoringService combines monitoring functionality
type MonitoringService struct {
	metrics       *MetricsCollector
	healthChecker *HealthChecker
	logger        *Logger
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService() *MonitoringService {
	logger := NewLogger(LogLevelInfo, &ConsoleLoggerOutput{})

	return &MonitoringService{
		metrics:       NewMetricsCollector(),
		healthChecker: NewHealthChecker(),
		logger:        logger,
	}
}

// GetMetrics returns the metrics collector
func (ms *MonitoringService) GetMetrics() *MetricsCollector {
	return ms.metrics
}

// GetHealthChecker returns the health checker
func (ms *MonitoringService) GetHealthChecker() *HealthChecker {
	return ms.healthChecker
}

// GetLogger returns the logger
func (ms *MonitoringService) GetLogger() *Logger {
	return ms.logger
}

// SetupDefaultChecks sets up default health checks
func (ms *MonitoringService) SetupDefaultChecks() {
	// Database health check
	ms.healthChecker.AddCheck("database", func(ctx context.Context) HealthStatus {
		// In a real implementation, this would check database connectivity
		return HealthStatus{
			Status:  "healthy",
			Message: "Database connection is healthy",
		}
	})

	// Memory health check
	ms.healthChecker.AddCheck("memory", func(ctx context.Context) HealthStatus {
		// In a real implementation, this would check memory usage
		return HealthStatus{
			Status:  "healthy",
			Message: "Memory usage is within limits",
		}
	})

	// External service health check
	ms.healthChecker.AddCheck("external_services", func(ctx context.Context) HealthStatus {
		// In a real implementation, this would check external service connectivity
		return HealthStatus{
			Status:  "healthy",
			Message: "External services are accessible",
		}
	})
}

// Global monitoring service instance
var GlobalMonitoringService = NewMonitoringService()

// Initialize monitoring system
func InitializeMonitoring() {
	GlobalMonitoringService.SetupDefaultChecks()
	GlobalMonitoringService.GetLogger().Info("Monitoring system initialized")
}
