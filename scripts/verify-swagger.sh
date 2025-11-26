#!/bin/bash

# Script to verify Swagger documentation setup
# This script checks that all Swagger files are present and properly configured

set -e

echo "🔍 Verifying Swagger Documentation Setup..."
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if required files exist
echo "📁 Checking required files..."

files=(
    "cmd/api/docs.go"
    "docs/docs.go"
    "docs/swagger.json"
    "docs/swagger.yaml"
    "docs/README.md"
    "docs/ERROR_CODES.md"
    "docs/API_EXAMPLES.md"
    "docs/SWAGGER_SETUP.md"
)

missing_files=0
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $file exists"
    else
        echo -e "${RED}✗${NC} $file is missing"
        missing_files=$((missing_files + 1))
    fi
done

echo ""

if [ $missing_files -gt 0 ]; then
    echo -e "${RED}❌ $missing_files file(s) missing${NC}"
    echo "Run: swag init -g cmd/api/docs.go -o docs --parseDependency --parseInternal"
    exit 1
fi

# Check if swagger.json is valid JSON
echo "🔍 Validating swagger.json..."
if command -v jq &> /dev/null; then
    if jq empty docs/swagger.json 2>/dev/null; then
        echo -e "${GREEN}✓${NC} swagger.json is valid JSON"
    else
        echo -e "${RED}✗${NC} swagger.json is invalid JSON"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠${NC} jq not installed, skipping JSON validation"
fi

echo ""

# Check if swagger.yaml is valid YAML
echo "🔍 Validating swagger.yaml..."
if command -v yq &> /dev/null; then
    if yq eval '.' docs/swagger.yaml > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} swagger.yaml is valid YAML"
    else
        echo -e "${RED}✗${NC} swagger.yaml is invalid YAML"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠${NC} yq not installed, skipping YAML validation"
fi

echo ""

# Check for key documentation elements in swagger.json
echo "🔍 Checking documentation content..."

if command -v jq &> /dev/null; then
    # Check for title
    title=$(jq -r '.info.title' docs/swagger.json)
    if [ "$title" = "ERPGo API" ]; then
        echo -e "${GREEN}✓${NC} API title is set correctly"
    else
        echo -e "${RED}✗${NC} API title is incorrect: $title"
    fi

    # Check for version
    version=$(jq -r '.info.version' docs/swagger.json)
    if [ -n "$version" ] && [ "$version" != "null" ]; then
        echo -e "${GREEN}✓${NC} API version is set: $version"
    else
        echo -e "${RED}✗${NC} API version is missing"
    fi

    # Check for contact info
    contact_email=$(jq -r '.info.contact.email' docs/swagger.json)
    if [ -n "$contact_email" ] && [ "$contact_email" != "null" ]; then
        echo -e "${GREEN}✓${NC} Contact email is set: $contact_email"
    else
        echo -e "${YELLOW}⚠${NC} Contact email is missing"
    fi

    # Check for security definitions
    security_defs=$(jq -r '.securityDefinitions | length' docs/swagger.json)
    if [ "$security_defs" -gt 0 ]; then
        echo -e "${GREEN}✓${NC} Security definitions present ($security_defs)"
    else
        echo -e "${YELLOW}⚠${NC} No security definitions found"
    fi

    # Check for paths
    paths_count=$(jq -r '.paths | length' docs/swagger.json)
    if [ "$paths_count" -gt 0 ]; then
        echo -e "${GREEN}✓${NC} API paths documented ($paths_count endpoints)"
    else
        echo -e "${RED}✗${NC} No API paths found"
    fi

    # Check for definitions/schemas
    defs_count=$(jq -r '.definitions | length' docs/swagger.json)
    if [ "$defs_count" -gt 0 ]; then
        echo -e "${GREEN}✓${NC} Data models documented ($defs_count models)"
    else
        echo -e "${YELLOW}⚠${NC} No data models found"
    fi
fi

echo ""

# Check if server is running (optional)
echo "🌐 Checking if API server is running..."
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health | grep -q "200"; then
    echo -e "${GREEN}✓${NC} API server is running"
    
    # Check Swagger UI endpoints
    echo ""
    echo "🔍 Checking Swagger UI endpoints..."
    
    endpoints=(
        "http://localhost:8080/api/docs/index.html"
        "http://localhost:8080/docs/index.html"
        "http://localhost:8080/swagger/index.html"
    )
    
    for endpoint in "${endpoints[@]}"; do
        status=$(curl -s -o /dev/null -w "%{http_code}" "$endpoint")
        if [ "$status" = "200" ]; then
            echo -e "${GREEN}✓${NC} $endpoint is accessible"
        else
            echo -e "${YELLOW}⚠${NC} $endpoint returned status $status"
        fi
    done
    
    echo ""
    echo -e "${GREEN}✅ Swagger UI is accessible at:${NC}"
    echo "   • http://localhost:8080/api/docs"
    echo "   • http://localhost:8080/docs"
    echo "   • http://localhost:8080/swagger/index.html"
else
    echo -e "${YELLOW}⚠${NC} API server is not running"
    echo "   Start with: go run cmd/api/main.go"
fi

echo ""
echo "═══════════════════════════════════════════════════════════"
echo -e "${GREEN}✅ Swagger documentation verification complete!${NC}"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📚 Documentation files:"
echo "   • Main docs: docs/README.md"
echo "   • Error codes: docs/ERROR_CODES.md"
echo "   • Examples: docs/API_EXAMPLES.md"
echo "   • Setup guide: docs/SWAGGER_SETUP.md"
echo ""
echo "🔄 To regenerate documentation:"
echo "   swag init -g cmd/api/docs.go -o docs --parseDependency --parseInternal"
echo ""
