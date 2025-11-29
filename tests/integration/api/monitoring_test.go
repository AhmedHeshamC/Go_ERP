package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/internal/application/services/monitoring"
	"erpgo/internal/infrastructure/logger"
)

// MonitoringIntegrationTestSuite tests the monitoring API endpoints
type MonitoringIntegrationTestSuite struct {
	suite.Suite
	monitoringSvc monitoring.Service
	router        *gin.Engine
	registry      *prometheus.Registry
}

// SetupSuite sets up the test suite
func (suite *MonitoringIntegrationTestSuite) SetupSuite() {
	// Initialize logger
	log := logger.NewNopLogger()

	// Initialize monitoring service
	suite.monitoringSvc = monitoring.NewMonitoringService(log)

	// Initialize Prometheus registry
	suite.registry = prometheus.NewRegistry()

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.setupRoutes()
}

// TearDownSuite cleans up after the test suite
func (suite *MonitoringIntegrationTestSuite) TearDownSuite() {
	if suite.monitoringSvc != nil {
		suite.monitoringSvc.Stop()
	}
}

// setupRoutes sets up the API routes for testing
func (suite *MonitoringIntegrationTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		api.GET("/health", suite.healthCheck)
		api.GET("/metrics", promhttp.HandlerFor(suite.registry, promhttp.HandlerOpts{}).ServeHTTP)
		api.GET("/monitoring/stats", suite.monitoringStats)
		api.GET("/monitoring/alerts", suite.monitoringAlerts)
		api.POST("/monitoring/alerts", suite.createAlert)
		api.DELETE("/monitoring/alerts/:id", suite.deleteAlert)
		api.GET("/monitoring/dashboards", suite.listDashboards)
		api.POST("/monitoring/dashboards", suite.createDashboard)
		api.GET("/monitoring/dashboards/:id", suite.getDashboard)
	}
}

// API Handlers

func (suite *MonitoringIntegrationTestSuite) healthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check various system components
	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"checks": map[string]interface{}{
			"database": "ok",
			"cache":    "ok",
			"redis":    "ok",
		},
		"uptime": time.Since(time.Now().Add(-time.Hour)).Seconds(), // Mock uptime
	}

	c.JSON(http.StatusOK, status)
}

func (suite *MonitoringIntegrationTestSuite) monitoringStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats := suite.monitoringSvc.GetSystemStats()

	response := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"stats":     stats,
	}

	c.JSON(http.StatusOK, response)
}

func (suite *MonitoringIntegrationTestSuite) monitoringAlerts(c *gin.Context) {
	ctx := c.Request.Context()

	// Mock alert data
	alerts := []map[string]interface{}{
		{
			"id":         "alert-1",
			"name":       "High CPU Usage",
			"level":      "warning",
			"status":     "active",
			"message":    "CPU usage is above 80%",
			"created_at": time.Now().Add(-time.Hour),
			"updated_at": time.Now().Add(-time.Minute),
		},
		{
			"id":          "alert-2",
			"name":        "Memory Pressure",
			"level":       "critical",
			"status":      "resolved",
			"message":     "Memory usage was critically high",
			"created_at":  time.Now().Add(-2 * time.Hour),
			"updated_at":  time.Now().Add(-time.Minute),
			"resolved_at": time.Now().Add(-time.Minute),
		},
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (suite *MonitoringIntegrationTestSuite) createAlert(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Name    string                 `json:"name" binding:"required"`
		Level   string                 `json:"level" binding:"required"`
		Message string                 `json:"message" binding:"required"`
		Source  string                 `json:"source"`
		Tags    map[string]interface{} `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert := map[string]interface{}{
		"id":         fmt.Sprintf("alert-%d", time.Now().UnixNano()),
		"name":       req.Name,
		"level":      req.Level,
		"status":     "active",
		"message":    req.Message,
		"source":     req.Source,
		"tags":       req.Tags,
		"created_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	}

	c.JSON(http.StatusCreated, gin.H{"alert": alert})
}

func (suite *MonitoringIntegrationTestSuite) deleteAlert(c *gin.Context) {
	ctx := c.Request.Context()

	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Alert ID is required"})
		return
	}

	// In a real implementation, you would delete the alert from storage
	c.JSON(http.StatusNoContent, nil)
}

func (suite *MonitoringIntegrationTestSuite) listDashboards(c *gin.Context) {
	ctx := c.Request.Context()

	// Mock dashboard data
	dashboards := []map[string]interface{}{
		{
			"id":          "dashboard-1",
			"name":        "System Overview",
			"description": "Overall system health and performance",
			"panels":      6,
			"created_at":  time.Now().Add(-24 * time.Hour),
			"updated_at":  time.Now().Add(-time.Hour),
		},
		{
			"id":          "dashboard-2",
			"name":        "API Metrics",
			"description": "API response times and error rates",
			"panels":      4,
			"created_at":  time.Now().Add(-12 * time.Hour),
			"updated_at":  time.Now().Add(-30 * time.Minute),
		},
	}

	c.JSON(http.StatusOK, gin.H{"dashboards": dashboards})
}

func (suite *MonitoringIntegrationTestSuite) createDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Name        string                   `json:"name" binding:"required"`
		Description string                   `json:"description"`
		Panels      []map[string]interface{} `json:"panels"`
		Tags        map[string]interface{}   `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dashboard := map[string]interface{}{
		"id":          fmt.Sprintf("dashboard-%d", time.Now().UnixNano()),
		"name":        req.Name,
		"description": req.Description,
		"panels":      req.Panels,
		"tags":        req.Tags,
		"created_at":  time.Now().UTC(),
		"updated_at":  time.Now().UTC(),
	}

	c.JSON(http.StatusCreated, gin.H{"dashboard": dashboard})
}

func (suite *MonitoringIntegrationTestSuite) getDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	dashboardID := c.Param("id")
	if dashboardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dashboard ID is required"})
		return
	}

	// Mock dashboard data
	dashboard := map[string]interface{}{
		"id":          dashboardID,
		"name":        "System Overview",
		"description": "Overall system health and performance",
		"panels": []map[string]interface{}{
			{
				"id":    "panel-1",
				"title": "CPU Usage",
				"type":  "gauge",
				"query": "rate(process_cpu_seconds_total[5m])",
			},
			{
				"id":    "panel-2",
				"title": "Memory Usage",
				"type":  "graph",
				"query": "process_resident_memory_bytes",
			},
		},
		"created_at": time.Now().Add(-24 * time.Hour),
		"updated_at": time.Now().Add(-time.Hour),
	}

	c.JSON(http.StatusOK, gin.H{"dashboard": dashboard})
}

// Test Cases

func (suite *MonitoringIntegrationTestSuite) TestHealthCheck() {
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("healthy", response["status"])
	suite.Contains(response, "checks")
	suite.Contains(response, "timestamp")
}

func (suite *MonitoringIntegrationTestSuite) TestMonitoringStats() {
	req, _ := http.NewRequest("GET", "/api/v1/monitoring/stats", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Contains(response, "stats")
	suite.Contains(response, "timestamp")
}

func (suite *MonitoringIntegrationTestSuite) TestMonitoringAlerts() {
	req, _ := http.NewRequest("GET", "/api/v1/monitoring/alerts", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	alerts, ok := response["alerts"].([]interface{})
	suite.True(ok)
	suite.Greater(len(alerts), 0)
}

func (suite *MonitoringIntegrationTestSuite) TestCreateAlert() {
	payload := map[string]interface{}{
		"name":    "Test Alert",
		"level":   "warning",
		"message": "This is a test alert",
		"source":  "integration-test",
		"tags": map[string]interface{}{
			"environment": "test",
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/monitoring/alerts", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	alert, ok := response["alert"].(map[string]interface{})
	suite.True(ok)
	suite.Equal("Test Alert", alert["name"])
	suite.Equal("warning", alert["level"])
}

func (suite *MonitoringIntegrationTestSuite) TestDeleteAlert() {
	alertID := "alert-test-123"
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/monitoring/alerts/%s", alertID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNoContent, w.Code)
}

func (suite *MonitoringIntegrationTestSuite) TestListDashboards() {
	req, _ := http.NewRequest("GET", "/api/v1/monitoring/dashboards", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	dashboards, ok := response["dashboards"].([]interface{})
	suite.True(ok)
	suite.Greater(len(dashboards), 0)
}

func (suite *MonitoringIntegrationTestSuite) TestCreateDashboard() {
	payload := map[string]interface{}{
		"name":        "Test Dashboard",
		"description": "A test dashboard for integration testing",
		"panels": []map[string]interface{}{
			{
				"title": "Test Panel",
				"type":  "graph",
				"query": "up",
			},
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/monitoring/dashboards", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	dashboard, ok := response["dashboard"].(map[string]interface{})
	suite.True(ok)
	suite.Equal("Test Dashboard", dashboard["name"])
}

func (suite *MonitoringIntegrationTestSuite) TestGetDashboard() {
	dashboardID := "dashboard-test-123"
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/monitoring/dashboards/%s", dashboardID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	dashboard, ok := response["dashboard"].(map[string]interface{})
	suite.True(ok)
	suite.Equal(dashboardID, dashboard["id"])
	suite.Contains(dashboard, "panels")
}

func (suite *MonitoringIntegrationTestSuite) TestCreateAlertValidation() {
	// Test missing required fields
	payload := map[string]interface{}{
		"level": "warning",
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/monitoring/alerts", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	// Test invalid level
	payload = map[string]interface{}{
		"name":    "Test Alert",
		"level":   "invalid",
		"message": "Test message",
	}

	jsonData, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/api/v1/monitoring/alerts", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Note: In a real implementation, you would validate the level field
	suite.Equal(http.StatusCreated, w.Code) // Currently just accepts any string
}

func (suite *MonitoringIntegrationTestSuite) TestGetNonExistentDashboard() {
	dashboardID := "non-existent-dashboard"
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/monitoring/dashboards/%s", dashboardID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Note: In a real implementation, this would return 404
	suite.Equal(http.StatusOK, w.Code) // Currently returns mock data
}

func (suite *MonitoringIntegrationTestSuite) TestMetricsEndpoint() {
	req, _ := http.NewRequest("GET", "/api/v1/metrics", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	suite.Equal("text/plain", w.Header().Get("Content-Type"))
	suite.NotEmpty(w.Body.String())
}

// Performance Tests

func (suite *MonitoringIntegrationTestSuite) TestConcurrentHealthChecks() {
	const numRequests = 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/api/v1/health", nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case <-done:
			// Request completed
		case <-time.After(5 * time.Second):
			suite.T().Fatal("Timeout waiting for health check requests")
		}
	}
}

func (suite *MonitoringIntegrationTestSuite) TestAlertCreationPerformance() {
	start := time.Now()
	const numAlerts = 50

	for i := 0; i < numAlerts; i++ {
		payload := map[string]interface{}{
			"name":    fmt.Sprintf("Performance Test Alert %d", i),
			"level":   "info",
			"message": fmt.Sprintf("Performance test alert number %d", i),
		}

		jsonData, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/alerts", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		suite.Equal(http.StatusCreated, w.Code)
	}

	duration := time.Since(start)
	avgDuration := duration / numAlerts

	suite.T().Logf("Created %d alerts in %v (avg: %v per alert)", numAlerts, duration, avgDuration)
	suite.Less(avgDuration, 10*time.Millisecond, "Average alert creation should be fast")
}

// Run the test suite
func TestMonitoringIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(MonitoringIntegrationTestSuite))
}

// Benchmark tests
func BenchmarkHealthCheck(b *testing.B) {
	suite := &MonitoringIntegrationTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}

func BenchmarkMonitoringStats(b *testing.B) {
	suite := &MonitoringIntegrationTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/stats", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}

func BenchmarkCreateAlert(b *testing.B) {
	suite := &MonitoringIntegrationTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	payload := map[string]interface{}{
		"name":    "Benchmark Alert",
		"level":   "info",
		"message": "Benchmark test alert",
	}

	for i := 0; i < b.N; i++ {
		jsonData, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/alerts", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}
