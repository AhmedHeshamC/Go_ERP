# ERPGo Deployment Guide

This comprehensive guide covers the deployment of ERPGo with production-ready configurations, CI/CD pipelines, and disaster recovery procedures.

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Architecture Overview](#architecture-overview)
3. [Pre-Deployment Checklist](#pre-deployment-checklist)
4. [Deployment Options](#deployment-options)
5. [Environment Configuration](#environment-configuration)
6. [Database Setup](#database-setup)
7. [Application Deployment](#application-deployment)
8. [Load Balancer Configuration](#load-balancer-configuration)
9. [SSL/TLS Setup](#ssltls-setup)
10. [Monitoring and Logging](#monitoring-and-logging)
11. [Backup and Recovery](#backup-and-recovery)
12. [Security Hardening](#security-hardening)
13. [Performance Optimization](#performance-optimization)
14. [Maintenance Procedures](#maintenance-procedures)
15. [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements

- **CPU**: 4 cores
- **Memory**: 8GB RAM
- **Storage**: 100GB SSD
- **Network**: 1Gbps connection

### Recommended Requirements

- **CPU**: 8 cores or more
- **Memory**: 16GB RAM or more
- **Storage**: 500GB SSD or more
- **Network**: 10Gbps connection

### Software Requirements

- **Operating System**: Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- **Go**: 1.21+
- **PostgreSQL**: 14+
- **Redis**: 6.2+
- **Nginx**: 1.20+
- **Docker**: 20.10+ (optional)
- **Kubernetes**: 1.24+ (optional)

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │────│   Web Server    │────│  Application    │
│    (Nginx)      │    │    (Nginx)      │    │     Server      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                              ┌────────────────────────┼────────────────────────┐
                              │                        │                        │
                     ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
                     │ PostgreSQL  │         │    Redis    │         │ File Storage│
                     │   Database  │         │     Cache   │         │   (S3/MinIO) │
                     └─────────────┘         └─────────────┘         └─────────────┘
```

## Pre-Deployment Checklist

### Security Checklist

- [ ] SSL/TLS certificates installed
- [ ] Firewall rules configured
- [ ] Security groups set up
- [ ] API authentication configured
- [ ] Environment variables secured
- [ ] Secret management implemented
- [ ] Access controls configured
- [ ] Security scanning completed

### Performance Checklist

- [ ] Database indexes optimized
- [ ] Connection pools configured
- [ ] Caching strategy implemented
- [ ] CDN configured (if applicable)
- [ ] Load testing completed
- [ ] Performance benchmarks established
- [ ] Monitoring tools configured
- [ ] Alert thresholds set

### Backup Checklist

- [ ] Database backup strategy
- [ ] File backup strategy
- [ ] Backup retention policy
- [ ] Disaster recovery plan
- [ ] Backup testing procedures
- [ ] Recovery testing completed

## Deployment Options

### Option 1: Traditional Deployment

#### Pros
- Full control over infrastructure
- Custom configurations possible
- Direct access to servers

#### Cons
- Manual setup required
- Higher operational overhead
- Scalability limitations

### Option 2: Docker Deployment

#### Pros
- Consistent environments
- Easy scaling
- Container orchestration support

#### Cons
- Learning curve
- Resource overhead
- Complex networking

### Option 3: Kubernetes Deployment

#### Pros
- Auto-scaling capabilities
- High availability
- Rolling deployments

#### Cons
- Complex setup
- Resource intensive
- Steep learning curve

## Environment Configuration

### Environment Variables

```bash
# Application Configuration
APP_NAME=erpgo
APP_ENV=production
APP_VERSION=1.0.0
APP_DEBUG=false
APP_TIMEZONE=UTC

# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_IDLE_TIMEOUT=60s

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=erpgo_production
DB_USER=erpgo_user
DB_PASSWORD=secure_password
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis_password
REDIS_DB=0
REDIS_POOL_SIZE=10

# JWT Configuration
JWT_SECRET=your_jwt_secret_key_here
JWT_EXPIRES_IN=24h
JWT_REFRESH_EXPIRES_IN=168h

# File Storage Configuration
STORAGE_TYPE=s3
STORAGE_BUCKET=erpgo-files
STORAGE_REGION=us-east-1
STORAGE_ACCESS_KEY=your_access_key
STORAGE_SECRET_KEY=your_secret_key

# Email Configuration
MAIL_DRIVER=smtp
MAIL_HOST=smtp.example.com
MAIL_PORT=587
MAIL_USERNAME=noreply@erpgo.example.com
MAIL_PASSWORD=email_password
MAIL_ENCRYPTION=tls
MAIL_FROM=noreply@erpgo.example.com

# Monitoring Configuration
PROMETHEUS_ENABLED=true
PROMETHEUS_PORT=9090
JAEGER_ENABLED=true
JAEGER_ENDPOINT=http://localhost:14268/api/traces

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout
```

### Configuration File

```yaml
# config/production.yml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

database:
  host: "${DB_HOST}"
  port: "${DB_PORT}"
  name: "${DB_NAME}"
  user: "${DB_USER}"
  password: "${DB_PASSWORD}"
  ssl_mode: "require"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

redis:
  host: "${REDIS_HOST}"
  port: "${REDIS_PORT}"
  password: "${REDIS_PASSWORD}"
  db: 0
  pool_size: 10

auth:
  jwt_secret: "${JWT_SECRET}"
  jwt_expires_in: "24h"
  jwt_refresh_expires_in: "168h"

storage:
  type: "${STORAGE_TYPE}"
  bucket: "${STORAGE_BUCKET}"
  region: "${STORAGE_REGION}"
  access_key: "${STORAGE_ACCESS_KEY}"
  secret_key: "${STORAGE_SECRET_KEY}"

monitoring:
  prometheus:
    enabled: true
    port: 9090
  jaeger:
    enabled: true
    endpoint: "http://localhost:14268/api/traces"
```

## Database Setup

### PostgreSQL Installation

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql postgresql-contrib

# CentOS/RHEL
sudo yum install postgresql-server postgresql-contrib
sudo postgresql-setup initdb
sudo systemctl enable postgresql
sudo systemctl start postgresql
```

### Database Creation

```sql
-- Create database user
CREATE USER erpgo_user WITH PASSWORD 'secure_password';

-- Create database
CREATE DATABASE erpgo_production OWNER erpgo_user;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE erpgo_production TO erpgo_user;

-- Connect to database
\c erpgo_production;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
```

### Database Migration

```bash
# Run database migrations
go run cmd/migrate/main.go up

# Or using migration tool
migrate -path migrations -database "postgres://erpgo_user:secure_password@localhost:5432/erpgo_production?sslmode=require" up
```

### Database Optimization

```sql
-- Create indexes for performance
CREATE INDEX CONCURRENTLY idx_orders_status ON orders(status);
CREATE INDEX CONCURRENTLY idx_orders_created_at ON orders(created_at);
CREATE INDEX CONCURRENTLY idx_orders_customer_id ON orders(customer_id);
CREATE INDEX CONCURRENTLY idx_customers_email ON customers(email);
CREATE INDEX CONCURRENTLY idx_products_sku ON products(sku);
CREATE INDEX CONCURRENTLY idx_order_items_order_id ON order_items(order_id);
CREATE INDEX CONCURRENTLY idx_order_items_product_id ON order_items(product_id);

-- Partition large tables (optional)
CREATE TABLE orders_2024 PARTITION OF orders
FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');
```

## Application Deployment

### Build Application

```bash
# Build for production
go build -ldflags "-s -w" -o bin/erpgo cmd/api/main.go

# Or using Makefile
make build-prod
```

### Systemd Service

```ini
# /etc/systemd/system/erpgo.service
[Unit]
Description=ERPGo Application
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=erpgo
Group=erpgo
WorkingDirectory=/opt/erpgo
ExecStart=/opt/erpgo/bin/erpgo -config /opt/erpgo/config/production.yml
Restart=always
RestartSec=10
Environment=GIN_MODE=release

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/erpgo/logs /opt/erpgo/storage

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start service
sudo systemctl enable erpgo
sudo systemctl start erpgo
sudo systemctl status erpgo
```

### Docker Deployment

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o erpgo cmd/api/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/erpgo .
COPY --from=builder /app/config ./config

EXPOSE 8080
CMD ["./erpgo"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  erpgo:
    build: .
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=production
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: erpgo_production
      POSTGRES_USER: erpgo_user
      POSTGRES_PASSWORD: secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    restart: unless-stopped

  redis:
    image: redis:6.2-alpine
    command: redis-server --requirepass redis_password
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - erpgo
    restart: unless-stopped

volumes:
  postgres_data:
```

## Load Balancer Configuration

### Nginx Configuration

```nginx
# /etc/nginx/nginx.conf
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Logging format
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for" '
                    'rt=$request_time uct="$upstream_connect_time" '
                    'uht="$upstream_header_time" urt="$upstream_response_time"';

    access_log /var/log/nginx/access.log main;

    # Performance optimizations
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/json
        application/javascript
        application/xml+rss
        application/atom+xml
        image/svg+xml;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=1r/s;

    # Upstream servers
    upstream erpgo_backend {
        least_conn;
        server 127.0.0.1:8080 max_fails=3 fail_timeout=30s;
        server 127.0.0.1:8081 max_fails=3 fail_timeout=30s;
        keepalive 32;
    }

    # Main server block
    server {
        listen 80;
        server_name api.erpgo.example.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name api.erpgo.example.com;

        # SSL configuration
        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;
        ssl_session_cache shared:SSL:10m;
        ssl_session_timeout 10m;

        # Security headers
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
        add_header X-Frame-Options DENY always;
        add_header X-Content-Type-Options nosniff always;
        add_header X-XSS-Protection "1; mode=block" always;
        add_header Referrer-Policy "strict-origin-when-cross-origin" always;

        # API routes
        location /api/ {
            limit_req zone=api burst=20 nodelay;

            proxy_pass http://erpgo_backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_cache_bypass $http_upgrade;
            proxy_read_timeout 300s;
            proxy_connect_timeout 75s;
        }

        # Login endpoint with stricter rate limiting
        location /api/v1/auth/login {
            limit_req zone=login burst=5 nodelay;

            proxy_pass http://erpgo_backend;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # Health check endpoint
        location /health {
            proxy_pass http://erpgo_backend;
            access_log off;
        }

        # Static files
        location /static/ {
            alias /var/www/erpgo/static/;
            expires 1y;
            add_header Cache-Control "public, immutable";
        }

        # Monitoring endpoints (restricted access)
        location /monitoring {
            allow 10.0.0.0/8;
            allow 172.16.0.0/12;
            allow 192.168.0.0/16;
            deny all;

            proxy_pass http://erpgo_backend;
        }
    }
}
```

## SSL/TLS Setup

### Let's Encrypt Certificate

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx

# Generate certificate
sudo certbot --nginx -d api.erpgo.example.com

# Auto-renewal
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet
```

### Manual Certificate Setup

```bash
# Generate private key
openssl genrsa -out erpgo.key 2048

# Generate CSR
openssl req -new -key erpgo.key -out erpgo.csr

# Submit CSR to CA and get certificate

# Configure Nginx
sudo cp erpgo.crt /etc/nginx/ssl/cert.pem
sudo cp erpgo.key /etc/nginx/ssl/key.pem
sudo nginx -t && sudo systemctl reload nginx
```

## Monitoring and Logging

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "erpgo_rules.yml"

scrape_configs:
  - job_name: 'erpgo'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: /metrics
    scrape_interval: 15s

  - job_name: 'postgres'
    static_configs:
      - targets: ['localhost:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']

  - job_name: 'nginx'
    static_configs:
      - targets: ['localhost:9113']
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "ERPGo Monitoring",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "singlestat",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m]) / rate(http_requests_total[5m])",
            "legendFormat": "Error Rate"
          }
        ]
      }
    ]
  }
}
```

### Log Aggregation

```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/erpgo/*.log
  fields:
    service: erpgo
    environment: production
  fields_under_root: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "erpgo-%{+yyyy.MM.dd}"

setup.kibana:
  host: "kibana:5601"
```

## Backup and Recovery

### Database Backup Script

```bash
#!/bin/bash
# backup_database.sh

BACKUP_DIR="/backups/database"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="erpgo_production"
DB_USER="erpgo_user"

# Create backup directory
mkdir -p $BACKUP_DIR

# Create database backup
pg_dump -h localhost -U $DB_USER -d $DB_NAME | gzip > $BACKUP_DIR/erpgo_$DATE.sql.gz

# Remove backups older than 30 days
find $BACKUP_DIR -name "erpgo_*.sql.gz" -mtime +30 -delete

# Upload to S3 (optional)
aws s3 cp $BACKUP_DIR/erpgo_$DATE.sql.gz s3://erpgo-backups/database/

echo "Database backup completed: erpgo_$DATE.sql.gz"
```

### File Backup Script

```bash
#!/bin/bash
# backup_files.sh

BACKUP_DIR="/backups/files"
SOURCE_DIR="/opt/erpgo/storage"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p $BACKUP_DIR

# Create file backup
tar -czf $BACKUP_DIR/files_$DATE.tar.gz -C $SOURCE_DIR .

# Remove backups older than 30 days
find $BACKUP_DIR -name "files_*.tar.gz" -mtime +30 -delete

# Upload to S3
aws s3 cp $BACKUP_DIR/files_$DATE.tar.gz s3://erpgo-backups/files/

echo "File backup completed: files_$DATE.tar.gz"
```

### Automated Backups

```bash
# Add to crontab
0 2 * * * /opt/erpgo/scripts/backup_database.sh
0 3 * * * /opt/erpgo/scripts/backup_files.sh
```

## Security Hardening

### Firewall Configuration

```bash
# UFW configuration
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### System Hardening

```bash
# Disable unnecessary services
sudo systemctl disable telnet
sudo systemctl disable rsh
sudo systemctl disable rlogin

# Configure fail2ban
sudo apt install fail2ban
sudo systemctl enable fail2ban
sudo systemctl start fail2ban

# Security updates
sudo apt update && sudo apt upgrade -y
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

### Application Security

```go
// Security middleware
func SecurityMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Security headers
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

        // Rate limiting
        if !rateLimiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

## Performance Optimization

### Database Optimization

```sql
-- Performance tuning configuration
-- postgresql.conf

# Memory settings
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB

# Connection settings
max_connections = 200
max_prepared_transactions = 200

# Checkpoint settings
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100

# Logging settings
log_min_duration_statement = 1000
log_checkpoints = on
log_connections = on
log_disconnections = on
```

### Application Optimization

```go
// Connection pool configuration
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)

// Redis pool configuration
redisPool := &redis.Pool{
    MaxIdle:     10,
    MaxActive:   100,
    IdleTimeout: 240 * time.Second,
    Dial: func() (redis.Conn, error) {
        return redis.Dial("tcp", redisAddr)
    },
}
```

### Caching Strategy

```go
// Multi-level caching
type CacheManager struct {
    l1Cache *sync.Map          // In-memory cache
    l2Cache *redis.Client      // Redis cache
    l3Cache *memcached.Client  // Memcached cache
}

func (cm *CacheManager) Get(key string) (interface{}, bool) {
    // L1 cache
    if value, ok := cm.l1Cache.Load(key); ok {
        return value, true
    }

    // L2 cache
    if value, err := cm.l2Cache.Get(key).Result(); err == nil {
        cm.l1Cache.Store(key, value)
        return value, true
    }

    // L3 cache
    if value, err := cm.l3Cache.Get(key); err == nil {
        cm.l1Cache.Store(key, value)
        cm.l2Cache.Set(key, value, time.Hour)
        return value, true
    }

    return nil, false
}
```

## Maintenance Procedures

### Rolling Deployment

```bash
#!/bin/bash
# deploy.sh

# Build new version
make build-prod

# Deploy to server 1
scp bin/erpgo user@server1:/opt/erpgo/bin/erpgo.new
ssh user@server1 "sudo systemctl stop erpgo && sudo mv /opt/erpgo/bin/erpgo.new /opt/erpgo/bin/erpgo && sudo systemctl start erpgo"

# Wait for health check
sleep 30
curl -f http://server1/health || exit 1

# Deploy to server 2
scp bin/erpgo user@server2:/opt/erpgo/bin/erpgo.new
ssh user@server2 "sudo systemctl stop erpgo && sudo mv /opt/erpgo/bin/erpgo.new /opt/erpgo/bin/erpgo && sudo systemctl start erpgo"

echo "Deployment completed successfully"
```

### Database Maintenance

```bash
#!/bin/bash
# maintain_database.sh

# Vacuum and analyze
psql -h localhost -U erpgo_user -d erpgo_production -c "VACUUM ANALYZE;"

# Reindex
psql -h localhost -U erpgo_user -d erpgo_production -c "REINDEX DATABASE erpgo_production;"

# Update statistics
psql -h localhost -U erpgo_user -d erpgo_production -c "ANALYZE;"

echo "Database maintenance completed"
```

## Troubleshooting

### Common Issues

#### Database Connection Issues

```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Check connection
psql -h localhost -U erpgo_user -d erpgo_production

# Check logs
sudo tail -f /var/log/postgresql/postgresql-14-main.log
```

#### Performance Issues

```bash
# Check system resources
top
htop
iostat -x 1

# Check database performance
psql -h localhost -U erpgo_user -d erpgo_production -c "SELECT * FROM pg_stat_activity;"

# Check slow queries
psql -h localhost -U erpgo_user -d erpgo_production -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

#### Memory Issues

```bash
# Check memory usage
free -h
cat /proc/meminfo

# Check application memory
ps aux | grep erpgo

# Check Go memory stats
curl http://localhost:8080/debug/pprof/heap
```

### Log Analysis

```bash
# Error logs
sudo tail -f /var/log/erpgo/error.log

# Access logs
sudo tail -f /var/log/nginx/access.log

# Application logs
sudo journalctl -u erpgo -f
```

### Health Check Script

```bash
#!/bin/bash
# health_check.sh

# Check application health
if ! curl -f http://localhost:8080/health; then
    echo "Application health check failed"
    exit 1
fi

# Check database connection
if ! pg_isready -h localhost -p 5432 -U erpgo_user; then
    echo "Database health check failed"
    exit 1
fi

# Check Redis connection
if ! redis-cli -h localhost -p 6379 ping; then
    echo "Redis health check failed"
    exit 1
fi

echo "All health checks passed"
```

## Support and Contact

For deployment support:

- **Documentation**: https://docs.erpgo.example.com
- **Support Email**: support@erpgo.example.com
- **Emergency Contact**: emergency@erpgo.example.com
- **Status Page**: https://status.erpgo.example.com

---

**Note**: This deployment guide covers production deployment scenarios. Always test deployments in a staging environment before deploying to production.