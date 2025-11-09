package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"erpgo/internal/interfaces/http/handlers"
	"erpgo/pkg/auth"
)

// SetupEmailVerificationRoutes sets up email verification routes
func SetupEmailVerificationRoutes(router *gin.Engine, emailVerificationHandler *handlers.EmailVerificationHandler, jwtService *auth.JWTService) {
	// Public email verification routes
	emailGroup := router.Group("/api/v1/email")
	{
		// Send verification email
		emailGroup.POST("/send-verification", emailVerificationHandler.SendVerificationEmail)

		// Verify email with token
		emailGroup.POST("/verify", emailVerificationHandler.VerifyEmail)

		// Resend verification email
		emailGroup.POST("/resend-verification", emailVerificationHandler.ResendVerificationEmail)

		// Verify password reset token
		emailGroup.GET("/verify-password-reset", emailVerificationHandler.VerifyPasswordResetToken)

		// Verify email change token
		emailGroup.GET("/verify-email-change", emailVerificationHandler.VerifyEmailChangeToken)
	}

	// Protected routes (require authentication)
	protectedEmailGroup := router.Group("/api/v1/email")
	protectedEmailGroup.Use(auth.AuthMiddleware(jwtService))
	{
		// Get verification status
		protectedEmailGroup.GET("/verification-status/:user_id", emailVerificationHandler.GetVerificationStatus)

		// Invalidate token
		protectedEmailGroup.POST("/invalidate-token", emailVerificationHandler.InvalidateToken)
	}

	// Admin routes (require admin role)
	adminEmailGroup := router.Group("/api/v1/admin/email")
	adminEmailGroup.Use(auth.AuthMiddleware(jwtService))
	// TODO: Add role-based authorization middleware
	{
		// Cleanup expired tokens
		adminEmailGroup.POST("/cleanup-expired-tokens", emailVerificationHandler.CleanupExpiredTokens)
	}
}

// SetupEmailVerificationWebRoutes sets up web routes for email verification
func SetupEmailVerificationWebRoutes(router *gin.Engine) {
	// Web routes for email verification (used in emails)
	webGroup := router.Group("/verify")
	{
		// Verify email (for link in emails)
		webGroup.GET("/email", func(c *gin.Context) {
			token := c.Query("token")
			if token == "" {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{
					"error": "Verification token is required",
				})
				return
			}

			// Redirect to the web application with the token
			redirectURL := "/email-verification?token=" + token
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		})

		// Password reset (for link in emails)
		webGroup.GET("/password-reset", func(c *gin.Context) {
			token := c.Query("token")
			if token == "" {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{
					"error": "Reset token is required",
				})
				return
			}

			// Redirect to the web application with the token
			redirectURL := "/password-reset?token=" + token
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		})

		// Email change verification (for link in emails)
		webGroup.GET("/email-change", func(c *gin.Context) {
			token := c.Query("token")
			if token == "" {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{
					"error": "Verification token is required",
				})
				return
			}

			// Redirect to the web application with the token
			redirectURL := "/email-change-verification?token=" + token
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		})
	}
}