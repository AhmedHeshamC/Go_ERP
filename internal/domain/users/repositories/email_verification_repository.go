package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"erpgo/internal/domain/users/entities"
)

// EmailVerificationRepository defines the interface for email verification data operations
type EmailVerificationRepository interface {
	// Verification token operations
	CreateVerification(ctx context.Context, verification *entities.EmailVerification) error
	GetVerificationByID(ctx context.Context, id uuid.UUID) (*entities.EmailVerification, error)
	GetVerificationByToken(ctx context.Context, token string) (*entities.EmailVerification, error)
	GetActiveVerificationByUserAndType(ctx context.Context, userID uuid.UUID, tokenType string) (*entities.EmailVerification, error)
	UpdateVerification(ctx context.Context, verification *entities.EmailVerification) error
	DeleteVerification(ctx context.Context, id uuid.UUID) error
	MarkTokenAsUsed(ctx context.Context, tokenID uuid.UUID) error

	// User verification operations
	GetUserVerifications(ctx context.Context, userID uuid.UUID, tokenType string) ([]*entities.EmailVerification, error)
	GetActiveVerificationsByEmail(ctx context.Context, email, tokenType string) ([]*entities.EmailVerification, error)
	DeactivateAllUserVerifications(ctx context.Context, userID uuid.UUID, tokenType string) error
	DeleteExpiredVerifications(ctx context.Context) (int64, error)

	// Verification statistics and cleanup
	GetVerificationStats(ctx context.Context, userID uuid.UUID) (*VerificationStats, error)
	CleanupExpiredVerifications(ctx context.Context, olderThan time.Duration) (int64, error)
}

// VerificationStats represents statistics for user verifications
type VerificationStats struct {
	TotalVerifications      int64     `json:"total_verifications"`
	ActiveVerifications     int64     `json:"active_verifications"`
	UsedVerifications       int64     `json:"used_verifications"`
	ExpiredVerifications    int64     `json:"expired_verifications"`
	LastVerificationSent    *time.Time `json:"last_verification_sent,omitempty"`
	LastVerificationUsed    *time.Time `json:"last_verification_used,omitempty"`
	VerificationTokenType   string    `json:"verification_token_type"`
}

// EmailRepository defines the interface for email sending operations
type EmailRepository interface {
	// Email sending operations
	SendEmail(ctx context.Context, content *entities.EmailContent) error
	SendVerificationEmail(ctx context.Context, email, token string) error
	SendPasswordResetEmail(ctx context.Context, email, token string) error
	SendEmailChangeVerification(ctx context.Context, email, token string) error

	// Email template operations
	GetEmailTemplate(ctx context.Context, templateType string) (*entities.EmailTemplate, error)
	CreateEmailTemplate(ctx context.Context, template *entities.EmailTemplate) error
	UpdateEmailTemplate(ctx context.Context, template *entities.EmailTemplate) error

	// Email tracking
	TrackEmailSent(ctx context.Context, emailID uuid.UUID, email, templateType string) error
	TrackEmailDelivered(ctx context.Context, emailID uuid.UUID) error
	TrackEmailOpened(ctx context.Context, emailID uuid.UUID) error
	TrackEmailClicked(ctx context.Context, emailID uuid.UUID, link string) error

	// Email statistics
	GetEmailStats(ctx context.Context, filter *EmailStatsFilter) (*EmailStats, error)
}

// EmailStatsFilter defines filtering options for email statistics
type EmailStatsFilter struct {
	TemplateType string     `json:"template_type"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Status       string     `json:"status,omitempty"`
	UserID       uuid.UUID  `json:"user_id,omitempty"`
}

// EmailStats represents email sending statistics
type EmailStats struct {
	TotalSent     int64 `json:"total_sent"`
	TotalDelivered int64 `json:"total_delivered"`
	TotalOpened   int64 `json:"total_opened"`
	TotalClicked  int64 `json:"total_clicked"`
	DeliveryRate  float64 `json:"delivery_rate"`
	OpenRate      float64 `json:"open_rate"`
	ClickRate     float64 `json:"click_rate"`
}

// EmailLog represents an email delivery log entry
type EmailLog struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	EmailID     uuid.UUID  `json:"email_id" db:"email_id"`
	ToEmail     string     `json:"to_email" db:"to_email"`
	TemplateType string    `json:"template_type" db:"template_type"`
	Status      string     `json:"status" db:"status"` // 'sent', 'delivered', 'opened', 'clicked', 'failed'
	SentAt      time.Time  `json:"sent_at" db:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	OpenedAt    *time.Time `json:"opened_at,omitempty" db:"opened_at"`
	ClickedAt   *time.Time `json:"clicked_at,omitempty" db:"clicked_at"`
	ErrorMessage string     `json:"error_message,omitempty" db:"error_message"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// EmailLogRepository defines the interface for email logging operations
type EmailLogRepository interface {
	// Log operations
	CreateEmailLog(ctx context.Context, log *EmailLog) error
	GetEmailLogByID(ctx context.Context, id uuid.UUID) (*EmailLog, error)
	GetEmailLogsByFilter(ctx context.Context, filter *EmailLogFilter) ([]*EmailLog, error)
	UpdateEmailLogStatus(ctx context.Context, id uuid.UUID, status string, timestamp *time.Time, errorMessage string) error

	// Statistics operations
	GetEmailStats(ctx context.Context, filter *EmailStatsFilter) (*EmailStats, error)
	GetEmailStatsByUser(ctx context.Context, userID uuid.UUID, timeRange time.Duration) (*EmailStats, error)
}

// EmailLogFilter defines filtering options for email logs
type EmailLogFilter struct {
	EmailID      uuid.UUID  `json:"email_id,omitempty"`
	ToEmail      string     `json:"to_email,omitempty"`
	TemplateType string     `json:"template_type,omitempty"`
	Status       string     `json:"status,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	UserID       uuid.UUID  `json:"user_id,omitempty"`
	Limit        int        `json:"limit,omitempty"`
	Offset       int        `json:"offset,omitempty"`
	OrderBy      string     `json:"order_by,omitempty"`
	SortOrder    string     `json:"sort_order,omitempty"`
}