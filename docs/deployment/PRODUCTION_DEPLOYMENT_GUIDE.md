# ERPGo Production Deployment Guide

This guide provides comprehensive instructions for deploying ERPGo in a production environment with high availability, security, monitoring, and disaster recovery capabilities.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Environment Setup](#environment-setup)
4. [SSL/TLS Configuration](#ssltls-configuration)
5. [Database Setup](#database-setup)
6. [Application Deployment](#application-deployment)
7. [Monitoring Setup](#monitoring-setup)
8. [Backup and Recovery](#backup-and-recovery)
9. [Maintenance Operations](#maintenance-operations)
10. [Troubleshooting](#troubleshooting)
11. [Security Considerations](#security-considerations)
12. [Performance Optimization](#performance-optimization)

## Prerequisites

### System Requirements

- **Operating System**: Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- **CPU**: Minimum 8 cores, Recommended 16+ cores
- **Memory**: Minimum 16GB RAM, Recommended 32GB+ RAM
- **Storage**: Minimum 100GB SSD, Recommended 500GB+ SSD
- **Network**: 1Gbps+ network connection
- **Docker**: Version 20.10+
- **Docker Compose**: Version 2.0+

### Software Dependencies

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Install additional tools
sudo apt-get update
sudo apt-get install -y nginx-certbot python3-certbot-nginx htop iotop
```

### Domain and SSL Setup

- Registered domain name (e.g., `yourdomain.com`)
- DNS A records pointing to your server IP
- SSL certificates (Let's Encrypt recommended)

## Architecture Overview

### Production Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Internet      │    │   CDN/CloudFlare│    │   Load Balancer │
│                 │───▶│                 │───▶│   (Nginx)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                       ┌─────────────────────────────────┼─────────────────────────────────┐
                       │                                 │                                 │
        ┌──────────────▼───────┐           ┌─────────────▼──────────┐        ┌────────────▼──────────┐
        │    ERPGo API         │           │    ERPGo API           │        │    ERPGo API           │
        │   (Instance 1)       │           │   (Instance 2)         │        │   (Instance 3)         │
        └──────────────┬───────┘           └─────────────┬──────────┘        └─────────────┬──────────┘
                       │                                 │                                 │
        ┌──────────────▼───────┐           ┌─────────────▼──────────┐        ┌────────────▼──────────┐
        │  PostgreSQL Primary  │           │   PostgreSQL Replica   │        │    Redis Master        │
        │   (Read/Write)       │           │    (Read Only)         │        │    (Cache)             │
        └──────────────────────┘           └───────────────────────┘        └───────────────────────┘
                                                                                          │
                                                                   ┌──────────────────────┘
                                                                   │
                                                   ┌──────────────▼──────────┐
                                                   │      Redis Replica       │
                                                   │      (Cache)             │
                                                   └───────────────────────────┘
```

### Components

1. **Load Balancer**: Nginx with SSL termination
2. **Application**: Multiple ERPGo API instances
3. **Database**: PostgreSQL with primary-replica configuration
4. **Cache**: Redis with master-replica configuration
5. **Monitoring**: Prometheus, Grafana, AlertManager
6. **Logging**: Loki with Promtail for log aggregation

## Environment Setup

### 1. Clone Repository

```bash
git clone https://github.com/your-org/erpgo.git
cd erpgo
```

### 2. Configure Environment Variables

```bash
# Copy production environment template
cp .env.production.example .env.production

# Edit configuration
nano .env.production
```

### 3. Critical Security Configuration

Edit `.env.production` and update these values:

```bash
# Database Security
POSTGRES_PASSWORD=GENERATE_STRONG_PASSWORD_HERE
POSTGRES_REPLICATION_PASSWORD=GENERATE_STRONG_REPLICATION_PASSWORD

# JWT Security
JWT_SECRET=GENERATE_256BIT_SECRET_HERE

# SSL Configuration
DOMAIN_NAME=yourdomain.com
SSL_REDIRECT=true

# Monitoring Security
GRAFANA_PASSWORD=GENERATE_STRONG_GRAFANA_PASSWORD
GRAFANA_SECRET_KEY=GENERATE_GRAFANA_SECRET_KEY
```

### 4. Create Required Directories

```bash
mkdir -p data/{postgres,redis,nats,uploads,logs,static}
mkdir -p data/{prometheus,grafana,alertmanager,loki}
mkdir -p backups/{postgres,deployment,disaster_recovery}
mkdir -p configs/{nginx/ssl,postgres,redis,logrotate}
```

### 5. Set Proper Permissions

```bash
# Set data directory permissions
sudo chown -R 1001:1001 data/uploads data/logs data/static
chmod 755 data

# Set backup directory permissions
chmod 700 backups
```

## SSL/TLS Configuration

### Option 1: Let's Encrypt (Recommended)

```bash
# Install Certbot
sudo apt-get install certbot python3-certbot-nginx

# Generate SSL certificate
sudo certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com

# Copy certificates to project directory
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem configs/nginx/ssl/cert.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem configs/nginx/ssl/key.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/chain.pem configs/nginx/ssl/chain.pem

# Set proper permissions
sudo chmod 600 configs/nginx/ssl/key.pem
sudo chmod 644 configs/nginx/ssl/cert.pem configs/nginx/ssl/chain.pem
```

### Option 2: Self-Signed (Testing Only)

```bash
# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout configs/nginx/ssl/key.pem \
    -out configs/nginx/ssl/cert.pem \
    -subj "/C=US/ST=State/L=City/O=Organization/CN=yourdomain.com"

# Create chain file
cp configs/nginx/ssl/cert.pem configs/nginx/ssl/chain.pem
```

### Auto-Renewal Setup

```bash
# Add to crontab for automatic renewal
echo "0 3 * * * certbot renew --quiet --deploy-hook 'docker-compose -f docker-compose.prod.yml restart nginx'" | sudo crontab -
```

## Database Setup

### 1. PostgreSQL Configuration

Create custom PostgreSQL configuration:

```bash
# Create postgresql.conf
cat > configs/postgres/postgresql.conf << EOF
# Connection Settings
listen_addresses = '*'
port = 5432
max_connections = 200

# Memory Settings
shared_buffers = 4GB
effective_cache_size = 12GB
work_mem = 64MB
maintenance_work_mem = 1GB

# WAL Settings
wal_level = replica
max_wal_size = 4GB
min_wal_size = 1GB
checkpoint_completion_target = 0.9

# Replication Settings
max_wal_senders = 3
wal_keep_segments = 32
hot_standby = on

# Logging
log_destination = 'stderr'
logging_collector = on
log_directory = 'pg_log'
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_min_duration_statement = 1000
log_checkpoints = on
log_connections = on
log_disconnections = on
EOF
```

### 2. Create Database Initialization Script

```bash
# Create replica setup script
cat > configs/postgres/replica-setup.sh << 'EOF'
#!/bin/bash
set -e

# Wait for primary to be ready
until pg_isready -h $POSTGRES_PRIMARY_HOST -p 5432 -U $POSTGRES_USER; do
  echo "Waiting for primary database..."
  sleep 2
done

# Create replica
pg_basebackup -h $POSTGRES_PRIMARY_HOST -D /var/lib/postgresql/data -U $POSTGRES_REPLICATION_USER -v -P -W

# Configure replica
cat >> /var/lib/postgresql/data/recovery.conf << EOL
standby_mode = 'on'
primary_conninfo = 'host=$POSTGRES_PRIMARY_HOST port=5432 user=$POSTGRES_REPLICATION_USER'
restore_command = 'cp /var/lib/postgresql/wal_archive/%f %p'
EOL

chmod 600 /var/lib/postgresql/data/recovery.conf
EOF

chmod +x configs/postgres/replica-setup.sh
```

## Application Deployment

### 1. Build Application Image

```bash
# Build production image
docker build -t erpgo:latest .

# Tag with version
docker tag erpgo:latest erpgo:v1.0.0
```

### 2. Initial Deployment

```bash
# Deploy infrastructure services first
docker-compose -f docker-compose.prod.yml up -d postgres-primary redis-master nats-server

# Wait for services to be ready
sleep 30

# Run database migrations
docker-compose -f docker-compose.prod.yml run --rm migrator

# Deploy application services
docker-compose -f docker-compose.prod.yml up -d api worker nginx

# Deploy monitoring stack
docker-compose -f docker-compose.prod.yml --profile monitoring up -d
```

### 3. Verify Deployment

```bash
# Check service status
docker-compose -f docker-compose.prod.yml ps

# Check health endpoints
curl https://yourdomain.com/health
curl https://yourdomain.com/api/v1/health

# Check database connectivity
docker exec erpgo-postgres-primary pg_isready -U erpgo -d erp

# Check Redis connectivity
docker exec erpgo-redis-master redis-cli ping
```

### 4. Production Deployment Script

Use the automated deployment script:

```bash
# Rolling update deployment
./scripts/deploy.sh deploy rolling v1.0.0

# Blue-green deployment (zero downtime)
./scripts/deploy.sh deploy blue-green v1.0.0

# Rollback if needed
./scripts/deploy.sh rollback ./backups/deployment_*
```

## Monitoring Setup

### 1. Grafana Dashboard Access

1. Navigate to `https://monitoring.yourdomain.com`
2. Login with admin credentials
3. Import pre-configured dashboards from `configs/grafana/dashboards/`

### 2. AlertManager Configuration

Configure alert routing in `configs/alertmanager/alertmanager.yml`:

```yaml
global:
  smtp_smarthost: 'smtp.yourdomain.com:587'
  smtp_from: 'alerts@yourdomain.com'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'

receivers:
  - name: 'web.hook'
    email_configs:
      - to: 'admin@yourdomain.com'
        subject: '[ERPGo Alert] {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          {{ end }}
```

### 3. Prometheus Targets

Verify all targets are up:

```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Expected targets:
# - erpgo-api:8080
# - postgres-exporter:9187
# - redis-exporter:9121
# - node-exporter:9100
# - grafana:3000
```

## Backup and Recovery

### 1. Automated Backups

Set up automated database backups:

```bash
# Add to crontab
echo "0 2 * * * /path/to/erpgo/scripts/backup/database-backup.sh backup full" | crontab -

# Test backup
./scripts/backup/database-backup.sh backup full
./scripts/backup/database-backup.sh list
```

### 2. Disaster Recovery

Test disaster recovery procedures:

```bash
# Create DR backup
./scripts/disaster-recovery.sh backup full

# Test RTO/RPO compliance
./scripts/disaster-recovery.sh test

# Simulate disaster and recover
./scripts/disaster-recovery.sh simulate partial
./scripts/disaster-recovery.sh restore ./backups/disaster_recovery/dr_backup_*
```

### 3. Backup Retention

Configure backup retention policies:

```bash
# Database backups: 30 days
# Application backups: 90 days
# Log files: 30 days
# Disaster recovery backups: 90 days
```

## Maintenance Operations

### 1. Log Rotation

Set up automated log rotation:

```bash
# Configure logrotate
./scripts/log-rotation.sh setup

# Test log rotation
./scripts/log-rotation.sh rotate

# Manual cleanup
./scripts/log-rotation.sh cleanup
```

### 2. System Updates

Update system packages:

```bash
# Update Docker images
docker-compose -f docker-compose.prod.yml pull

# Restart services with new images
./scripts/deploy.sh deploy rolling latest

# Update system packages
sudo apt update && sudo apt upgrade -y
```

### 3. Performance Monitoring

Monitor system performance:

```bash
# Check resource usage
docker stats

# Check disk usage
df -h

# Check memory usage
free -h

# Check network connections
netstat -tulpn
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Issues

```bash
# Check database logs
docker logs erpgo-postgres-primary

# Test connectivity
docker exec erpgo-postgres-primary pg_isready -U erpgo -d erp

# Check connection settings
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "SHOW max_connections;"
```

#### 2. Redis Connection Issues

```bash
# Check Redis logs
docker logs erpgo-redis-master

# Test connectivity
docker exec erpgo-redis-master redis-cli ping

# Check memory usage
docker exec erpgo-redis-master redis-cli info memory
```

#### 3. Application Startup Issues

```bash
# Check application logs
docker logs erpgo-api

# Check environment variables
docker exec erpgo-api env | grep -E "(DATABASE|REDIS|JWT)"

# Test health endpoint
curl http://localhost:8080/health
```

#### 4. SSL Certificate Issues

```bash
# Check certificate validity
openssl x509 -in configs/nginx/ssl/cert.pem -text -noout

# Test SSL configuration
nginx -t -c configs/nginx/nginx.conf

# Check certificate expiration
openssl x509 -in configs/nginx/ssl/cert.pem -noout -dates
```

### Performance Issues

#### 1. High CPU Usage

```bash
# Check process CPU usage
docker stats --no-stream

# Check application goroutines
curl http://localhost:8080/debug/pprof/goroutine?debug=1

# Profile CPU usage
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
```

#### 2. High Memory Usage

```bash
# Check memory usage by container
docker stats --no-stream | grep erpgo

# Check application memory
curl http://localhost:8080/debug/pprof/heap?debug=1

# Check database memory
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "SELECT * FROM pg_stat_activity;"
```

#### 3. Database Performance

```bash
# Check slow queries
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# Check connection pool
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "SELECT state, count(*) FROM pg_stat_activity GROUP BY state;"

# Analyze query performance
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "EXPLAIN ANALYZE SELECT * FROM users LIMIT 10;"
```

## Security Considerations

### 1. Network Security

```bash
# Firewall configuration
sudo ufw enable
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw deny 5432/tcp   # PostgreSQL (internal only)
sudo ufw deny 6379/tcp   # Redis (internal only)
```

### 2. Application Security

- Change all default passwords
- Use strong, unique passwords
- Enable SSL/TLS everywhere
- Implement rate limiting
- Use environment variables for secrets
- Regular security updates

### 3. Database Security

```sql
-- Create read-only user for reporting
CREATE USER reporting_user WITH PASSWORD 'strong_password';
GRANT CONNECT ON DATABASE erp TO reporting_user;
GRANT USAGE ON SCHEMA public TO reporting_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO reporting_user;

-- Audit user actions
CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    user_id UUID,
    action VARCHAR(100),
    table_name VARCHAR(100),
    timestamp TIMESTAMP DEFAULT NOW(),
    details JSONB
);
```

## Performance Optimization

### 1. Database Optimization

```sql
-- Create indexes for frequently queried columns
CREATE INDEX CONCURRENTLY idx_users_email_active ON users(email, is_active);
CREATE INDEX CONCURRENTLY idx_orders_customer_status ON orders(customer_id, status);
CREATE INDEX CONCURRENTLY idx_products_category_active ON products(category_id, is_active);

-- Update table statistics
ANALYZE users;
ANALYZE orders;
ANALYZE products;
```

### 2. Application Optimization

```bash
# Tune Go runtime
export GOMAXPROCS=8
export GOGC=100
export GOMEMLIMIT=8GiB

# Configure connection pools
DATABASE_MAX_CONNECTIONS=100
DATABASE_MAX_IDLE_CONNECTIONS=20
REDIS_POOL_SIZE=100
```

### 3. Nginx Optimization

```nginx
# Enable gzip compression
gzip on;
gzip_vary on;
gzip_min_length 1000;
gzip_types text/plain text/css application/json application/javascript;

# Enable caching
location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
    expires 1y;
    add_header Cache-Control "public, immutable";
}
```

## Emergency Procedures

### 1. Service Outage

```bash
# Check all services
docker-compose -f docker-compose.prod.yml ps

# Restart failed services
docker-compose -f docker-compose.prod.yml restart

# Check logs for errors
docker-compose -f docker-compose.prod.yml logs --tail=100
```

### 2. Database Corruption

```bash
# Stop application
docker-compose -f docker-compose.prod.yml stop api worker

# Restore from latest backup
./scripts/backup/database-backup.sh restore latest_backup.sql

# Verify data integrity
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "SELECT COUNT(*) FROM users;"

# Start application
docker-compose -f docker-compose.prod.yml start api worker
```

### 3. Full System Recovery

```bash
# Complete disaster recovery
./scripts/disaster-recovery.sh restore ./backups/disaster_recovery/dr_backup_*

# Verify all services
./scripts/deploy.sh health

# Send notification to team
echo "ERPGo system recovery completed" | mail -s "System Recovery Complete" team@yourdomain.com
```

## Support and Maintenance

### Monitoring Dashboards

- **System Overview**: `https://monitoring.yourdomain.com/d/overview`
- **Application Metrics**: `https://monitoring.yourdomain.com/d/application`
- **Database Performance**: `https://monitoring.yourdomain.com/d/database`
- **Infrastructure**: `https://monitoring.yourdomain.com/d/infrastructure`

### Contact Information

- **Emergency Contact**: +1-555-EMERGENCY
- **Support Email**: support@yourdomain.com
- **Documentation**: https://docs.erpgo.com
- **Status Page**: https://status.erpgo.com

### Regular Maintenance Schedule

- **Daily**: Log rotation, backup verification
- **Weekly**: Security updates, performance review
- **Monthly**: System updates, capacity planning
- **Quarterly**: Disaster recovery testing, security audit

## Conclusion

This production deployment guide provides a comprehensive framework for deploying and maintaining ERPGo in a production environment. Follow the procedures carefully and ensure all security measures are implemented before going live.

For additional support or questions, refer to the project documentation or contact the support team.