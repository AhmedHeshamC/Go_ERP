# ERPGo Quick Start Guide

## Prerequisites
- Go 1.21+
- PostgreSQL 13+
- Redis 6+
- Docker & Docker Compose (optional, for easy setup)

## Option 1: Docker Compose (Recommended)

```bash
# Start all services with Docker Compose
docker-compose up -d

# The ERPGo API will be available at http://localhost:8080
# Swagger UI: http://localhost:8080/docs
# Grafana: http://localhost:3000
# Prometheus: http://localhost:9090
```

## Option 2: Manual Setup

### 1. Set up PostgreSQL
```bash
# Create database
createdb erpgo

# Run migrations (if you have migration files)
# Or create tables manually using the schema in docs/
```

### 2. Set up Redis
```bash
# Start Redis server
redis-server
```

### 3. Configure Environment
```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your database credentials
# Required minimum:
# JWT_SECRET=your-super-secret-jwt-key-min-32-characters-long
# DB_HOST=localhost
# DB_USER=your-db-user
# DB_PASSWORD=your-db-password
# DB_NAME=erpgo
```

### 4. Run the Application
```bash
# Build the binary
go build -o erpgo ./cmd/api

# Run with environment variables
export JWT_SECRET="your-super-secret-jwt-key-min-32-characters-long"
./erpgo
```

## API Documentation

Once running, you can access:
- **Interactive API Docs**: http://localhost:8080/docs
- **Health Check**: http://localhost:8080/health
- **API Base URL**: http://localhost:8080/api/v1

## Default Users

After initial setup, you can create users via:
- Registration API: POST /api/v1/auth/register
- Admin interface (if implemented)

## Development

For development with hot reload:
```bash
# Install air for hot reload
go install github.com/air-verse/air@latest

# Run with hot reload
air
```

## Troubleshooting

1. **JWT_SECRET required**: Always set a secure JWT secret
2. **Database connection**: Ensure PostgreSQL is running and credentials are correct
3. **Redis connection**: Redis is required for caching and session management
4. **Port conflicts**: Default port is 8080, change in .env if needed

## Production Deployment

For production deployment, see the comprehensive Deployment Guide:
- `docs/DEPLOYMENT.md`
- `docker-compose.prod.yml`
- Production environment templates in `configs/environments/`
