# Backup Operations Runbook

## Overview

This runbook provides step-by-step procedures for managing database backups in the ERPGo system.

## Quick Reference

| Operation | Command | Frequency |
|-----------|---------|-----------|
| Manual Backup | `./scripts/backup/database-backup.sh backup full` | On-demand |
| Automated Backup | Runs via cron | Every 6 hours |
| List Backups | `./scripts/backup/database-backup.sh list` | As needed |
| Restore Backup | `./scripts/backup/database-backup.sh restore <file>` | Emergency only |
| Verify Backup | Automatic after creation | Every backup |
| Cleanup Old Backups | Automatic | Daily |

## Backup Schedule

### Automated Schedule

Backups run automatically every 6 hours:
- 00:00 UTC
- 06:00 UTC
- 12:00 UTC
- 18:00 UTC

### Retention Policy

| Type | Retention | Purpose |
|------|-----------|---------|
| Daily | 7 days | Recent recovery |
| Weekly | 4 weeks | Short-term history |
| Monthly | 12 months | Long-term compliance |

## Common Operations

### 1. Create Manual Backup

**When to use**: Before major changes, deployments, or migrations

```bash
# Navigate to project root
cd /opt/erpgo

# Create full backup
./scripts/backup/database-backup.sh backup full

# Expected output:
# [SUCCESS] Backup created successfully: full_backup_20231215_143022.sql (Size: 245M)
# [SUCCESS] Backup integrity verified
# [SUCCESS] Backup process completed successfully!
```

**Verification**:
```bash
# Check backup was created
ls -lh /backups/postgres/*.sql

# Check backup log
tail -20 /backups/postgres/logs/backup_report.log
```

### 2. List Available Backups

```bash
# List all backups
./scripts/backup/database-backup.sh list

# List only recent backups
ls -lht /backups/postgres/*.sql* | head -10

# List backups by type
ls -lh /backups/postgres/*_daily.sql*
ls -lh /backups/postgres/*_weekly.sql*
ls -lh /backups/postgres/*_monthly.sql*
```

### 3. Verify Backup Integrity

Backups are automatically verified after creation, but you can manually verify:

```bash
# Verify specific backup
BACKUP_FILE="/backups/postgres/automated_full_backup_20231215_120000_daily.sql.gz.enc"

# If encrypted, decrypt first
openssl enc -aes-256-cbc -d \
  -in "$BACKUP_FILE" \
  -out /tmp/verify_backup.sql \
  -pass pass:"$BACKUP_ENCRYPTION_KEY"

# Copy to container
docker cp /tmp/verify_backup.sql erpgo-postgres-primary:/tmp/verify.sql

# Verify with pg_restore
docker exec erpgo-postgres-primary pg_restore \
  --list \
  --format=custom \
  /tmp/verify.sql

# Clean up
docker exec erpgo-postgres-primary rm -f /tmp/verify.sql
rm -f /tmp/verify_backup.sql
```

### 4. Restore from Backup

**⚠️ WARNING**: This will overwrite current database. Always create emergency backup first!

```bash
# Step 1: Create emergency backup
./scripts/backup/database-backup.sh backup full

# Step 2: Stop application services
docker-compose stop api worker

# Step 3: Restore from backup
BACKUP_FILE="/backups/postgres/automated_full_backup_20231215_120000_daily.sql.gz.enc"
./scripts/backup/database-backup.sh restore "$BACKUP_FILE"

# Step 4: Restart services
docker-compose up -d

# Step 5: Verify
curl -f http://localhost:8080/health/ready
```

### 5. Monitor Backup Status

```bash
# Check automated backup logs
tail -f /backups/postgres/logs/automated-backup.log

# Check for backup failures
grep -i "error\|fail" /backups/postgres/logs/automated-backup.log

# Check backup success rate
grep -c "SUCCESS" /backups/postgres/logs/automated-backup.log
grep -c "ERROR" /backups/postgres/logs/automated-backup.log

# Check disk space
df -h /backups/postgres
```

### 6. Manual Cleanup

```bash
# Clean up backups older than retention policy
./scripts/backup/automated-backup.sh cleanup

# Manually delete specific backup
rm /backups/postgres/old_backup_file.sql

# Check space saved
df -h /backups/postgres
```

## Troubleshooting

### Problem: Backup Creation Fails

**Symptoms**:
- Error in backup logs
- No backup file created
- Alert notification received

**Diagnosis**:
```bash
# Check database connectivity
docker exec erpgo-postgres-primary pg_isready -U erpgo -d erp

# Check disk space
df -h /backups/postgres

# Check database logs
docker logs erpgo-postgres-primary --tail 50

# Check backup script logs
tail -50 /backups/postgres/logs/automated-backup.log
```

**Solutions**:

1. **Database not accessible**:
   ```bash
   # Restart database
   docker-compose restart postgres-primary
   
   # Wait for ready
   timeout 30 bash -c 'until docker exec erpgo-postgres-primary pg_isready; do sleep 2; done'
   
   # Retry backup
   ./scripts/backup/automated-backup.sh backup full
   ```

2. **Insufficient disk space**:
   ```bash
   # Clean up old backups
   ./scripts/backup/automated-backup.sh cleanup
   
   # Or manually delete old backups
   find /backups/postgres -name "*.sql*" -mtime +30 -delete
   
   # Retry backup
   ./scripts/backup/automated-backup.sh backup full
   ```

3. **Permission issues**:
   ```bash
   # Fix permissions
   sudo chown -R $(whoami):$(whoami) /backups/postgres
   chmod -R 755 /backups/postgres
   
   # Retry backup
   ./scripts/backup/automated-backup.sh backup full
   ```

### Problem: Backup Verification Fails

**Symptoms**:
- Backup file created but verification fails
- Error: "Backup integrity check failed"

**Diagnosis**:
```bash
# Check backup file size
ls -lh /backups/postgres/failed_backup.sql

# Try to list backup contents
docker exec erpgo-postgres-primary pg_restore \
  --list \
  /path/to/backup.sql
```

**Solutions**:

1. **Corrupted backup file**:
   ```bash
   # Delete corrupted backup
   rm /backups/postgres/corrupted_backup.sql
   
   # Create new backup
   ./scripts/backup/automated-backup.sh backup full
   ```

2. **Encryption issues**:
   ```bash
   # Verify encryption key is correct
   echo $BACKUP_ENCRYPTION_KEY
   
   # Try manual decryption
   openssl enc -aes-256-cbc -d \
     -in backup.sql.enc \
     -out test_decrypt.sql \
     -pass pass:"$BACKUP_ENCRYPTION_KEY"
   ```

### Problem: Restore Fails

**Symptoms**:
- Restore command fails
- Database in inconsistent state
- Application errors after restore

**Diagnosis**:
```bash
# Check database status
docker exec erpgo-postgres-primary pg_isready

# Check for active connections
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT COUNT(*) FROM pg_stat_activity WHERE datname='erp';"

# Check restore logs
tail -50 /backups/postgres/logs/restore_*.log
```

**Solutions**:

1. **Active connections preventing restore**:
   ```bash
   # Stop all application services
   docker-compose stop api worker
   
   # Terminate active connections
   docker exec erpgo-postgres-primary psql -U erpgo -d postgres -c \
     "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='erp';"
   
   # Retry restore
   ./scripts/backup/database-backup.sh restore /path/to/backup.sql
   ```

2. **Incompatible backup version**:
   ```bash
   # Check PostgreSQL version
   docker exec erpgo-postgres-primary psql --version
   
   # Check backup PostgreSQL version
   docker exec erpgo-postgres-primary pg_restore --version
   
   # If versions don't match, may need to upgrade/downgrade
   ```

3. **Restore to different database**:
   ```bash
   # Create new database for testing
   docker exec erpgo-postgres-primary psql -U erpgo -d postgres -c \
     "CREATE DATABASE erp_restore;"
   
   # Restore to test database
   ./scripts/backup/database-backup.sh restore /path/to/backup.sql erp_restore
   
   # Verify data
   docker exec erpgo-postgres-primary psql -U erpgo -d erp_restore -c \
     "SELECT COUNT(*) FROM users;"
   ```

### Problem: Automated Backups Not Running

**Symptoms**:
- No recent backups in last 6 hours
- Cron job not executing
- No entries in automated-backup.log

**Diagnosis**:
```bash
# Check cron job
crontab -l | grep automated-backup

# Check cron logs
grep -i backup /var/log/syslog

# Check script permissions
ls -l /opt/erpgo/scripts/backup/automated-backup.sh
```

**Solutions**:

1. **Cron job not configured**:
   ```bash
   # Setup cron job
   ./scripts/backup/automated-backup.sh setup-cron
   
   # Verify cron job
   crontab -l
   ```

2. **Script not executable**:
   ```bash
   # Make script executable
   chmod +x /opt/erpgo/scripts/backup/automated-backup.sh
   
   # Test manual execution
   ./scripts/backup/automated-backup.sh backup full
   ```

3. **Environment variables not set**:
   ```bash
   # Check environment variables in cron
   # Add to crontab:
   SHELL=/bin/bash
   BASH_ENV=/opt/erpgo/.env.production
   
   0 */6 * * * /opt/erpgo/scripts/backup/automated-backup.sh backup full
   ```

### Problem: Backup Size Growing Too Large

**Symptoms**:
- Backup files consuming too much disk space
- Backup creation taking too long
- Disk space alerts

**Diagnosis**:
```bash
# Check backup sizes
du -sh /backups/postgres/*.sql* | sort -h

# Check database size
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT pg_size_pretty(pg_database_size('erp'));"

# Check largest tables
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT schemaname, tablename, 
   pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
   FROM pg_tables 
   WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
   ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC 
   LIMIT 10;"
```

**Solutions**:

1. **Implement data archiving**:
   ```bash
   # Archive old data to separate table
   docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
     "CREATE TABLE orders_archive AS 
      SELECT * FROM orders WHERE created_at < NOW() - INTERVAL '1 year';"
   
   # Delete archived data from main table
   docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
     "DELETE FROM orders WHERE created_at < NOW() - INTERVAL '1 year';"
   
   # Vacuum to reclaim space
   docker exec erpgo-postgres-primary psql -U erpgo -d erp -c "VACUUM FULL;"
   ```

2. **Adjust retention policy**:
   ```bash
   # Reduce retention periods
   export BACKUP_RETENTION_DAILY_DAYS=5
   export BACKUP_RETENTION_WEEKLY_WEEKS=3
   export BACKUP_RETENTION_MONTHLY_MONTHS=6
   
   # Clean up old backups
   ./scripts/backup/automated-backup.sh cleanup
   ```

3. **Use incremental backups** (future enhancement):
   ```bash
   # Enable WAL archiving for point-in-time recovery
   # This allows smaller incremental backups
   ```

## Monitoring and Alerts

### Key Metrics to Monitor

1. **Backup Success Rate**: Should be 100%
2. **Backup Duration**: Should be < 10 minutes for typical database
3. **Backup Size**: Monitor for unexpected growth
4. **Disk Space**: Should have 3x database size available
5. **Last Successful Backup**: Should be < 6 hours ago

### Alert Conditions

| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| Backup Failed | Backup fails after retry | Critical | Investigate immediately |
| Backup Delayed | No backup in 8 hours | Warning | Check cron job |
| Low Disk Space | < 20% free space | Warning | Clean up or expand |
| Backup Size Spike | > 50% increase | Info | Investigate data growth |
| Verification Failed | Backup integrity check fails | Critical | Create new backup |

### Prometheus Queries

```promql
# Backup success rate (last 24 hours)
rate(backup_success_total[24h]) / rate(backup_attempts_total[24h])

# Time since last successful backup
time() - backup_last_success_timestamp

# Backup duration
backup_duration_seconds

# Backup size
backup_size_bytes
```

## Best Practices

### Before Major Changes

1. Always create manual backup before:
   - Database migrations
   - Major deployments
   - Schema changes
   - Data imports/exports

2. Verify backup integrity
3. Test restore procedure on non-production environment
4. Document backup location and timestamp

### Regular Maintenance

1. **Weekly**:
   - Review backup logs for errors
   - Verify automated backups are running
   - Check disk space usage

2. **Monthly**:
   - Test restore procedure
   - Review retention policy
   - Update documentation

3. **Quarterly**:
   - Full disaster recovery drill
   - Review and update procedures
   - Audit backup security

### Security

1. **Encryption**:
   - Always encrypt backups
   - Rotate encryption keys quarterly
   - Store keys in secure vault

2. **Access Control**:
   - Limit backup access to authorized personnel
   - Use separate credentials for backup operations
   - Audit backup access logs

3. **Storage**:
   - Store backups in separate location from database
   - Use S3 Glacier for long-term storage
   - Enable versioning on S3 buckets

## Emergency Procedures

### Complete Data Loss

If all data is lost and no recent backup available:

1. Restore from oldest available backup
2. Manually reconstruct recent data from:
   - Application logs
   - External system integrations
   - User reports
3. Document data loss extent
4. Communicate with affected users

### Backup System Failure

If backup system completely fails:

1. Immediately create manual backup
2. Store in alternative location
3. Investigate backup system failure
4. Implement temporary backup solution
5. Fix and test backup system
6. Resume automated backups

## Appendix

### Environment Variables

```bash
# Required
POSTGRES_DB=erp
POSTGRES_USER=erpgo
POSTGRES_PASSWORD=<secure-password>
POSTGRES_PRIMARY_HOST=postgres-primary

# Optional
BACKUP_ENCRYPTION_KEY=<32-byte-key>
BACKUP_RETENTION_DAILY_DAYS=7
BACKUP_RETENTION_WEEKLY_WEEKS=4
BACKUP_RETENTION_MONTHLY_MONTHS=12
BACKUP_MAX_RETRIES=1
BACKUP_STORAGE_TYPE=local
BACKUP_S3_BUCKET=<bucket-name>
BACKUP_S3_REGION=<region>
```

### Useful SQL Queries

```sql
-- Check database size
SELECT pg_size_pretty(pg_database_size('erp'));

-- Check table sizes
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
  pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS index_size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 20;

-- Check active connections
SELECT 
  datname,
  usename,
  application_name,
  client_addr,
  state,
  query_start
FROM pg_stat_activity
WHERE datname = 'erp';

-- Check long-running queries
SELECT 
  pid,
  now() - query_start AS duration,
  query,
  state
FROM pg_stat_activity
WHERE state != 'idle'
  AND now() - query_start > interval '5 minutes'
ORDER BY duration DESC;
```

---

**Document Version**: 1.0  
**Last Updated**: 2024-01-15  
**Next Review**: 2024-04-15  
**Owner**: DevOps Team
