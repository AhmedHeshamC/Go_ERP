# ERPGo Troubleshooting Guide & Runbooks

## Overview

This guide provides comprehensive troubleshooting procedures and runbooks for common issues encountered in the ERPGo system. It covers application, database, infrastructure, and performance issues with step-by-step resolution procedures.

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Application Issues](#application-issues)
3. [Database Issues](#database-issues)
4. [Authentication & Authorization](#authentication--authorization)
5. [Performance Issues](#performance-issues)
6. [Network & Connectivity](#network--connectivity)
7. [Infrastructure Issues](#infrastructure-issues)
8. [Data Integrity Issues](#data-integrity-issues)
9. [Security Issues](#security-issues)
10. [Monitoring & Alerting](#monitoring--alerting)
11. [Emergency Procedures](#emergency-procedures)
12. [Common Error Codes](#common-error-codes)

## Quick Diagnostics

### Health Check Endpoints

Always start with these basic health checks:

```bash
# Application health
curl http://localhost:8080/health

# Detailed health with dependencies
curl http://localhost:8080/health/detailed

# Database connectivity
curl http://localhost:8080/health/database

# Cache connectivity
curl http://localhost:8080/health/cache
```

### System Status Commands

```bash
# Check application logs
docker-compose logs -f erpgo-api

# Check system resources
top
htop
df -h
free -h

# Check network connectivity
ping -c 4 google.com
netstat -tulpn | grep :8080
```

### Docker Environment Diagnostics

```bash
# Check container status
docker-compose ps

# Check container resource usage
docker stats

# Inspect specific container
docker inspect erpgo-api

# Execute commands in container
docker-compose exec erpgo-api ps aux
```

## Application Issues

### Application Won't Start

#### Symptoms
- Application fails to start
- Crash loops in Docker
- Port binding errors

#### Runbook: Application Startup Issues

1. **Check Configuration**
   ```bash
   # Verify environment variables
   docker-compose config

   # Check .env file
   cat .env | grep -E "(DATABASE_URL|REDIS_URL|JWT_SECRET)"

   # Validate configuration syntax
   docker-compose config --quiet
   ```

2. **Check Port Availability**
   ```bash
   # Check if port is in use
   lsof -i :8080

   # Kill conflicting process if needed
   kill -9 <PID>

   # Or change port in configuration
   sed -i 's/SERVER_PORT=8080/SERVER_PORT=8081/' .env
   ```

3. **Check Dependencies**
   ```bash
   # Verify database connection
   docker-compose exec erpgo-api go run cmd/healthcheck/main.go database

   # Verify Redis connection
   docker-compose exec erpgo-api go run cmd/healthcheck/main.go redis

   # Check all services
   docker-compose exec erpgo-api go run cmd/healthcheck/main.go all
   ```

4. **Check Logs for Errors**
   ```bash
   # Application logs
   docker-compose logs erpgo-api

   # Follow logs in real-time
   docker-compose logs -f erpgo-api

   # Check recent errors
   docker-compose logs erpgo-api | grep -i error | tail -20
   ```

5. **Rebuild Application**
   ```bash
   # Rebuild Docker image
   docker-compose build --no-cache erpgo-api

   # Restart with fresh container
   docker-compose up -d --force-recreate erpgo-api
   ```

### 500 Internal Server Errors

#### Symptoms
- API endpoints returning 500 errors
- Application crashes on specific requests
- Unhandled exceptions

#### Runbook: 500 Error Resolution

1. **Identify Error Pattern**
   ```bash
   # Check recent 500 errors
   docker-compose logs erpgo-api | grep "500" | tail -10

   # Look for stack traces
   docker-compose logs erpgo-api | grep -A 10 -B 5 "panic\|runtime error"

   # Check error patterns
   docker-compose logs erpgo-api | grep -i error | tail -20
   ```

2. **Check Database Connection**
   ```bash
   # Test database connectivity
   docker-compose exec postgres psql -U erpgo -d erp -c "SELECT 1;"

   # Check database logs
   docker-compose logs postgres | tail -20

   # Check connection pool
   docker-compose exec erpgo-api curl http://localhost:8080/debug/pprof/heap
   ```

3. **Check Resource Limits**
   ```bash
   # Memory usage
   docker stats erpgo-api

   # Check Go memory stats
   curl http://localhost:8080/debug/pprof/heap > heap.prof
   go tool pprof heap.prof

   # Check goroutines
   curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
   go tool pprof goroutine.prof
   ```

4. **Validate Request Data**
   ```bash
   # Test with valid request
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","password":"password"}'

   # Check request size limits
   curl -v -X POST http://localhost:8080/api/v1/products \
     -H "Content-Type: application/json" \
     -d '{"name":"test"}'
   ```

### Memory Leaks

#### Symptoms
- Memory usage continuously increasing
- Container restarts due to OOM
- Performance degradation over time

#### Runbook: Memory Leak Detection

1. **Monitor Memory Usage**
   ```bash
   # Real-time memory monitoring
   watch -n 1 'docker stats --no-stream erpgo-api'

   # Memory profiling
   curl http://localhost:8080/debug/pprof/heap > heap_$(date +%s).prof

   # Compare profiles over time
   go tool pprof -base heap_1640000000.prof heap_1640000600.prof
   ```

2. **Check Goroutine Leaks**
   ```bash
   # Goroutine profiling
   curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
   go tool pprof goroutine.prof

   # Look for stuck goroutines
   curl http://localhost:8080/debug/pprof/goroutine?debug=1
   ```

3. **Database Connection Analysis**
   ```bash
   # Check open connections
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT count(*) FROM pg_stat_activity WHERE state = 'active';
   "

   # Check connection pool settings
   docker-compose exec erpgo-api env | grep -i db
   ```

4. **Restart Services**
   ```bash
   # Graceful restart
   docker-compose restart erpgo-api

   # Force restart if needed
   docker-compose stop erpgo-api
   docker-compose up -d erpgo-api
   ```

## Database Issues

### Database Connection Issues

#### Symptoms
- "Database connection failed" errors
- Connection timeouts
- Too many connections error

#### Runbook: Database Connection Resolution

1. **Check Database Status**
   ```bash
   # PostgreSQL service status
   docker-compose ps postgres

   # Database logs
   docker-compose logs postgres | tail -20

   # Test direct connection
   docker-compose exec postgres psql -U erpgo -d erp -c "SELECT version();"
   ```

2. **Verify Connection Parameters**
   ```bash
   # Check environment variables
   docker-compose exec erpgo-api env | grep DATABASE_URL

   # Parse connection string
   docker-compose exec erpgo-api go run cmd/debug/main.go parse-db-url

   # Test connection from application container
   docker-compose exec erpgo-api pg_isready -h postgres -p 5432 -U erpgo
   ```

3. **Check Connection Limits**
   ```bash
   # Current connections
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT count(*) as active_connections
     FROM pg_stat_activity
     WHERE state = 'active';
   "

   # Max connections setting
   docker-compose exec postgres psql -U erpgo -d erp -c "SHOW max_connections;"

   # Connection statistics
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT datname, numbackends, xact_commit, xact_rollback
     FROM pg_stat_database
     WHERE datname = 'erp';
   "
   ```

4. **Restart Database Service**
   ```bash
   # Graceful restart
   docker-compose restart postgres

   # Force restart (data preserved in volumes)
   docker-compose stop postgres
   docker-compose up -d postgres

   # Verify recovery
   docker-compose exec postgres psql -U erpgo -d erp -c "SELECT 1;"
   ```

### Slow Query Performance

#### Symptoms
- API responses taking > 5 seconds
- Database queries timing out
- High CPU usage on database

#### Runbook: Slow Query Resolution

1. **Identify Slow Queries**
   ```bash
   # Enable slow query logging
   docker-compose exec postgres psql -U erpgo -d erp -c "
     ALTER SYSTEM SET log_min_duration_statement = 1000;
     SELECT pg_reload_conf();
   "

   # Check running queries
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT pid, now() - pg_stat_activity.query_start AS duration, query
     FROM pg_stat_activity
     WHERE state = 'active' AND now() - query_start > interval '5 seconds';
   "

   # Query statistics
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT query, calls, total_time, mean_time, rows
     FROM pg_stat_statements
     ORDER BY mean_time DESC
     LIMIT 10;
   "
   ```

2. **Analyze Query Plans**
   ```bash
   # Explain specific query
   docker-compose exec postgres psql -U erpgo -d erp -c "
     EXPLAIN ANALYZE SELECT * FROM orders WHERE customer_id = 'some-uuid';
   "

   # Check index usage
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read
     FROM pg_stat_user_indexes
     ORDER BY idx_scan DESC;
   "
   ```

3. **Update Statistics**
   ```bash
   # Update table statistics
   docker-compose exec postgres psql -U erpgo -d erp -c "ANALYZE;"

   # Rebuild indexes if needed
   docker-compose exec postgres psql -U erpgo -d erp -c "
     REINDEX DATABASE erp;
   "
   ```

4. **Optimize Configuration**
   ```bash
   # Check current settings
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SHOW shared_buffers;
     SHOW work_mem;
     SHOW effective_cache_size;
   "

   # Temporary optimization (restart required)
   # Edit postgresql.conf in container or use environment variables
   ```

### Database Locks and Deadlocks

#### Symptoms
- Queries hanging indefinitely
- "Deadlock detected" errors
- Application timeouts

#### Runbook: Database Lock Resolution

1. **Identify Locks**
   ```bash
   # Check for locks
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT blocked_locks.pid AS blocked_pid,
            blocked_activity.usename AS blocked_user,
            blocking_locks.pid AS blocking_pid,
            blocking_activity.usename AS blocking_user,
            blocked_activity.query AS blocked_statement,
            blocking_activity.query AS current_statement_in_blocking_process
     FROM pg_catalog.pg_locks blocked_locks
     JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
     JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
     JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
     WHERE NOT blocked_locks.granted;
   "

   # Check transaction status
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT pid, state, query, now() - query_start AS duration
     FROM pg_stat_activity
     WHERE state != 'idle'
     ORDER BY duration DESC;
   "
   ```

2. **Resolve Deadlocks**
   ```bash
   # Kill blocking transaction (use with caution)
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT pg_terminate_backend(<blocking_pid>);
   "

   # Or cancel the query
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT pg_cancel_backend(<blocking_pid>);
   "
   ```

3. **Prevent Future Deadlocks**
   ```bash
   # Review application code for proper transaction ordering
   # Ensure consistent order of table access
   # Keep transactions short
   # Use appropriate isolation levels
   ```

## Authentication & Authorization

### Login Failures

#### Symptoms
- Users unable to login
- "Invalid credentials" errors
- JWT token generation failures

#### Runbook: Login Issue Resolution

1. **Verify User Credentials**
   ```bash
   # Check if user exists
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT id, email, is_active, is_verified
     FROM users
     WHERE email = 'user@example.com';
   "

   # Check password hash format
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT id, email, password_hash LIKE '$2a$%' AS valid_bcrypt
     FROM users
     WHERE email = 'user@example.com';
   "
   ```

2. **Test Authentication Flow**
   ```bash
   # Direct API test
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com","password":"password"}' \
     -v

   # Check response headers
   curl -I -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com","password":"password"}'
   ```

3. **Verify JWT Configuration**
   ```bash
   # Check JWT secret
   docker-compose exec erpgo-api env | grep JWT_SECRET

   # Test token generation
   docker-compose exec erpgo-api go run cmd/debug/main.go test-jwt

   # Verify token format
   echo "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." | cut -d. -f2 | base64 -d
   ```

4. **Reset User Password**
   ```bash
   # Generate new password hash
   docker-compose exec erpgo-api go run cmd/tools/main.go hash-password "newpassword"

   # Update user password
   docker-compose exec postgres psql -U erpgo -d erp -c "
     UPDATE users
     SET password_hash = '$2a$12$...', updated_at = NOW()
     WHERE email = 'user@example.com';
   "
   ```

### JWT Token Issues

#### Symptoms
- "Invalid token" errors
- Token expiration before expected
- Authorization failures

#### Runbook: JWT Token Issues

1. **Verify Token Format**
   ```bash
   # Decode JWT token
   echo "your.jwt.token" | cut -d. -f2 | base64 -d 2>/dev/null | jq .

   # Check token structure
   echo "your.jwt.token" | awk -F. '{print NF}'
   ```

2. **Check Token Expiration**
   ```bash
   # Check JWT configuration
   docker-compose exec erpgo-api env | grep JWT_EXPIRY

   # Test token expiration
   docker-compose exec erpgo-api go run cmd/debug/main.go check-token "your.jwt.token"
   ```

3. **Refresh Token Process**
   ```bash
   # Test refresh token
   curl -X POST http://localhost:8080/api/v1/auth/refresh \
     -H "Content-Type: application/json" \
     -d '{"refresh_token":"your_refresh_token"}'
   ```

### Permission Issues

#### Symptoms
- Access denied errors
- Role assignment problems
- Permission evaluation failures

#### Runbook: Permission Issues

1. **Check User Roles**
   ```bash
   # Get user roles
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT u.email, r.name, ur.assigned_at
     FROM users u
     JOIN user_roles ur ON u.id = ur.user_id
     JOIN roles r ON ur.role_id = r.id
     WHERE u.email = 'user@example.com';
   "
   ```

2. **Verify Role Permissions**
   ```bash
   # Check role permissions
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT r.name, r.permissions
     FROM roles r
     WHERE r.name = 'admin';
   "

   # Test permission evaluation
   docker-compose exec erpgo-api go run cmd/debug/main.go check-permissions \
     user-id resource-id action
   ```

3. **Assign Missing Roles**
   ```bash
   # Add role to user
   docker-compose exec postgres psql -U erpgo -d erp -c "
     INSERT INTO user_roles (user_id, role_id, assigned_by)
     SELECT u.id, r.id, u.id
     FROM users u, roles r
     WHERE u.email = 'user@example.com' AND r.name = 'admin';
   "
   ```

## Performance Issues

### High Response Times

#### Symptoms
- API responses > 5 seconds
- Database query timeouts
- User interface lag

#### Runbook: Performance Resolution

1. **Application Profiling**
   ```bash
   # Enable pprof endpoints
   curl http://localhost:8080/debug/pprof/profile > cpu.prof
   go tool pprof cpu.prof

   # Memory profiling
   curl http://localhost:8080/debug/pprof/heap > heap.prof
   go tool pprof heap.prof

   # Goroutine analysis
   curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
   go tool pprof goroutine.prof
   ```

2. **Database Performance**
   ```bash
   # Check slow queries
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT query, mean_time, calls
     FROM pg_stat_statements
     WHERE mean_time > 1000
     ORDER BY mean_time DESC;
   "

   # Check table sizes
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT schemaname, tablename,
            pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
     FROM pg_tables
     WHERE schemaname = 'public'
     ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
   "
   ```

3. **Cache Performance**
   ```bash
   # Check Redis hit rate
   docker-compose exec redis redis-cli info stats | grep keyspace

   # Check memory usage
   docker-compose exec redis redis-cli info memory

   # Test cache performance
   docker-compose exec erpgo-api go run cmd/debug/main.go test-cache
   ```

### High CPU Usage

#### Symptoms
- CPU usage > 80%
- System responsiveness issues
- Container throttling

#### Runbook: High CPU Resolution

1. **Identify CPU Consumers**
   ```bash
   # System processes
   top
   htop

   # Container processes
   docker exec erpgo-api top

   # Go CPU profiling
   curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
   go tool pprof cpu.prof
   ```

2. **Check Application Loops**
   ```bash
   # Goroutine inspection
   curl http://localhost:8080/debug/pprof/goroutine?debug=2

   # Look for blocking operations
   curl http://localhost:8080/debug/pprof/block
   go tool pprof block.prof
   ```

3. **Optimize Configuration**
   ```bash
   # Check Go runtime settings
   docker-compose exec erpgo-api env | grep GOMAXPROCS

   # Adjust CPU limits in docker-compose.yml
   # Set GOMAXPROCS environment variable
   ```

## Network & Connectivity

### Service Discovery Issues

#### Symptoms
- Services cannot communicate
- DNS resolution failures
- Connection refused errors

#### Runbook: Service Discovery Resolution

1. **Check Docker Network**
   ```bash
   # List networks
   docker network ls

   # Inspect network
   docker network inspect erpgo_erpgo-network

   # Test connectivity between containers
   docker-compose exec erpgo-api ping postgres
   docker-compose exec erpgo-api ping redis
   ```

2. **Verify Service Names**
   ```bash
   # Check service names in network
   docker-compose exec erpgo-api nslookup postgres

   # Test connection using service name
   docker-compose exec erpgo-api nc -z postgres 5432
   ```

3. **Recreate Network**
   ```bash
   # Stop all services
   docker-compose down

   # Remove network
   docker network rm erpgo_erpgo-network

   # Restart services
   docker-compose up -d
   ```

### SSL/TLS Issues

#### Symptoms
- Certificate errors
- HTTPS connection failures
- Handshake timeouts

#### Runbook: SSL/TLS Resolution

1. **Check Certificate Validity**
   ```bash
   # Verify certificate
   openssl s_client -connect localhost:443 -showcerts

   # Check certificate dates
   openssl x509 -in cert.pem -noout -dates

   # Verify certificate chain
   openssl verify cert.pem
   ```

2. **Test HTTPS Connection**
   ```bash
   # Test with curl
   curl -v https://localhost:8443/health

   # Test with specific CA
   curl --cac ca.pem https://localhost:8443/health
   ```

3. **Regenerate Certificates**
   ```bash
   # Generate self-signed certificate
   openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

   # Or use Let's Encrypt
   certbot --nginx -d yourdomain.com
   ```

## Infrastructure Issues

### Disk Space Issues

#### Symptoms
- Disk usage > 90%
- Write failures
- Log rotation issues

#### Runbook: Disk Space Resolution

1. **Check Disk Usage**
   ```bash
   # Overall disk usage
   df -h

   # Directory usage
   du -sh /var/lib/docker/*
   du -sh /var/log/*

   # Find large files
   find / -type f -size +1G 2>/dev/null
   ```

2. **Clean Up Docker Resources**
   ```bash
   # Remove unused images
   docker image prune -a

   # Remove unused containers
   docker container prune

   # Remove unused volumes
   docker volume prune

   # System prune
   docker system prune -a
   ```

3. **Log Management**
   ```bash
   # Rotate application logs
   logrotate -f /etc/logrotate.d/erpgo

   # Clear old logs
   find /var/log -name "*.log" -mtime +30 -delete

   # Configure log limits in docker-compose.yml
   ```

### Memory Issues

#### Symptoms
- Out of memory errors
- Container OOM kills
- Swap usage

#### Runbook: Memory Resolution

1. **Check Memory Usage**
   ```bash
   # System memory
   free -h
   cat /proc/meminfo

   # Container memory
   docker stats --no-stream

   # Process memory
   ps aux --sort=-%mem | head
   ```

2. **Optimize Memory Usage**
   ```bash
   # Check Go memory settings
   docker-compose exec erpgo-api env | grep GOMEMLIMIT

   # Tune garbage collection
   export GOGC=100
   export GODEBUG=gctrace=1
   ```

3. **Add Swap Space**
   ```bash
   # Create swap file
   fallocate -l 2G /swapfile
   chmod 600 /swapfile
   mkswap /swapfile
   swapon /swapfile

   # Add to fstab
   echo '/swapfile none swap sw 0 0' >> /etc/fstab
   ```

## Data Integrity Issues

### Inconsistent Data

#### Symptoms
- Foreign key constraint violations
- Data inconsistencies
- Calculation errors

#### Runbook: Data Integrity Resolution

1. **Identify Issues**
   ```bash
   # Check foreign key violations
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT conname, conrelid::regclass, confrelid::regclass
     FROM pg_constraint
     WHERE contype = 'f' AND convalidated = false;
   "

   # Check data consistency
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT 'orders_total_calculation' as check_name, COUNT(*) as violations
     FROM orders
     WHERE total_amount != (subtotal + tax_amount + shipping_amount - discount_amount);
   "
   ```

2. **Repair Data**
   ```bash
   # Fix order totals
   docker-compose exec postgres psql -U erpgo -d erp -c "
     UPDATE orders
     SET total_amount = subtotal + tax_amount + shipping_amount - discount_amount
     WHERE total_amount != (subtotal + tax_amount + shipping_amount - discount_amount);
   "

   # Recalculate inventory quantities
   docker-compose exec postgres psql -U erpgo -d erp -c "
     UPDATE products
     SET stock_quantity = (
       SELECT COALESCE(SUM(quantity_available), 0)
       FROM product_inventory
       WHERE product_id = products.id
     );
   "
   ```

3. **Prevent Future Issues**
   ```bash
   # Add constraints
   docker-compose exec postgres psql -U erpgo -d erp -c "
     ALTER TABLE orders ADD CONSTRAINT check_total_calculation
       CHECK (total_amount = subtotal + tax_amount + shipping_amount - discount_amount);
   "

   # Add triggers
   docker-compose exec postgres psql -U erpgo -d erp -c "
     CREATE TRIGGER update_product_stock
       AFTER UPDATE ON product_inventory
       FOR EACH ROW EXECUTE FUNCTION update_product_stock();
   "
   ```

## Security Issues

### Unauthorized Access Attempts

#### Symptoms
- Brute force login attempts
- Suspicious API calls
- Failed authentication spikes

#### Runbook: Security Incident Response

1. **Identify Threat**
   ```bash
   # Check failed login attempts
   docker-compose logs erpgo-api | grep "failed login" | tail -50

   # Check IP patterns
   docker-compose logs erpgo-api | grep "failed login" | \
     awk '{print $1}' | sort | uniq -c | sort -nr

   # Check for suspicious patterns
   docker-compose logs erpgo-api | grep -E "sql injection|xss|path traversal"
   ```

2. **Block Malicious IPs**
   ```bash
   # Add to firewall
   iptables -A INPUT -s <malicious_ip> -j DROP

   # Or use fail2ban
   fail2ban-client set erpgo banip <malicious_ip>
   ```

3. **Review Security Settings**
   ```bash
   # Check rate limiting
   docker-compose exec erpgo-api env | grep RATE_LIMIT

   # Review authentication settings
   docker-compose exec erpgo-api env | grep JWT

   # Check SSL/TLS configuration
   openssl s_client -connect localhost:443 -showcerts
   ```

## Monitoring & Alerting

### Setting Up Alerts

#### Critical Metrics to Monitor

1. **Application Metrics**
   - HTTP response times > 5 seconds
   - Error rate > 5%
   - Memory usage > 80%
   - CPU usage > 80%

2. **Database Metrics**
   - Connection pool usage > 90%
   - Slow queries > 1 second
   - Lock wait time > 10 seconds
   - Disk usage > 85%

3. **Infrastructure Metrics**
   - Disk usage > 90%
   - Memory usage > 90%
   - Container restarts
   - Service availability

#### Alert Configuration Example

```yaml
# Prometheus alert rules
groups:
  - name: erpgo_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: High error rate detected

      - alert: DatabaseConnectionPool
        expr: db_connections_active / db_connections_max > 0.9
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: Database connection pool nearly full
```

## Emergency Procedures

### Complete System Outage

#### Runbook: System Recovery

1. **Assess Situation**
   ```bash
   # Check all services
   docker-compose ps

   # Check system resources
   free -h && df -h

   # Check recent logs
   docker-compose logs --tail=100
   ```

2. **Emergency Restart**
   ```bash
   # Stop all services
   docker-compose down

   # Check system health
   systemctl status docker

   # Restart services
   docker-compose up -d

   # Verify health
   curl http://localhost:8080/health
   ```

3. **Data Recovery**
   ```bash
   # If database is corrupted
   docker-compose stop postgres
   # Restore from backup
   pg_restore -h localhost -U erpgo -d erp backup.dump

   # Verify data integrity
   docker-compose exec postgres psql -U erpgo -d erp -c "SELECT COUNT(*) FROM users;"
   ```

### Data Corruption

#### Runbook: Data Recovery

1. **Stop Application**
   ```bash
   docker-compose stop erpgo-api
   ```

2. **Assess Damage**
   ```bash
   # Check database consistency
   docker-compose exec postgres pg_dumpall --schema-only > schema.sql

   # Look for corruption signs
   docker-compose logs postgres | grep -i error
   ```

3. **Restore from Backup**
   ```bash
   # Select appropriate backup
   ls -la /backups/database/

   # Restore database
   docker-compose exec postgres psql -U erpgo -d erp -c "DROP DATABASE erp;"
   docker-compose exec postgres psql -U erpgo -d postgres -c "CREATE DATABASE erp;"
   pg_restore -h localhost -U erpgo -d erp /backups/database/erp_latest.dump
   ```

4. **Verify Recovery**
   ```bash
   # Check critical tables
   docker-compose exec postgres psql -U erpgo -d erp -c "
     SELECT 'users' as table_name, COUNT(*) as record_count FROM users
     UNION ALL
     SELECT 'products' as table_name, COUNT(*) as record_count FROM products
     UNION ALL
     SELECT 'orders' as table_name, COUNT(*) as record_count FROM orders;
   "

   # Test application
   docker-compose start erpgo-api
   curl http://localhost:8080/health
   ```

## Common Error Codes

### HTTP Status Codes

| Code | Meaning | Common Causes | Resolution |
|------|---------|---------------|------------|
| 400 | Bad Request | Invalid JSON, missing fields | Validate request format |
| 401 | Unauthorized | Invalid/missing token | Refresh authentication |
| 403 | Forbidden | Insufficient permissions | Check user roles |
| 404 | Not Found | Resource doesn't exist | Verify resource ID |
| 409 | Conflict | Duplicate resource | Check for existing data |
| 422 | Unprocessable Entity | Validation failed | Fix validation errors |
| 429 | Too Many Requests | Rate limit exceeded | Reduce request rate |
| 500 | Internal Error | Server error | Check application logs |
| 503 | Service Unavailable | Service down | Check service health |

### Database Error Codes

| Code | Meaning | Common Causes | Resolution |
|------|---------|---------------|------------|
| 23505 | Unique Violation | Duplicate key | Check existing data |
| 23503 | Foreign Key Violation | Invalid reference | Verify related data |
| 23514 | Check Violation | Constraint failure | Fix data constraints |
| 08006 | Connection Failed | Database down | Restart database |
| 08001 | Connection Error | Network issues | Check connectivity |

### Application Error Codes

| Code | Meaning | Common Causes | Resolution |
|------|---------|---------------|------------|
| AUTH_001 | Invalid credentials | Wrong password | Reset password |
| AUTH_002 | Token expired | Session timeout | Refresh token |
| AUTH_003 | Invalid token | Corrupted token | Re-authenticate |
| PROD_001 | Product not found | Invalid SKU | Verify product ID |
| PROD_002 | Insufficient stock | Low inventory | Update stock levels |
| ORDER_001 | Order not found | Invalid order ID | Check order number |
| ORDER_002 | Invalid status | Wrong status flow | Verify status transition |

---

**Important**: Always document any incident resolution steps and update runbooks based on new learnings. Regular review and practice of these procedures ensures faster resolution during actual incidents.