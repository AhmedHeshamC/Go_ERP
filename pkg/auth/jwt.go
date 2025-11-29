package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Username string    `json:"username"`
	Roles    []string  `json:"roles"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	secretKey     []byte
	issuer        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	redisClient   redis.Cmdable
	prefix        string
}

// NewJWTService creates a new JWT service instance
func NewJWTService(secret, issuer string, accessExpiry, refreshExpiry time.Duration) *JWTService {
	return &JWTService{
		secretKey:     []byte(secret),
		issuer:        issuer,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		redisClient:   nil, // Will be set separately
		prefix:        "jwt:blacklist:",
	}
}

// NewJWTServiceWithRedis creates a new JWT service instance with Redis client
func NewJWTServiceWithRedis(secret, issuer string, accessExpiry, refreshExpiry time.Duration, redisClient redis.Cmdable) *JWTService {
	return &JWTService{
		secretKey:     []byte(secret),
		issuer:        issuer,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		redisClient:   redisClient,
		prefix:        "jwt:blacklist:",
	}
}

// SetRedisClient sets the Redis client for token blacklist functionality
func (j *JWTService) SetRedisClient(redisClient redis.Cmdable) {
	j.redisClient = redisClient
}

// GenerateTokenPair generates both access and refresh tokens
func (j *JWTService) GenerateTokenPair(userID uuid.UUID, email, username string, roles []string) (accessToken, refreshToken string, err error) {
	// Generate access token
	accessToken, err = j.GenerateAccessToken(userID, email, username, roles)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err = j.GenerateRefreshToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// GenerateAccessToken generates a new JWT access token
func (j *JWTService) GenerateAccessToken(userID uuid.UUID, email, username string, roles []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Email:    email,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    j.issuer,
			Subject:   userID.String(),
			Audience:  []string{"erpgo-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessExpiry)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// GenerateRefreshToken generates a new JWT refresh token
func (j *JWTService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		ID:        uuid.New().String(),
		Issuer:    j.issuer,
		Subject:   userID.String(),
		Audience:  []string{"erpgo-refresh"},
		ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshExpiry)),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateAccessToken validates an access token and returns the claims
func (j *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check if token is blacklisted
		if j.redisClient != nil {
			ctx := context.Background()
			isBlacklisted, err := j.IsTokenBlacklisted(ctx, tokenString)
			if err != nil {
				return nil, fmt.Errorf("failed to check token blacklist: %w", err)
			}
			if isBlacklisted {
				return nil, errors.New("token is blacklisted")
			}

			// Check if user is invalidated
			isUserInvalidated, err := j.IsUserInvalidated(ctx, claims.UserID)
			if err != nil {
				return nil, fmt.Errorf("failed to check user invalidation: %w", err)
			}
			if isUserInvalidated {
				return nil, errors.New("user tokens are invalidated")
			}
		}

		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (j *JWTService) ValidateRefreshToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		// Extract user ID from subject
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid user ID in token: %w", err)
		}

		return userID, nil
	}

	return uuid.Nil, errors.New("invalid refresh token claims")
}

// GetTokenExpiration returns the expiration time of a token
func (j *JWTService) GetTokenExpiration(tokenString string) (*time.Time, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		if claims.ExpiresAt != nil {
			return &claims.ExpiresAt.Time, nil
		}
		return nil, errors.New("token has no expiration time")
	}

	return nil, errors.New("invalid token claims")
}

// IsTokenExpired checks if a token is expired
func (j *JWTService) IsTokenExpired(tokenString string) bool {
	exp, err := j.GetTokenExpiration(tokenString)
	if err != nil {
		return true
	}
	return time.Now().After(*exp)
}

// RefreshAccessToken generates a new access token using a valid refresh token
func (j *JWTService) RefreshAccessToken(refreshTokenString string, userID uuid.UUID, email, username string, roles []string) (string, error) {
	// Validate the refresh token
	tokenUserID, err := j.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Ensure the user ID matches
	if tokenUserID != userID {
		return "", errors.New("refresh token user ID mismatch")
	}

	// Generate new access token
	return j.GenerateAccessToken(userID, email, username, roles)
}

// ExtractTokenFromBearer extracts token from Bearer authorization header
func ExtractTokenFromBearer(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is empty")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("authorization header format must be Bearer {token}")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", errors.New("authorization header format must be Bearer {token}")
	}

	return strings.TrimSpace(token), nil
}

// ValidateToken is a generic method that validates both access and refresh tokens
func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	// First try to validate as access token
	claims, err := j.ValidateAccessToken(tokenString)
	if err == nil {
		return claims, nil
	}

	// If access token validation fails, try refresh token
	refreshUserID, err := j.ValidateRefreshToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Return minimal claims for refresh token
	return &Claims{
		UserID: refreshUserID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: refreshUserID.String(),
		},
	}, nil
}

// GetAccessExpiry returns the access token expiration duration
func (j *JWTService) GetAccessExpiry() time.Duration {
	return j.accessExpiry
}

// GetRefreshExpiry returns the refresh token expiration duration
func (j *JWTService) GetRefreshExpiry() time.Duration {
	return j.refreshExpiry
}

// Token Blacklist Management

// BlacklistToken adds a token to the blacklist with expiration
func (j *JWTService) BlacklistToken(ctx context.Context, tokenString string) error {
	if j.redisClient == nil {
		// If no Redis client, we can't blacklist tokens
		// In production, this should be an error, but for now we'll just log
		return nil
	}

	// Parse token to get expiration time
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Calculate expiration time
	expiresAt := claims.ExpiresAt.Time
	if expiresAt.IsZero() {
		// If no expiration in token, use default expiry time
		expiresAt = time.Now().Add(j.accessExpiry)
	}

	// Add to blacklist with expiration
	key := j.prefix + tokenString
	err = j.redisClient.Set(ctx, key, "blacklisted", time.Until(expiresAt)).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (j *JWTService) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if j.redisClient == nil {
		// If no Redis client, we can't check blacklist
		return false, nil
	}

	key := j.prefix + tokenString
	result, err := j.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Token not blacklisted
		}
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return result == "blacklisted", nil
}

// InvalidateToken invalidates a token immediately (adds to blacklist)
func (j *JWTService) InvalidateToken(ctx context.Context, tokenString string) error {
	if j.redisClient == nil {
		return errors.New("redis client not configured for token invalidation")
	}

	// Add to blacklist with immediate expiration (will be cleaned up by TTL)
	key := j.prefix + tokenString
	err := j.redisClient.Set(ctx, key, "invalidated", j.accessExpiry).Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate token: %w", err)
	}

	return nil
}

// InvalidateUserTokens invalidates all tokens for a user by adding the user ID to a user blacklist
func (j *JWTService) InvalidateUserTokens(ctx context.Context, userID uuid.UUID) error {
	if j.redisClient == nil {
		return errors.New("redis client not configured for token invalidation")
	}

	// Add user to blacklist with expiry
	key := j.prefix + "user:" + userID.String()
	err := j.redisClient.Set(ctx, key, "user_invalidated", j.accessExpiry).Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate user tokens: %w", err)
	}

	return nil
}

// IsUserInvalidated checks if a user's tokens should be invalidated
func (j *JWTService) IsUserInvalidated(ctx context.Context, userID uuid.UUID) (bool, error) {
	if j.redisClient == nil {
		return false, nil
	}

	key := j.prefix + "user:" + userID.String()
	result, err := j.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // User not invalidated
		}
		return false, fmt.Errorf("failed to check user invalidation: %w", err)
	}

	return result == "user_invalidated", nil
}

// RefreshTokenRotation validates refresh token and generates new token pair, invalidating old tokens
func (j *JWTService) RefreshTokenRotation(ctx context.Context, refreshTokenString string, userID uuid.UUID, email, username string, roles []string) (accessToken, refreshToken string, err error) {
	// Validate refresh token
	refreshClaims, err := j.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Ensure the refresh token belongs to the same user
	if refreshClaims != userID {
		return "", "", errors.New("refresh token user mismatch")
	}

	// Invalidate the old refresh token
	if err := j.InvalidateToken(ctx, refreshTokenString); err != nil {
		// Log error but don't fail the refresh process
		// In production, you might want to handle this more strictly
		fmt.Printf("Warning: failed to invalidate old refresh token: %v\n", err)
	}

	// Generate new token pair
	return j.GenerateTokenPair(userID, email, username, roles)
}
