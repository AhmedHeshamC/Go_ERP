package health

import (
	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for health check endpoints
type Handler struct {
	checker *HealthChecker
}

// NewHandler creates a new health check handler
func NewHandler(checker *HealthChecker) *Handler {
	return &Handler{
		checker: checker,
	}
}

// LivenessHandler handles /health/live endpoint
// Returns 200 if the application is running
// Validates: Requirements 8.1
func (h *Handler) LivenessHandler(c *gin.Context) {
	ctx := c.Request.Context()
	status := h.checker.CheckLiveness(ctx)

	c.JSON(status.HTTPStatusCode(), gin.H{
		"status":    status.Status,
		"timestamp": status.Timestamp,
		"message":   "Service is alive",
	})
}

// ReadinessHandler handles /health/ready endpoint
// Returns 200 if ready to serve traffic, 503 if not ready
// Checks database and Redis connectivity
// Validates: Requirements 8.1, 8.2, 8.3, 8.4
func (h *Handler) ReadinessHandler(c *gin.Context) {
	ctx := c.Request.Context()
	status := h.checker.CheckReadiness(ctx)

	response := gin.H{
		"status":    status.Status,
		"timestamp": status.Timestamp,
	}

	// Add detailed status for each dependency
	// Validates: Requirements 8.3
	if len(status.Checks) > 0 {
		checks := make(map[string]gin.H)
		for name, check := range status.Checks {
			checks[name] = gin.H{
				"status":    check.Status,
				"message":   check.Message,
				"duration":  check.Duration.String(),
				"timestamp": check.Timestamp,
			}
			// Include details if check failed
			if check.Status == StatusUnhealthy && len(check.Details) > 0 {
				checks[name]["details"] = check.Details
			}
		}
		response["checks"] = checks
	}

	// Set appropriate message
	if status.Status == StatusHealthy {
		response["message"] = "Service is ready"
	} else {
		response["message"] = "Service is not ready"
	}

	c.JSON(status.HTTPStatusCode(), response)
}

// RegisterRoutes registers health check routes with the Gin router
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	health := router.Group("/health")
	{
		health.GET("/live", h.LivenessHandler)
		health.GET("/ready", h.ReadinessHandler)
	}
}
