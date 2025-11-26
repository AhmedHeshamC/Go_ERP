# Recovery Test Plan

## Overview

This document outlines the testing procedures for validating disaster recovery capabilities. Regular testing ensures that recovery procedures work as expected and that RTO/RPO objectives can be met.

## Test Schedule

| Test Type | Frequency | Duration | Participants |
|-----------|-----------|----------|--------------|
| Backup Verification | Daily | 5 minutes | Automated |
| Restore Test | Monthly | 1 hour | DevOps Engineer |
| Database Failover | Quarterly | 2 hours | DevOps + DBA |
| Full DR Drill | Quarterly | 4 hours | All Engineering |
| Region Failover | Annually | 8 hours | All Engineering + Management |

## Monthly Restore Test

**Objective**: Verify that backups can be successfully restored

**Duration**: 1 hour

**Prerequisites**:
- Non-production environment available
- Recent backup file identified
- Database credentials available

### Test Procedure

#### 1. Preparation (10 minutes)

```bash
# Identify test backup
BACKUP_FILE=$(ls -t /backups/postgres/automated_*_daily.sql* | head -1)
echo "Testing backup: $BACKUP_FILE"

# Record start time
START_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
echo "Test started: $START_TIME"

# Create test environment
docker-compose -f docker-compose.test.yml up -d postgres-test
```

#### 2. Restore Backup (30 minutes)

```bash
# Decrypt backup if encrypted
if [[ "$BACKUP_FILE" == *.enc ]]; then
    openssl enc -aes-256-cbc -d \
      -in "$BACKUP_FILE" \
      -out /tmp/test_restore.sql \
      -pass pass:"$BACKUP_ENCRYPTION_KEY"
    RESTORE_FILE="/tmp/test_restore.sql"
else
    RESTORE_FILE="$BACKUP_FILE"
fi

# Copy to test container
docker cp "$RESTORE_FILE" erpgo-postgres-test:/tmp/restore.sql

# Restore database
docker exec erpgo-postgres-test pg_restore \
  --verbose \
  --clean \
  --if-exists \
  --no-owner \
  --no-privileges \
  --dbname=erp_test \
  /tmp/restore.sql

# Record restore completion time
RESTORE_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
```

#### 3. Verification (15 minutes)

```bash
# Verify database connectivity
docker exec erpgo-postgres-test pg_isready -U erpgo -d erp_test

# Check critical tables exist
docker exec erpgo-postgres-test psql -U erpgo -d erp_test -c \
  "SELECT COUNT(*) as user_count FROM users;
   SELECT COUNT(*) as order_count FROM orders;
   SELECT COUNT(*) as product_count FROM products;"

# Verify data integrity
docker exec erpgo-postgres-test psql -U erpgo -d erp_test -c \
  "SELECT COUNT(*) as orphaned_orders 
   FROM order_items oi 
   LEFT JOIN orders o ON oi.order_id = o.id 
   WHERE o.id IS NULL;"

# Check foreign key constraints
docker exec erpgo-postgres-test psql -U erpgo -d erp_test -c \
  "SELECT conname, conrelid::regclass 
   FROM pg_constraint 
   WHERE contype = 'f' 
   AND NOT pg_catalog.pg_constraint_is_valid(oid);"
```

#### 4. Cleanup (5 minutes)

```bash
# Stop test environment
docker-compose -f docker-compose.test.yml down -v

# Clean up temporary files
rm -f /tmp/test_restore.sql

# Record end time
END_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
```

#### 5. Documentation

Record test results:

```bash
# Calculate duration
DURATION=$(($(date -d "$END_TIME" +%s) - $(date -d "$START_TIME" +%s)))

# Create test report
cat > /tmp/restore_test_report.txt <<EOF
=== Monthly Restore Test Report ===
Test Date: $(date +%Y-%m-%d)
Backup File: $BACKUP_FILE
Start Time: $START_TIME
End Time: $END_TIME
Duration: ${DURATION}s

Results:
- Restore: SUCCESS
- Data Integrity: VERIFIED
- Constraints: VALID

RTO Target: 4 hours
Actual RTO: ${DURATION}s ($(($DURATION / 60)) minutes)

Notes:
- All critical tables restored successfully
- No orphaned records found
- All foreign key constraints valid

Next Test: $(date -d "+1 month" +%Y-%m-%d)
EOF

# Save report
cp /tmp/restore_test_report.txt /backups/postgres/logs/restore_test_$(date +%Y%m%d).txt
```

### Success Criteria

- ✅ Backup restores without errors
- ✅ All critical tables present
- ✅ Data integrity checks pass
- ✅ Foreign key constraints valid
- ✅ Restore completes within RTO (4 hours)

### Failure Response

If test fails:
1. Document failure details
2. Create incident ticket
3. Investigate root cause
4. Fix issues
5. Retest within 24 hours

---

## Quarterly Database Failover Test

**Objective**: Verify database failover to replica works correctly

**Duration**: 2 hours

**Prerequisites**:
- Database replication configured
- Replica database running
- Non-production environment

### Test Procedure

#### 1. Pre-Failover Checks (15 minutes)

```bash
# Check replication status
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT * FROM pg_stat_replication;"

# Check replication lag
docker exec erpgo-postgres-replica psql -U erpgo -d erp -c \
  "SELECT EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())) AS lag_seconds;"

# Record current data state
docker exec erpgo-postgres-primary psql -U erpgo -d erp -c \
  "SELECT COUNT(*) FROM users;" > /tmp/pre_failover_counts.txt
```

#### 2. Simulate Primary Failure (5 minutes)

```bash
# Stop primary database
docker-compose stop postgres-primary

# Verify primary is down
docker exec erpgo-postgres-primary pg_isready || echo "Primary is down"
```

#### 3. Promote Replica (10 minutes)

```bash
# Promote replica to primary
docker exec erpgo-postgres-replica pg_ctl promote -D /var/lib/postgresql/data

# Wait for promotion to complete
sleep 10

# Verify replica is now accepting writes
docker exec erpgo-postgres-replica psql -U erpgo -d erp -c \
  "SELECT pg_is_in_recovery();"  # Should return false
```

#### 4. Update Application Configuration (10 minutes)

```bash
# Update connection string to point to new primary
export POSTGRES_PRIMARY_HOST=postgres-replica

# Restart application services
docker-compose restart api worker

# Wait for services to be ready
sleep 30
```

#### 5. Verification (20 minutes)

```bash
# Verify application can connect
curl -f http://localhost:8080/health/ready

# Verify data consistency
docker exec erpgo-postgres-replica psql -U erpgo -d erp -c \
  "SELECT COUNT(*) FROM users;" > /tmp/post_failover_counts.txt

# Compare counts
diff /tmp/pre_failover_counts.txt /tmp/post_failover_counts.txt

# Test write operations
docker exec erpgo-postgres-replica psql -U erpgo -d erp -c \
  "INSERT INTO audit_logs (event_type, action, success) 
   VALUES ('test', 'failover_test', true);"

# Verify write succeeded
docker exec erpgo-postgres-replica psql -U erpgo -d erp -c \
  "SELECT * FROM audit_logs WHERE action = 'failover_test';"
```

#### 6. Restore Original Configuration (30 minutes)

```bash
# Start original primary
docker-compose start postgres-primary

# Configure as replica of new primary
# (This step depends on your replication setup)

# Or restore from backup and resync
```

#### 7. Documentation

Record failover test results:

```bash
cat > /tmp/failover_test_report.txt <<EOF
=== Quarterly Failover Test Report ===
Test Date: $(date +%Y-%m-%d)
Test Type: Database Failover

Timeline:
- Primary failure detected: [TIME]
- Replica promotion started: [TIME]
- Replica promotion completed: [TIME]
- Application reconnected: [TIME]
- Verification completed: [TIME]

Results:
- Failover: SUCCESS
- Data Loss: NONE
- Application Downtime: [DURATION]

RTO Target: 5 minutes
Actual RTO: [DURATION]

RPO Target: 0 minutes
Actual RPO: 0 minutes (no data loss)

Issues Encountered:
- [List any issues]

Lessons Learned:
- [Document learnings]

Next Test: $(date -d "+3 months" +%Y-%m-%d)
EOF
```

### Success Criteria

- ✅ Replica promoted successfully
- ✅ Application reconnects automatically
- ✅ No data loss
- ✅ Failover completes within RTO (5 minutes)
- ✅ Write operations work on new primary

---

## Quarterly Full DR Drill

**Objective**: Test complete disaster recovery process

**Duration**: 4 hours

**Prerequisites**:
- All team members available
- DR environment ready
- Communication channels established
- Backup files identified

### Test Procedure

#### Phase 1: Scenario Setup (30 minutes)

```bash
# Scenario: Complete data center failure
# All primary systems are down
# Must recover from backups

# Stop all services
docker-compose down

# Simulate data loss (in test environment only!)
# DO NOT RUN IN PRODUCTION
docker volume rm erpgo_postgres_data
```

#### Phase 2: Emergency Response (15 minutes)

1. Incident declared
2. Team assembled
3. Communication channels opened
4. Roles assigned:
   - Incident Commander
   - Database Recovery Lead
   - Application Recovery Lead
   - Communications Lead

#### Phase 3: Assessment (30 minutes)

```bash
# Identify last good backup
ls -lt /backups/postgres/*.sql* | head -5

# Calculate data loss window
BACKUP_TIME=$(stat -c %y "$BACKUP_FILE")
CURRENT_TIME=$(date)
echo "Data loss window: $BACKUP_TIME to $CURRENT_TIME"

# Verify backup integrity
./scripts/backup/database-backup.sh verify "$BACKUP_FILE"
```

#### Phase 4: Recovery Execution (2 hours)

```bash
# Follow disaster recovery procedures
export RECOVERY_BACKUP="$BACKUP_FILE"
export RECOVERY_TYPE="full"
export RECOVERY_TARGET="production"

# Execute recovery
./scripts/backup/disaster-recovery.sh
```

#### Phase 5: Verification (45 minutes)

```bash
# Verify all services running
docker-compose ps

# Check health endpoints
curl -f http://localhost:8080/health/ready

# Verify critical functionality
# - User authentication
# - Order creation
# - Product search
# - Payment processing

# Run smoke tests
./scripts/test-critical-paths.sh
```

#### Phase 6: Post-Drill Review (30 minutes)

Team discussion:
1. What went well?
2. What could be improved?
3. Were RTO/RPO met?
4. Any procedure updates needed?
5. Training gaps identified?

### Success Criteria

- ✅ All services recovered
- ✅ Data integrity verified
- ✅ Critical functionality working
- ✅ RTO met (4 hours)
- ✅ RPO met (1 hour)
- ✅ Team coordination effective
- ✅ Documentation accurate

---

## Annual Region Failover Test

**Objective**: Test complete region failover

**Duration**: 8 hours

**Prerequisites**:
- Secondary region infrastructure deployed
- Cross-region replication configured
- DNS failover capability
- All stakeholders notified

### Test Procedure

#### Phase 1: Planning (1 hour)

- Review failover procedures
- Verify secondary region readiness
- Establish communication channels
- Assign roles and responsibilities

#### Phase 2: Pre-Failover Verification (1 hour)

```bash
# Verify secondary region infrastructure
ssh secondary-region "docker-compose ps"

# Check replication lag
ssh secondary-region "docker exec postgres psql -c 'SELECT replication_lag;'"

# Verify DNS configuration
dig api.example.com

# Test secondary region connectivity
curl -f https://secondary-region.example.com/health
```

#### Phase 3: Failover Execution (2 hours)

```bash
# Update DNS to point to secondary region
# Promote secondary database
# Start services in secondary region
# Verify traffic routing

# Follow region failover procedures in DR document
```

#### Phase 4: Verification (2 hours)

```bash
# Verify all services in secondary region
# Test critical user journeys
# Monitor error rates and latency
# Verify data consistency
```

#### Phase 5: Failback (1 hour)

```bash
# Restore primary region
# Sync data from secondary to primary
# Update DNS back to primary
# Verify primary region operation
```

#### Phase 6: Post-Test Review (1 hour)

- Document timeline
- Calculate actual RTO/RPO
- Identify improvements
- Update procedures

### Success Criteria

- ✅ Secondary region activated successfully
- ✅ User traffic routed correctly
- ✅ No data loss
- ✅ RTO met (4 hours)
- ✅ RPO met (1 hour)
- ✅ Failback successful

---

## Test Metrics

### Key Metrics to Track

| Metric | Target | Measurement |
|--------|--------|-------------|
| Backup Success Rate | 100% | Daily automated backups |
| Restore Time | < 4 hours | Monthly restore tests |
| Failover Time | < 5 minutes | Quarterly failover tests |
| Data Loss | 0 records | All recovery tests |
| Test Success Rate | 100% | All scheduled tests |

### Reporting

Monthly report should include:
- Number of tests conducted
- Success/failure rate
- Average recovery times
- Issues identified
- Improvements implemented

---

## Continuous Improvement

### After Each Test

1. Document lessons learned
2. Update procedures if needed
3. Fix identified issues
4. Schedule follow-up tests
5. Share results with team

### Quarterly Review

1. Analyze test metrics
2. Review RTO/RPO targets
3. Update DR procedures
4. Conduct team training
5. Plan next quarter's tests

---

## Appendix

### Test Checklist Template

```markdown
## Recovery Test Checklist

**Test Date**: ___________
**Test Type**: ___________
**Tester**: ___________

### Pre-Test
- [ ] Backup identified
- [ ] Test environment ready
- [ ] Team notified
- [ ] Documentation reviewed

### During Test
- [ ] Start time recorded
- [ ] Procedures followed
- [ ] Issues documented
- [ ] End time recorded

### Post-Test
- [ ] Results verified
- [ ] Report created
- [ ] Issues logged
- [ ] Procedures updated

### Sign-Off
- [ ] Test successful
- [ ] Report reviewed
- [ ] Next test scheduled

**Tester Signature**: ___________
**Date**: ___________
```

### Test Report Template

```markdown
## Recovery Test Report

**Test ID**: RT-YYYY-MM-DD-##
**Test Date**: ___________
**Test Type**: ___________
**Tester**: ___________

### Objective
[Describe test objective]

### Procedure
[Summarize procedure followed]

### Results
- Start Time: ___________
- End Time: ___________
- Duration: ___________
- Status: SUCCESS / FAILURE

### Metrics
- RTO Target: ___________
- RTO Actual: ___________
- RPO Target: ___________
- RPO Actual: ___________

### Issues
[List any issues encountered]

### Recommendations
[List recommendations for improvement]

### Next Steps
[List follow-up actions]

**Approved By**: ___________
**Date**: ___________
```

---

**Document Version**: 1.0  
**Last Updated**: 2024-01-15  
**Next Review**: 2024-04-15  
**Owner**: DevOps Team
