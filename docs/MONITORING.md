# ERPGo Monitoring & Observability Guide

## Overview

This guide provides comprehensive documentation for the monitoring and observability stack of ERPGo, including Prometheus, Grafana, Loki, and AlertManager configurations.

## Architecture

The monitoring stack consists of the following components:

- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **Loki**: Log aggregation and storage
- **Promtail**: Log collection agent
- **AlertManager**: Alert routing and notification
- **Node Exporter**: System metrics collection
- **PostgreSQL Exporter**: Database metrics
- **Redis Exporter**: Cache metrics

## Table of Contents

1. [Prometheus Configuration](#prometheus-configuration)
2. [Grafana Dashboards](#grafana-dashboards)
3. [Log Aggregation](#log-aggregation)
4. [Alerting Rules](#alerting-rules)
5. [Monitoring Services](#monitoring-services)
6. [Troubleshooting](#troubleshooting)
7. [Maintenance](#maintenance)

## Prometheus Configuration

### Configuration File: `configs/prometheus.yml`

Prometheus is configured to scrape metrics from multiple sources:

#### Key Scrape Configurations

**ERPGo Application Metrics**
```yaml
- job_name: 'erpgo-api'
  static_configs:
    - targets: ['api:8080']
  metrics_path: '/metrics'
  scrape_interval: 15s
  scrape_timeout: 10s
```

**PostgreSQL Database Metrics**
```yaml
- job_name: 'postgres-exporter'
  static_configs:
    - targets: ['postgres-exporter:9187']
  scrape_interval: 30s
```

**Redis Cache Metrics**
```yaml
- job_name: 'redis-exporter'
  static_configs:
    - targets: ['redis-exporter:9121']
  scrape_interval: 30s
```

**System Metrics**
```yaml
- job_name: 'node-exporter'
  static_configs:
    - targets: ['node-exporter:9100']
  scrape_interval: 30s
```

### Recording Rules

Prometheus uses recording rules for efficient querying:

- **Request Rate**: `job:http_requests:rate5m`
- **Error Rate**: `job:http_requests_errors:rate5m`
- **Response Duration**: `job:http_request_duration_seconds:mean5m`
- **Database Connection Pool**: `job:db_connections_pool_usage`
- **Cache Hit Rate**: `job:cache_hit_rate`

### Performance Tuning

- **Scrape Interval**: 15s for application metrics, 30s for infrastructure
- **Retention**: 15 days default
- **Storage**: Local TSDB with remote write capability

## Grafana Dashboards

### Available Dashboards

1. **System Performance Dashboard** (`erpgo-system-performance`)
   - HTTP request rates
   - Response time percentiles
   - Memory usage
   - Goroutine count
   - Database connections
   - Error rates
   - Cache hit rates

2. **Database Performance Dashboard** (`erpgo-database-performance`)
   - Connection pool usage
   - Query rates
   - Query response times
   - Slow query rate
   - Database errors
   - Memory usage

3. **Business Metrics Dashboard** (`erpgo-business-metrics`)
   - Order creation rates
   - Revenue metrics
   - User activity
   - Product engagement
   - Order and cart value distribution

4. **Security Metrics Dashboard** (`erpgo-security-metrics`)
   - Authentication failures
   - Rate limit violations
   - Blocked requests
   - Threat detection
   - JWT token management

### Dashboard Access

- **URL**: http://localhost:3000
- **Default Credentials**: admin/admin (change on first login)
- **Data Sources**: Prometheus and Loki

### Customizing Dashboards

1. Navigate to the desired dashboard
2. Click "Edit" (pencil icon)
3. Modify panels as needed
4. Save changes

## Log Aggregation

### Loki Configuration

**Configuration File**: `configs/loki.yml`

Loki is configured for:
- **Storage**: Local filesystem
- **Schema**: v11 with boltdb-shipper
- **Indexing**: 24-hour periods
- **Replication**: Single instance (factor: 1)

### Promtail Configuration

**Configuration File**: `configs/promtail.yml`

Promtail collects logs from multiple sources:

#### Log Sources

1. **ERPGo Application Logs**
   - Path: `/var/log/erpgo/*.log`
   - Format: JSON structured logging
   - Labels: level, component, correlation_id, trace_id, user_id

2. **System Logs**
   - Path: `/var/log/syslog`
   - Format: Syslog format
   - Labels: host, process

3. **Nginx Access Logs**
   - Path: `/var/log/nginx/access.log`
   - Format: Combined log format
   - Labels: method, status, remote_addr

4. **Nginx Error Logs**
   - Path: `/var/log/nginx/error.log`
   - Format: Nginx error log format
   - Labels: level

5. **PostgreSQL Logs**
   - Path: `/var/log/postgresql/*.log`
   - Format: PostgreSQL log format
   - Labels: level

6. **Redis Logs**
   - Path: `/var/log/redis/redis-server.log`
   - Format: Redis log format
   - Labels: role

7. **Docker Container Logs**
   - Source: Docker socket
   - Labels: container, stream

### Log Querying

Use LogQL (Log Query Language) to search logs:

**Basic Query**:
```logql
{job="erpgo-app"} |= "error"
```

**With Labels**:
```logql
{job="erpgo-app", level="error", component="auth"}
```

**Time Range**:
```logql
{job="erpgo-app"}[5m]
```

**Aggregation**:
```logql
count_over_time({job="erpgo-app"}[5m])
```

## Alerting Rules

### AlertManager Configuration

**Configuration File**: `configs/alertmanager.yml`

#### Alert Routing

- **Critical Alerts**: Immediate notification to admin and ops teams
- **Warning Alerts**: Standard notification to dev team
- **Default**: Basic notification to admin

#### Notification Channels

1. **Email**: Primary notification channel
   - SMTP server: localhost:587
   - From: alerts@erpgo.local

### Alert Rules

**Configuration File**: `configs/alert_rules.yml`

#### System Health Alerts

- **High Error Rate**: > 5% for 2 minutes (warning), > 10% for 1 minute (critical)
- **High Response Time**: > 1s for 3 minutes (warning), > 2s for 1 minute (critical)
- **Memory Usage**: > 80% for 5 minutes (warning), > 95% for 2 minutes (critical)
- **Goroutine Count**: > 1000 for 5 minutes (warning), > 5000 for 2 minutes (critical)

#### Database Alerts

- **High Connection Count**: > 20 for 5 minutes (warning), > 50 for 2 minutes (critical)
- **Slow Queries**: 95th percentile > 0.5s for 3 minutes (warning)
- **Database Errors**: Any errors detected

#### Cache Alerts

- **Low Hit Rate**: < 80% for 10 minutes (warning), < 50% for 5 minutes (critical)

#### Business Metrics Alerts

- **Low Order Rate**: < 0.1 orders/hour for 15 minutes (warning)
- **High Cancellation Rate**: > 20% for 10 minutes (warning)
- **Low User Activity**: < 0.5 logins/hour for 30 minutes (warning)

#### Security Alerts

- **High Auth Failures**: > 5 per minute for 3 minutes (warning), > 20 per minute for 1 minute (critical)
- **Low Stock Alerts**: > 0.1 alerts per minute for 5 minutes (warning)

### Custom Alert Rules

Create custom alerts by adding to `configs/alert_rules.yml`:

```yaml
- alert: CustomAlert
  expr: your_metric_expression
  for: duration
  labels:
    severity: warning
    service: your-service
  annotations:
    summary: "Alert summary"
    description: "Alert description"
```

## Monitoring Services

### Service Health Checks

All monitoring services should be healthy:

```bash
# Check Prometheus
curl http://localhost:9090/-/healthy

# Check Grafana
curl http://localhost:3000/api/health

# Check Loki
curl http://localhost:3100/ready

# Check AlertManager
curl http://localhost:9093/-/healthy
```

### Service Ports

| Service | Port | Purpose |
|---------|------|---------|
| ERPGo API | 8080 | Application metrics |
| Prometheus | 9090 | Metrics collection |
| Grafana | 3000 | Visualization |
| Loki | 3100 | Log aggregation |
| Promtail | 9080 | Log collection |
| AlertManager | 9093 | Alert routing |
| Node Exporter | 9100 | System metrics |
| PostgreSQL Exporter | 9187 | Database metrics |
| Redis Exporter | 9121 | Cache metrics |

### Metrics Reference

#### Application Metrics

**HTTP Metrics**
- `erpgo_http_requests_total`: Total HTTP requests by status code
- `erpgo_http_request_duration_seconds`: Request duration histogram
- `erpgo_http_request_size_bytes`: Request body size
- `erpgo_http_response_size_bytes`: Response body size

**Database Metrics**
- `erpgo_database_connections`: Connection pool statistics
- `erpgo_database_queries_total`: Query counters by type
- `erpgo_database_query_duration_seconds`: Query duration histogram
- `erpgo_database_errors_total`: Database error counters
- `erpgo_database_timeouts_total`: Database timeout counters

**Cache Metrics**
- `erpgo_cache_operations_total`: Cache operation counters
- `erpgo_cache_hit_rate`: Cache hit ratio
- `erpgo_cache_size_bytes`: Cache memory usage

**System Metrics**
- `erpgo_system_memory_bytes`: Memory usage statistics
- `erpgo_goroutines`: Goroutine count
- `erpgo_gc_duration_seconds`: Garbage collection duration

**Business Metrics**
- `erpgo_orders_created_total`: Order creation counters
- `erpgo_orders_completed_total`: Order completion counters
- `erpgo_orders_cancelled_total`: Order cancellation counters
- `erpgo_user_registrations_total`: User registration counters
- `erpgo_user_logins_total`: User login counters
- `erpgo_revenue_total`: Revenue counters

**Security Metrics**
- `erpgo_auth_failures_total`: Authentication failure counters
- `erpgo_auth_successes_total`: Authentication success counters
- `erpgo_rate_limit_violations_total`: Rate limit violation counters
- `erpgo_blocked_requests_total`: Blocked request counters
- `erpgo_security_events_total`: Security event counters

## Troubleshooting

### Common Issues

#### Prometheus Not Scraping Metrics

**Symptoms**: No data in dashboards
**Solutions**:
1. Check target status: http://localhost:9090/targets
2. Verify metrics endpoint is accessible: `curl http://localhost:8080/metrics`
3. Check network connectivity between Prometheus and targets
4. Review Prometheus logs

#### Grafana Not Showing Data

**Symptoms**: Dashboards show "No Data"
**Solutions**:
1. Verify data source configuration
2. Test Prometheus connection in Grafana
3. Check if Prometheus is actually collecting metrics
4. Verify time range in dashboard

#### Loki Not Receiving Logs

**Symptoms**: No logs in Grafana Explore
**Solutions**:
1. Check Promtail logs: `docker logs promtail`
2. Verify log file paths and permissions
3. Test Promtail configuration: `promtail -config.file=promtail.yml -dry-run`
4. Check Loki ingestion: `curl http://localhost:3100/loki/api/v1/push`

#### AlertManager Not Sending Alerts

**Symptoms**: No email notifications
**Solutions**:
1. Check AlertManager UI: http://localhost:9093
2. Verify SMTP configuration
3. Test email delivery manually
4. Check alert rule syntax

### Performance Issues

#### High Memory Usage in Prometheus

**Causes**:
- Too many metrics
- High cardinality labels
- Long retention periods

**Solutions**:
- Reduce scrape intervals
- Drop unnecessary metrics
- Implement metric filtering
- Adjust retention settings

#### Slow Grafana Dashboards

**Causes**:
- Complex queries
- Large time ranges
- Too many panels

**Solutions**:
- Optimize queries
- Use recording rules
- Reduce time ranges
- Implement caching

### Log Analysis

#### Finding Errors

```logql
{job="erpgo-app"} |= "error" | logfmt
```

#### Authentication Issues

```logql
{job="erpgo-app", component="auth"} | logfmt
```

#### Database Issues

```logql
{job="postgresql"} |= "ERROR" | logfmt
```

#### Nginx Issues

```logql
{job="nginx-error"} | logfmt
```

## Maintenance

### Daily Tasks

1. **Check Dashboard Health**: Ensure all dashboards are displaying data
2. **Review Alerts**: Acknowledge and resolve any active alerts
3. **Log Review**: Check for error patterns in application logs
4. **Performance Check**: Monitor response times and error rates

### Weekly Tasks

1. **Metric Review**: Analyze trends in key metrics
2. **Alert Tuning**: Adjust alert thresholds based on usage patterns
3. **Capacity Planning**: Review resource utilization trends
4. **Backup Verification**: Ensure monitoring data backups are working

### Monthly Tasks

1. **Retention Management**: Clean up old metrics and logs
2. **Performance Optimization**: Review and optimize slow queries
3. **Dashboard Updates**: Update dashboards based on new requirements
4. **Documentation Review**: Update monitoring documentation

### Backup and Recovery

#### Prometheus Data Backup

```bash
# Backup Prometheus data
docker exec prometheus tar -czf - /prometheus > prometheus-backup.tar.gz

# Restore Prometheus data
docker exec -i prometheus tar -xzf - < prometheus-backup.tar.gz
```

#### Grafana Dashboard Backup

```bash
# Export all dashboards
curl -u admin:admin http://localhost:3000/api/search\?type=dash-db | \
  jq -r '.[] | .uri' | while read dashboard; do
    curl -u admin:admin "http://localhost:3000/api/dashboards/db/$dashboard" | \
      jq > "backup-$(basename $dashboard).json"
  done
```

#### Configuration Backup

```bash
# Backup monitoring configurations
tar -czf monitoring-configs-backup.tar.gz configs/
```

### Scaling Considerations

#### Horizontal Scaling

- **Prometheus**: Use federation for multi-cluster setups
- **Loki**: Use distributed mode for high volume
- **Grafana**: Use multiple instances behind load balancer

#### Resource Planning

- **Prometheus**: 2GB RAM + 50GB disk per 1M samples/sec
- **Loki**: 4GB RAM + 100GB disk per 10GB logs/day
- **Grafana**: 1GB RAM + small disk for dashboards

### Security Considerations

- **Authentication**: Enable authentication for Grafana and Prometheus
- **Authorization**: Implement role-based access control
- **Network Security**: Use firewalls to restrict access
- **Data Encryption**: Encrypt sensitive monitoring data
- **Audit Logging**: Enable audit logging for all monitoring components

## Conclusion

This monitoring stack provides comprehensive observability for ERPGo, enabling proactive detection of issues and performance optimization. Regular maintenance and monitoring of the monitoring infrastructure itself ensures reliable operation.

For additional support or questions, refer to the [ERPGo documentation](../README.md) or create an issue in the project repository.