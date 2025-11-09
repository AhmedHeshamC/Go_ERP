package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"erpgo/pkg/auth"
)

func main() {
	// Parse command line flags
	var (
		email    = flag.String("email", "user@example.com", "User email for token")
		username = flag.String("username", "user", "Username for token")
		roles    = flag.String("roles", "user", "Comma-separated list of roles")
		userID   = flag.String("user-id", "", "Specific user ID (generates if empty)")
		secret   = flag.String("secret", "", "JWT secret key (uses env if empty)")
		issuer   = flag.String("issuer", "", "JWT issuer (uses env if empty)")
		access   = flag.String("access-expiry", "", "Access token expiry (uses env if empty)")
		refresh  = flag.String("refresh-expiry", "", "Refresh token expiry (uses env if empty)")
		jsonOut  = flag.Bool("json", false, "Output as JSON")
		help     = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		fmt.Println("JWT Token Generator for ERPGo")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  go run ./cmd/generate-jwt [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  go run ./cmd/generate-jwt -email admin@company.com -username admin -roles admin,user")
		fmt.Println("  go run ./cmd/generate-jwt -json -email test@test.com")
		fmt.Println("  go run ./cmd/generate-jwt -user-id 550e8400-e29b-41d4-a716-446655440000")
		os.Exit(0)
	}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file")
	}

	// Get JWT configuration
	if *secret == "" {
		*secret = os.Getenv("JWT_SECRET")
		if *secret == "" {
			*secret = "your-secret-key-change-in-production"
		}
	}

	if *issuer == "" {
		*issuer = os.Getenv("APP_NAME")
		if *issuer == "" {
			*issuer = "erpgo-api"
		}
	}

	// Parse expiry times
	accessExpiryStr := *access
	if accessExpiryStr == "" {
		accessExpiryStr = os.Getenv("JWT_ACCESS_EXPIRY")
		if accessExpiryStr == "" {
			accessExpiryStr = "15m"
		}
	}

	refreshExpiryStr := *refresh
	if refreshExpiryStr == "" {
		refreshExpiryStr = os.Getenv("JWT_REFRESH_EXPIRY")
		if refreshExpiryStr == "" {
			refreshExpiryStr = "168h"
		}
	}

	accessExpiry, err := time.ParseDuration(accessExpiryStr)
	if err != nil {
		log.Fatalf("Invalid access expiry duration: %v", err)
	}

	refreshExpiry, err := time.ParseDuration(refreshExpiryStr)
	if err != nil {
		log.Fatalf("Invalid refresh expiry duration: %v", err)
	}

	// Parse user ID
	var targetUserID uuid.UUID
	if *userID == "" {
		targetUserID = uuid.New()
	} else {
		targetUserID, err = uuid.Parse(*userID)
		if err != nil {
			log.Fatalf("Invalid user ID format: %v", err)
		}
	}

	// Parse roles
	roleList := strings.Split(*roles, ",")
	for i, role := range roleList {
		roleList[i] = strings.TrimSpace(role)
	}

	// Create JWT service
	jwtService := auth.NewJWTService(*secret, *issuer, accessExpiry, refreshExpiry)

	// Generate token pair
	accessToken, refreshToken, err := jwtService.GenerateTokenPair(targetUserID, *email, *username, roleList)
	if err != nil {
		log.Fatalf("Failed to generate tokens: %v", err)
	}

	// Validate the access token to get claims
	claims, err := jwtService.ValidateAccessToken(accessToken)
	if err != nil {
		log.Fatalf("Failed to validate access token: %v", err)
	}

	if *jsonOut {
		// JSON output
		output := map[string]interface{}{
			"user": map[string]interface{}{
				"id":       claims.UserID.String(),
				"email":    claims.Email,
				"username": claims.Username,
				"roles":    claims.Roles,
			},
			"access_token": map[string]interface{}{
				"token":     accessToken,
				"expires_in": int(accessExpiry.Seconds()),
				"expires_at": claims.ExpiresAt.Time.Format(time.RFC3339),
			},
			"refresh_token": map[string]interface{}{
				"token":     refreshToken,
				"expires_in": int(refreshExpiry.Seconds()),
			},
			"generated_at": time.Now().Format(time.RFC3339),
		}

		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Human-readable output
		fmt.Println("=== JWT TOKENS ===")
		fmt.Printf("Generated at: %s\n\n", time.Now().Format(time.RFC3339))

		fmt.Printf("User ID: %s\n", claims.UserID.String())
		fmt.Printf("Email: %s\n", claims.Email)
		fmt.Printf("Username: %s\n", claims.Username)
		fmt.Printf("Roles: %v\n\n", claims.Roles)

		fmt.Println("ACCESS TOKEN:")
		fmt.Println(accessToken)
		fmt.Printf("Expires in: %v\n", accessExpiry)
		fmt.Printf("Expires at: %s\n\n", claims.ExpiresAt.Time.Format(time.RFC3339))

		fmt.Println("REFRESH TOKEN:")
		fmt.Println(refreshToken)
		fmt.Printf("Expires in: %v\n\n", refreshExpiry)

		fmt.Println("=== USAGE ===")
		fmt.Println("Authorization header:")
		fmt.Printf("Authorization: Bearer %s\n", accessToken)
	}
}