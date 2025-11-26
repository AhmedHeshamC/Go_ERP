# ERPGo Operational Runbooks

This document provides runbooks for common operational issues and alerts. Each runbook is linked to specific alert rules and provides step-by-step guidance for investigation and resolution.

## Quick Reference

| Alert | Severity | Runbook | Alert Rule |
|-------|----------|---------|------------|
| High Error Rate | Critical | [Link](#high-error-rate) | `HighErrorRate` |
| Database Down | Critical | [Link](#database-connectivity-issues) | `PostgreSQLDown` |
| High Latency | Warning | [Link](#high-latency-incidents) | `HighResponseTime` |
| Auth Failures | Warning | [Link](#authentication-failures) | `HighFailedLoginRate` |
| Connection Pool | Critical | [Link](#database-connection-pool-exhaustion) | `DatabaseConnectionPoolExhausted` |
| Low Cache Hit | Warning | [Link](#low-cache-hit-rate) | `LowCacheHitRate` |
| Deployment | N/A | [Link](#deployment-procedures) | N/A |

## Table of Contents

1. [High Error Rate](#high-error-rate)
2. [Database Connectivity Issues](#database-connectivity-issues)
3. [High Latency Incidents](#high-latency-incidents)
4. [Authentication Failures](#authentication-failures)
5. [Database Connection Pool Exhaustion](#database-connection-pool-exhaustion)
6. [Low Cache Hit Rate](#low-cache-hit-rate)
7. [Deployment Procedures](#deployment-procedures)
8. [Emergency Rollback](#emergency-rollback)
9. [General Troubleshooting](#general-troubleshooting)

---

## High Error Rate

**Alert Name:** `HighErrorRate`

**Severity:** Critical

**Description:** The application is experiencing an error rate above 5% over the last 5 minutes.

### Symptoms
- HTTP 5xx errors are being returned to clients
- Error rate metric shows > 5%
- Users may be experiencing service degradation

### Investigation Steps

1. **Check the error logs:**
   ```bash
   kubectl logs -l app=erpgo-api --tail=100 | grep ERROR
   ```

2. **Identify the most common error types:**
   ```bash
   # Check Grafana dashboard for error breakdown by endpoint
   # Or query Prometheus:
   # rate(http_requests_total{status=~"5.."}[5m]) by (endpoint, status)
   ```

3. **Check recent deployments:**
   ```bash
   kubectl rollout history deployment/erpgo-api
   ```

4. **Check database connectivity:**
   ```bash
   kubectl exec -it deployment/erpgo-api -- psql -h $DB_HOST -U $DB_USER -c "SELECT 1"
   ```

5. **Check external service dependencies:**
   - Verify Redis is accessible
   - Check any third-party API status pages

### Resolution Steps

1. **If caused by recent deployment:**
   ```bash
   kubectl rollout undo deployment/erpgo-api
   ```

2. **If database connection issues:**
   - Check database connection pool metrics
   - Verify database is not overloaded
   - Consider scaling database or application

3. **If external service issues:**
   - Enable circuit breakers if available
   - Check service status pages
   - Contact vendor support if needed

4. **If application bug:**
   - Identify the problematic code path from logs
   - Create hotfix if critical
   - Otherwise, create bug ticket and monitor

### Prevention
- Implement comprehensive error handling
- Add circuit breakers for external dependencies
- Increase test coverage for critical paths
- Set up canary deployments

**Related Alerts:**
- Alert Rule: `HighErrorRate` in `configs/prometheus/alert_rules.yml`
- Severity: Critical
- Threshold: Error rate > 5% for 5 minutes

**Related Documentation:**
- [Error Handling Design](../ARCHITECTURE_OVERVIEW.md#error-handling)
- [Monitoring Guide](../MONITORING.md)

---

## Database Connectivity Issues

**Alert Name:** `PostgreSQLDown`

**Severity:** Critical

**Description:** PostgreSQL database is not responding or unreachable.

**Related Alerts:**
- Alert Rule: `PostgreSQLDown` in `configs/prometheus/alert_rules.yml`
- Severity: Critical
- Threshold: Database down for > 1 minute

### Symptoms
- All database operations failing
- Application returning 500 errors
- Health check endpoints failing (`/health/ready` returns 503)
- Database metrics not being collected
- Connection timeout errors in logs

### Investigation Steps

1. **Check database pod/container status:**
   ```bash
   # Kubernetes
   kubectl get pods -l app=postgres
   kubectl describe pod <postgres-pod-name>
   
   # Docker Compose
   docker-compose ps postgres
   docker-compose logs postgres --tail=100
   ```

2. **Check database logs for errors:**
   ```bash
   # Kubernetes
   kubectl logs -l app=postgres --tail=200
   
   # Docker Compose
   docker-compose logs postgres --tail=200
   
   # Look for:
   # - Out of memory errors
   # - Disk full errors
   # - Corruption messages
   # - Connection limit exceeded
   ```

3. **Verify network connectivity:**
   ```bash
   # From application pod
   kubectl exec -it deployment/erpgo-api -- nc -zv postgres-service 5432
   
   # Check DNS resolution
   kubectl exec -it deployment/erpgo-api -- nslookup postgres-service
   ```

4. **Check database disk space:**
   ```bash
   kubectl exec -it <postgres-pod> -- df -h /var/lib/postgresql/data
   ```

5. **Check database process and connections:**
   ```bash
   kubectl exec -it <postgres-pod> -- ps aux | grep postgres
   
   # Check active connections
   kubectl exec -it <postgres-pod> -- psql -U $DB_USER -c "SELECT count(*) FROM pg_stat_activity;"
   ```

6. **Review recent changes:**
   - Check recent deployments
   - Review configuration changes
   - Check for infrastructure changes

### Resolution Steps

1. **If pod/container is not running:**
   ```bash
   # Check pod events
   kubectl describe pod <postgres-pod>
   
   # Check for resource constraints
   kubectl top pod <postgres-pod>
   
   # Restart if needed
   kubectl delete pod <postgres-pod>  # Will be recreated by StatefulSet
   ```

2. **If disk is full:**
   ```bash
   # Check WAL files
   kubectl exec -it <postgres-pod> -- ls -lh /var/lib/postgresql/data/pg_wal/
   
   # Archive old WAL files (if archiving is configured)
   kubectl exec -it <postgres-pod> -- pg_archivecleanup /var/lib/postgresql/data/pg_wal <oldest-wal-to-keep>
   
   # Or expand disk volume (requires downtime)
   # Follow cloud provider's volume expansion procedure
   ```

3. **If connection limit exceeded:**
   ```sql
   -- Check current connections
   SELECT count(*), state FROM pg_stat_activity GROUP BY state;
   
   -- Kill idle connections (if safe)
   SELECT pg_terminate_backend(pid)
   FROM pg_stat_activity
   WHERE state = 'idle' AND state_change < now() - interval '10 minutes';
   
   -- Increase max_connections (requires restart)
   ALTER SYSTEM SET max_connections = 200;
   -- Then restart database
   ```

4. **If database is corrupted:**
   ```bash
   # Stop application to prevent further writes
   kubectl scale deployment/erpgo-api --replicas=0
   
   # Attempt recovery
   kubectl exec -it <postgres-pod> -- pg_resetwal /var/lib/postgresql/data
   
   # If recovery fails, restore from backup
   ./scripts/backup/disaster-recovery.sh restore <backup-timestamp>
   ```

5. **If network issues:**
   ```bash
   # Check service endpoints
   kubectl get endpoints postgres-service
   
   # Verify service configuration
   kubectl get service postgres-service -o yaml
   
   # Check network policies
   kubectl get networkpolicies
   ```

### Prevention
- Implement automated backups (every 6 hours)
- Monitor disk space proactively (alert at 80%)
- Set up database replication for high availability
- Regular backup restore testing (monthly)
- Implement proper resource limits and requests
- Use connection pooling in application
- Monitor connection count trends

**Related Documentation:**
- [Backup Procedures](./BACKUP_RUNBOOK.md)
- [Disaster Recovery](./DISASTER_RECOVERY_PROCEDURES.md)
- [Database Configuration](../DATABASE_SCHEMA.md)

---

## High Latency Incidents

**Alert Name:** `HighResponseTime`

**Severity:** Warning

**Description:** 95th percentile response time exceeds 1 second, indicating performance degradation.

**Related Alerts:**
- Alert Rule: `HighResponseTime` in `configs/prometheus/alert_rules.yml`
- Severity: Warning
- Threshold: p95 > 1 second for 5 minutes

### Symptoms
- Slow page loads for users
- Increased timeout errors
- High response time metrics
- User complaints about performance
- Increased queue depths

### Investigation Steps

1. **Identify slow endpoints:**
   ```bash
   # Check Grafana dashboard: "ERPGo System Performance"
   # Or query Prometheus directly:
   
   # Top 5 slowest endpoints
   topk(5, histogram_quantile(0.95, 
     rate(http_request_duration_seconds_bucket[5m])) by (endpoint))
   ```

2. **Check database query performance:**
   ```sql
   -- Connect to database
   kubectl exec -it <postgres-pod> -- psql -U $DB_USER -d erpgo
   
   -- Top 10 slowest queries
   SELECT 
     query,
     mean_exec_time,
     calls,
     total_exec_time
   FROM pg_stat_statements
   ORDER BY mean_exec_time DESC
   LIMIT 10;
   
   -- Check for long-running queries
   SELECT 
     pid,
     now() - pg_stat_activity.query_start AS duration,
     query,
     state
   FROM pg_stat_activity
   WHERE state != 'idle'
   ORDER BY duration DESC
   LIMIT 10;
   ```

3. **Check cache hit rate:**
   ```bash
   # Query Prometheus
   # cache_hit_rate should be > 70%
   
   # Check Redis performance
   kubectl exec -it <redis-pod> -- redis-cli INFO stats
   kubectl exec -it <redis-pod> -- redis-cli --latency
   ```

4. **Check system resources:**
   ```bash
   # Pod resource usage
   kubectl top pods -l app=erpgo-api
   
   # Node resource usage
   kubectl top nodes
   
   # Check for CPU throttling
   kubectl describe pod <erpgo-api-pod> | grep -A 5 "cpu"
   ```

5. **Check for external service delays:**
   ```bash
   # Review distributed traces in Jaeger
   # Look for spans with high duration
   
   # Check third-party service status
   # - Payment gateway status page
   # - Email service status page
   # - Any other external dependencies
   ```

6. **Analyze request patterns:**
   ```bash
   # Check for traffic spikes
   # Query Prometheus: rate(http_requests_total[5m])
   
   # Check for unusual request patterns
   kubectl logs -l app=erpgo-api --tail=1000 | grep -E "POST|PUT|DELETE" | awk '{print $7}' | sort | uniq -c | sort -rn
   ```

### Resolution Steps

1. **If database queries are slow:**
   ```sql
   -- Add missing indexes
   -- Example: If user lookups by email are slow
   CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
   
   -- Analyze query plans
   EXPLAIN ANALYZE <slow-query>;
   
   -- Update statistics
   ANALYZE;
   
   -- Consider query caching for frequently accessed data
   ```

2. **If cache hit rate is low (<70%):**
   ```bash
   # Check cache memory
   kubectl exec -it <redis-pod> -- redis-cli INFO memory
   
   # Increase cache TTL for stable data (update config)
   kubectl set env deployment/erpgo-api CACHE_TTL_PERMISSIONS=600
   
   # Warm up cache after deployment
   ./scripts/warm-cache.sh
   
   # Increase Redis memory if needed
   kubectl set resources deployment/redis --limits=memory=4Gi
   ```

3. **If CPU/memory constrained:**
   ```bash
   # Scale horizontally (preferred)
   kubectl scale deployment/erpgo-api --replicas=5
   
   # Or scale vertically
   kubectl set resources deployment/erpgo-api \
     --limits=cpu=2,memory=4Gi \
     --requests=cpu=1,memory=2Gi
   
   # Monitor after scaling
   watch kubectl top pods -l app=erpgo-api
   ```

4. **If external service delays:**
   ```bash
   # Implement/adjust timeouts
   kubectl set env deployment/erpgo-api \
     HTTP_CLIENT_TIMEOUT=10s \
     DB_QUERY_TIMEOUT=5s
   
   # Enable circuit breakers (if implemented)
   kubectl set env deployment/erpgo-api CIRCUIT_BREAKER_ENABLED=true
   
   # Consider async processing for non-critical operations
   ```

5. **If N+1 query problem:**
   ```bash
   # Review application logs for query patterns
   kubectl logs -l app=erpgo-api | grep "SELECT" | wc -l
   
   # Fix in code: Use JOINs or eager loading
   # Deploy fix as hotfix if critical
   ```

### Prevention
- Regular performance testing (weekly load tests)
- Implement comprehensive caching strategy
- Optimize database queries proactively
- Use CDN for static assets
- Implement request timeouts (default: 30s)
- Monitor query performance trends
- Set up performance budgets
- Regular code reviews focusing on performance

**Related Documentation:**
- [Performance Optimization Guide](../ARCHITECTURE_OVERVIEW.md#performance-optimization)
- [Caching Strategy](../ARCHITECTURE_OVERVIEW.md#caching-strategy)
- [Load Testing](../../tests/load/README.md)

---

## Authentication Failures

**Alert Name:** `HighFailedLoginRate`

**Severity:** Warning

**Description:** Failed login attempts exceed 10 per minute, indicating potential security issues or user problems.

**Related Alerts:**
- Alert Rule: `HighFailedLoginRate` in `configs/prometheus/alert_rules.yml`
- Severity: Warning
- Threshold: > 10 failed attempts per minute for 5 minutes

### Symptoms
- High rate of authentication failures
- Potential brute force attack
- Account lockouts
- Security alerts
- User complaints about being locked out

### Investigation Steps

1. **Check failed login sources and patterns:**
   ```bash
   # Analyze failed login attempts by IP
   kubectl logs -l app=erpgo-api --tail=5000 | \
     grep "authentication failed" | \
     awk '{print $5}' | \
     sort | uniq -c | sort -rn | head -20
   
   # Check for distributed attack (many IPs)
   kubectl logs -l app=erpgo-api --tail=5000 | \
     grep "authentication failed" | \
     awk '{print $6}' | \
     sort -u | wc -l
   ```

2. **Check rate limiter metrics:**
   ```bash
   # Query Prometheus
   # auth_login_failure_total
   # auth_rate_limit_exceeded_total
   # auth_active_locked_accounts
   
   # Check Redis for rate limit data
   kubectl exec -it <redis-pod> -- redis-cli KEYS "ratelimit:*" | wc -l
   ```

3. **Identify targeted accounts:**
   ```bash
   # Check audit logs for targeted usernames
   kubectl logs -l app=erpgo-api --tail=5000 | \
     grep "failed login" | \
     grep -o "username=[^ ]*" | \
     sort | uniq -c | sort -rn | head -20
   
   # Query audit log database
   kubectl exec -it <postgres-pod> -- psql -U $DB_USER -d erpgo -c \
     "SELECT user_id, COUNT(*) as attempts 
      FROM audit_logs 
      WHERE event_type = 'login_failed' 
        AND timestamp > NOW() - INTERVAL '1 hour'
      GROUP BY user_id 
      ORDER BY attempts DESC 
      LIMIT 20;"
   ```

4. **Analyze attack patterns:**
   ```bash
   # Check for credential stuffing (many usernames, few IPs)
   # Check for brute force (few usernames, many attempts)
   # Check for distributed attack (many IPs, many usernames)
   
   # Time-based analysis
   kubectl logs -l app=erpgo-api --tail=10000 | \
     grep "authentication failed" | \
     awk '{print $1, $2}' | \
     cut -d: -f1 | \
     sort | uniq -c
   ```

5. **Check for legitimate issues:**
   ```bash
   # Check for recent password policy changes
   kubectl logs -l app=erpgo-api | grep "password policy"
   
   # Check JWT validation errors
   kubectl logs -l app=erpgo-api | grep "token validation failed"
   
   # Check for clock skew issues
   kubectl exec -it deployment/erpgo-api -- date
   ```

### Resolution Steps

1. **If brute force attack detected (single IP, many attempts):**
   ```bash
   # Block attacking IP at load balancer/firewall
   # AWS ALB example:
   aws wafv2 update-ip-set \
     --name blocked-ips \
     --id <ip-set-id> \
     --addresses <attacking-ip>/32
   
   # Or update network policy
   kubectl apply -f - <<EOF
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: block-attacker
   spec:
     podSelector:
       matchLabels:
         app: erpgo-api
     policyTypes:
     - Ingress
     ingress:
     - from:
       - ipBlock:
           cidr: 0.0.0.0/0
           except:
           - <attacking-ip>/32
   EOF
   ```

2. **If credential stuffing attack (many usernames, distributed IPs):**
   ```bash
   # Force password reset for affected accounts
   kubectl exec -it <postgres-pod> -- psql -U $DB_USER -d erpgo -c \
     "UPDATE users 
      SET password_reset_required = true 
      WHERE id IN (
        SELECT DISTINCT user_id 
        FROM audit_logs 
        WHERE event_type = 'login_failed' 
          AND timestamp > NOW() - INTERVAL '1 hour'
        GROUP BY user_id 
        HAVING COUNT(*) > 5
      );"
   
   # Send notification emails
   ./scripts/notify-password-reset.sh
   
   # Enable MFA requirement
   kubectl set env deployment/erpgo-api REQUIRE_MFA=true
   ```

3. **If distributed attack (many IPs, many attempts):**
   ```bash
   # Tighten rate limits temporarily
   kubectl set env deployment/erpgo-api \
     AUTH_MAX_ATTEMPTS=3 \
     AUTH_LOCKOUT_DURATION=30m \
     AUTH_RATE_LIMIT_WINDOW=10m
   
   # Enable CAPTCHA for login
   kubectl set env deployment/erpgo-api CAPTCHA_ENABLED=true
   
   # Consider enabling WAF rules
   ```

4. **If legitimate user issues:**
   ```bash
   # Unlock specific accounts
   kubectl exec -it <redis-pod> -- redis-cli DEL "ratelimit:user:<user-id>"
   
   # Check authentication service health
   kubectl logs -l app=erpgo-api | grep "auth service"
   
   # Verify JWT secret is correct
   kubectl get secret erpgo-secrets -o jsonpath='{.data.JWT_SECRET}' | base64 -d | wc -c
   # Should be >= 32 bytes
   ```

5. **Immediate mitigation for ongoing attack:**
   ```bash
   # Enable maintenance mode for login endpoint only
   kubectl set env deployment/erpgo-api MAINTENANCE_MODE_AUTH=true
   
   # Or temporarily disable new logins
   kubectl set env deployment/erpgo-api ALLOW_NEW_LOGINS=false
   
   # Monitor attack subsiding
   watch 'kubectl logs -l app=erpgo-api --tail=100 | grep "authentication failed" | wc -l'
   ```

### Prevention
- Implement CAPTCHA after 3 failed attempts
- Require MFA for all accounts (especially admin)
- Implement IP-based rate limiting (5 attempts per 15 min)
- Monitor for credential stuffing patterns
- Regular security awareness training
- Implement account lockout (5 failures = 15 min lockout)
- Use breach detection services (HaveIBeenPwned API)
- Implement progressive delays on failed attempts

**Related Documentation:**
- [Authentication Security](../AUTHENTICATION.md)
- [Security Best Practices](../SECURITY_BEST_PRACTICES.md)
- [Rate Limiting Configuration](../ARCHITECTURE_OVERVIEW.md#rate-limiting)

---

## Database Connection Pool Exhaustion

**Alert Name:** `DatabaseConnectionPoolExhausted`

**Severity:** Critical

**Description:** Database connection pool utilization is above 90%.

### Symptoms
- Slow API responses
- Timeout errors
- "Too many connections" errors in logs
- Connection pool metrics showing high utilization

### Investigation Steps

1. **Check current pool statistics:**
   ```bash
   # View Grafana dashboard: Database Performance
   # Or query Prometheus:
   # db_pool_utilization_percent
   # db_pool_acquired_connections
   # db_pool_idle_connections
   ```

2. **Identify long-running queries:**
   ```sql
   SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
   FROM pg_stat_activity
   WHERE state != 'idle'
   ORDER BY duration DESC
   LIMIT 10;
   ```

3. **Check for connection leaks:**
   ```bash
   # Review application logs for unclosed connections
   kubectl logs -l app=erpgo-api | grep "connection leak"
   ```

4. **Check application instance count:**
   ```bash
   kubectl get pods -l app=erpgo-api
   ```

### Resolution Steps

1. **Immediate: Kill long-running queries (if safe):**
   ```sql
   SELECT pg_terminate_backend(pid)
   FROM pg_stat_activity
   WHERE state != 'idle' AND now() - pg_stat_activity.query_start > interval '5 minutes';
   ```

2. **Scale up connection pool:**
   ```bash
   # Update environment variable
   kubectl set env deployment/erpgo-api DB_MAX_CONNECTIONS=150
   ```

3. **Scale application horizontally:**
   ```bash
   kubectl scale deployment/erpgo-api --replicas=5
   ```

4. **Optimize queries:**
   - Review slow query log
   - Add missing indexes
   - Optimize N+1 queries

### Prevention
- Monitor connection pool metrics proactively
- Implement connection timeouts
- Use connection pooling best practices
- Regular query performance reviews
- Implement read replicas for read-heavy workloads

---

## Database Down

**Alert Name:** `PostgreSQLDown`

**Severity:** Critical

**Description:** PostgreSQL database is not responding.

### Symptoms
- All database operations failing
- Application returning 500 errors
- Health check endpoints failing
- Database metrics not being collected

### Investigation Steps

1. **Check database pod status:**
   ```bash
   kubectl get pods -l app=postgres
   kubectl describe pod <postgres-pod-name>
   ```

2. **Check database logs:**
   ```bash
   kubectl logs -l app=postgres --tail=100
   ```

3. **Check database disk space:**
   ```bash
   kubectl exec -it <postgres-pod> -- df -h
   ```

4. **Check database process:**
   ```bash
   kubectl exec -it <postgres-pod> -- ps aux | grep postgres
   ```

### Resolution Steps

1. **If pod is not running:**
   ```bash
   kubectl get pod <postgres-pod> -o yaml
   # Check for resource constraints, image pull errors, etc.
   ```

2. **If disk is full:**
   ```bash
   # Clean up old WAL files (if safe)
   kubectl exec -it <postgres-pod> -- pg_archivecleanup /var/lib/postgresql/data/pg_wal
   # Or expand disk volume
   ```

3. **If database is corrupted:**
   ```bash
   # Restore from latest backup
   ./scripts/restore-database.sh <backup-timestamp>
   ```

4. **If configuration issue:**
   ```bash
   # Verify postgresql.conf settings
   kubectl exec -it <postgres-pod> -- cat /var/lib/postgresql/data/postgresql.conf
   ```

### Prevention
- Implement automated backups
- Monitor disk space proactively
- Set up database replication
- Regular backup restore testing
- Implement proper resource limits

---

## High Latency

**Alert Name:** `HighResponseTime`

**Severity:** Warning

**Description:** 95th percentile response time exceeds 1 second.

### Symptoms
- Slow page loads for users
- Increased timeout errors
- High response time metrics
- User complaints about performance

### Investigation Steps

1. **Identify slow endpoints:**
   ```bash
   # Check Grafana dashboard: API Performance
   # Or query Prometheus:
   # histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) by (endpoint)
   ```

2. **Check database query performance:**
   ```sql
   SELECT query, mean_exec_time, calls
   FROM pg_stat_statements
   ORDER BY mean_exec_time DESC
   LIMIT 10;
   ```

3. **Check cache hit rate:**
   ```bash
   # Query Prometheus:
   # cache_hit_rate
   ```

4. **Check system resources:**
   ```bash
   kubectl top pods -l app=erpgo-api
   kubectl top nodes
   ```

5. **Check for external service delays:**
   - Review distributed traces in Jaeger
   - Check third-party service status

### Resolution Steps

1. **If database queries are slow:**
   - Add missing indexes
   - Optimize query plans
   - Consider query caching

2. **If cache hit rate is low:**
   - Increase cache TTL for stable data
   - Warm up cache after deployment
   - Increase cache memory allocation

3. **If CPU/memory constrained:**
   ```bash
   # Scale horizontally
   kubectl scale deployment/erpgo-api --replicas=5
   
   # Or scale vertically
   kubectl set resources deployment/erpgo-api --limits=cpu=2,memory=4Gi
   ```

4. **If external service delays:**
   - Implement timeouts
   - Add circuit breakers
   - Consider async processing

### Prevention
- Regular performance testing
- Implement caching strategy
- Optimize database queries
- Use CDN for static assets
- Implement request timeouts

---

## High Authentication Failures

**Alert Name:** `HighFailedLoginRate`

**Severity:** Warning

**Description:** Failed login attempts exceed 10 per minute.

### Symptoms
- High rate of authentication failures
- Potential brute force attack
- Account lockouts
- Security alerts

### Investigation Steps

1. **Check failed login sources:**
   ```bash
   # Query logs for failed login attempts
   kubectl logs -l app=erpgo-api | grep "authentication failed" | awk '{print $5}' | sort | uniq -c | sort -rn
   ```

2. **Check rate limiter metrics:**
   ```bash
   # Query Prometheus:
   # auth_login_failure_total
   # auth_rate_limit_exceeded_total
   # auth_active_locked_accounts
   ```

3. **Identify targeted accounts:**
   ```bash
   # Check audit logs
   kubectl logs -l app=erpgo-api | grep "failed login" | grep -o "user=[^ ]*" | sort | uniq -c | sort -rn
   ```

4. **Check for distributed attack:**
   ```bash
   # Count unique IP addresses
   kubectl logs -l app=erpgo-api | grep "authentication failed" | awk '{print $6}' | sort -u | wc -l
   ```

### Resolution Steps

1. **If brute force attack detected:**
   ```bash
   # Block attacking IP addresses at firewall/load balancer level
   # Update WAF rules if available
   ```

2. **If credential stuffing attack:**
   - Force password reset for affected accounts
   - Enable MFA for all users
   - Notify affected users

3. **If legitimate user issues:**
   - Check for password policy changes
   - Verify authentication service is working
   - Check for clock skew issues (JWT validation)

4. **Immediate mitigation:**
   ```bash
   # Temporarily increase rate limit thresholds if needed
   kubectl set env deployment/erpgo-api AUTH_MAX_ATTEMPTS=3 AUTH_LOCKOUT_DURATION=30m
   ```

### Prevention
- Implement CAPTCHA after N failed attempts
- Use MFA for all accounts
- Implement IP-based rate limiting
- Monitor for credential stuffing patterns
- Regular security awareness training

---

## Low Cache Hit Rate

**Alert Name:** `LowCacheHitRate`

**Severity:** Warning

**Description:** Cache hit rate has dropped below 70%.

### Symptoms
- Increased database load
- Slower response times
- Higher latency for cached endpoints
- Increased database query count

### Investigation Steps

1. **Check current cache metrics:**
   ```bash
   # Query Prometheus:
   # cache_hit_rate by (cache_type)
   # cache_operations_total by (operation, result)
   ```

2. **Check cache memory usage:**
   ```bash
   kubectl exec -it <redis-pod> -- redis-cli INFO memory
   ```

3. **Check cache eviction rate:**
   ```bash
   # Query Prometheus:
   # rate(cache_evictions_total[5m])
   ```

4. **Identify cache misses by key pattern:**
   ```bash
   kubectl logs -l app=erpgo-api | grep "cache miss" | awk '{print $5}' | sort | uniq -c | sort -rn
   ```

### Resolution Steps

1. **If cache memory is full:**
   ```bash
   # Increase Redis memory limit
   kubectl set resources deployment/redis --limits=memory=4Gi
   
   # Or adjust eviction policy
   kubectl exec -it <redis-pod> -- redis-cli CONFIG SET maxmemory-policy allkeys-lru
   ```

2. **If cache is being cleared frequently:**
   - Review cache invalidation logic
   - Increase TTL for stable data
   - Implement cache warming strategy

3. **If new deployment cleared cache:**
   ```bash
   # Warm up cache
   ./scripts/warm-cache.sh
   ```

4. **If cache keys are poorly distributed:**
   - Review cache key naming strategy
   - Implement cache key prefixing
   - Consider cache sharding

### Prevention
- Monitor cache metrics proactively
- Implement cache warming on deployment
- Use appropriate TTL values
- Regular cache performance reviews
- Implement cache preloading for critical data

---

## Deployment Procedures

**Purpose:** Standard operating procedures for deploying ERPGo to production environments.

**Related Documentation:**
- [Deployment Guide](../DEPLOYMENT_GUIDE.md)
- [Launch Checklist](./LAUNCH_CHECKLIST.md)
- [Rollback Procedures](./ROLLBACK_PROCEDURES.md)

### Pre-Deployment Checklist

**24-48 Hours Before Deployment:**

- [ ] All tests passing (unit, integration, e2e, property-based)
  ```bash
  make test
  go test -race ./...
  ```

- [ ] Security scan completed with no critical findings
  ```bash
  ./scripts/security-scan.sh
  gosec ./...
  nancy go.sum
  ```

- [ ] Load tests passed with acceptable performance
  ```bash
  ./scripts/load-test.sh
  # Verify: p99 < 500ms, error rate < 0.1%
  ```

- [ ] Database migrations tested in staging
  ```bash
  # Run migrations in staging
  kubectl exec -it deployment/erpgo-api-staging -- ./erpgo migrate up
  
  # Verify data integrity
  ./scripts/verify-migration.sh
  ```

- [ ] Rollback plan documented and tested
  ```bash
  # Test rollback in staging
  kubectl rollout undo deployment/erpgo-api-staging
  ```

- [ ] Monitoring dashboards ready
  - [ ] Grafana dashboards updated
  - [ ] Alert rules configured
  - [ ] Runbooks linked to alerts

- [ ] Team notification sent
  ```
  Subject: Production Deployment - [Date/Time]
  - What: ERPGo v[version]
  - When: [Date] at [Time] [Timezone]
  - Duration: ~30 minutes
  - Impact: None expected (zero-downtime)
  - Rollback: Available within 5 minutes
  ```

- [ ] Backup verified
  ```bash
  # Verify latest backup exists and is valid
  ./scripts/backup/database-backup.sh verify
  ```

**1 Hour Before Deployment:**

- [ ] Verify staging environment is healthy
  ```bash
  curl https://staging.erpgo.com/health/ready
  ```

- [ ] Check production system health
  ```bash
  curl https://api.erpgo.com/health/ready
  kubectl get pods -l app=erpgo-api
  kubectl top nodes
  ```

- [ ] Verify no ongoing incidents
  - Check PagerDuty
  - Check Grafana for anomalies
  - Check error rates

- [ ] On-call engineer available
  - Primary on-call confirmed
  - Secondary on-call confirmed
  - Database team on standby

### Deployment Steps

**Step 1: Pre-Deployment Backup**

```bash
# Create pre-deployment backup
./scripts/backup/database-backup.sh manual "pre-deployment-v$(cat VERSION)"

# Verify backup completed successfully
./scripts/backup/database-backup.sh verify
```

**Step 2: Deploy Database Migrations (if any)**

```bash
# Run migrations in production
kubectl exec -it deployment/erpgo-api -- ./erpgo migrate up

# Verify migrations applied successfully
kubectl exec -it deployment/erpgo-api -- ./erpgo migrate status

# Check database health
kubectl exec -it <postgres-pod> -- psql -U $DB_USER -d erpgo -c "SELECT version();"
```

**Step 3: Deploy Application (Blue-Green or Canary)**

**Option A: Blue-Green Deployment**

```bash
# Deploy new version to "green" environment
kubectl apply -f k8s/deployment-green.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=erpgo-api,version=green --timeout=300s

# Run smoke tests against green environment
./scripts/smoke-test.sh https://green.erpgo.com

# Switch traffic to green
kubectl patch service erpgo-api -p '{"spec":{"selector":{"version":"green"}}}'

# Monitor for 10 minutes
watch -n 10 'kubectl logs -l app=erpgo-api,version=green --tail=50 | grep ERROR'

# If successful, scale down blue
kubectl scale deployment/erpgo-api-blue --replicas=0
```

**Option B: Canary Deployment**

```bash
# Deploy canary with 10% traffic
kubectl apply -f k8s/deployment-canary.yaml
kubectl scale deployment/erpgo-api-canary --replicas=1

# Monitor canary for 10 minutes
watch 'kubectl logs -l app=erpgo-api,version=canary --tail=50 | grep ERROR'

# Check canary metrics
# Error rate should be similar to main deployment
# Latency should be similar to main deployment

# If healthy, increase to 50%
kubectl scale deployment/erpgo-api-canary --replicas=5
kubectl scale deployment/erpgo-api --replicas=5

# Monitor for 10 minutes

# If healthy, complete rollout
kubectl scale deployment/erpgo-api-canary --replicas=10
kubectl scale deployment/erpgo-api --replicas=0

# Rename canary to main
kubectl patch deployment erpgo-api-canary -p '{"metadata":{"name":"erpgo-api"}}'
```

**Step 4: Post-Deployment Verification**

```bash
# Run smoke tests
./scripts/smoke-test.sh https://api.erpgo.com

# Verify health checks
curl https://api.erpgo.com/health/live
curl https://api.erpgo.com/health/ready

# Check key metrics
# - Error rate < 0.1%
# - p99 latency < 500ms
# - All pods running

# Verify critical functionality
./scripts/verify-deployment.sh

# Check logs for errors
kubectl logs -l app=erpgo-api --tail=100 | grep -i error
```

**Step 5: Monitor for 1 Hour**

```bash
# Watch key metrics in Grafana
# - Request rate
# - Error rate
# - Response time
# - Database connections
# - Cache hit rate

# Monitor logs
kubectl logs -l app=erpgo-api -f | grep -E "ERROR|WARN"

# Check for alerts
# No critical alerts should fire
```

### Post-Deployment Tasks

- [ ] Update deployment log
  ```bash
  echo "$(date): Deployed v$(cat VERSION) to production" >> deployments.log
  ```

- [ ] Send completion notification
  ```
  Subject: Production Deployment Complete - v[version]
  - Status: Success
  - Duration: [actual duration]
  - Issues: None / [list any issues]
  - Rollback: Not needed
  ```

- [ ] Update documentation (if needed)
  - API documentation
  - Configuration changes
  - New features

- [ ] Schedule post-deployment review (within 24 hours)
  - Review metrics
  - Identify any issues
  - Document lessons learned

### Deployment Rollback

See [Emergency Rollback](#emergency-rollback) section below.

---

## Emergency Rollback

**When to Rollback:**
- Error rate > 5%
- p99 latency > 2 seconds
- Critical functionality broken
- Data corruption detected
- Security vulnerability discovered

### Rollback Steps

**Step 1: Initiate Rollback**

```bash
# Immediate rollback to previous version
kubectl rollout undo deployment/erpgo-api

# Or rollback to specific revision
kubectl rollout history deployment/erpgo-api
kubectl rollout undo deployment/erpgo-api --to-revision=<revision-number>

# Wait for rollback to complete
kubectl rollout status deployment/erpgo-api
```

**Step 2: Verify Rollback**

```bash
# Check pod status
kubectl get pods -l app=erpgo-api

# Verify version
kubectl exec -it deployment/erpgo-api -- ./erpgo version

# Run smoke tests
./scripts/smoke-test.sh https://api.erpgo.com

# Check metrics
# - Error rate should decrease
# - Latency should improve
```

**Step 3: Rollback Database Migrations (if needed)**

```bash
# Only if migrations were applied and are causing issues
kubectl exec -it deployment/erpgo-api -- ./erpgo migrate down

# Verify migration rollback
kubectl exec -it deployment/erpgo-api -- ./erpgo migrate status

# Restore from backup if migration rollback fails
./scripts/backup/disaster-recovery.sh restore <backup-timestamp>
```

**Step 4: Notify Team**

```
Subject: URGENT - Production Rollback Initiated
- Reason: [brief description]
- Action: Rolled back to v[previous-version]
- Status: [In Progress / Complete]
- Impact: [description of user impact]
- Next Steps: [investigation plan]
```

**Step 5: Post-Rollback Actions**

- [ ] Verify system is stable
- [ ] Document rollback reason
- [ ] Create incident report
- [ ] Schedule post-mortem
- [ ] Fix issues in development
- [ ] Plan re-deployment

---

## General Troubleshooting

### Accessing Logs
```bash
# Application logs
kubectl logs -l app=erpgo-api --tail=100 -f

# Database logs
kubectl logs -l app=postgres --tail=100 -f

# Redis logs
kubectl logs -l app=redis --tail=100 -f
```

### Accessing Metrics
- Grafana: https://grafana.erpgo.com
- Prometheus: https://prometheus.erpgo.com
- Jaeger: https://jaeger.erpgo.com

### Emergency Contacts
- On-call Engineer: Check PagerDuty
- Database Team: db-team@erpgo.com
- Security Team: security@erpgo.com
- DevOps Team: devops@erpgo.com

### Escalation Path
1. On-call Engineer (immediate)
2. Team Lead (15 minutes)
3. Engineering Manager (30 minutes)
4. CTO (1 hour for critical issues)

---

## Post-Incident Actions

After resolving any incident:

1. **Document the incident:**
   - Create incident report
   - Document timeline
   - Note resolution steps

2. **Conduct post-mortem (for P0/P1):**
   - Schedule within 48 hours
   - Identify root cause
   - Create action items

3. **Update runbooks:**
   - Add new learnings
   - Update resolution steps
   - Improve prevention measures

4. **Implement improvements:**
   - Address root causes
   - Improve monitoring
   - Update alerts if needed
