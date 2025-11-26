# Disaster Recovery Procedures

## Overview

This document outlines the disaster recovery (DR) procedures for the ERPGo system. These procedures are designed to minimize downtime and data loss in the event of various failure scenarios.

## Recovery Objectives

- **RTO (Recovery Time Objective)**: Maximum acceptable downtime
- **RPO (Recovery Point Objective)**: Maximum acceptable data loss

### Service Level Objectives

| Scenario | RTO | RPO | Priority |
|----------|-----|-----|----------|
| Database Failure | 5 minutes | 0 minutes | P0 - Critical |
| Application Failure | 2 minutes | 0 minutes | P0 - Critical |
| Region Failure | 4 hours | 1 hour | P1 - High |
| Data Corruption | 4 hours | 1 hour | P1 - High |
| Security Breach | Varies | Varies | P0 - Critical |

## Backup Strategy

### Automated Backups

Backups are performed automatically every 6 hours with the following retention policy:

- **Daily Backups**: Retained for 7 days
- **Weekly Backups**: Retained for 4 weeks (28 days)
- **Monthly Backups**: Retained for 1 year (12 months)

### Backup Locations

- **Primary**: Local storage on backup server
- **Secondary**: S3 Glacier storage (if configured)
- **Encryption**: All backups are encrypted using AES-256-CBC

### Backup Verification

All backups are automatically verified for integrity after creation using `pg_restore --list` to ensure they can be restored successfully.

## Disaster Recovery Scenarios

## 1. Database Failure Recovery

**Scenario**: Primary database server fails or becomes unresponsive

**RTO**: 5 minutes  
**RPO**: 0 minutes (using replication)

### Detection

- Health check endpoint returns 503
- Prometheus alert: `DatabaseDown`
- Application logs show connection errors

### Recovery Procedure

#### Step 1: Verify Database Status

```bash
# Check if database is responsive
docker exec erpgo-postgres-primary pg_isready -U erpgo -d erp

# Check database logs
docker logs erpgo-postgres-primary --tail 100
```

#### Step 2: Attempt Database Restart

```bash
# Restart database container
docker-compose restart postgres-primary

# Wait for database to be ready (max 30 seconds)
timeout 30 bash -c 'until docker exec erpgo-postgres-primary pg_isready -U erpgo -d erp; do sleep 2; done'
```

#### Step 3: Failover to Replica (if available)

If primary database cannot be recovered:

```bash
# Promote replica to primary
docker exec erpgo-postgres-replica pg_ctl promote -D /var/lib/postgresql/data

# Update application configuration to point to new primary
export POSTGRES_PRIMARY_HOST=postgres-replica

# Restart application services
docker-compose restart api worker
```

#### Step 4: Verify Recovery

```bash
# Check database connectivity
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "SELECT 1;"

# Check application health
curl -f http://localhost:8080/health/ready

# Verify data integrity
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "SELECT COUNT(*) FROM users;"
```

#### Step 5: Monitor and Document

- Monitor application metrics for 30 minutes
- Document incident in incident log
- Schedule post-mortem meeting

### Rollback Procedure

If failover causes issues:

```bash
# Revert to original primary
export POSTGRES_PRIMARY_HOST=postgres-primary

# Restart services
docker-compose restart api worker
```

---

## 2. Backup Restore Procedure

**Scenario**: Need to restore database from backup due to data corruption or loss

**RTO**: 4 hours  
**RPO**: 1 hour (6-hour backup interval)

### Prerequisites

- Backup file location
- Backup encryption key (if encrypted)
- Database credentials
- Sufficient disk space

### Recovery Procedure

#### Step 1: Identify Backup to Restore

```bash
# List available backups
./scripts/backup/database-backup.sh list

# Identify the most recent valid backup before the incident
# Example: /backups/postgres/automated_full_backup_20231215_120000_daily.sql.gz.enc
```

#### Step 2: Create Emergency Backup

Before restoring, create an emergency backup of current state:

```bash
# Create emergency backup
./scripts/backup/database-backup.sh backup full

# This creates a backup in case we need to revert
```

#### Step 3: Stop Application Services

```bash
# Stop API and worker services to prevent new connections
docker-compose stop api worker

# Verify no active connections
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT COUNT(*) FROM pg_stat_activity WHERE datname='erp';"
```

#### Step 4: Restore Database

```bash
# Set environment variables
export RECOVERY_BACKUP="/backups/postgres/automated_full_backup_20231215_120000_daily.sql.gz.enc"
export BACKUP_ENCRYPTION_KEY="your-encryption-key"
export RECOVERY_TYPE="database"
export RECOVERY_TARGET="production"

# Run disaster recovery script
./scripts/backup/disaster-recovery.sh
```

Or manually restore:

```bash
# Decrypt backup if encrypted
openssl enc -aes-256-cbc -d \
  -in /backups/postgres/backup.sql.enc \
  -out /tmp/backup.sql \
  -pass pass:"$BACKUP_ENCRYPTION_KEY"

# Copy backup to container
docker cp /tmp/backup.sql erpgo-postgres-primary:/tmp/restore.sql

# Restore database
docker exec erpgo-postgres-primary pg_restore \
  --verbose \
  --clean \
  --if-exists \
  --no-owner \
  --no-privileges \
  --dbname=erp \
  /tmp/restore.sql

# Clean up
docker exec erpgo-postgres-primary rm -f /tmp/restore.sql
rm -f /tmp/backup.sql
```

#### Step 5: Run Database Migrations

```bash
# Ensure database schema is up to date
docker-compose run --rm migrator
```

#### Step 6: Restart Application Services

```bash
# Start all services
docker-compose up -d

# Wait for services to be ready
sleep 30
```

#### Step 7: Verify Recovery

```bash
# Check database connectivity
docker exec erpgo-postgres-primary pg_isready -U erpgo -d erp

# Verify critical data
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT COUNT(*) FROM users; SELECT COUNT(*) FROM orders;"

# Check application health
curl -f http://localhost:8080/health/ready

# Test critical API endpoints
curl -f http://localhost:8080/api/v1/health
```

#### Step 8: Validate Data Integrity

```bash
# Run data integrity checks
# Check for orphaned records
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT COUNT(*) FROM order_items oi 
   LEFT JOIN orders o ON oi.order_id = o.id 
   WHERE o.id IS NULL;"

# Verify foreign key constraints
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT conname, conrelid::regclass, confrelid::regclass 
   FROM pg_constraint 
   WHERE contype = 'f';"
```

#### Step 9: Monitor and Document

- Monitor application for 2 hours
- Check error rates and latency
- Document recovery time and data loss
- Update incident log

### Rollback Procedure

If restore causes issues:

```bash
# Restore from emergency backup created in Step 2
export RECOVERY_BACKUP="/path/to/emergency_backup.sql"
./scripts/backup/disaster-recovery.sh
```

---

## 3. Region Failover Procedure

**Scenario**: Primary region becomes unavailable

**RTO**: 4 hours  
**RPO**: 1 hour

### Prerequisites

- Secondary region infrastructure deployed
- Database replication configured
- DNS failover capability
- S3 cross-region replication enabled

### Recovery Procedure

#### Step 1: Verify Primary Region Status

```bash
# Check primary region health
curl -f https://primary-region.example.com/health/ready

# Check database replication lag
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn();"
```

#### Step 2: Activate Secondary Region

```bash
# Promote secondary database to primary
ssh secondary-region "docker exec erpgo-postgres-replica pg_ctl promote"

# Update DNS to point to secondary region
# This depends on your DNS provider (Route53, CloudFlare, etc.)
# Example with AWS Route53:
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch file://failover-dns.json
```

#### Step 3: Start Services in Secondary Region

```bash
# SSH to secondary region
ssh secondary-region

# Start all services
cd /opt/erpgo
docker-compose up -d

# Verify services are running
docker-compose ps
```

#### Step 4: Verify Secondary Region

```bash
# Check health endpoints
curl -f https://secondary-region.example.com/health/ready

# Test critical functionality
curl -f https://secondary-region.example.com/api/v1/users/me \
  -H "Authorization: Bearer $TEST_TOKEN"
```

#### Step 5: Monitor and Communicate

- Send notification to all users about region failover
- Monitor secondary region performance
- Document failover time and any issues
- Plan primary region recovery

### Rollback Procedure

When primary region is restored:

```bash
# Sync data from secondary to primary
# Stop writes to secondary
# Restore primary database from secondary backup
# Update DNS back to primary region
# Restart services in primary region
```

---

## 4. Data Corruption Recovery

**Scenario**: Data corruption detected in database

**RTO**: 4 hours  
**RPO**: 1 hour

### Detection

- Data validation checks fail
- Foreign key constraint violations
- Unexpected NULL values in required fields
- User reports of missing or incorrect data

### Recovery Procedure

#### Step 1: Assess Corruption Scope

```bash
# Check for constraint violations
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT conname, conrelid::regclass 
   FROM pg_constraint 
   WHERE NOT pg_catalog.pg_constraint_is_valid(oid);"

# Check for orphaned records
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT 'order_items' as table_name, COUNT(*) as orphaned_count
   FROM order_items oi 
   LEFT JOIN orders o ON oi.order_id = o.id 
   WHERE o.id IS NULL;"
```

#### Step 2: Identify Last Known Good Backup

```bash
# List recent backups
./scripts/backup/database-backup.sh list

# Identify backup before corruption occurred
# Check backup timestamps against incident timeline
```

#### Step 3: Restore from Backup

Follow the "Backup Restore Procedure" above (Section 2)

#### Step 4: Replay Transactions (if possible)

If transaction logs are available:

```bash
# Restore to point-in-time before corruption
# This requires WAL archiving to be enabled
docker exec erpgo-postgres-primary pg_restore \
  --target-time='2023-12-15 14:30:00' \
  /path/to/backup.sql
```

#### Step 5: Validate Data Integrity

```bash
# Run comprehensive data validation
docker exec erpgo-postgres-primary psql -U erpgo -d erp -f /scripts/validate_data.sql

# Check critical business metrics
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT 
     (SELECT COUNT(*) FROM users) as user_count,
     (SELECT COUNT(*) FROM orders) as order_count,
     (SELECT SUM(total_amount) FROM orders WHERE status='completed') as total_revenue;"
```

---

## 5. Security Breach Response

**Scenario**: Security breach detected (unauthorized access, data leak, etc.)

**RTO**: Varies based on severity  
**RPO**: Varies based on severity

### Immediate Actions (First 15 minutes)

#### Step 1: Contain the Breach

```bash
# Immediately block suspicious IP addresses
# Update firewall rules
iptables -A INPUT -s <suspicious-ip> -j DROP

# Disable compromised user accounts
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "UPDATE users SET is_active = false WHERE id IN (<compromised-user-ids>);"

# Rotate all secrets immediately
# Update JWT secrets
export JWT_SECRET=$(openssl rand -base64 32)
export REFRESH_SECRET=$(openssl rand -base64 32)

# Restart services with new secrets
docker-compose restart api worker
```

#### Step 2: Assess Impact

```bash
# Check audit logs for unauthorized access
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT * FROM audit_logs 
   WHERE timestamp > NOW() - INTERVAL '24 hours' 
   AND (event_type = 'login' OR event_type = 'data_access')
   ORDER BY timestamp DESC;"

# Check for data exfiltration
# Review access logs for unusual patterns
docker logs erpgo-api --since 24h | grep -i "download\|export"
```

#### Step 3: Notify Stakeholders

- Notify security team immediately
- Notify management within 1 hour
- Prepare customer communication (if data breach)
- Contact legal team if required

### Investigation Phase (Hours 1-4)

#### Step 4: Collect Evidence

```bash
# Create forensic backup
./scripts/backup/database-backup.sh backup full

# Export audit logs
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "COPY (SELECT * FROM audit_logs WHERE timestamp > NOW() - INTERVAL '7 days') 
   TO '/tmp/audit_logs_forensic.csv' CSV HEADER;"

# Collect application logs
docker logs erpgo-api > /tmp/api_logs_forensic.log
docker logs erpgo-postgres-primary > /tmp/db_logs_forensic.log
```

#### Step 5: Identify Attack Vector

- Review authentication logs
- Check for SQL injection attempts
- Review API access patterns
- Check for privilege escalation

#### Step 6: Close Security Gaps

```bash
# Apply security patches
# Update dependencies
go get -u all
go mod tidy

# Rebuild and redeploy
docker-compose build
docker-compose up -d

# Enable additional security measures
# - Enable 2FA for all admin accounts
# - Implement IP whitelisting
# - Add rate limiting
# - Enable audit logging for all operations
```

### Recovery Phase (Hours 4-24)

#### Step 7: Restore Clean State

If data was compromised:

```bash
# Restore from last known good backup before breach
export RECOVERY_BACKUP="/path/to/pre-breach-backup.sql"
./scripts/backup/disaster-recovery.sh
```

#### Step 8: Reset All Credentials

```bash
# Force password reset for all users
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "UPDATE users SET password_reset_required = true;"

# Invalidate all active sessions
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "DELETE FROM sessions;"

# Rotate all API keys
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "UPDATE api_keys SET revoked = true;"
```

#### Step 9: Enhanced Monitoring

```bash
# Enable verbose audit logging
export AUDIT_LOG_LEVEL=debug

# Add additional security alerts
# Monitor for:
# - Failed login attempts
# - Unusual data access patterns
# - API rate limit violations
# - Privilege escalation attempts
```

### Post-Incident (Days 1-7)

#### Step 10: Post-Mortem

- Document timeline of events
- Identify root cause
- Document lessons learned
- Update security procedures
- Implement preventive measures

#### Step 11: Customer Communication

If customer data was affected:

- Notify affected customers within 72 hours
- Provide details of breach (what data, when, how)
- Explain remediation steps taken
- Offer credit monitoring if applicable
- Document all communications

---

## Testing and Validation

### Quarterly DR Drills

Conduct full disaster recovery drills every quarter:

1. **Q1**: Database failure and restore
2. **Q2**: Region failover
3. **Q3**: Data corruption recovery
4. **Q4**: Full system recovery

### Monthly Backup Tests

- Verify backup integrity
- Test restore procedure on non-production environment
- Measure restore time
- Document any issues

### Continuous Monitoring

- Monitor backup success rate (target: 100%)
- Monitor backup size trends
- Alert on backup failures
- Track RTO/RPO metrics

---

## Contact Information

### Emergency Contacts

| Role | Name | Phone | Email |
|------|------|-------|-------|
| On-Call Engineer | TBD | TBD | oncall@example.com |
| Database Admin | TBD | TBD | dba@example.com |
| Security Lead | TBD | TBD | security@example.com |
| CTO | TBD | TBD | cto@example.com |

### Escalation Path

1. **Level 1**: On-Call Engineer (0-15 minutes)
2. **Level 2**: Database Admin + Security Lead (15-30 minutes)
3. **Level 3**: CTO + Management (30-60 minutes)

---

## Appendix

### A. Backup Script Locations

- Database backup: `./scripts/backup/database-backup.sh`
- Automated backup: `./scripts/backup/automated-backup.sh`
- Disaster recovery: `./scripts/backup/disaster-recovery.sh`

### B. Configuration Files

- Production config: `.env.production`
- Backup config: Environment variables in automated-backup.sh
- Docker compose: `docker-compose.prod.yml`

### C. Monitoring Dashboards

- Grafana: http://grafana.example.com
- Prometheus: http://prometheus.example.com
- Backup status: Check `/backups/postgres/logs/backup_report.log`

### D. Useful Commands

```bash
# Check backup status
tail -f /backups/postgres/logs/automated-backup.log

# List recent backups
ls -lah /backups/postgres/ | grep automated

# Check database size
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT pg_size_pretty(pg_database_size('erp'));"

# Check replication lag
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()));"
```

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2024-01-15 | System | Initial version |

---

**Last Updated**: 2024-01-15  
**Next Review**: 2024-04-15  
**Owner**: DevOps Team
