package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"erpgo/pkg/auth"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file")
	}

	// Get JWT configuration from environment
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key-change-in-production" // fallback
	}

	issuer := os.Getenv("APP_NAME")
	if issuer == "" {
		issuer = "erpgo-api" // fallback
	}

	// Parse expiry times
	accessExpiryStr := os.Getenv("JWT_ACCESS_TOKEN_EXPIRATION")
	if accessExpiryStr == "" {
		accessExpiryStr = "15m" // fallback
	}

	refreshExpiryStr := os.Getenv("JWT_REFRESH_TOKEN_EXPIRATION")
	if refreshExpiryStr == "" {
		refreshExpiryStr = "168h" // fallback (7 days)
	}

	accessExpiry, err := time.ParseDuration(accessExpiryStr)
	if err != nil {
		log.Fatalf("Invalid access expiry duration: %v", err)
	}

	refreshExpiry, err := time.ParseDuration(refreshExpiryStr)
	if err != nil {
		log.Fatalf("Invalid refresh expiry duration: %v", err)
	}

	// Create JWT service
	jwtService := auth.NewJWTService(secret, issuer, accessExpiry, refreshExpiry)

	// Generate token for a test user
	userID := uuid.New()
	email := "test@example.com"
	username := "testuser"
	roles := []string{"user", "admin"}

	// Generate token pair
	accessToken, refreshToken, err := jwtService.GenerateTokenPair(userID, email, username, roles)
	if err != nil {
		log.Fatalf("Failed to generate tokens: %v", err)
	}

	// Export tokens
	fmt.Println("=== JWT TOKENS EXPORT ===")
	fmt.Printf("Generated at: %s\n\n", time.Now().Format(time.RFC3339))

	fmt.Printf("User ID: %s\n", userID.String())
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Roles: %v\n\n", roles)

	fmt.Println("ACCESS TOKEN:")
	fmt.Println(accessToken)
	fmt.Printf("Expires in: %v\n\n", accessExpiry)

	fmt.Println("REFRESH TOKEN:")
	fmt.Println(refreshToken)
	fmt.Printf("Expires in: %v\n\n", refreshExpiry)

	// Validate the access token to show its contents
	claims, err := jwtService.ValidateAccessToken(accessToken)
	if err != nil {
		log.Fatalf("Failed to validate access token: %v", err)
	}

	fmt.Println("ACCESS TOKEN CLAIMS:")
	fmt.Printf("User ID: %s\n", claims.UserID.String())
	fmt.Printf("Email: %s\n", claims.Email)
	fmt.Printf("Username: %s\n", claims.Username)
	fmt.Printf("Roles: %v\n", claims.Roles)
	fmt.Printf("Issuer: %s\n", claims.Issuer)
	fmt.Printf("Subject: %s\n", claims.Subject)
	fmt.Printf("Audience: %v\n", claims.Audience)
	fmt.Printf("Issued At: %s\n", claims.IssuedAt.Format(time.RFC3339))
	fmt.Printf("Expires At: %s\n", claims.ExpiresAt.Time.Format(time.RFC3339))
	fmt.Printf("Token ID: %s\n", claims.ID)
}