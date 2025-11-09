#!/bin/bash

# JWT Token Generator Script for ERPGo
# This script provides easy access to JWT token generation

set -e

# Default values
EMAIL="user@example.com"
USERNAME="user"
ROLES="user"
JSON_OUTPUT=false
SHOW_HELP=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--email)
            EMAIL="$2"
            shift 2
            ;;
        -u|--username)
            USERNAME="$2"
            shift 2
            ;;
        -r|--roles)
            ROLES="$2"
            shift 2
            ;;
        -j|--json)
            JSON_OUTPUT=true
            shift
            ;;
        -h|--help)
            SHOW_HELP=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            SHOW_HELP=true
            shift
            ;;
    esac
done

# Show help
if [ "$SHOW_HELP" = true ]; then
    cat << EOF
JWT Token Generator for ERPGo

Usage: $0 [OPTIONS]

Options:
    -e, --email EMAIL      User email for token (default: user@example.com)
    -u, --username USER    Username for token (default: user)
    -r, --roles ROLES      Comma-separated list of roles (default: user)
    -j, --json             Output as JSON
    -h, --help             Show this help message

Examples:
    $0                                    # Generate default user token
    $0 -e admin@company.com -u admin -r "admin,user,manager"
    $0 -j -e test@test.com -r "user"
    $0 --email dev@company.com --username dev --roles "admin,user" --json

Environment Variables:
    JWT_SECRET         Must be set in .env file
    APP_NAME           Used as JWT issuer (default: ERPGo)
    JWT_ACCESS_TOKEN_EXPIRATION  Access token expiry (default: 15m)
    JWT_REFRESH_TOKEN_EXPIRATION  Refresh token expiry (default: 168h)
EOF
    exit 0
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "Error: .env file not found. Please create one with JWT_SECRET configured."
    exit 1
fi

# Build command
CMD="go run ./cmd/generate-jwt"
CMD="$CMD -email '$EMAIL' -username '$USERNAME' -roles '$ROLES'"

if [ "$JSON_OUTPUT" = true ]; then
    CMD="$CMD -json"
fi

# Execute command
echo "Generating JWT token..."
echo "Email: $EMAIL"
echo "Username: $USERNAME"
echo "Roles: $ROLES"
echo ""

eval $CMD