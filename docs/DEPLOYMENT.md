# ERPGo Deployment Guide

## Overview

This guide covers various deployment strategies for the ERPGo application, including development, staging, and production environments. ERPGo can be deployed using Docker, cloud platforms, or traditional server setups.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Environment Configuration](#environment-configuration)
3. [Docker Deployment](#docker-deployment)
4. [Cloud Deployment](#cloud-deployment)
5. [Database Setup](#database-setup)
6. [Monitoring and Logging](#monitoring-and-logging)
7. [Security Considerations](#security-considerations)
8. [Maintenance and Updates](#maintenance-and-updates)
9. [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

**Minimum Requirements:**
- CPU: 2 cores
- RAM: 4GB
- Storage: 20GB SSD
- PostgreSQL: 13+
- Redis: 6+

**Recommended Requirements:**
- CPU: 4+ cores
- RAM: 8GB+
- Storage: 50GB+ SSD
- Load balancer
- SSL/TLS certificate

### Software Dependencies

- Go 1.24+ (for building from source)
- Docker 20.10+ (for containerized deployment)
- Docker Compose 2.0+
- PostgreSQL 13+
- Redis 6+
- Nginx (reverse proxy)

## Environment Configuration

### Environment Variables

Create a `.env` file in your deployment directory:

```bash
# Application Configuration
APP_NAME=erpgo-api
APP_VERSION=1.0.0
APP_ENV=production
APP_PORT=8080
APP_HOST=0.0.0.0
APP_DEBUG=false

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=erpgo_user
DB_PASSWORD=secure_password
DB_NAME=erpgo_prod
DB_SSL_MODE=require
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
DB_CONNECTION_MAX_LIFETIME=300s

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis_password
REDIS_DB=0
REDIS_POOL_SIZE=10

# JWT Configuration
JWT_SECRET=your-super-secure-jwt-secret-key-min-32-chars
JWT_ACCESS_TOKEN_DURATION=15m
JWT_REFRESH_TOKEN_DURATION=168h
JWT_ISSUER=erpgo-api

# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=noreply@yourcompany.com
SMTP_PASSWORD=your-smtp-password
SMTP_FROM=ERPGo System <noreply@yourcompany.com>

# File Storage
STORAGE_TYPE=local  # local, s3
STORAGE_PATH=/app/storage
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_S3_BUCKET=your-s3-bucket

# Security
BCRYPT_COST=12
SESSION_SECRET=session-secret-key
RATE_LIMIT_LOGIN=5
RATE_LIMIT_REGISTER=3

# Monitoring
ENABLE_METRICS=true
METRICS_PORT=9090
LOG_LEVEL=info
LOG_FORMAT=json
```

### Production Environment

```bash
# .env.production
APP_ENV=production
APP_DEBUG=false
APP_PORT=8080
APP_HOST=0.0.0.0
DB_SSL_MODE=require
LOG_LEVEL=warn
BCRYPT_COST=12
ENABLE_METRICS=true
```

### Staging Environment

```bash
# .env.staging
APP_ENV=staging
APP_DEBUG=false
APP_PORT=8081
DB_SSL_MODE=require
LOG_LEVEL=info
BCRYPT_COST=10
ENABLE_METRICS=true
```

## Docker Deployment

### Docker Compose (Recommended)

1. **Create docker-compose.yml:**

```yaml
version: '3.8'

services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
      - "9090:9090"  # Metrics port
    environment:
      - APP_ENV=production
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    volumes:
      - ./storage:/app/storage
      - ./logs:/app/logs
    restart: unless-stopped
    networks:
      - erpgo-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: erpgo_prod
      POSTGRES_USER: erpgo_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    restart: unless-stopped
    networks:
      - erpgo-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U erpgo_user"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    restart: unless-stopped
    networks:
      - erpgo-network
    healthcheck:
      test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
      - ./nginx/ssl:/etc/nginx/ssl
      - ./logs/nginx:/var/log/nginx
    depends_on:
      - api
    restart: unless-stopped
    networks:
      - erpgo-network

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
    restart: unless-stopped
    networks:
      - erpgo-network

volumes:
  postgres_data:
  redis_data:
  prometheus_data:

networks:
  erpgo-network:
    driver: bridge
```

2. **Create Dockerfile:**

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/api/main.go

# Generate Swagger documentation
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g cmd/api/main.go --output docs

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create app user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs

# Create storage directory
RUN mkdir -p /app/storage /app/logs && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run application
CMD ["./main"]
```

3. **Deploy using Docker Compose:**

```bash
# Clone the repository
git clone <repository-url>
cd Go_ERP

# Set up environment
cp .env.example .env
# Edit .env with production values

# Build and start services
docker-compose up -d

# Run database migrations
docker-compose exec api migrate -path /app/migrations -database $DATABASE_URL up

# Generate Swagger docs
docker-compose exec api swag init -g cmd/api/main.go --output /app/docs

# Check service status
docker-compose ps
docker-compose logs api
```

### Kubernetes Deployment

Create Kubernetes manifests for production deployment:

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: erpgo

---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: erpgo-config
  namespace: erpgo
data:
  APP_ENV: "production"
  DB_HOST: "postgres-service"
  REDIS_HOST: "redis-service"

---
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: erpgo-secrets
  namespace: erpgo
type: Opaque
data:
  db-password: <base64-encoded-password>
  jwt-secret: <base64-encoded-jwt-secret>
  redis-password: <base64-encoded-redis-password>

---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: erpgo-api
  namespace: erpgo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: erpgo-api
  template:
    metadata:
      labels:
        app: erpgo-api
    spec:
      containers:
      - name: erpgo-api
        image: erpgo/api:latest
        ports:
        - containerPort: 8080
        env:
        - name: APP_ENV
          valueFrom:
            configMapKeyRef:
              name: erpgo-config
              key: APP_ENV
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: erpgo-secrets
              key: db-password
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: erpgo-service
  namespace: erpgo
spec:
  selector:
    app: erpgo-api
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP

---
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: erpgo-ingress
  namespace: erpgo
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - api.yourcompany.com
    secretName: erpgo-tls
  rules:
  - host: api.yourcompany.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: erpgo-service
            port:
              number: 80
```

## Cloud Deployment

### AWS Deployment

#### 1. ECS (Elastic Container Service)

```json
{
  "family": "erpgo-task",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "erpgo-api",
      "image": "your-account.dkr.ecr.region.amazonaws.com/erpgo:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "APP_ENV",
          "value": "production"
        }
      ],
      "secrets": [
        {
          "name": "DB_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:erpgo/db-password"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/erpgo",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

#### 2. Database Setup

```bash
# Create RDS PostgreSQL instance
aws rds create-db-instance \
  --db-instance-identifier erpgo-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version 15.4 \
  --master-username erpgo_user \
  --master-user-password secure_password \
  --allocated-storage 20 \
  --vpc-security-group-ids sg-xxxxxxxxx \
  --db-subnet-group-name default \
  --backup-retention-period 7 \
  --multi-az \
  --storage-type gp2

# Create ElastiCache Redis instance
aws elasticache create-cache-cluster \
  --cache-cluster-id erpgo-redis \
  --cache-node-type cache.t3.micro \
  --engine redis \
  --num-cache-nodes 1 \
  --security-group-ids sg-xxxxxxxxx \
  --subnet-group-name default
```

### Google Cloud Platform Deployment

#### Cloud Run Deployment

```bash
# Build and push Docker image
gcloud builds submit --tag gcr.io/PROJECT_ID/erpgo-api

# Deploy to Cloud Run
gcloud run deploy erpgo-api \
  --image gcr.io/PROJECT_ID/erpgo-api \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars APP_ENV=production \
  --set-secrets DB_PASSWORD=erpgo-db-password:latest
```

### Azure Deployment

#### Container Instances

```bash
# Create resource group
az group create --name erpgo-rg --location eastus

# Deploy container
az container create \
  --resource-group erpgo-rg \
  --name erpgo-api \
  --image yourregistry.azurecr.io/erpgo:latest \
  --cpu 1 \
  --memory 2 \
  --ports 8080 \
  --environment-variables APP_ENV=production \
  --secure-environment-variables DB_PASSWORD=secure_password
```

## Database Setup

### Production Database Configuration

1. **PostgreSQL Configuration:**

```sql
-- postgresql.conf
listen_addresses = '*'
port = 5432
max_connections = 200
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
```

2. **Database Security:**

```sql
-- Create dedicated user
CREATE USER erpgo_user WITH PASSWORD 'secure_password';
CREATE DATABASE erpgo_prod OWNER erpgo_user;

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON DATABASE erpgo_prod TO erpgo_user;

-- Enable row-level security if needed
ALTER DATABASE erpgo_prod SET row_security = on;
```

3. **Database Migrations:**

```bash
# Install migration tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations -database "postgres://user:password@host:port/dbname?sslmode=require" up

# Verify migration status
migrate -path migrations -database "postgres://user:password@host:port/dbname?sslmode=require" version
```

### Database Performance Tuning

```sql
-- Create indexes for better performance
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_products_active ON products(is_active);
CREATE INDEX CONCURRENTLY idx_orders_status ON orders(status);
CREATE INDEX CONCURRENTLY idx_inventory_product_warehouse ON inventory(product_id, warehouse_id);

-- Partition large tables (optional)
CREATE TABLE inventory_transactions_2024 PARTITION OF inventory_transactions
FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');
```

## Monitoring and Logging

### Prometheus Metrics

The application exposes metrics on port 9090:

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'erpgo-api'
    static_configs:
      - targets: ['api:9090']
    metrics_path: /metrics
    scrape_interval: 30s

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### Log Management

1. **Structured Logging:**

```go
// Example structured log
logger.Info().
    Str("user_id", userID).
    Str("action", "product_created").
    Int("product_id", productID).
    Dur("duration", time.Since(start)).
    Msg("Product created successfully")
```

2. **Log Aggregation (ELK Stack):**

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.8.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
    ports:
      - "9200:9200"
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data

  logstash:
    image: docker.elastic.co/logstash/logstash:8.8.0
    ports:
      - "5044:5044"
    volumes:
      - ./monitoring/logstash.conf:/usr/share/logstash/pipeline/logstash.conf

  kibana:
    image: docker.elastic.co/kibana/kibana:8.8.0
    ports:
      - "5601:5601"
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
```

### Health Checks

The application provides comprehensive health checks:

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health check
curl http://localhost:8080/health/detailed

Response:
{
  "status": "healthy",
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "disk_space": "healthy"
  },
  "version": "1.0.0",
  "uptime": "72h30m15s"
}
```

## Security Considerations

### SSL/TLS Configuration

```nginx
# nginx/nginx.conf
server {
    listen 443 ssl http2;
    server_name api.yourcompany.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://api:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Security Headers

```go
// Security middleware
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Next()
    }
}
```

### Firewall Configuration

```bash
# UFW rules
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw allow 9090/tcp  # Metrics (internal only)
ufw deny 5432/tcp   # PostgreSQL (internal only)
ufw deny 6379/tcp   # Redis (internal only)
ufw enable
```

## Maintenance and Updates

### Application Updates

```bash
# Zero-downtime deployment
# 1. Deploy new version
docker-compose pull
docker-compose up -d --no-deps api

# 2. Health check
curl http://localhost:8080/health

# 3. If healthy, continue
# 4. Roll back if needed
docker-compose rollback api
```

### Database Backups

```bash
# Automated backups
#!/bin/bash
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Create database backup
pg_dump -h localhost -U erpgo_user -d erpgo_prod > $BACKUP_DIR/erpgo_$DATE.sql

# Compress backup
gzip $BACKUP_DIR/erpgo_$DATE.sql

# Clean old backups (keep 30 days)
find $BACKUP_DIR -name "erpgo_*.sql.gz" -mtime +30 -delete
```

### Log Rotation

```bash
# logrotate.conf
/app/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 appuser appuser
    postrotate
        docker-compose exec api kill -USR1 1
    endscript
}
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Issues

```bash
# Check database connectivity
docker-compose exec api ping -c 3 postgres

# Check database logs
docker-compose logs postgres

# Test connection manually
psql -h localhost -U erpgo_user -d erpgo_prod
```

#### 2. Redis Connection Issues

```bash
# Check Redis connectivity
docker-compose exec api redis-cli -h redis ping

# Check Redis logs
docker-compose logs redis

# Monitor Redis usage
redis-cli -h redis info memory
```

#### 3. Application Issues

```bash
# Check application logs
docker-compose logs api

# Check application status
curl http://localhost:8080/health

# Check resource usage
docker stats erpgo_api
```

### Performance Monitoring

```bash
# Check response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/api/v1/users

# Monitor database queries
docker-compose exec postgres psql -U erpgo_user -d erpgo_prod -c "
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;"
```

### Security Monitoring

```bash
# Monitor failed login attempts
grep "Failed login attempt" /app/logs/api.log | tail -20

# Check for unusual activity
tail -f /app/logs/api.log | grep "ERROR\|WARN"

# Monitor rate limiting
curl http://localhost:8080/metrics | grep rate_limit
```

---

For additional support and troubleshooting, refer to the [Developer Guide](DEVELOPER_GUIDE.md) and [Authentication Guide](AUTHENTICATION.md).