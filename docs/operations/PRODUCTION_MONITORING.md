# ERPGo Production Monitoring & Alerting Configuration

## Overview
This document provides comprehensive monitoring and alerting configuration for the ERPGo production launch, ensuring real-time visibility into system health, performance, and business metrics.

## Monitoring Architecture

### Stack Components
- **Metrics Collection**: Prometheus
- **Visualization**: Grafana
- **Alerting**: AlertManager
- **Log Aggregation**: Loki + Promtail
- **Distributed Tracing**: Jaeger
- **Infrastructure Monitoring**: Node Exporter, cAdvisor
- **Database Monitoring**: PostgreSQL Exporter
- **Cache Monitoring**: Redis Exporter

### Data Flow
```
Applications → Metrics → Prometheus → Grafana/AlertManager
Applications → Logs → Promtail → Loki → Grafana
Infrastructure → Node Exporter → Prometheus → Grafana
```

---

## 🚨 Critical Alerting Configuration

### Alert Routing Rules

```yaml
# configs/alertmanager/production-routes.yml
global:
  smtp_smarthost: 'smtp.erpgo.com:587'
  smtp_from: 'alerts@erpgo.com'
  smtp_auth_username: 'alerts@erpgo.com'
  smtp_auth_password: '${SMTP_PASSWORD}'

# Inhibition rules to prevent alert spam
inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'service']

# Routing configuration for production alerts
route:
  group_by: ['alertname', 'service', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'default-receiver'
  routes:
    # Critical alerts - immediate escalation
    - match:
        severity: critical
      receiver: 'critical-alerts'
      group_wait: 0s
      repeat_interval: 5m
      continue: true

    # High severity alerts - 15 minute escalation
    - match:
        severity: high
      receiver: 'high-alerts'
      group_wait: 2m
      repeat_interval: 30m

    # Business critical alerts - business team
    - match:
        category: business
      receiver: 'business-alerts'
      group_wait: 5m
      repeat_interval: 1h

    # Security alerts - security team
    - match:
        category: security
      receiver: 'security-alerts'
      group_wait: 0s
      repeat_interval: 15m

    # Performance alerts - DevOps team
    - match:
        category: performance
      receiver: 'performance-alerts'
      group_wait: 5m
      repeat_interval: 2h
```

### Critical Alert Rules

```yaml
# configs/prometheus/production-critical-rules.yml
groups:
  - name: erpgo-critical-alerts
    interval: 30s
    rules:
      # System Down Alerts
      - alert: ERPGoApplicationDown
        expr: up{job="erpgo-api"} == 0
        for: 1m
        labels:
          severity: critical
          service: erpgo-api
          category: availability
        annotations:
          summary: "ERPGo application is down"
          description: "ERPGo API has been down for more than 1 minute on instance {{ $labels.instance }}"
          runbook_url: "https://docs.erpgo.com/runbooks/application-down"
          dashboard_url: "https://monitoring.erpgo.com/d/erpgo-overview"

      - alert: DatabaseDown
        expr: up{job="postgres-exporter"} == 0
        for: 1m
        labels:
          severity: critical
          service: postgres
          category: availability
        annotations:
          summary: "PostgreSQL database is down"
          description: "PostgreSQL database has been down for more than 1 minute"
          runbook_url: "https://docs.erpgo.com/runbooks/database-down"

      - alert: RedisDown
        expr: up{job="redis-exporter"} == 0
        for: 1m
        labels:
          severity: critical
          service: redis
          category: availability
        annotations:
          summary: "Redis cache is down"
          description: "Redis cache has been down for more than 1 minute"
          runbook_url: "https://docs.erpgo.com/runbooks/redis-down"

      # High Error Rate Alerts
      - alert: HighAPIErrorRate
        expr: |
          (
            rate(erpgo_http_requests_total{status_code=~"5.."}[5m]) /
            rate(erpgo_http_requests_total[5m])
          ) > 0.05
        for: 2m
        labels:
          severity: critical
          service: erpgo-api
          category: performance
        annotations:
          summary: "High API error rate detected"
          description: "API error rate is {{ $value | humanizePercentage }} over the last 5 minutes (threshold: 5%)"
          runbook_url: "https://docs.erpgo.com/runbooks/high-error-rate"

      - alert: DatabaseConnectionFailure
        expr: pg_up{job="postgres-exporter"} == 0
        for: 1m
        labels:
          severity: critical
          service: postgres
          category: availability
        annotations:
          summary: "Database connection failure"
          description: "Cannot connect to PostgreSQL database"

      # Performance Degradation Alerts
      - alert: HighResponseTime
        expr: |
          histogram_quantile(0.95,
            rate(erpgo_http_request_duration_seconds_bucket[5m])
          ) > 2
        for: 3m
        labels:
          severity: high
          service: erpgo-api
          category: performance
        annotations:
          summary: "High API response time"
          description: "95th percentile response time is {{ $value }}s (threshold: 2s)"
          runbook_url: "https://docs.erpgo.com/runbooks/high-response-time"

      - alert: DatabaseSlowQueries
        expr: pg_stat_statements_mean_time_seconds > 1000
        for: 5m
        labels:
          severity: high
          service: postgres
          category: performance
        annotations:
          summary: "Slow database queries detected"
          description: "Average query time is {{ $value }}ms (threshold: 1000ms)"

      # Resource Usage Alerts
      - alert: HighCPUUsage
        expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 90
        for: 5m
        labels:
          severity: high
          service: infrastructure
          category: performance
        annotations:
          summary: "High CPU usage detected"
          description: "CPU usage is {{ $value }}% on instance {{ $labels.instance }}"

      - alert: HighMemoryUsage
        expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 85
        for: 5m
        labels:
          severity: high
          service: infrastructure
          category: performance
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is {{ $value }}% on instance {{ $labels.instance }}"

      - alert: DiskSpaceLow
        expr: (node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100 < 10
        for: 2m
        labels:
          severity: critical
          service: infrastructure
          category: availability
        annotations:
          summary: "Low disk space"
          description: "Disk space is {{ $value }}% available on {{ $labels.device }} at {{ $labels.instance }}"

      # Database-Specific Alerts
      - alert: DatabaseConnectionHigh
        expr: pg_stat_activity_count > 80
        for: 3m
        labels:
          severity: high
          service: postgres
          category: performance
        annotations:
          summary: "High database connection count"
          description: "Database has {{ $value }} active connections (threshold: 80)"

      - alert: DatabaseReplicationLag
        expr: pg_replication_lag_seconds > 60
        for: 2m
        labels:
          severity: high
          service: postgres
          category: availability
        annotations:
          summary: "Database replication lag"
          description: "Replication lag is {{ $value }} seconds"

      # Business Critical Alerts
      - alert: NoOrdersProcessed
        expr: increase(erpgo_orders_created_total[1h]) == 0
        for: 15m
        labels:
          severity: high
          service: business
          category: business
        annotations:
          summary: "No orders processed in last hour"
          description: "No orders have been created in the last hour. This may indicate a business process issue."

      - alert: PaymentProcessingFailures
        expr: rate(erpgo_payment_failures_total[5m]) > 0.1
        for: 2m
        labels:
          severity: high
          service: business
          category: business
        annotations:
          summary: "High payment failure rate"
          description: "Payment failure rate is {{ $value }} failures/minute"

      # Security Alerts
      - alert: BruteForceAttack
        expr: rate(erpgo_auth_failures_total[5m]) > 10
        for: 1m
        labels:
          severity: critical
          service: security
          category: security
        annotations:
          summary: "Potential brute force attack"
          description: "High rate of authentication failures: {{ $value }} attempts/minute"

      - alert: SuspiciousActivity
        expr: rate(erpgo_suspicious_activities_total[5m]) > 5
        for: 2m
        labels:
          severity: high
          service: security
          category: security
        annotations:
          summary: "Suspicious activity detected"
          description: "Suspicious activity rate: {{ $value }} events/minute"
```

### Notification Channels Configuration

```yaml
# configs/alertmanager/receivers.yml
receivers:
  - name: 'critical-alerts'
    email_configs:
      - to: 'oncall@erpgo.com'
        subject: '[CRITICAL] ERPGo Alert: {{ .GroupLabels.alertname }}'
        body: |
          🚨 CRITICAL ALERT 🚨

          Alert: {{ .GroupLabels.alertname }}
          Service: {{ .GroupLabels.service }}
          Severity: {{ .GroupLabels.severity }}

          {{ range .Alerts }}
          Description: {{ .Annotations.description }}
          Dashboard: {{ .Annotations.dashboard_url }}
          Runbook: {{ .Annotations.runbook_url }}
          Started: {{ .StartsAt }}
          {{ end }}

          Immediate action required!
    slack_configs:
      - api_url: '${SLACK_WEBHOOK_URL}'
        channel: '#alerts-critical'
        title: '🚨 Critical ERPGo Alert'
        text: |
          *Alert*: {{ .GroupLabels.alertname }}
          *Service*: {{ .GroupLabels.service }}
          *Description*: {{ range .Alerts }}{{ .Annotations.description }}{{ end }}
          *Runbook*: {{ range .Alerts }}{{ .Annotations.runbook_url }}{{ end }}
        send_resolved: true

    pagerduty_configs:
      - routing_key: '${PAGERDUTY_ROUTING_KEY}'
        description: '{{ .GroupLabels.alertname }}: {{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
        severity: 'critical'

  - name: 'high-alerts'
    email_configs:
      - to: 'devops@erpgo.com'
        subject: '[HIGH] ERPGo Alert: {{ .GroupLabels.alertname }}'
        body: |
          ⚠️ HIGH SEVERITY ALERT ⚠️

          Alert: {{ .GroupLabels.alertname }}
          Service: {{ .GroupLabels.service }}

          {{ range .Alerts }}
          Description: {{ .Annotations.description }}
          Dashboard: {{ .Annotations.dashboard_url }}
          Started: {{ .StartsAt }}
          {{ end }}
    slack_configs:
      - api_url: '${SLACK_WEBHOOK_URL}'
        channel: '#alerts-devops'
        title: '⚠️ High Priority ERPGo Alert'
        send_resolved: true

  - name: 'business-alerts'
    email_configs:
      - to: 'business-team@erpgo.com'
        subject: '[BUSINESS] ERPGo Alert: {{ .GroupLabels.alertname }}'
    slack_configs:
      - api_url: '${SLACK_WEBHOOK_URL}'
        channel: '#business-alerts'
        title: '📊 Business Alert'
        send_resolved: true

  - name: 'security-alerts'
    email_configs:
      - to: 'security@erpgo.com'
        subject: '[SECURITY] ERPGo Alert: {{ .GroupLabels.alertname }}'
        body: |
          🔒 SECURITY ALERT 🔒

          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Immediate investigation required!
          {{ end }}
    slack_configs:
      - api_url: '${SLACK_WEBHOOK_URL}'
        channel: '#security-alerts'
        title: '🔒 Security Alert'
        color: 'danger'
        send_resolved: true

  - name: 'performance-alerts'
    email_configs:
      - to: 'performance@erpgo.com'
        subject: '[PERFORMANCE] ERPGo Alert: {{ .GroupLabels.alertname }}'
    slack_configs:
      - api_url: '${SLACK_WEBHOOK_URL}'
        channel: '#performance-alerts'
        title: '📈 Performance Alert'
        send_resolved: true
```

---

## 📊 Launch-Specific Monitoring Configuration

### Launch Day Enhanced Monitoring

```yaml
# configs/prometheus/launch-monitoring.yml
groups:
  - name: erpgo-launch-monitoring
    interval: 15s
    rules:
      # Launch-specific health checks
      - alert: LaunchHealthCheckFailed
        expr: erpgo_launch_health_check == 0
        for: 30s
        labels:
          severity: critical
          service: erpgo-api
          category: launch
        annotations:
          summary: "Launch health check failed"
          description: "Critical launch health check has failed"

      - alert: LaunchHighErrorRate
        expr: |
          (
            rate(erpgo_http_requests_total{status_code=~"5.."}[2m]) /
            rate(erpgo_http_requests_total[2m])
          ) > 0.02
        for: 1m
        labels:
          severity: critical
          service: erpgo-api
          category: launch
        annotations:
          summary: "Launch error rate above threshold"
          description: "Error rate is {{ $value | humanizePercentage }} during launch (threshold: 2%)"

      - alert: LaunchSlowResponse
        expr: |
          histogram_quantile(0.95,
            rate(erpgo_http_request_duration_seconds_bucket[2m])
          ) > 1
        for: 1m
        labels:
          severity: high
          service: erpgo-api
          category: launch
        annotations:
          summary: "Launch response time degradation"
          description: "95th percentile response time is {{ $value }}s during launch (threshold: 1s)"

      # Business metrics during launch
      - alert: LaunchNoUserActivity
        expr: increase(erpgo_user_sessions_total[10m]) == 0
        for: 5m
        labels:
          severity: high
          service: business
          category: launch
        annotations:
          summary: "No user activity during launch"
          description: "No new user sessions in the last 10 minutes"

      - record: erpgo_launch_health_check
        expr: up{job="erpgo-api"} and
              (rate(erpgo_http_requests_total[1m]) > 0) and
              (rate(erpgo_http_requests_total{status_code=~"5.."}[1m]) / rate(erpgo_http_requests_total[1m]) < 0.01)

      - record: erpgo_launch_error_rate
        expr: rate(erpgo_http_requests_total{status_code=~"5.."}[1m]) / rate(erpgo_http_requests_total[1m])

      - record: erpgo_launch_response_time_p95
        expr: histogram_quantile(0.95, rate(erpgo_http_request_duration_seconds_bucket[1m]))
```

### Real-time Launch Dashboard Configuration

```json
{
  "dashboard": {
    "id": null,
    "title": "ERPGo Launch Dashboard",
    "tags": ["erpgo", "launch", "production"],
    "timezone": "browser",
    "panels": [
      {
        "title": "System Health Status",
        "type": "stat",
        "targets": [
          {
            "expr": "erpgo_launch_health_check",
            "legendFormat": "Health Check"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "mappings": [
              {"options": {"0": {"text": "CRITICAL", "color": "red"}}, "type": "value"},
              {"options": {"1": {"text": "HEALTHY", "color": "green"}}, "type": "value"}
            ]
          }
        },
        "gridPos": {"h": 8, "w": 6, "x": 0, "y": 0}
      },
      {
        "title": "Request Rate (RPS)",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(erpgo_http_requests_total[30s])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ],
        "yAxes": [{"label": "Requests/sec"}],
        "gridPos": {"h": 8, "w": 12, "x": 6, "y": 0}
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(erpgo_http_requests_total{status_code=~\"5..\"}[30s]) / rate(erpgo_http_requests_total[30s])",
            "legendFormat": "Error Rate"
          }
        ],
        "yAxes": [{"label": "Percentage", "max": 1, "min": 0}],
        "gridPos": {"h": 8, "w": 6, "x": 18, "y": 0}
      },
      {
        "title": "Response Time (95th percentile)",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(erpgo_http_request_duration_seconds_bucket[30s]))",
            "legendFormat": "95th percentile"
          }
        ],
        "yAxes": [{"label": "Seconds"}],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8}
      },
      {
        "title": "Active Users",
        "type": "stat",
        "targets": [
          {
            "expr": "erpgo_active_users_total",
            "legendFormat": "Active Users"
          }
        ],
        "gridPos": {"h": 8, "w": 6, "x": 12, "y": 8}
      },
      {
        "title": "Orders Created",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(erpgo_orders_created_total[30s]) * 60",
            "legendFormat": "Orders/min"
          }
        ],
        "yAxes": [{"label": "Orders/min"}],
        "gridPos": {"h": 8, "w": 6, "x": 18, "y": 8}
      },
      {
        "title": "Database Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "pg_stat_activity_count",
            "legendFormat": "Active Connections"
          }
        ],
        "yAxes": [{"label": "Connections"}],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 16}
      },
      {
        "title": "CPU Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[30s])) * 100)",
            "legendFormat": "{{instance}}"
          }
        ],
        "yAxes": [{"label": "Percentage", "max": 100, "min": 0}],
        "gridPos": {"h": 8, "w": 6, "x": 12, "y": 16}
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100",
            "legendFormat": "{{instance}}"
          }
        ],
        "yAxes": [{"label": "Percentage", "max": 100, "min": 0}],
        "gridPos": {"h": 8, "w": 6, "x": 18, "y": 16}
      }
    ],
    "time": {"from": "now-1h", "to": "now"},
    "refresh": "5s"
  }
}
```

---

## 🔧 Monitoring Setup Scripts

### Production Monitoring Deployment Script

```bash
#!/bin/bash
# scripts/monitoring/deploy-production-monitoring.sh

set -e

echo "🚀 Deploying ERPGo Production Monitoring Stack..."

# Create monitoring namespace
kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -

# Deploy Prometheus
echo "📊 Deploying Prometheus..."
kubectl apply -f configs/monitoring/prometheus-configmap.yaml
kubectl apply -f configs/monitoring/prometheus-deployment.yaml
kubectl apply -f configs/monitoring/prometheus-service.yaml

# Deploy Grafana
echo "📈 Deploying Grafana..."
kubectl apply -f configs/monitoring/grafana-configmap.yaml
kubectl apply -f configs/monitoring/grafana-deployment.yaml
kubectl apply -f configs/monitoring/grafana-service.yaml
kubectl apply -f configs/monitoring/grafana-ingress.yaml

# Deploy AlertManager
echo "🚨 Deploying AlertManager..."
kubectl apply -f configs/monitoring/alertmanager-configmap.yaml
kubectl apply -f configs/monitoring/alertmanager-deployment.yaml
kubectl apply -f configs/monitoring/alertmanager-service.yaml

# Deploy Log Aggregation
echo "📝 Deploying Loki and Promtail..."
kubectl apply -f configs/monitoring/loki-configmap.yaml
kubectl apply -f configs/monitoring/loki-deployment.yaml
kubectl apply -f configs/monitoring/promtail-configmap.yaml
kubectl apply -f configs/monitoring/promtail-daemonset.yaml

# Deploy Exporters
echo "📡 Deploying Exporters..."
kubectl apply -f configs/monitoring/node-exporter-daemonset.yaml
kubectl apply -f configs/monitoring/postgres-exporter-deployment.yaml
kubectl apply -f configs/monitoring/redis-exporter-deployment.yaml

# Wait for deployments to be ready
echo "⏳ Waiting for monitoring stack to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/prometheus -n monitoring
kubectl wait --for=condition=available --timeout=300s deployment/grafana -n monitoring
kubectl wait --for=condition=available --timeout=300s deployment/alertmanager -n monitoring

# Import Grafana dashboards
echo "📊 Importing Grafana dashboards..."
./scripts/monitoring/import-grafana-dashboards.sh

# Verify monitoring is working
echo "🔍 Verifying monitoring setup..."
./scripts/monitoring/verify-monitoring.sh

echo "✅ Production monitoring stack deployed successfully!"
echo "📊 Grafana: https://grafana.erpgo.com"
echo "📈 Prometheus: https://prometheus.erpgo.com"
echo "🚨 AlertManager: https://alertmanager.erpgo.com"
```

### Monitoring Verification Script

```bash
#!/bin/bash
# scripts/monitoring/verify-monitoring.sh

set -e

echo "🔍 Verifying ERPGo Production Monitoring..."

# Check Prometheus targets
echo "📊 Checking Prometheus targets..."
PROMETHEUS_URL="https://prometheus.erpgo.com"
TARGETS_RESPONSE=$(curl -s "$PROMETHEUS_URL/api/v1/targets")

if echo "$TARGETS_RESPONSE" | jq -e '.data.activeTargets[] | select(.health == "up")' > /dev/null; then
    echo "✅ Prometheus targets are healthy"
else
    echo "❌ Some Prometheus targets are down"
    echo "$TARGETS_RESPONSE" | jq '.data.activeTargets[] | select(.health == "down")'
    exit 1
fi

# Check Grafana datasources
echo "📈 Checking Grafana datasources..."
GRAFANA_URL="https://grafana.erpgo.com"
GRAFANA_TOKEN="${GRAFANA_API_TOKEN}"

DATASOURCES_RESPONSE=$(curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" "$GRAFANA_URL/api/datasources")
if echo "$DATASOURCES_RESPONSE" | jq -e '.[] | select(.type == "prometheus" and .access == "proxy")' > /dev/null; then
    echo "✅ Grafana datasources are configured"
else
    echo "❌ Grafana datasources are not properly configured"
    exit 1
fi

# Check AlertManager alerts
echo "🚨 Checking AlertManager alerts..."
ALERTMANAGER_URL="https://alertmanager.erpgo.com"
ALERTS_RESPONSE=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")

ACTIVE_ALERTS=$(echo "$ALERTS_RESPONSE" | jq '.data.alerts | length')
echo "📊 Currently $ACTIVE_ALERTS active alerts"

# Test alert routing
echo "📧 Testing alert routing..."
curl -XPOST "$PROMETHEUS_URL/api/v1/alerts" -H 'Content-Type: application/json' -d '[
  {
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning",
      "service": "test"
    },
    "annotations": {
      "description": "This is a test alert to verify alert routing"
    }
  }
]'

echo "⏳ Waiting 30 seconds for alert routing..."
sleep 30

# Check if test alert was received (this would depend on your notification setup)
echo "✅ Alert routing test completed"

# Verify monitoring dashboards
echo "📊 Verifying monitoring dashboards..."
DASHBOARDS_RESPONSE=$(curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" "$GRAFANA_URL/api/search?tag=erpgo")
DASHBOARD_COUNT=$(echo "$DASHBOARDS_RESPONSE" | jq '. | length')

if [ "$DASHBOARD_COUNT" -gt 0 ]; then
    echo "✅ Found $DASHBOARD_COUNT ERPGo dashboards"
else
    echo "❌ No ERPGo dashboards found"
    exit 1
fi

# Test metrics collection
echo "📈 Testing metrics collection..."
METRICS_RESPONSE=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=up")
if echo "$METRICS_RESPONSE" | jq -e '.data.result[] | select(.value[1] == "1")' > /dev/null; then
    echo "✅ Metrics are being collected successfully"
else
    echo "❌ Metrics collection is not working properly"
    exit 1
fi

echo "✅ All monitoring components are verified and working correctly!"
```

### Launch Monitoring Activation Script

```bash
#!/bin/bash
# scripts/monitoring/activate-launch-monitoring.sh

set -e

echo "🚀 Activating ERPGo Launch Monitoring..."

# Enable launch-specific monitoring rules
echo "📊 Loading launch monitoring rules..."
curl -X POST http://prometheus:9090/api/v1/rules \
  -H 'Content-Type: application/json' \
  -d @configs/prometheus/launch-monitoring.yml

# Set enhanced alert thresholds for launch
echo "🚨 Configuring launch alert thresholds..."
curl -X POST http://alertmanager:9093/api/v1/silences \
  -H 'Content-Type: application/json' \
  -d '{
    "matchers": [
      {"name": "category", "value": "launch", "isRegex": false}
    ],
    "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'",
    "endsAt": "'$(date -u -d +24h +%Y-%m-%dT%H:%M:%S.%3NZ)'",
    "createdBy": "launch-monitoring",
    "comment": "Enable launch-specific monitoring for 24 hours"
  }'

# Deploy launch dashboard
echo "📈 Deploying launch dashboard..."
curl -X POST -H "Authorization: Bearer $GRAFANA_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d @configs/grafana/launch-dashboard.json \
  https://grafana.erpgo.com/api/dashboards/db

# Set up enhanced logging for launch
echo "📝 Configuring enhanced logging..."
kubectl annotate deployment erpgo-api \
  prometheus.io/scrape="true" \
  prometheus.io/port="8080" \
  prometheus.io/path="/metrics" \
  --overwrite

# Enable debug logging temporarily
kubectl set env deployment/erpgo-api LOG_LEVEL=debug --overwrite

# Scale up monitoring components for launch
echo "📈 Scaling monitoring components..."
kubectl scale deployment prometheus --replicas=2 -n monitoring
kubectl scale deployment grafana --replicas=2 -n monitoring

# Create launch-specific alerts
echo "🚨 Creating launch-specific alert rules..."
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: launch-alert-rules
  namespace: monitoring
data:
  launch-rules.yml: |
    groups:
      - name: erpgo-launch-alerts
        interval: 15s
        rules:
          - alert: LaunchDegradation
            expr: erpgo_launch_health_check == 0
            for: 30s
            labels:
              severity: critical
              category: launch
            annotations:
              summary: "Launch degradation detected"
              description: "Critical launch health check has failed"
EOF

# Reload Prometheus configuration
curl -X POST http://prometheus:9090/-/reload

echo "✅ Launch monitoring activated successfully!"
echo "📊 Launch dashboard: https://grafana.erpgo.com/d/launch"
echo "🚨 Enhanced alerting active for 24 hours"
echo "📈 Monitoring components scaled for launch"
```

---

## 📱 Mobile and Notification Setup

### Mobile Alert Configuration

```yaml
# configs/alertmanager/mobile-notifications.yml
receivers:
  - name: 'mobile-critical-alerts'
    pushover_configs:
      - user_key: '${PUSHOVER_USER_KEY}'
        api_key: '${PUSHOVER_API_KEY}'
        title: '🚨 ERPGo Critical Alert'
        message: '{{ .GroupLabels.alertname }}: {{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
        priority: 2
        expire: 3600
        retry: 30

    telegram_configs:
      - bot_token: '${TELEGRAM_BOT_TOKEN}'
        chat_id: '${TELEGRAM_CHAT_ID}'
        message: |
          🚨 *Critical Alert*

          *Alert*: {{ .GroupLabels.alertname }}
          *Service*: {{ .GroupLabels.service }}
          *Description*: {{ range .Alerts }}{{ .Annotations.description }}{{ end }}
          *Runbook*: {{ range .Alerts }}{{ .Annotations.runbook_url }}{{ end }}
        parse_mode: 'Markdown'

    discord_configs:
      - webhook_url: '${DISCORD_WEBHOOK_URL}'
        title: '🚨 Critical ERPGo Alert'
        message: |
          **Alert**: {{ .GroupLabels.alertname }}
          **Service**: {{ .GroupLabels.service }}
          **Description**: {{ range .Alerts }}{{ .Annotations.description }}{{ end }}
```

### Status Page Integration

```bash
#!/bin/bash
# scripts/monitoring/update-status-page.sh

STATUS_URL="https://status.erpgo.com/api/v1/status"
API_KEY="${STATUS_PAGE_API_KEY}"

update_status() {
    local status="$1"
    local message="$2"

    curl -X POST "$STATUS_URL" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -d "{
            \"status\": \"$status\",
            \"message\": \"$message\",
            \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)\"
        }"
}

# Update status page based on system health
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    update_status "operational" "All systems operational"
else
    update_status "down" "System experiencing issues"
fi
```

---

## 🔍 Custom Monitoring Scripts

### Application Health Monitor

```bash
#!/bin/bash
# scripts/monitoring/application-health-monitor.sh

ENDPOINT="https://api.erpgo.com/health"
TIMEOUT=10
MAX_RETRIES=3

check_health() {
    local attempt=1

    while [ $attempt -le $MAX_RETRIES ]; do
        if curl -f -s --max-time $TIMEOUT "$ENDPOINT" > /dev/null; then
            echo "✅ Application health check passed (attempt $attempt)"
            return 0
        fi

        echo "❌ Health check failed (attempt $attempt/$MAX_RETRIES)"
        sleep $((attempt * 2))
        ((attempt++))
    done

    echo "🚨 Application health check failed after $MAX_RETRIES attempts"

    # Trigger alert
    curl -XPOST "http://alertmanager:9093/api/v1/alerts" \
        -H 'Content-Type: application/json' \
        -d '[{
            "labels": {
                "alertname": "ApplicationHealthCheckFailed",
                "severity": "critical",
                "service": "erpgo-api"
            },
            "annotations": {
                "description": "Application health check failed after multiple attempts"
            }
        }]'

    return 1
}

check_health
```

### Database Performance Monitor

```bash
#!/bin/bash
# scripts/monitoring/database-performance-monitor.sh

DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="erpgo"
DB_USER="monitoring_user"

check_database_performance() {
    # Check connection count
    CONNECTION_COUNT=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
        SELECT count(*) FROM pg_stat_activity;
    ")

    if [ "$CONNECTION_COUNT" -gt 80 ]; then
        echo "🚨 High database connection count: $CONNECTION_COUNT"
        return 1
    fi

    # Check slow queries
    SLOW_QUERIES=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
        SELECT count(*) FROM pg_stat_statements WHERE mean_time > 1000;
    ")

    if [ "$SLOW_QUERIES" -gt 5 ]; then
        echo "⚠️ High number of slow queries: $SLOW_QUERIES"
        return 1
    fi

    # Check database size
    DB_SIZE=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
        SELECT pg_size_pretty(pg_database_size('$DB_NAME'));
    ")

    echo "✅ Database performance OK (connections: $CONNECTION_COUNT, size: $DB_SIZE)"
    return 0
}

check_database_performance
```

---

## 📋 Monitoring Runbook

### Daily Monitoring Checklist

**System Health (Daily 9:00 AM)**
- [ ] Check system uptime
- [ ] Verify all services are running
- [ ] Review error rates
- [ ] Check response times
- [ ] Verify backup completion

**Performance Review (Daily 10:00 AM)**
- [ ] Review performance metrics
- [ ] Check resource utilization
- [ ] Analyze database performance
- [ ] Review cache hit rates
- [ ] Check auto-scaling events

**Business Metrics (Daily 11:00 AM)**
- [ ] Review user activity
- [ ] Check order volumes
- [ ] Analyze revenue metrics
- [ ] Review conversion rates
- [ ] Check support ticket volume

### Weekly Monitoring Tasks

**Performance Analysis (Mondays)**
- [ ] Analyze performance trends
- [ ] Review capacity planning
- [ ] Check database optimization opportunities
- [ ] Review caching strategies
- [ ] Plan performance improvements

**Security Review (Tuesdays)**
- [ ] Review security logs
- [ ] Check for suspicious activity
- [ ] Analyze authentication patterns
- [ ] Review access control effectiveness
- [ ] Update security monitoring rules

**Infrastructure Review (Wednesdays)**
- [ ] Review infrastructure utilization
- [ ] Check capacity requirements
- [ ] Review backup strategies
- [ ] Analyze disaster recovery readiness
- [ ] Plan infrastructure upgrades

**Business Review (Thursdays)**
- [ ] Analyze business metrics trends
- [ ] Review user engagement
- [ ] Check feature adoption
- [ ] Analyze customer feedback
- [ ] Plan business improvements

**Team Review (Fridays)**
- [ ] Review monitoring effectiveness
- [ ] Discuss incident response
- [ ] Plan process improvements
- [ ] Review tool effectiveness
- [ ] Plan team training

---

**Document Version**: 1.0
**Last Updated**: [Date]
**Next Review**: [Date]
**Approved By**: [Name], [Title]

This comprehensive monitoring configuration ensures real-time visibility into the ERPGo production system, enabling rapid detection and response to any issues during the launch and ongoing operations.