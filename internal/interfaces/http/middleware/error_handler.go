package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "erpgo/pkg/errors"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details,omitempty"`
	Fields        map[string][]string    `json:"fields,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Timestamp     string                 `json:"timestamp"`
}

// HandleError converts application errors to HTTP responses
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Get correlation ID from context
	correlationID, _ := c.Get("correlation_id")
	correlationIDStr, _ := correlationID.(string)

	// Check if it's an AppError
	if appErr, ok := err.(*apperrors.AppError); ok {
		handleAppError(c, appErr, correlationIDStr)
		return
	}

	// Check for specific error types
	switch e := err.(type) {
	case *apperrors.NotFoundError:
		handleAppError(c, e.AppError, correlationIDStr)
	case *apperrors.ValidationError:
		handleValidationError(c, e, correlationIDStr)
	case *apperrors.ConflictError:
		handleAppError(c, e.AppError, correlationIDStr)
	case *apperrors.UnauthorizedError:
		handleAppError(c, e.AppError, correlationIDStr)
	case *apperrors.ForbiddenError:
		handleAppError(c, e.AppError, correlationIDStr)
	case *apperrors.RateLimitError:
		handleRateLimitError(c, e, correlationIDStr)
	case *apperrors.BadRequestError:
		handleAppError(c, e.AppError, correlationIDStr)
	case *apperrors.TimeoutError:
		handleAppError(c, e.AppError, correlationIDStr)
	case *apperrors.UnavailableError:
		handleAppError(c, e.AppError, correlationIDStr)
	case *apperrors.InternalError:
		handleAppError(c, e.AppError, correlationIDStr)
	default:
		// Unknown error - treat as internal server error
		handleGenericError(c, err, correlationIDStr)
	}
}

// handleAppError handles AppError types
func handleAppError(c *gin.Context, appErr *apperrors.AppError, correlationID string) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:          string(appErr.Code),
			Message:       appErr.Message,
			Details:       appErr.Details,
			CorrelationID: correlationID,
			Timestamp:     appErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	c.JSON(appErr.StatusCode, response)
}

// handleValidationError handles ValidationError with field details
func handleValidationError(c *gin.Context, valErr *apperrors.ValidationError, correlationID string) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:          string(valErr.Code),
			Message:       valErr.Message,
			Details:       valErr.Details,
			Fields:        valErr.Fields,
			CorrelationID: correlationID,
			Timestamp:     valErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	c.JSON(valErr.StatusCode, response)
}

// handleRateLimitError handles RateLimitError with retry-after header
func handleRateLimitError(c *gin.Context, rateLimitErr *apperrors.RateLimitError, correlationID string) {
	// Set Retry-After header
	if rateLimitErr.RetryAfter > 0 {
		c.Header("Retry-After", fmt.Sprintf("%d", int(rateLimitErr.RetryAfter.Seconds())))
	}

	response := ErrorResponse{
		Error: ErrorDetail{
			Code:          string(rateLimitErr.Code),
			Message:       rateLimitErr.Message,
			Details:       rateLimitErr.Details,
			CorrelationID: correlationID,
			Timestamp:     rateLimitErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	c.JSON(rateLimitErr.StatusCode, response)
}

// handleGenericError handles unknown errors
func handleGenericError(c *gin.Context, err error, correlationID string) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:          string(apperrors.ErrCodeInternal),
			Message:       "An internal error occurred",
			CorrelationID: correlationID,
			Timestamp:     time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	c.JSON(http.StatusInternalServerError, response)
}
