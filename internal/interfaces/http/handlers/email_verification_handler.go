package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"erpgo/internal/application/services/email"
	"erpgo/internal/domain/users/entities"
)

// EmailVerificationHandler handles email verification endpoints
type EmailVerificationHandler struct {
	emailVerificationService email.Service
}

// NewEmailVerificationHandler creates a new email verification handler
func NewEmailVerificationHandler(emailVerificationService email.Service) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		emailVerificationService: emailVerificationService,
	}
}

// SendVerificationEmail handles the request to send a verification email
func (h *EmailVerificationHandler) SendVerificationEmail(c *gin.Context) {
	var req entities.EmailVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	err := h.emailVerificationService.SendVerificationEmail(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "EMAIL_SEND_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email sent successfully",
		"code":    "EMAIL_SENT",
	})
}

// VerifyEmail handles the request to verify an email with a token
func (h *EmailVerificationHandler) VerifyEmail(c *gin.Context) {
	var req entities.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	response, err := h.emailVerificationService.VerifyEmail(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "VERIFICATION_FAILED",
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// ResendVerificationEmail handles the request to resend a verification email
func (h *EmailVerificationHandler) ResendVerificationEmail(c *gin.Context) {
	var req entities.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	err := h.emailVerificationService.ResendVerificationEmail(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "EMAIL_RESEND_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email resent successfully",
		"code":    "EMAIL_RESENT",
	})
}

// GetVerificationStatus handles the request to get verification status
func (h *EmailVerificationHandler) GetVerificationStatus(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
			"code":  "INVALID_USER_ID",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	status, err := h.emailVerificationService.GetVerificationStatus(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "STATUS_CHECK_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"code":   "STATUS_RETRIEVED",
	})
}

// VerifyPasswordResetToken handles the request to verify a password reset token
func (h *EmailVerificationHandler) VerifyPasswordResetToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Token is required",
			"code":  "TOKEN_REQUIRED",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	userID, err := h.emailVerificationService.VerifyPasswordResetToken(ctx, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "TOKEN_INVALID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID.String(),
		"code":    "TOKEN_VALID",
	})
}

// VerifyEmailChangeToken handles the request to verify an email change token
func (h *EmailVerificationHandler) VerifyEmailChangeToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Token is required",
			"code":  "TOKEN_REQUIRED",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	userID, newEmail, err := h.emailVerificationService.VerifyEmailChangeToken(ctx, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "TOKEN_INVALID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":   userID.String(),
		"new_email": newEmail,
		"code":      "TOKEN_VALID",
	})
}

// InvalidateToken handles the request to invalidate a token
func (h *EmailVerificationHandler) InvalidateToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	err := h.emailVerificationService.InvalidateToken(ctx, req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "TOKEN_INVALIDATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token invalidated successfully",
		"code":    "TOKEN_INVALIDATED",
	})
}

// CleanupExpiredTokens handles the request to cleanup expired tokens (admin only)
func (h *EmailVerificationHandler) CleanupExpiredTokens(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	count, err := h.emailVerificationService.CleanupExpiredTokens(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "CLEANUP_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Expired tokens cleaned up successfully",
		"deleted_count": count,
		"code":          "CLEANUP_SUCCESS",
	})
}