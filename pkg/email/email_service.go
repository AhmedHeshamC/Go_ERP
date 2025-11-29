package email

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"erpgo/internal/domain/users/entities"
	"github.com/google/uuid"
	"github.com/jordan-wright/email"
)

// SMTPService implements EmailService using SMTP
type SMTPService struct {
	config *entities.EmailConfig
	auth   smtp.Auth
}

// NewSMTPService creates a new SMTP email service
func NewSMTPService(config *entities.EmailConfig) *SMTPService {
	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)

	return &SMTPService{
		config: config,
		auth:   auth,
	}
}

// SendVerificationEmail sends a verification email to the user
func (s *SMTPService) SendVerificationEmail(email, token string) error {
	subject := "Verify Your Email Address"
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.getBaseURL(), token)

	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>Welcome to ERPGo!</h2>
			<p>Thank you for registering. Please verify your email address by clicking the link below:</p>
			<p><a href="%s">Verify Email Address</a></p>
			<p>If you didn't create an account, you can safely ignore this email.</p>
			<p>This link will expire in 24 hours.</p>
			<br>
			<p>Best regards,<br>The ERPGo Team</p>
		</body>
		</html>
	`, verificationURL)

	textBody := fmt.Sprintf(`
		Welcome to ERPGo!

		Thank you for registering. Please verify your email address by visiting:
		%s

		If you didn't create an account, you can safely ignore this email.
		This link will expire in 24 hours.

		Best regards,
		The ERPGo Team
	`, verificationURL)

	content := &entities.EmailContent{
		ToEmail:  email,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	return s.SendEmail(content)
}

// SendPasswordResetEmail sends a password reset email to the user
func (s *SMTPService) SendPasswordResetEmail(email, token string) error {
	subject := "Reset Your Password"
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.getBaseURL(), token)

	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>Password Reset Request</h2>
			<p>We received a request to reset your password for your ERPGo account.</p>
			<p>Click the link below to reset your password:</p>
			<p><a href="%s">Reset Password</a></p>
			<p>If you didn't request a password reset, you can safely ignore this email.</p>
			<p>This link will expire in 1 hour.</p>
			<br>
			<p>Best regards,<br>The ERPGo Team</p>
		</body>
		</html>
	`, resetURL)

	textBody := fmt.Sprintf(`
		Password Reset Request

		We received a request to reset your password for your ERPGo account.
		Visit the link below to reset your password:
		%s

		If you didn't request a password reset, you can safely ignore this email.
		This link will expire in 1 hour.

		Best regards,
		The ERPGo Team
	`, resetURL)

	content := &entities.EmailContent{
		ToEmail:  email,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	return s.SendEmail(content)
}

// SendEmailChangeVerification sends an email change verification email
func (s *SMTPService) SendEmailChangeVerification(email, token string) error {
	subject := "Confirm Your New Email Address"
	verificationURL := fmt.Sprintf("%s/confirm-email-change?token=%s", s.getBaseURL(), token)

	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>Confirm Your New Email Address</h2>
			<p>You requested to change your email address for your ERPGo account.</p>
			<p>Please confirm your new email address by clicking the link below:</p>
			<p><a href="%s">Confirm Email Address</a></p>
			<p>If you didn't request this change, you can safely ignore this email.</p>
			<p>This link will expire in 30 minutes.</p>
			<br>
			<p>Best regards,<br>The ERPGo Team</p>
		</body>
		</html>
	`, verificationURL)

	textBody := fmt.Sprintf(`
		Confirm Your New Email Address

		You requested to change your email address for your ERPGo account.
		Please confirm your new email address by visiting:
		%s

		If you didn't request this change, you can safely ignore this email.
		This link will expire in 30 minutes.

		Best regards,
		The ERPGo Team
	`, verificationURL)

	content := &entities.EmailContent{
		ToEmail:  email,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	return s.SendEmail(content)
}

// SendEmail sends an email using SMTP
func (s *SMTPService) SendEmail(content *entities.EmailContent) error {
	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	e.To = []string{content.ToEmail}
	e.Subject = content.Subject
	e.Text = []byte(content.TextBody)
	e.HTML = []byte(content.HTMLBody)

	// Set headers
	e.Headers.Add("X-Mailer", "ERPGo Mailer")
	e.Headers.Add("X-Priority", "3")
	e.Headers.Add("MIME-Version", "1.0")
	e.Headers.Add("Content-Type", "text/html; charset=UTF-8")

	// Configure SMTP server address
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Send email with TLS if configured
	if s.config.UseTLS || s.config.UseSSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.config.SMTPHost,
			MinVersion:         tls.VersionTLS12, // G402: Set minimum TLS version to 1.2
		}

		if s.config.UseSSL {
			// SSL (implicit TLS on port 465)
			return e.SendWithTLS(addr, s.auth, tlsConfig)
		} else {
			// TLS (explicit TLS on port 587)
			return e.SendWithStartTLS(addr, s.auth, tlsConfig)
		}
	}

	// Send without TLS
	return e.Send(addr, s.auth)
}

// getBaseURL returns the base URL for the application
func (s *SMTPService) getBaseURL() string {
	// This should be configurable, but for now return a default
	return "https://app.erpgo.example.com"
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	max := big.NewInt(int64(len(charset)))

	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			// Fallback to less secure method if crypto/rand fails
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}

	return string(b)
}

// GenerateHexToken generates a cryptographically secure hexadecimal token
func GenerateHexToken(length int) string {
	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to UUID-based token
		return uuid.New().String()
	}
	return hex.EncodeToString(bytes)
}

// ValidateEmailFormat validates email format using net/mail
func ValidateEmailFormat(email string) bool {
	if email == "" {
		return false
	}

	// Use net/mail for basic email validation
	_, err := mail.ParseAddress(email)
	return err == nil && strings.Contains(email, "@")
}

// GetDomainFromEmail extracts the domain from an email address
func GetDomainFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// IsDisposableEmail checks if the email domain is known to be disposable
func IsDisposableEmail(email string) bool {
	domain := GetDomainFromEmail(email)

	disposableDomains := []string{
		"10minutemail.com",
		"guerrillamail.com",
		"mailinator.com",
		"tempmail.org",
		"yopmail.com",
		"throwaway.email",
		"temp-mail.org",
		"maildrop.cc",
		"sharklasers.com",
		"mailnesia.com",
	}

	for _, disposable := range disposableDomains {
		if strings.Contains(domain, disposable) {
			return true
		}
	}

	return false
}

// EmailRateLimiter represents a simple rate limiter for email sending
type EmailRateLimiter struct {
	sendAttempts map[string][]time.Time
	maxAttempts  int
	timeWindow   time.Duration
}

// NewEmailRateLimiter creates a new email rate limiter
func NewEmailRateLimiter(maxAttempts int, timeWindow time.Duration) *EmailRateLimiter {
	return &EmailRateLimiter{
		sendAttempts: make(map[string][]time.Time),
		maxAttempts:  maxAttempts,
		timeWindow:   timeWindow,
	}
}

// CanSendEmail checks if an email can be sent to the given address
func (r *EmailRateLimiter) CanSendEmail(email string) bool {
	now := time.Now()

	// Clean old attempts
	r.cleanOldAttempts(email, now)

	// Check current attempts
	attempts, exists := r.sendAttempts[email]
	if !exists {
		return true
	}

	return len(attempts) < r.maxAttempts
}

// RecordEmailAttempt records an email send attempt
func (r *EmailRateLimiter) RecordEmailAttempt(email string) {
	now := time.Now()
	r.sendAttempts[email] = append(r.sendAttempts[email], now)
}

// cleanOldAttempts removes old attempts outside the time window
func (r *EmailRateLimiter) cleanOldAttempts(email string, now time.Time) {
	attempts, exists := r.sendAttempts[email]
	if !exists {
		return
	}

	var validAttempts []time.Time
	cutoff := now.Add(-r.timeWindow)

	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}

	if len(validAttempts) > 0 {
		r.sendAttempts[email] = validAttempts
	} else {
		delete(r.sendAttempts, email)
	}
}
