package entities

import (
	"time"

	"github.com/google/uuid"
)

// EmailVerification represents an email verification token
type EmailVerification struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	Email        string     `json:"email" db:"email"`
	Token        string     `json:"token" db:"token"`
	TokenType    string     `json:"token_type" db:"token_type"` // "verification", "password_reset", "email_change"
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	IsUsed       bool       `json:"is_used" db:"is_used"`
	UsedAt       *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// EmailVerificationRequest represents a request to send verification email
type EmailVerificationRequest struct {
	Email    string `json:"email" validate:"required,email"`
	UserID   string `json:"user_id,omitempty"`
	TokenType string `json:"token_type,omitempty"` // "verification", "password_reset", "email_change"
}

// EmailVerificationResponse represents the response for email verification
type EmailVerificationResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// VerifyEmailRequest represents a request to verify email with token
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyEmailResponse represents the response for email verification
type VerifyEmailResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
	UserID  string `json:"user_id,omitempty"`
}

// ResendVerificationRequest represents a request to resend verification email
type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// Token type constants
const (
	TokenTypeVerification   = "verification"
	TokenTypePasswordReset  = "password_reset"
	TokenTypeEmailChange    = "email_change"
)

// Token expiration durations
const (
	DefaultVerificationExpiration = 24 * time.Hour  // 24 hours
	DefaultPasswordResetExpiration = 1 * time.Hour   // 1 hour
	DefaultEmailChangeExpiration  = 30 * time.Minute // 30 minutes
)

// IsExpired checks if the verification token has expired
func (ev *EmailVerification) IsExpired() bool {
	return time.Now().After(ev.ExpiresAt)
}

// IsValid checks if the verification token is valid (not used and not expired)
func (ev *EmailVerification) IsValid() bool {
	return !ev.IsUsed && !ev.IsExpired()
}

// MarkAsUsed marks the verification token as used
func (ev *EmailVerification) MarkAsUsed() {
	now := time.Now()
	ev.IsUsed = true
	ev.UsedAt = &now
	ev.UpdatedAt = now
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject    string `json:"subject"`
	HTMLBody   string `json:"html_body"`
	TextBody   string `json:"text_body"`
	FromEmail  string `json:"from_email"`
	FromName   string `json:"from_name"`
}

// EmailContent represents the content of a verification email
type EmailContent struct {
	ToEmail   string `json:"to_email"`
	Subject   string `json:"subject"`
	HTMLBody  string `json:"html_body"`
	TextBody  string `json:"text_body"`
}

// EmailConfig represents email service configuration
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
	UseTLS       bool   `json:"use_tls"`
	UseSSL       bool   `json:"use_ssl"`
}

// EmailService represents the email service interface
type EmailService interface {
	SendVerificationEmail(email, token string) error
	SendPasswordResetEmail(email, token string) error
	SendEmailChangeVerification(email, token string) error
	SendEmail(content *EmailContent) error
}

// GenerateVerificationToken creates a new email verification token
func GenerateVerificationToken(userID uuid.UUID, email, tokenType string, expiration time.Duration) *EmailVerification {
	return &EmailVerification{
		ID:        uuid.New(),
		UserID:    userID,
		Email:     email,
		Token:     generateSecureToken(),
		TokenType: tokenType,
		ExpiresAt: time.Now().Add(expiration),
		IsUsed:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() string {
	// This should use a proper cryptographic random generator
	// For now, using UUID as a placeholder
	return uuid.New().String()
}