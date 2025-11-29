package monitoring

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// MonitoringMiddleware provides HTTP request monitoring
type MonitoringMiddleware struct {
	metrics *MetricsCollector
	logger  *Logger
}

// NewMonitoringMiddleware creates a new monitoring middleware
func NewMonitoringMiddleware(metrics *MetricsCollector, logger *Logger) *MonitoringMiddleware {
	return &MonitoringMiddleware{
		metrics: metrics,
		logger:  logger,
	}
}

// Middleware returns the HTTP middleware function
func (mm *MonitoringMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Record metrics
		mm.metrics.RecordRequestDuration(duration)
		if wrapped.statusCode >= 400 {
			mm.metrics.RecordError()
		}

		// Log request
		mm.logger.Info("HTTP request processed", map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrapped.statusCode,
			"duration_ms": duration.Milliseconds(),
			"user_agent":  r.UserAgent(),
			"remote_addr": r.RemoteAddr,
		})
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// SetupMonitoringRoutes sets up monitoring endpoints
func SetupMonitoringRoutes(router *mux.Router, monitoringService *MonitoringService) {
	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		health := monitoringService.GetHealthChecker().GetOverallHealth(ctx)

		w.Header().Set("Content-Type", "application/json")
		if health.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else if health.Status == "degraded" {
			w.WriteHeader(http.StatusOK) // Still return 200 for degraded
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		// Write JSON response
		w.Write([]byte(`{"status":"` + health.Status + `","message":"` + health.Message + `"}`))
	}).Methods("GET")

	// Detailed health check endpoint
	router.HandleFunc("/health/detailed", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		checks := monitoringService.GetHealthChecker().CheckHealth(ctx)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Write JSON response
		w.Write([]byte(`{"checks":` + formatHealthChecks(checks) + `}`))
	}).Methods("GET")

	// Metrics endpoint (for Prometheus scraping)
	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// In a real implementation, this would return Prometheus metrics
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("# Prometheus metrics endpoint\n"))
	}).Methods("GET")

	// Monitoring dashboard endpoint
	router.HandleFunc("/monitoring", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(getMonitoringDashboardHTML()))
	}).Methods("GET")
}

// formatHealthChecks formats health checks for JSON response
func formatHealthChecks(checks map[string]HealthStatus) string {
	var result string = "{"
	for name, check := range checks {
		if len(result) > 1 {
			result += ","
		}
		result += `"` + name + `":` + `{"status":"` + check.Status + `","message":"` + check.Message + `"}`
	}
	result += "}"
	return result
}

// getMonitoringDashboardHTML returns a simple monitoring dashboard HTML
func getMonitoringDashboardHTML() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <title>ERPGo Monitoring Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 10px 0; padding: 10px; border: 1px solid #ddd; }
        .healthy { background-color: #d4edda; }
        .degraded { background-color: #fff3cd; }
        .unhealthy { background-color: #f8d7da; }
        .metric-value { font-weight: bold; }
    </style>
</head>
<body>
    <h1>ERPGo Monitoring Dashboard</h1>
    <div id="health-status">
        <h2>System Health</h2>
        <div id="health-content">Loading...</div>
    </div>
    <div id="metrics">
        <h2>System Metrics</h2>
        <div id="metrics-content">Loading...</div>
    </div>
    <script>
        function fetchHealth() {
            fetch('/health/detailed')
                .then(response => response.json())
                .then(data => {
                    let html = '';
                    for (const [name, check] of Object.entries(data.checks)) {
                        html += '<div class="metric ' + check.status + '">';
                        html += '<strong>' + name + '</strong>: ' + check.status;
                        html += ' - ' + check.message + '</div>';
                    }
                    document.getElementById('health-content').innerHTML = html;
                });
        }

        function fetchMetrics() {
            fetch('/metrics')
                .then(response => response.text())
                .then(data => {
                    document.getElementById('metrics-content').innerHTML = '<pre>' + data + '</pre>';
                });
        }

        // Update every 30 seconds
        setInterval(fetchHealth, 30000);
        setInterval(fetchMetrics, 30000);

        // Initial load
        fetchHealth();
        fetchMetrics();
    </script>
</body>
</html>
`
}
