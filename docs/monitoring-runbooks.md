# ERPGo Monitoring Runbooks

## Overview

This document provides detailed runbooks for handling common monitoring scenarios and incidents in the ERPGo system. Each runbook includes step-by-step procedures for identifying, diagnosing, and resolving specific issues.

## Table of Contents

1. [Emergency Response Runbooks](#emergency-response-runbooks)
2. [Performance Issues](#performance-issues)
3. [Database Issues](#database-issues)
4. [Security Incidents](#security-incidents)
5. [Infrastructure Issues](#infrastructure-issues)
6. [Business Logic Issues](#business-logic-issues)
7. [Preventive Maintenance](#preventive-maintenance)

---

## Emergency Response Runbooks

### Runbook: Application Down

**Severity**: Critical
**Response Time**: < 5 minutes
**Alert**: `ApplicationDown`

#### Symptoms
- ERPGo API unreachable
- All HTTP requests failing
- Health check endpoints returning 503

#### Immediate Actions (5 minutes)

1. **Verify Alert Validity**
   ```bash
   # Check if application is actually down
   curl -f http://localhost:8080/health || echo "Application is down"

   # Check container status
   docker ps | grep erpgo-api
   ```

2. **Check Application Logs**
   ```bash
   # Check recent application logs for errors
   docker logs --tail=100 erpgo-api-container

   # Or with Promtail/Loki
   # Query in Grafana: {job="erpgo-app"} |= "panic" or |= "fatal"
   ```

3. **Verify Infrastructure**
   ```bash
   # Check database connectivity
   docker exec erpgo-api-container ping postgres

   # Check Redis connectivity
   docker exec erpgo-api-container ping redis
   ```

#### Diagnosis (15 minutes)

1. **Container Issues**
   ```bash
   # Restart container if needed
   docker restart erpgo-api-container

   # Check resource usage
   docker stats erpgo-api-container
   ```

2. **Database Connection Issues**
   ```bash
   # Check PostgreSQL status
   docker exec postgres pg_isready

   # Check connection limits
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT count(*) FROM pg_stat_activity;"
   ```

3. **Resource Exhaustion**
   ```bash
   # Check memory usage
   free -h

   # Check disk space
   df -h

   # Check CPU usage
   top -p $(pgrep erpgo)
   ```

#### Resolution Steps

1. **Quick Fix - Restart Services**
   ```bash
   # Restart in proper order
   docker restart postgres redis
   sleep 10
   docker restart erpgo-api-container
   ```

2. **If Restart Fails - Check Configuration**
   ```bash
   # Validate configuration files
   docker exec erpgo-api-container ./erpgo --config-check

   # Check environment variables
   docker exec erpgo-api-container env | grep -E "(DATABASE|REDIS)"
   ```

3. **Manual Recovery**
   ```bash
   # Scale up if using orchestrator
   kubectl scale deployment erpgo-api --replicas=2

   # Or manually start new instance
   docker run -d --name erpgo-api-recovery erpgo:latest
   ```

#### Verification
- Health check returns 200
- Dashboards show metrics flowing
- Sample API calls succeed

#### Escalation
- If unresolved after 30 minutes, escalate to DevOps team
- Consider failover to backup instance if available

---

### Runbook: High Error Rate

**Severity**: Warning/Critical
**Response Time**: < 10 minutes
**Alerts**: `HighErrorRate`, `CriticalErrorRate`

#### Symptoms
- 5xx response rate > 5% (warning) or > 10% (critical)
- Users reporting application failures
- Increased application error logs

#### Immediate Actions

1. **Verify Error Metrics**
   ```bash
   # Check current error rate in Prometheus
   curl "http://localhost:9090/api/v1/query?query=rate(erpgo_http_requests_total{status_code=~\"5..\"}[5m])/rate(erpgo_http_requests_total[5m])*100"
   ```

2. **Identify Error Pattern**
   ```sql
   -- Check database for recent errors
   SELECT level, COUNT(*), message
   FROM application_logs
   WHERE timestamp >= NOW() - INTERVAL '10 minutes'
   AND level = 'ERROR'
   GROUP BY level, message
   ORDER BY COUNT(*) DESC;
   ```

3. **Check Logs for Root Cause**
   ```logql
   # Query in Grafana Loki
   {job="erpgo-app", level="error"}[5m]
   ```

#### Diagnosis

1. **Database Connection Issues**
   ```bash
   # Check database health
   docker exec postgres pg_isready
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT count(*) FROM pg_stat_activity WHERE state = 'active';"
   ```

2. **Resource Constraints**
   ```bash
   # Check memory and CPU
   docker stats erpgo-api-container --no-stream

   # Check goroutine count
   curl http://localhost:8080/metrics | grep erpgo_goroutines
   ```

3. **External Service Dependencies**
   ```bash
   # Check external API connectivity
   curl -I https://external-service.com/health
   ```

#### Resolution Steps

1. **Database Issues**
   ```bash
   # Increase connection pool if needed
   # Update configuration: DB_MAX_CONNECTIONS=50

   # Restart application after config change
   docker restart erpgo-api-container
   ```

2. **Memory Issues**
   ```bash
   # Add more memory to container
   docker stop erpgo-api-container
   docker run -d --memory=2g --name erpgo-api-container erpgo:latest
   ```

3. **External Service Failures**
   ```bash
   # Implement circuit breakers for external calls
   # Update application configuration to enable fallbacks
   ```

#### Verification
- Error rate drops below threshold
- Application functions normally
- User complaints cease

#### Prevention
- Implement proper error handling
- Add circuit breakers for external services
- Set up automated recovery procedures

---

## Performance Issues

### Runbook: High Response Time

**Severity**: Warning/Critical
**Alerts**: `HighResponseTime`, `CriticalResponseTime`

#### Symptoms
- 95th percentile response time > 1s (warning) or > 2s (critical)
- Users reporting slow application performance
- Database query timeouts

#### Immediate Actions

1. **Check Response Time Metrics**
   ```bash
   # Query Prometheus for current response times
   curl "http://localhost:9090/api/v1/query?query=histogram_quantile(0.95,rate(erpgo_http_request_duration_seconds_bucket[5m]))"
   ```

2. **Identify Slow Endpoints**
   ```sql
   -- Check slow queries in database
   SELECT query, calls, total_time, mean_time
   FROM pg_stat_statements
   WHERE mean_time > 100
   ORDER BY mean_time DESC
   LIMIT 10;
   ```

3. **Check Resource Utilization**
   ```bash
   # CPU and memory usage
   docker stats erpgo-api-container

   # Database performance
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT * FROM pg_stat_activity WHERE state = 'active' AND query_start < now() - interval '1 minute';"
   ```

#### Diagnosis

1. **Database Performance**
   ```bash
   # Check slow queries
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

   # Check indexes
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT schemaname, tablename, attname, n_distinct, correlation FROM pg_stats WHERE tablename IN ('orders', 'products', 'customers');"
   ```

2. **Application Performance**
   ```bash
   # Check goroutine leaks
   curl http://localhost:8080/metrics | grep erpgo_goroutines

   # Check memory allocation
   curl http://localhost:8080/metrics | grep erpgo_system_memory_bytes
   ```

3. **Network Issues**
   ```bash
   # Check network latency
   ping postgres
   ping redis

   # Check bandwidth usage
   ifstat
   ```

#### Resolution Steps

1. **Database Optimization**
   ```sql
   -- Create missing indexes
   CREATE INDEX CONCURRENTLY idx_orders_created_at ON orders(created_at);
   CREATE INDEX CONCURRENTLY idx_products_category_id ON products(category_id);

   -- Update statistics
   ANALYZE;
   ```

2. **Caching Optimization**
   ```bash
   # Check cache hit rate
   curl http://localhost:8080/metrics | grep erpgo_cache_hit_rate

   # Clear cache if needed
   docker exec redis redis-cli FLUSHDB
   ```

3. **Horizontal Scaling**
   ```bash
   # Scale application
   docker-compose up -d --scale api=3

   # Add load balancer if not already present
   ```

#### Verification
- Response times return to normal levels
- Database query times improve
- User experience improves

#### Prevention
- Regular database maintenance
- Performance testing
- Proper indexing strategy

---

## Database Issues

### Runbook: Database Connection Pool Exhaustion

**Severity**: Critical
**Alerts**: `DatabaseConnectionHigh`, `DatabaseConnectionCritical`

#### Symptoms
- Database connection errors
- Application timeouts
- High active connection count

#### Immediate Actions

1. **Check Connection Count**
   ```bash
   # Current connections
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT count(*) FROM pg_stat_activity;"

   # Connection by state
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT state, count(*) FROM pg_stat_activity GROUP BY state;"
   ```

2. **Identify Long-Running Queries**
   ```bash
   # Find queries running > 5 minutes
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT pid, now() - query_start AS duration, query FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '5 minutes';"
   ```

3. **Check Application Configuration**
   ```bash
   # Verify connection pool settings
   docker exec erpgo-api-container env | grep DB_MAX_CONNECTIONS
   ```

#### Diagnosis

1. **Connection Leaks**
   ```sql
   -- Look for idle connections from application
   SELECT pid, state, query_start, application_name
   FROM pg_stat_activity
   WHERE state = 'idle'
   AND application_name = 'erpgo';
   ```

2. **Long-Running Queries**
   ```sql
   -- Identify blocking queries
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
   ```

#### Resolution Steps

1. **Terminate Long-Running Queries**
   ```bash
   # Kill specific long-running queries
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT pg_terminate_backend(pid FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '10 minutes');"
   ```

2. **Increase Connection Pool**
   ```bash
   # Update application configuration
   DB_MAX_CONNECTIONS=100

   # Restart application
   docker restart erpgo-api-container
   ```

3. **Optimize Connection Usage**
   ```bash
   # Enable connection pooling in PostgreSQL
   # Add to postgresql.conf: max_connections = 200

   # Restart PostgreSQL
   docker restart postgres
   ```

#### Verification
- Connection count returns to normal
- Application connections succeed
- Performance improves

#### Prevention
- Monitor connection usage
- Implement connection timeout
- Regular connection pool review

---

## Security Incidents

### Runbook: High Authentication Failure Rate

**Severity**: Warning/Critical
**Alerts**: `HighAuthFailureRate`, `CriticalAuthFailureRate`

#### Symptoms
- High rate of failed login attempts
- Possible brute force attack
- Account lockouts increasing

#### Immediate Actions

1. **Verify Alert**
   ```bash
   # Check current auth failure rate
   curl "http://localhost:9090/api/v1/query?query=rate(erpgo_auth_failures_total[5m])"
   ```

2. **Check Source IPs**
   ```sql
   -- Check failed login attempts by IP
   SELECT ip_address, COUNT(*) as failure_count
   FROM auth_attempts
   WHERE success = false
   AND timestamp >= NOW() - INTERVAL '1 hour'
   GROUP BY ip_address
   HAVING COUNT(*) > 10
   ORDER BY failure_count DESC;
   ```

3. **Review Security Logs**
   ```logql
   # Query in Grafana Loki
   {job="erpgo-app", component="auth"} |= "failed" | logfmt
   ```

#### Diagnosis

1. **Identify Attack Patterns**
   ```bash
   # Check for distributed attack
   docker logs erpgo-api-container | grep "auth.*failed" | awk '{print $1}' | sort | uniq -c | sort -nr

   # Check for specific user targeting
   docker logs erpgo-api-container | grep "auth.*failed.*admin" | wc -l
   ```

2. **Check Rate Limiting**
   ```bash
   # Verify rate limiting is working
   curl "http://localhost:9090/api/v1/query?query=rate(erpgo_rate_limit_violations_total[5m])"
   ```

#### Resolution Steps

1. **Block Malicious IPs**
   ```bash
   # Add IPs to firewall blocklist
   iptables -A INPUT -s MALICIOUS_IP -j DROP

   # Or block at application level
   # Update application configuration to block IPs
   ```

2. **Enable Additional Security Measures**
   ```bash
   # Enable CAPTCHA for login attempts
   # Update application: LOGIN_CAPTCHA_ENABLED=true

   # Enable account lockout
   # Update application: ACCOUNT_LOCKOUT_THRESHOLD=5
   ```

3. **Increase Monitoring**
   ```bash
   # Add more detailed logging
   # Update application: SECURITY_LOG_LEVEL=debug

   # Set up immediate notifications for security events
   ```

#### Verification
- Attack blocked or mitigated
- Legitimate users can still login
- No further security alerts

#### Prevention
- Implement multi-factor authentication
- Regular security audits
- IP reputation checking

---

## Infrastructure Issues

### Runbook: High Memory Usage

**Severity**: Warning/Critical
**Alerts**: `HighMemoryUsage`, `CriticalMemoryUsage`

#### Symptoms
- Application using > 80% (warning) or > 95% (critical) of memory
- Out of memory errors
- Application restarts

#### Immediate Actions

1. **Check Memory Usage**
   ```bash
   # System memory
   free -h

   # Container memory
   docker stats erpgo-api-container --no-stream

   # Application memory metrics
   curl http://localhost:8080/metrics | grep erpgo_system_memory_bytes
   ```

2. **Check for Memory Leaks**
   ```bash
   # Check goroutine count
   curl http://localhost:8080/metrics | grep erpgo_goroutines

   # Check heap allocation
   curl http://localhost:8080/metrics | grep heap_alloc
   ```

#### Diagnosis

1. **Memory Profiling**
   ```bash
   # Enable pprof profiling
   curl http://localhost:8080/debug/pprof/heap > heap.prof

   # Analyze with go tool
   go tool pprof heap.prof
   ```

2. **Check Database Memory Usage**
   ```bash
   # PostgreSQL memory usage
   docker exec postgres psql -U erpgo_user -d erpgo_db -c "SELECT name, setting FROM pg_settings WHERE name LIKE '%memory%';"
   ```

#### Resolution Steps

1. **Restart Application**
   ```bash
   # Graceful restart to clear memory
   docker restart erpgo-api-container
   ```

2. **Increase Memory Limits**
   ```bash
   # Add more memory to container
   docker stop erpgo-api-container
   docker run -d --memory=4g --name erpgo-api-container erpgo:latest
   ```

3. **Optimize Memory Usage**
   ```bash
   # Update Go garbage collection settings
   GODEBUG=gctrace=1

   # Adjust application memory pools
   MEMORY_POOL_SIZE=1000
   ```

#### Verification
- Memory usage returns to normal
- Application performance stable
- No more memory alerts

#### Prevention
- Regular memory profiling
- Memory leak detection
- Proper resource allocation

---

## Business Logic Issues

### Runbook: Low Order Rate

**Severity**: Warning
**Alerts**: `LowOrderRate`

#### Symptoms
- Order creation rate < 0.1 per hour for 15 minutes
- Revenue drop
- Possible checkout issues

#### Immediate Actions

1. **Verify Alert**
   ```bash
   # Check current order rate
   curl "http://localhost:9090/api/v1/query?query=rate(erpgo_orders_created_total[1h])"
   ```

2. **Check Order Creation Endpoints**
   ```bash
   # Test order creation manually
   curl -X POST http://localhost:8080/api/v1/orders \
     -H "Content-Type: application/json" \
     -d '{"customer_id": "test", "items": []}'
   ```

3. **Check Payment Processing**
   ```bash
   # Check payment gateway status
   curl -I https://payment-gateway.com/health

   # Check payment logs
   docker logs erpgo-api-container | grep -i payment
   ```

#### Diagnosis

1. **Checkout Flow Issues**
   ```sql
   -- Check recent order failures
   SELECT error_type, COUNT(*)
   FROM order_attempts
   WHERE timestamp >= NOW() - INTERVAL '1 hour'
   AND success = false
   GROUP BY error_type;
   ```

2. **Inventory Issues**
   ```sql
   -- Check for out-of-stock items
   SELECT product_id, stock_level
   FROM inventory
   WHERE stock_level = 0;
   ```

#### Resolution Steps

1. **Fix Checkout Issues**
   ```bash
   # Restart order service if needed
   docker restart erpgo-api-container

   # Update payment gateway configuration
   PAYMENT_GATEWAY_URL=https://backup-gateway.com
   ```

2. **Manual Order Processing**
   ```bash
   # Enable manual order processing
   MANUAL_ORDER_PROCESSING=true

   # Notify customer service team
   ```

#### Verification
- Order creation rate returns to normal
- Revenue generation resumes
- Customer complaints addressed

#### Prevention
- Monitor payment gateway health
- Regular checkout flow testing
- Backup payment providers

---

## Preventive Maintenance

### Daily Health Checks

1. **Application Health**
   ```bash
   curl -f http://localhost:8080/health
   ```

2. **Database Health**
   ```bash
   docker exec postgres pg_isready
   ```

3. **Cache Health**
   ```bash
   docker exec redis redis-cli ping
   ```

4. **Monitor Status**
   ```bash
   curl http://localhost:9090/-/healthy
   curl http://localhost:3000/api/health
   ```

### Weekly Performance Review

1. **Response Time Trends**
   ```bash
   # Check weekly response time trends
   curl "http://localhost:9090/api/v1/query_range?query=histogram_quantile(0.95,rate(erpgo_http_request_duration_seconds_bucket[5m]))&start=$(date -d '1 week ago' +%s)&end=$(date +%s)&step=300"
   ```

2. **Error Rate Analysis**
   ```bash
   # Analyze error patterns
   # Use Grafana to create weekly error reports
   ```

3. **Resource Utilization**
   ```bash
   # Review memory and CPU trends
   docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}"
   ```

### Monthly Maintenance

1. **Database Maintenance**
   ```sql
   -- Update statistics
   ANALYZE;

   -- Reindex if needed
   REINDEX DATABASE erpgo_db;

   -- Clean up old logs
   DELETE FROM application_logs WHERE timestamp < NOW() - INTERVAL '30 days';
   ```

2. **Log Cleanup**
   ```bash
   # Rotate application logs
   logrotate -f /etc/logrotate.d/erpgo

   # Clean old monitoring data
   # Configure Prometheus retention policies
   ```

3. **Security Updates**
   ```bash
   # Check for security updates
   docker-compose pull

   # Update containers safely
   docker-compose up -d --no-deps postgres
   ```

### Automated Recovery Scripts

```bash
#!/bin/bash
# auto-recovery.sh

# Function to restart services if down
restart_if_down() {
    local service=$1
    if ! curl -f http://localhost:${2}/health > /dev/null 2>&1; then
        echo "Service $service is down, restarting..."
        docker restart $service
    fi
}

# Function to clear cache if hit rate low
clear_cache_if_needed() {
    local hit_rate=$(curl -s http://localhost:9090/api/v1/query?query=erpgo_cache_hit_rate | jq -r '.data.result[0].value[1]')
    if (( $(echo "$hit_rate < 0.5" | bc -l) )); then
        echo "Cache hit rate low ($hit_rate), clearing cache..."
        docker exec redis redis-cli FLUSHDB
    fi
}

# Execute recovery procedures
restart_if_down "erpgo-api-container" "8080"
clear_cache_if_needed

echo "Auto-recovery completed at $(date)"
```

## Contact Information

- **DevOps Team**: devops@erpgo.com
- **Development Team**: dev@erpgo.com
- **Security Team**: security@erpgo.com
- **On-Call Rotation**: +1-555-ERP-GO1

## Escalation Matrix

| Severity | Response Time | Escalation | Contact |
|----------|---------------|------------|---------|
| Critical | 5 minutes | 30 minutes | DevOps Lead |
| High | 15 minutes | 1 hour | Development Lead |
| Medium | 1 hour | 4 hours | Team Lead |
| Low | 4 hours | 1 day | Team Member |

This runbook should be reviewed and updated regularly to ensure it remains current with the evolving ERPGo infrastructure and procedures.