package monitoring

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector collects and exports application metrics
type MetricsCollector struct {
	// HTTP metrics
	httpRequestsTotal     *prometheus.CounterVec
	httpRequestDuration   *prometheus.HistogramVec
	httpRequestSizeBytes  *prometheus.HistogramVec
	httpResponseSizeBytes *prometheus.HistogramVec

	// Application metrics
	ordersCreated       *prometheus.CounterVec
	ordersCompleted     *prometheus.CounterVec
	ordersCancelled     *prometheus.CounterVec
	orderProcessingTime *prometheus.HistogramVec
	orderValue          *prometheus.HistogramVec

	// Product metrics
	productsCreated  *prometheus.CounterVec
	productsViewed   *prometheus.CounterVec
	productSearches  *prometheus.CounterVec
	inventoryUpdates *prometheus.CounterVec
	lowStockAlerts   *prometheus.CounterVec

	// User metrics
	userRegistrations *prometheus.CounterVec
	userLogins        *prometheus.CounterVec
	activeSessions    *prometheus.GaugeVec
	authFailures      *prometheus.CounterVec

	// System metrics
	databaseConnections *prometheus.GaugeVec
	databaseQueryTime   *prometheus.HistogramVec
	cacheOperations     *prometheus.CounterVec
	cacheHitRate        *prometheus.GaugeVec
	systemMemoryUsage   *prometheus.GaugeVec
	systemCPUUsage      *prometheus.GaugeVec
	goroutineCount      *prometheus.GaugeVec

	// Business metrics
	revenueTotal   *prometheus.CounterVec
	cartValue      *prometheus.HistogramVec
	conversionRate *prometheus.GaugeVec

	// Error metrics
	errorTotal *prometheus.CounterVec
	panicTotal *prometheus.CounterVec

	// Custom metrics registry
	customMetrics      map[string]prometheus.Metric
	customMetricsMutex sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		// HTTP metrics
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code", "user_agent"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),
		httpRequestSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{"method", "endpoint"},
		),
		httpResponseSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{"method", "endpoint"},
		),

		// Order metrics
		ordersCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_orders_created_total",
				Help: "Total number of orders created",
			},
			[]string{"status", "payment_method", "customer_type"},
		),
		ordersCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_orders_completed_total",
				Help: "Total number of orders completed",
			},
			[]string{"fulfillment_method", "customer_type"},
		),
		ordersCancelled: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_orders_cancelled_total",
				Help: "Total number of orders cancelled",
			},
			[]string{"reason", "customer_type"},
		),
		orderProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_order_processing_seconds",
				Help:    "Time spent processing orders",
				Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
			},
			[]string{"order_type"},
		),
		orderValue: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_order_value_dollars",
				Help:    "Order value in dollars",
				Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
			},
			[]string{"customer_type", "payment_method"},
		),

		// Product metrics
		productsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_products_created_total",
				Help: "Total number of products created",
			},
			[]string{"category", "created_by"},
		),
		productsViewed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_products_viewed_total",
				Help: "Total number of product views",
			},
			[]string{"product_id", "category"},
		),
		productSearches: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_product_searches_total",
				Help: "Total number of product searches",
			},
			[]string{"search_type", "results_count"},
		),
		inventoryUpdates: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_inventory_updates_total",
				Help: "Total number of inventory updates",
			},
			[]string{"operation", "product_id", "warehouse_id"},
		),
		lowStockAlerts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_low_stock_alerts_total",
				Help: "Total number of low stock alerts",
			},
			[]string{"product_id", "warehouse_id", "severity"},
		),

		// User metrics
		userRegistrations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_user_registrations_total",
				Help: "Total number of user registrations",
			},
			[]string{"user_type", "source"},
		),
		userLogins: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_user_logins_total",
				Help: "Total number of user logins",
			},
			[]string{"user_type", "status"},
		),
		activeSessions: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_active_sessions",
				Help: "Number of active user sessions",
			},
			[]string{"user_type"},
		),
		authFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"reason", "ip_address"},
		),

		// System metrics
		databaseConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_database_connections",
				Help: "Number of database connections",
			},
			[]string{"database", "state"},
		),
		databaseQueryTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_database_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"operation", "table"},
		),
		cacheOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_cache_operations_total",
				Help: "Total number of cache operations",
			},
			[]string{"operation", "cache_type", "result"},
		),
		cacheHitRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_cache_hit_rate",
				Help: "Cache hit rate percentage",
			},
			[]string{"cache_type"},
		),
		systemMemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_system_memory_bytes",
				Help: "System memory usage in bytes",
			},
			[]string{"type"},
		),
		systemCPUUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_system_cpu_percent",
				Help: "System CPU usage percentage",
			},
			[]string{"cpu"},
		),
		goroutineCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_goroutines",
				Help: "Number of goroutines",
			},
			[]string{},
		),

		// Business metrics
		revenueTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_revenue_total",
				Help: "Total revenue in dollars",
			},
			[]string{"currency", "source"},
		),
		cartValue: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_cart_value_dollars",
				Help:    "Shopping cart value in dollars",
				Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500},
			},
			[]string{"customer_type"},
		),
		conversionRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_conversion_rate",
				Help: "Conversion rate percentage",
			},
			[]string{"period", "source"},
		),

		// Error metrics
		errorTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_errors_total",
				Help: "Total number of errors",
			},
			[]string{"error_type", "component", "severity"},
		),
		panicTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "erpgo_panics_total",
				Help: "Total number of panics",
			},
			[]string{"component"},
		),

		customMetrics: make(map[string]prometheus.Metric),
	}

	// Start system metrics collection
	go mc.collectSystemMetrics()

	return mc
}

// RecordHTTPRequest records HTTP request metrics
func (mc *MetricsCollector) RecordHTTPRequest(method, endpoint, statusCode, userAgent string, duration time.Duration, requestSize, responseSize int64) {
	mc.httpRequestsTotal.WithLabelValues(method, endpoint, statusCode, userAgent).Inc()
	mc.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())

	if requestSize > 0 {
		mc.httpRequestSizeBytes.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	}
	if responseSize > 0 {
		mc.httpResponseSizeBytes.WithLabelValues(method, endpoint).Observe(float64(responseSize))
	}
}

// RecordRequest records HTTP request metrics (simplified version for middleware)
func (mc *MetricsCollector) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	statusStr := fmt.Sprintf("%d", statusCode)
	mc.httpRequestsTotal.WithLabelValues(method, path, statusStr, "").Inc()
	mc.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordOrderCreated records an order creation event
func (mc *MetricsCollector) RecordOrderCreated(status, paymentMethod, customerType string, value float64) {
	mc.ordersCreated.WithLabelValues(status, paymentMethod, customerType).Inc()
	mc.orderValue.WithLabelValues(customerType, paymentMethod).Observe(value)
}

// RecordOrderCompleted records an order completion event
func (mc *MetricsCollector) RecordOrderCompleted(fulfillmentMethod, customerType string, processingTime time.Duration) {
	mc.ordersCompleted.WithLabelValues(fulfillmentMethod, customerType).Inc()
	mc.orderProcessingTime.WithLabelValues("completed").Observe(processingTime.Seconds())
}

// RecordOrderCancelled records an order cancellation event
func (mc *MetricsCollector) RecordOrderCancelled(reason, customerType string, processingTime time.Duration) {
	mc.ordersCancelled.WithLabelValues(reason, customerType).Inc()
	mc.orderProcessingTime.WithLabelValues("cancelled").Observe(processingTime.Seconds())
}

// RecordProductViewed records a product view event
func (mc *MetricsCollector) RecordProductViewed(productID, category string) {
	mc.productsViewed.WithLabelValues(productID, category).Inc()
}

// RecordProductSearch records a product search event
func (mc *MetricsCollector) RecordProductSearch(searchType string, resultsCount int) {
	resultCategory := "no_results"
	if resultsCount > 0 && resultsCount <= 10 {
		resultCategory = "1_10"
	} else if resultsCount > 10 && resultsCount <= 50 {
		resultCategory = "11_50"
	} else if resultsCount > 50 {
		resultCategory = "50_plus"
	}
	mc.productSearches.WithLabelValues(searchType, resultCategory).Inc()
}

// RecordInventoryUpdate records an inventory update event
func (mc *MetricsCollector) RecordInventoryUpdate(operation, productID, warehouseID string) {
	mc.inventoryUpdates.WithLabelValues(operation, productID, warehouseID).Inc()
}

// RecordLowStockAlert records a low stock alert
func (mc *MetricsCollector) RecordLowStockAlert(productID, warehouseID, severity string) {
	mc.lowStockAlerts.WithLabelValues(productID, warehouseID, severity).Inc()
}

// RecordUserRegistration records a user registration event
func (mc *MetricsCollector) RecordUserRegistration(userType, source string) {
	mc.userRegistrations.WithLabelValues(userType, source).Inc()
}

// RecordUserLogin records a user login event
func (mc *MetricsCollector) RecordUserLogin(userType, status string) {
	mc.userLogins.WithLabelValues(userType, status).Inc()
}

// SetActiveSessions sets the number of active sessions
func (mc *MetricsCollector) SetActiveSessions(userType string, count int) {
	mc.activeSessions.WithLabelValues(userType).Set(float64(count))
}

// RecordAuthFailure records an authentication failure
func (mc *MetricsCollector) RecordAuthFailure(reason, ipAddress string) {
	mc.authFailures.WithLabelValues(reason, ipAddress).Inc()
}

// SetDatabaseConnections sets the number of database connections
func (mc *MetricsCollector) SetDatabaseConnections(database, state string, count int) {
	mc.databaseConnections.WithLabelValues(database, state).Set(float64(count))
}

// RecordDatabaseQuery records a database query execution
func (mc *MetricsCollector) RecordDatabaseQuery(operation, table string, duration time.Duration) {
	mc.databaseQueryTime.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordCacheOperation records a cache operation
func (mc *MetricsCollector) RecordCacheOperation(operation, cacheType, result string) {
	mc.cacheOperations.WithLabelValues(operation, cacheType, result).Inc()
}

// SetCacheHitRate sets the cache hit rate
func (mc *MetricsCollector) SetCacheHitRate(cacheType string, rate float64) {
	mc.cacheHitRate.WithLabelValues(cacheType).Set(rate)
}

// RecordRevenue records revenue amount
func (mc *MetricsCollector) RecordRevenue(currency, source string, amount float64) {
	mc.revenueTotal.WithLabelValues(currency, source).Add(amount)
}

// RecordCartValue records shopping cart value
func (mc *MetricsCollector) RecordCartValue(customerType string, value float64) {
	mc.cartValue.WithLabelValues(customerType).Observe(value)
}

// SetConversionRate sets the conversion rate
func (mc *MetricsCollector) SetConversionRate(period, source string, rate float64) {
	mc.conversionRate.WithLabelValues(period, source).Set(rate)
}

// RecordError records an error event
func (mc *MetricsCollector) RecordError(errorType, component, severity string) {
	mc.errorTotal.WithLabelValues(errorType, component, severity).Inc()
}

// RecordPanic records a panic event
func (mc *MetricsCollector) RecordPanic(component string) {
	mc.panicTotal.WithLabelValues(component).Inc()
}

// collectSystemMetrics collects system-level metrics
func (mc *MetricsCollector) collectSystemMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Memory statistics
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		mc.systemMemoryUsage.WithLabelValues("alloc").Set(float64(m.Alloc))
		mc.systemMemoryUsage.WithLabelValues("total_alloc").Set(float64(m.TotalAlloc))
		mc.systemMemoryUsage.WithLabelValues("sys").Set(float64(m.Sys))
		mc.systemMemoryUsage.WithLabelValues("heap_alloc").Set(float64(m.HeapAlloc))
		mc.systemMemoryUsage.WithLabelValues("heap_sys").Set(float64(m.HeapSys))
		mc.systemMemoryUsage.WithLabelValues("heap_idle").Set(float64(m.HeapIdle))
		mc.systemMemoryUsage.WithLabelValues("heap_inuse").Set(float64(m.HeapInuse))

		// Goroutine count
		mc.goroutineCount.WithLabelValues().Set(float64(runtime.NumGoroutine()))

		// GC statistics
		mc.systemMemoryUsage.WithLabelValues("num_gc").Set(float64(m.NumGC))
		mc.systemMemoryUsage.WithLabelValues("gc_cpu_fraction").Set(m.GCCPUFraction)
	}
}

// AddCustomMetric adds a custom metric
func (mc *MetricsCollector) AddCustomMetric(name string, metric prometheus.Metric) {
	mc.customMetricsMutex.Lock()
	defer mc.customMetricsMutex.Unlock()
	mc.customMetrics[name] = metric
}

// GetMetricsHandler returns the Prometheus metrics handler
func (mc *MetricsCollector) GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsMiddleware returns a Gin middleware for collecting HTTP metrics
func (mc *MetricsCollector) MetricsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		// Get request size
		var requestSize int64
		if c.Request.ContentLength > 0 {
			requestSize = c.Request.ContentLength
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get response size
		var responseSize int64
		responseSize = int64(c.Writer.Size())

		// Get user agent (truncate if too long)
		userAgent := c.Request.UserAgent()
		if len(userAgent) > 100 {
			userAgent = userAgent[:100]
		}

		// Record metrics
		statusCode := c.Writer.Status()
		mc.RecordHTTPRequest(
			c.Request.Method,
			c.FullPath(),
			string(rune(statusCode)),
			userAgent,
			duration,
			requestSize,
			responseSize,
		)

		// Record errors
		if statusCode >= 400 {
			severity := "warning"
			if statusCode >= 500 {
				severity = "error"
			}
			mc.RecordError("http_error", "api", severity)
		}
	})
}

// Global metrics collector instance
var GlobalMetricsCollector = NewMetricsCollector()
