package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/email"
)

// Service defines the interface for email verification operations
type Service interface {
	// Email verification operations
	SendVerificationEmail(ctx context.Context, req *entities.EmailVerificationRequest) error
	VerifyEmail(ctx context.Context, req *entities.VerifyEmailRequest) (*entities.VerifyEmailResponse, error)
	ResendVerificationEmail(ctx context.Context, req *entities.ResendVerificationRequest) error
	VerifyPasswordResetToken(ctx context.Context, token string) (uuid.UUID, error)
	VerifyEmailChangeToken(ctx context.Context, token string) (uuid.UUID, string, error)

	// Token management
	InvalidateToken(ctx context.Context, token string) error
	CleanupExpiredTokens(ctx context.Context) (int64, error)

	// Verification status
	GetVerificationStatus(ctx context.Context, userID uuid.UUID) (*VerificationStatus, error)
}

// ServiceImpl implements the email verification service
type ServiceImpl struct {
	userRepo             repositories.UserRepository
	emailVerificationRepo repositories.EmailVerificationRepository
	emailService          *email.SMTPService
	rateLimiter          *email.EmailRateLimiter
}

// NewService creates a new email verification service
func NewService(
	userRepo repositories.UserRepository,
	emailVerificationRepo repositories.EmailVerificationRepository,
	emailService *email.SMTPService,
) Service {
	rateLimiter := email.NewEmailRateLimiter(
		5,                // max 5 attempts
		1*time.Hour,      // per hour
	)

	return &ServiceImpl{
		userRepo:             userRepo,
		emailVerificationRepo: emailVerificationRepo,
		emailService:          emailService,
		rateLimiter:          rateLimiter,
	}
}

// SendVerificationEmail sends a verification email to the user
func (s *ServiceImpl) SendVerificationEmail(ctx context.Context, req *entities.EmailVerificationRequest) error {
	// Check rate limiting
	if !s.rateLimiter.CanSendEmail(req.Email) {
		return errors.New("too many verification emails sent. Please try again later")
	}

	// Find user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if user is already verified
	if user.IsVerified {
		return errors.New("user is already verified")
	}

	// Generate verification token
	token := email.GenerateHexToken(64) // 64 characters = 32 bytes
	expiration := entities.DefaultVerificationExpiration

	verification := entities.GenerateVerificationToken(
		user.ID,
		user.Email,
		entities.TokenTypeVerification,
		expiration,
	)

	verification.Token = token

	// Deactivate any existing verification tokens for this user
	err = s.emailVerificationRepo.DeactivateAllUserVerifications(ctx, user.ID, entities.TokenTypeVerification)
	if err != nil {
		return fmt.Errorf("failed to deactivate existing tokens: %w", err)
	}

	// Save new verification token
	err = s.emailVerificationRepo.CreateVerification(ctx, verification)
	if err != nil {
		return fmt.Errorf("failed to create verification token: %w", err)
	}

	// Send verification email
	err = s.emailService.SendVerificationEmail(user.Email, token)
	if err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	// Record the attempt
	s.rateLimiter.RecordEmailAttempt(user.Email)

	return nil
}

// VerifyEmail verifies an email verification token
func (s *ServiceImpl) VerifyEmail(ctx context.Context, req *entities.VerifyEmailRequest) (*entities.VerifyEmailResponse, error) {
	// Find verification token
	verification, err := s.emailVerificationRepo.GetVerificationByToken(ctx, req.Token)
	if err != nil {
		return &entities.VerifyEmailResponse{
			Message: "Invalid verification token",
			Success: false,
		}, nil
	}

	// Check if token is valid
	if !verification.IsValid() {
		return &entities.VerifyEmailResponse{
			Message: "Verification token is expired or already used",
			Success: false,
		}, nil
	}

	// Check token type
	if verification.TokenType != entities.TokenTypeVerification {
		return &entities.VerifyEmailResponse{
			Message: "Invalid token type",
			Success: false,
		}, nil
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, verification.UserID)
	if err != nil {
		return &entities.VerifyEmailResponse{
			Message: "User not found",
			Success: false,
		}, nil
	}

	// Check if email matches
	if user.Email != verification.Email {
		return &entities.VerifyEmailResponse{
			Message: "Email does not match",
			Success: false,
		}, nil
	}

	// Mark token as used
	err = s.emailVerificationRepo.MarkTokenAsUsed(ctx, verification.ID)
	if err != nil {
		return &entities.VerifyEmailResponse{
			Message: "Failed to verify token",
			Success: false,
		}, nil
	}

	// Mark user as verified
	user.IsVerified = true
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return &entities.VerifyEmailResponse{
			Message: "Failed to update user verification status",
			Success: false,
		}, nil
	}

	return &entities.VerifyEmailResponse{
		Message: "Email verified successfully",
		Success: true,
		UserID:  user.ID.String(),
	}, nil
}

// ResendVerificationEmail resends a verification email
func (s *ServiceImpl) ResendVerificationEmail(ctx context.Context, req *entities.ResendVerificationRequest) error {
	// Check rate limiting
	if !s.rateLimiter.CanSendEmail(req.Email) {
		return errors.New("too many verification emails sent. Please try again later")
	}

	// Find user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if user is already verified
	if user.IsVerified {
		return errors.New("user is already verified")
	}

	// Check for existing active verification token
	existingToken, err := s.emailVerificationRepo.GetActiveVerificationByUserAndType(
		ctx,
		user.ID,
		entities.TokenTypeVerification,
	)
	if err == nil && existingToken != nil && !existingToken.IsExpired() {
		// Resend existing token if it's still valid
		err = s.emailService.SendVerificationEmail(user.Email, existingToken.Token)
		if err != nil {
			return fmt.Errorf("failed to resend verification email: %w", err)
		}

		s.rateLimiter.RecordEmailAttempt(user.Email)
		return nil
	}

	// Generate new token
	verificationReq := &entities.EmailVerificationRequest{
		Email:    req.Email,
		UserID:   user.ID.String(),
		TokenType: entities.TokenTypeVerification,
	}

	return s.SendVerificationEmail(ctx, verificationReq)
}

// VerifyPasswordResetToken verifies a password reset token
func (s *ServiceImpl) VerifyPasswordResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	verification, err := s.emailVerificationRepo.GetVerificationByToken(ctx, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid token: %w", err)
	}

	if !verification.IsValid() {
		return uuid.Nil, errors.New("token is expired or already used")
	}

	if verification.TokenType != entities.TokenTypePasswordReset {
		return uuid.Nil, errors.New("invalid token type")
	}

	return verification.UserID, nil
}

// VerifyEmailChangeToken verifies an email change token
func (s *ServiceImpl) VerifyEmailChangeToken(ctx context.Context, token string) (uuid.UUID, string, error) {
	verification, err := s.emailVerificationRepo.GetVerificationByToken(ctx, token)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid token: %w", err)
	}

	if !verification.IsValid() {
		return uuid.Nil, "", errors.New("token is expired or already used")
	}

	if verification.TokenType != entities.TokenTypeEmailChange {
		return uuid.Nil, "", errors.New("invalid token type")
	}

	return verification.UserID, verification.Email, nil
}

// InvalidateToken invalidates a verification token
func (s *ServiceImpl) InvalidateToken(ctx context.Context, token string) error {
	verification, err := s.emailVerificationRepo.GetVerificationByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("token not found: %w", err)
	}

	return s.emailVerificationRepo.MarkTokenAsUsed(ctx, verification.ID)
}

// CleanupExpiredTokens removes expired verification tokens
func (s *ServiceImpl) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	return s.emailVerificationRepo.DeleteExpiredVerifications(ctx)
}

// GetVerificationStatus gets the verification status for a user
func (s *ServiceImpl) GetVerificationStatus(ctx context.Context, userID uuid.UUID) (*VerificationStatus, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	stats, err := s.emailVerificationRepo.GetVerificationStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification stats: %w", err)
	}

	return &VerificationStatus{
		UserID:            userID,
		Email:             user.Email,
		IsVerified:        user.IsVerified,
		HasActiveToken:    stats.ActiveVerifications > 0,
		LastSentAt:        stats.LastVerificationSent,
		CanResend:         s.rateLimiter.CanSendEmail(user.Email),
		TotalVerifications: stats.TotalVerifications,
	}, nil
}

// VerificationStatus represents the verification status of a user
type VerificationStatus struct {
	UserID            uuid.UUID  `json:"user_id"`
	Email             string     `json:"email"`
	IsVerified        bool       `json:"is_verified"`
	HasActiveToken    bool       `json:"has_active_token"`
	LastSentAt        *time.Time `json:"last_sent_at,omitempty"`
	CanResend         bool       `json:"can_resend"`
	TotalVerifications int64     `json:"total_verifications"`
}