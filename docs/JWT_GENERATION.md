# JWT Token Generation for ERPGo

This document explains how to generate JWT tokens for the ERPGo application.

## Overview

ERPGo provides two tools for generating JWT tokens:

1. **Simple Export Tool** (`cmd/export-jwt`) - Basic token generation with default values
2. **Flexible Generator** (`cmd/generate-jwt`) - Advanced token generation with customization
3. **Shell Script** (`scripts/generate-jwt.sh`) - Easy-to-use wrapper script

## Prerequisites

1. Set up your `.env` file with a secure `JWT_SECRET`:

```bash
# Generate a secure JWT secret
openssl rand -base64 32

# Add it to your .env file
JWT_SECRET=your-generated-secret-here
```

## Usage

### 1. Simple Export Tool

Generate tokens with default test user data:

```bash
go run ./cmd/export-jwt
```

### 2. Flexible Generator

Generate tokens with custom user data:

```bash
# Basic usage
go run ./cmd/generate-jwt -email admin@company.com -username admin -roles admin,user,manager

# JSON output
go run ./cmd/generate-jwt -json -email test@example.com

# Custom expiry times
go run ./cmd/generate-jwt -access-expiry 1h -refresh-expiry 24h

# Help
go run ./cmd/generate-jwt --help
```

### 3. Shell Script (Recommended)

Easy-to-use wrapper with sensible defaults:

```bash
# Generate default user token
./scripts/generate-jwt.sh

# Generate admin token
./scripts/generate-jwt.sh -e admin@company.com -u admin -r "admin,user,manager"

# Generate JSON output
./scripts/generate-jwt.sh -j -e test@test.com

# Show help
./scripts/generate-jwt.sh --help
```

## Token Structure

The generated JWT tokens include the following claims:

- `user_id`: Unique user identifier (UUID)
- `email`: User email address
- `username`: User username
- `roles`: Array of user roles
- `iss`: Issuer (application name)
- `sub`: Subject (user ID)
- `aud`: Audience (api/refresh)
- `exp`: Expiration time
- `iat`: Issued at time
- `jti`: JWT ID (unique token identifier)

## Using the Tokens

### Authorization Header

Include the access token in API requests:

```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     http://localhost:8080/api/v1/users
```

### Token Refresh

Use the refresh token to get new access tokens:

```bash
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"refresh_token":"YOUR_REFRESH_TOKEN"}' \
     http://localhost:8080/api/v1/auth/refresh
```

## Environment Variables

The token generation tools respect these environment variables:

- `JWT_SECRET`: Secret key for signing tokens (required)
- `APP_NAME`: JWT issuer (default: ERPGo)
- `JWT_ACCESS_TOKEN_EXPIRATION`: Access token expiry (default: 15m)
- `JWT_REFRESH_TOKEN_EXPIRATION`: Refresh token expiry (default: 168h)

## Security Notes

1. **Never commit** your `.env` file to version control
2. **Use strong secrets** for `JWT_SECRET` in production
3. **Set appropriate expiry times** based on your security requirements
4. **Store refresh tokens securely** on the client side
5. **Implement token rotation** in production for enhanced security

## Troubleshooting

### JWT_SECRET Validation Error

If you get "JWT_SECRET must be set to a secure value":

1. Check your `.env` file exists
2. Ensure `JWT_SECRET` is set and not the default value
3. Generate a new secret: `openssl rand -base64 32`

### Database Connection Issues

The API requires a database connection. For development:

1. Ensure PostgreSQL is running
2. Create the database: `createdb erpgo`
3. Update your `.env` file with correct database credentials

### Token Expiration

- Access tokens expire in 15 minutes by default
- Refresh tokens expire in 7 days by default
- Use the refresh token to get new access tokens before expiration