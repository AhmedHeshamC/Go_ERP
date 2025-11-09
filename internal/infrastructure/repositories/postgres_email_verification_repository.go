package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/database"
)

// PostgresEmailVerificationRepository implements EmailVerificationRepository for PostgreSQL
type PostgresEmailVerificationRepository struct {
	db *database.Database
}

// NewPostgresEmailVerificationRepository creates a new PostgreSQL email verification repository
func NewPostgresEmailVerificationRepository(db *database.Database) *PostgresEmailVerificationRepository {
	return &PostgresEmailVerificationRepository{
		db: db,
	}
}

// CreateVerification creates a new email verification token
func (r *PostgresEmailVerificationRepository) CreateVerification(ctx context.Context, verification *entities.EmailVerification) error {
	query := `
		INSERT INTO email_verifications (id, user_id, email, token, token_type, expires_at, is_used, used_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(ctx, query,
		verification.ID,
		verification.UserID,
		verification.Email,
		verification.Token,
		verification.TokenType,
		verification.ExpiresAt,
		verification.IsUsed,
		verification.UsedAt,
		verification.CreatedAt,
		verification.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create verification: %w", err)
	}

	return nil
}

// GetVerificationByID retrieves a verification by ID
func (r *PostgresEmailVerificationRepository) GetVerificationByID(ctx context.Context, id uuid.UUID) (*entities.EmailVerification, error) {
	query := `
		SELECT id, user_id, email, token, token_type, expires_at, is_used, used_at, created_at, updated_at
		FROM email_verifications
		WHERE id = $1`

	verification := &entities.EmailVerification{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.Email,
		&verification.Token,
		&verification.TokenType,
		&verification.ExpiresAt,
		&verification.IsUsed,
		&verification.UsedAt,
		&verification.CreatedAt,
		&verification.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("verification not found")
		}
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	return verification, nil
}

// GetVerificationByToken retrieves a verification by token
func (r *PostgresEmailVerificationRepository) GetVerificationByToken(ctx context.Context, token string) (*entities.EmailVerification, error) {
	query := `
		SELECT id, user_id, email, token, token_type, expires_at, is_used, used_at, created_at, updated_at
		FROM email_verifications
		WHERE token = $1`

	verification := &entities.EmailVerification{}
	err := r.db.QueryRow(ctx, query, token).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.Email,
		&verification.Token,
		&verification.TokenType,
		&verification.ExpiresAt,
		&verification.IsUsed,
		&verification.UsedAt,
		&verification.CreatedAt,
		&verification.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("verification not found")
		}
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	return verification, nil
}

// GetActiveVerificationByUserAndType retrieves an active verification for a user by type
func (r *PostgresEmailVerificationRepository) GetActiveVerificationByUserAndType(ctx context.Context, userID uuid.UUID, tokenType string) (*entities.EmailVerification, error) {
	query := `
		SELECT id, user_id, email, token, token_type, expires_at, is_used, used_at, created_at, updated_at
		FROM email_verifications
		WHERE user_id = $1 AND token_type = $2 AND is_used = FALSE AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC
		LIMIT 1`

	verification := &entities.EmailVerification{}
	err := r.db.QueryRow(ctx, query, userID, tokenType).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.Email,
		&verification.Token,
		&verification.TokenType,
		&verification.ExpiresAt,
		&verification.IsUsed,
		&verification.UsedAt,
		&verification.CreatedAt,
		&verification.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no active verification found")
		}
		return nil, fmt.Errorf("failed to get active verification: %w", err)
	}

	return verification, nil
}

// UpdateVerification updates an existing verification
func (r *PostgresEmailVerificationRepository) UpdateVerification(ctx context.Context, verification *entities.EmailVerification) error {
	query := `
		UPDATE email_verifications
		SET user_id = $2, email = $3, token = $4, token_type = $5, expires_at = $6,
		    is_used = $7, used_at = $8, updated_at = $9
		WHERE id = $1`

	verification.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, query,
		verification.ID,
		verification.UserID,
		verification.Email,
		verification.Token,
		verification.TokenType,
		verification.ExpiresAt,
		verification.IsUsed,
		verification.UsedAt,
		verification.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update verification: %w", err)
	}

	return nil
}

// DeleteVerification deletes a verification
func (r *PostgresEmailVerificationRepository) DeleteVerification(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM email_verifications WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete verification: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no verification deleted")
	}

	if rowsAffected == 0 {
		return fmt.Errorf("verification not found")
	}

	return nil
}

// MarkTokenAsUsed marks a token as used
func (r *PostgresEmailVerificationRepository) MarkTokenAsUsed(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE email_verifications
		SET is_used = TRUE, used_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no verification deleted")
	}

	if rowsAffected == 0 {
		return fmt.Errorf("verification not found")
	}

	return nil
}

// GetUserVerifications retrieves all verifications for a user
func (r *PostgresEmailVerificationRepository) GetUserVerifications(ctx context.Context, userID uuid.UUID, tokenType string) ([]*entities.EmailVerification, error) {
	query := `
		SELECT id, user_id, email, token, token_type, expires_at, is_used, used_at, created_at, updated_at
		FROM email_verifications
		WHERE user_id = $1`

	args := []interface{}{userID}
	if tokenType != "" {
		query += " AND token_type = $2 ORDER BY created_at DESC"
		args = append(args, tokenType)
	} else {
		query += " ORDER BY created_at DESC"
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get user verifications: %w", err)
	}
	defer rows.Close()

	var verifications []*entities.EmailVerification
	for rows.Next() {
		verification := &entities.EmailVerification{}
		err := rows.Scan(
			&verification.ID,
			&verification.UserID,
			&verification.Email,
			&verification.Token,
			&verification.TokenType,
			&verification.ExpiresAt,
			&verification.IsUsed,
			&verification.UsedAt,
			&verification.CreatedAt,
			&verification.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan verification: %w", err)
		}
		verifications = append(verifications, verification)
	}

	return verifications, nil
}

// GetActiveVerificationsByEmail retrieves active verifications by email
func (r *PostgresEmailVerificationRepository) GetActiveVerificationsByEmail(ctx context.Context, email, tokenType string) ([]*entities.EmailVerification, error) {
	query := `
		SELECT id, user_id, email, token, token_type, expires_at, is_used, used_at, created_at, updated_at
		FROM email_verifications
		WHERE email = $1 AND token_type = $2 AND is_used = FALSE AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, email, tokenType)
	if err != nil {
		return nil, fmt.Errorf("failed to get active verifications by email: %w", err)
	}
	defer rows.Close()

	var verifications []*entities.EmailVerification
	for rows.Next() {
		verification := &entities.EmailVerification{}
		err := rows.Scan(
			&verification.ID,
			&verification.UserID,
			&verification.Email,
			&verification.Token,
			&verification.TokenType,
			&verification.ExpiresAt,
			&verification.IsUsed,
			&verification.UsedAt,
			&verification.CreatedAt,
			&verification.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan verification: %w", err)
		}
		verifications = append(verifications, verification)
	}

	return verifications, nil
}

// DeactivateAllUserVerifications deactivates all verifications for a user by type
func (r *PostgresEmailVerificationRepository) DeactivateAllUserVerifications(ctx context.Context, userID uuid.UUID, tokenType string) error {
	query := `
		UPDATE email_verifications
		SET is_used = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND token_type = $2 AND is_used = FALSE`

	_, err := r.db.Exec(ctx, query, userID, tokenType)
	if err != nil {
		return fmt.Errorf("failed to deactivate user verifications: %w", err)
	}

	return nil
}

// DeleteExpiredVerifications deletes expired verifications
func (r *PostgresEmailVerificationRepository) DeleteExpiredVerifications(ctx context.Context) (int64, error) {
	query := `DELETE FROM email_verifications WHERE expires_at < CURRENT_TIMESTAMP`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired verifications: %w", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected, nil
}

// GetVerificationStats retrieves verification statistics for a user
func (r *PostgresEmailVerificationRepository) GetVerificationStats(ctx context.Context, userID uuid.UUID) (*repositories.VerificationStats, error) {
	query := `
		SELECT
			COUNT(*) as total_verifications,
			COUNT(CASE WHEN is_used = FALSE AND expires_at > CURRENT_TIMESTAMP THEN 1 END) as active_verifications,
			COUNT(CASE WHEN is_used = TRUE THEN 1 END) as used_verifications,
			COUNT(CASE WHEN expires_at <= CURRENT_TIMESTAMP THEN 1 END) as expired_verifications,
			MAX(created_at) as last_verification_sent,
			MAX(used_at) as last_verification_used,
			token_type
		FROM email_verifications
		WHERE user_id = $1
		GROUP BY token_type
		ORDER BY last_verification_sent DESC
		LIMIT 1`

	stats := &repositories.VerificationStats{}
	var tokenType sql.NullString
	var lastSent, lastUsed sql.NullTime

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&stats.TotalVerifications,
		&stats.ActiveVerifications,
		&stats.UsedVerifications,
		&stats.ExpiredVerifications,
		&lastSent,
		&lastUsed,
		&tokenType,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Return empty stats for user with no verifications
			return &repositories.VerificationStats{
				TotalVerifications:   0,
				ActiveVerifications:  0,
				UsedVerifications:    0,
				ExpiredVerifications: 0,
				VerificationTokenType: entities.TokenTypeVerification,
			}, nil
		}
		return nil, fmt.Errorf("failed to get verification stats: %w", err)
	}

	if lastSent.Valid {
		stats.LastVerificationSent = &lastSent.Time
	}

	if lastUsed.Valid {
		stats.LastVerificationUsed = &lastUsed.Time
	}

	if tokenType.Valid {
		stats.VerificationTokenType = tokenType.String
	}

	return stats, nil
}

// CleanupExpiredVerifications removes expired verifications older than specified duration
func (r *PostgresEmailVerificationRepository) CleanupExpiredVerifications(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM email_verifications WHERE expires_at < $1`

	result, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired verifications: %w", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected, nil
}