# ERPGo Monitoring & Alerting Guide

## Overview

This guide covers the comprehensive monitoring and alerting strategy for ERPGo, including infrastructure monitoring, application performance monitoring (APM), log aggregation, and incident management procedures.

## Table of Contents

1. [Monitoring Architecture](#monitoring-architecture)
2. [Metrics Collection](#metrics-collection)
3. [Log Management](#log-management)
4. [Alerting Strategy](#alerting-strategy)
5. [Dashboard Configuration](#dashboard-configuration)
6. [Performance Monitoring](#performance-monitoring)
7. [Security Monitoring](#security-monitoring)
8. [Incident Management](#incident-management)
9. [Monitoring Setup](#monitoring-setup)
10. [Best Practices](#best-practices)

## Monitoring Architecture

### System Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Applications  │────│   Metrics       │────│   Prometheus    │
│   (ERPGo API)   │    │   Collection    │    │   (Time Series) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Infrastructure│────│   Log           │────│   Loki          │
│   (Docker/VM)   │    │   Aggregation   │    │   (Log Storage) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   External      │────│   Tracing       │────│   Jaeger        │
│   Services      │    │   Collection    │    │   (Distributed) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                       ┌─────────────────┐
                       │   Grafana       │
                       │   (Visualization)│
                       └─────────────────┘
                                │
                       ┌─────────────────┐
                       │   AlertManager  │
                       │   (Alerting)     │
                       └─────────────────┘
```

### Data Flow

1. **Metrics**: Applications → Prometheus → Grafana/AlertManager
2. **Logs**: Applications → Promtail → Loki → Grafana
3. **Traces**: Applications → Jaeger → Grafana
4. **Alerts**: Prometheus → AlertManager → Notification Channels

## Metrics Collection

### Application Metrics

#### Custom Metrics in Go

```go
// pkg/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // HTTP request counter
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "erpgo_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status_code"},
    )

    // Request duration histogram
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "erpgo_http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )

    // Database connections
    dbConnectionsActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "erpgo_database_connections_active",
            Help: "Number of active database connections",
        },
    )

    // Business metrics
    ordersCreated = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "erpgo_orders_created_total",
            Help: "Total number of orders created",
        },
    )

    usersActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "erpgo_users_active",
            Help: "Number of active users",
        },
    )
)

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint string, statusCode int, duration float64) {
    httpRequestsTotal.WithLabelValues(method, endpoint, fmt.Sprintf("%d", statusCode)).Inc()
    httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// UpdateDatabaseConnections updates database connection metrics
func UpdateDatabaseConnections(active, idle, max int) {
    dbConnectionsActive.Set(float64(active))
}
```

#### Database Metrics

```go
// pkg/metrics/database.go
package metrics

import (
    "database/sql"
    "time"
)

type DatabaseMetrics struct {
    db *sql.DB
}

func NewDatabaseMetrics(db *sql.DB) *DatabaseMetrics {
    return &DatabaseMetrics{db: db}
}

func (dm *DatabaseMetrics) Collect() {
    stats := dm.db.Stats()

    dbConnectionsActive.Set(float64(stats.OpenConnections))
    dbConnectionsIdle.Set(float64(stats.Idle))
    dbConnectionsInUse.Set(float64(stats.InUse))

    dbWaitDuration.Observe(stats.WaitDuration.Seconds())
    dbMaxIdleClosed.Inc(float64(stats.MaxIdleClosed))
    dbMaxIdleTimeClosed.Inc(float64(stats.MaxIdleTimeClosed))
}
```

#### Infrastructure Metrics

##### Node Exporter Metrics

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']
    scrape_interval: 15s
    metrics_path: /metrics
```

##### Docker Container Metrics

```yaml
  - job_name: 'docker-exporter'
    static_configs:
      - targets: ['docker-exporter:9323']
    scrape_interval: 30s
```

##### Database Exporter

```yaml
  - job_name: 'postgres-exporter'
    static_configs:
      - targets: ['postgres-exporter:9187']
    scrape_interval: 30s
```

### Prometheus Configuration

```yaml
# configs/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'erpgo-production'
    replica: 'prometheus-1'

rule_files:
  - "alert_rules.yml"
  - "recording_rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  # ERPGo Application
  - job_name: 'erpgo-api'
    static_configs:
      - targets: ['api:8080']
    metrics_path: /metrics
    scrape_interval: 15s
    scrape_timeout: 10s

  # Infrastructure
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

  - job_name: 'postgres-exporter'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis-exporter'
    static_configs:
      - targets: ['redis-exporter:9121']

  - job_name: 'docker-exporter'
    static_configs:
      - targets: ['docker-exporter:9323']

# Recording rules for better performance
recording_rules:
  - name: erpgo.rules
    rules:
      # Request rate
      - record: job:http_requests:rate5m
        expr: rate(erpgo_http_requests_total[5m])

      # Error rate
      - record: job:http_requests_errors:rate5m
        expr: rate(erpgo_http_requests_total{status_code=~"5.."}[5m])

      # Response time percentiles
      - record: job:http_request_duration_seconds:p95:5m
        expr: histogram_quantile(0.95, rate(erpgo_http_request_duration_seconds_bucket[5m]))

      # Database performance
      - record: job:db_connections_usage_percent
        expr: (erpgo_database_connections_active / erpgo_database_connections_max) * 100

      # Business metrics
      - record: business:orders_per_minute
        expr: rate(erpgo_orders_created_total[5m]) * 60

      - record: business:revenue_per_hour
        expr: rate(erpgo_revenue_total[5m]) * 3600
```

## Log Management

### Structured Logging with Zerolog

```go
// pkg/logger/logger.go
package logger

import (
    "os"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

type Logger struct {
    logger zerolog.Logger
}

func NewLogger(level string) *Logger {
    zerologTimeFieldFormat := zerolog.TimeFormatRFC3339

    l := zerolog.New(os.Stdout).
        With().
        Timestamp().
        Stack().
        Logger()

    switch level {
    case "debug":
        l = l.Level(zerolog.DebugLevel)
    case "info":
        l = l.Level(zerolog.InfoLevel)
    case "warn":
        l = l.Level(zerolog.WarnLevel)
    case "error":
        l = l.Level(zerolog.ErrorLevel)
    default:
        l = l.Level(zerolog.InfoLevel)
    }

    return &Logger{logger: l}
}

func (l *Logger) WithRequestID(requestID string) *Logger {
    return &Logger{
        logger: l.logger.With().Str("request_id", requestID).Logger(),
    }
}

func (l *Logger) WithUserID(userID string) *Logger {
    return &Logger{
        logger: l.logger.With().Str("user_id", userID).Logger(),
    }
}

func (l *Logger) Info() *zerolog.Event {
    return l.logger.Info()
}

func (l *Logger) Error() *zerolog.Event {
    return l.logger.Error()
}

func (l *Logger) Debug() *zerolog.Event {
    return l.logger.Debug()
}
```

### Log Configuration

```yaml
# configs/loki.yml
auth_enabled: false

server:
  http_listen_port: 3100

ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
    final_sleep: 0s
  chunk_idle_period: 1h
  max_chunk_age: 1h
  chunk_target_size: 1048576
  chunk_retain_period: 30s

schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

storage_config:
  boltdb_shipper:
    active_index_directory: /loki/boltdb-shipper-active
    cache_location: /loki/boltdb-shipper-cache
    shared_store: filesystem
  filesystem:
    directory: /loki/chunks

limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: 168h

chunk_store_config:
  max_look_back_period: 0s

table_manager:
  retention_deletes_enabled: false
  retention_period: 0s
```

### Promtail Configuration

```yaml
# configs/promtail.yml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  # Application logs
  - job_name: erpgo-logs
    static_configs:
      - targets:
          - localhost
        labels:
          job: erpgo-api
          __path__: /var/log/erpgo/*.log

    pipeline_stages:
      - json:
          expressions:
            level: level
            request_id: request_id
            user_id: user_id
            method: method
            endpoint: endpoint
            status_code: status_code
            duration: duration
            error: error

      - labels:
          level:
          method:
          endpoint:
          status_code:

      - timestamp:
          format: RFC3339
          source: time

  # Docker container logs
  - job_name: docker-logs
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        refresh_interval: 5s
        port: 8080

    pipeline_stages:
      - json:
          expressions:
            level: level
            request_id: request_id
            method: method
            endpoint: endpoint
            status_code: status_code

      - labels:
          level:
          method:
          endpoint:
          status_code:

      - timestamp:
          format: RFC3339
          source: time
```

## Alerting Strategy

### Alerting Rules

```yaml
# configs/alert_rules.yml
groups:
  - name: erpgo-application
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          (
            rate(erpgo_http_requests_total{status_code=~"5.."}[5m]) /
            rate(erpgo_http_requests_total[5m])
          ) > 0.05
        for: 5m
        labels:
          severity: critical
          service: erpgo-api
        annotations:
          summary: "High error rate detected in ERPGo API"
          description: |
            Error rate is {{ $value | humanizePercentage }} over the last 5 minutes.
            Current error rate: {{ $value | humanizePercentage }}
            Threshold: 5%

      # High response time
      - alert: HighResponseTime
        expr: |
          histogram_quantile(0.95,
            rate(erpgo_http_request_duration_seconds_bucket[5m])
          ) > 2
        for: 5m
        labels:
          severity: warning
          service: erpgo-api
        annotations:
          summary: "High response time detected"
          description: |
            95th percentile response time is {{ $value }}s.
            Threshold: 2s

      # Application down
      - alert: ApplicationDown
        expr: up{job="erpgo-api"} == 0
        for: 1m
        labels:
          severity: critical
          service: erpgo-api
        annotations:
          summary: "ERPGo API is down"
          description: "ERPGo API has been down for more than 1 minute"

      # High memory usage
      - alert: HighMemoryUsage
        expr: |
          (process_resident_memory_bytes / 1024 / 1024) > 500
        for: 5m
        labels:
          severity: warning
          service: erpgo-api
        annotations:
          summary: "High memory usage detected"
          description: |
            Memory usage is {{ $value }}MB.
            Threshold: 500MB

  - name: erpgo-database
    rules:
      # Database connection issues
      - alert: DatabaseConnectionHigh
        expr: |
          pg_stat_activity_count > 80
        for: 2m
        labels:
          severity: warning
          service: postgres
        annotations:
          summary: "High database connection count"
          description: |
            Database has {{ $value }} active connections.
            Threshold: 80

      # Slow queries
      - alert: SlowQueries
        expr: |
          pg_stat_statements_mean_time_seconds > 1000
        for: 5m
        labels:
          severity: warning
          service: postgres
        annotations:
          summary: "Slow database queries detected"
          description: |
            Average query time is {{ $value }}ms.
            Threshold: 1000ms

      # Disk space low
      - alert: DiskSpaceLow
        expr: |
          (node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100 < 10
        for: 5m
        labels:
          severity: critical
          service: infrastructure
        annotations:
          summary: "Low disk space"
          description: |
            Disk space is {{ $value }}% available.
            Threshold: 10%

  - name: erpgo-business
    rules:
      # No orders for extended period
      - alert: NoOrdersCreated
        expr: |
          increase(erpgo_orders_created_total[1h]) == 0
        for: 15m
        labels:
          severity: warning
          service: business
        annotations:
          summary: "No orders created in the last hour"
          description: |
            No orders have been created in the last hour.
            This might indicate a business process issue.

      # Order processing backlog
      - alert: OrderProcessingBacklog
        expr: |
          erpgo_orders_pending_count > 100
        for: 10m
        labels:
          severity: warning
          service: business
        annotations:
          summary: "High order processing backlog"
          description: |
            {{ $value }} orders are pending processing.
            Threshold: 100
```

### AlertManager Configuration

```yaml
# configs/alertmanager.yml
global:
  smtp_smarthost: 'smtp.example.com:587'
  smtp_from: 'alerts@erpgo.example.com'
  smtp_auth_username: 'alerts@erpgo.example.com'
  smtp_auth_password: 'smtp-password'

# Inhibition rules to prevent alert spam
inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'service']

# Routing configuration
route:
  group_by: ['alertname', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
    # Critical alerts go to on-call
    - match:
        severity: critical
      receiver: 'on-call'
      group_wait: 0s
      repeat_interval: 5m

    # Business alerts go to business team
    - match:
        service: business
      receiver: 'business-team'
      group_wait: 5m
      repeat_interval: 30m

    # Infrastructure alerts go to DevOps
    - match:
        service: infrastructure
      receiver: 'devops-team'
      group_wait: 5m
      repeat_interval: 15m

# Notification receivers
receivers:
  - name: 'default'
    email_configs:
      - to: 'team@erpgo.example.com'
        subject: '[ERPGo Alert] {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Labels: {{ range .Labels.SortedPairs }}{{ .Name }}={{ .Value }} {{ end }}
          {{ end }}

  - name: 'on-call'
    email_configs:
      - to: 'oncall@erpgo.example.com'
        subject: '[CRITICAL] ERPGo Alert: {{ .GroupLabels.alertname }}'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#alerts-critical'
        title: 'Critical ERPGo Alert'
        text: |
          {{ range .Alerts }}
          *Alert*: {{ .Annotations.summary }}
          *Description*: {{ .Annotations.description }}
          *Runbook*: https://docs.erpgo.example.com/runbooks/{{ .Labels.alertname }}
          {{ end }}

  - name: 'business-team'
    email_configs:
      - to: 'business-team@erpgo.example.com'
        subject: '[Business] ERPGo Alert: {{ .GroupLabels.alertname }}'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#business-alerts'
        title: 'Business Alert'
        text: |
          {{ range .Alerts }}
          *Alert*: {{ .Annotations.summary }}
          *Description*: {{ .Annotations.description }}
          {{ end }}

  - name: 'devops-team'
    email_configs:
      - to: 'devops@erpgo.example.com'
        subject: '[DevOps] ERPGo Alert: {{ .GroupLabels.alertname }}'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#devops-alerts'
        title: 'DevOps Alert'
        text: |
          {{ range .Alerts }}
          *Alert*: {{ .Annotations.summary }}
          *Description*: {{ .Annotations.description }}
          {{ end }}

# Templates for custom alert formatting
templates:
  - '/etc/alertmanager/templates/*.tmpl'
```

## Dashboard Configuration

### Grafana Dashboard for Application Metrics

```json
{
  "dashboard": {
    "title": "ERPGo Application Dashboard",
    "tags": ["erpgo", "application"],
    "timezone": "browser",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(erpgo_http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ],
        "yAxes": [
          {
            "label": "Requests/sec"
          }
        ]
      },
      {
        "title": "Response Time (95th percentile)",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(erpgo_http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ],
        "yAxes": [
          {
            "label": "Seconds"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(erpgo_http_requests_total{status_code=~\"5..\"}[5m]) / rate(erpgo_http_requests_total[5m])",
            "legendFormat": "Error Rate"
          }
        ],
        "yAxes": [
          {
            "label": "Percentage",
            "max": 1,
            "min": 0
          }
        ]
      },
      {
        "title": "Database Connections",
        "type": "singlestat",
        "targets": [
          {
            "expr": "erpgo_database_connections_active",
            "legendFormat": "Active Connections"
          }
        ]
      },
      {
        "title": "Orders Created",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(erpgo_orders_created_total[5m]) * 60",
            "legendFormat": "Orders/min"
          }
        ]
      },
      {
        "title": "Active Users",
        "type": "singlestat",
        "targets": [
          {
            "expr": "erpgo_users_active",
            "legendFormat": "Active Users"
          }
        ]
      }
    ]
  }
}
```

### Business Metrics Dashboard

```json
{
  "dashboard": {
    "title": "ERPGo Business Metrics",
    "tags": ["erpgo", "business"],
    "panels": [
      {
        "title": "Orders per Hour",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(erpgo_orders_created_total[1h]) * 3600",
            "legendFormat": "Orders/Hour"
          }
        ]
      },
      {
        "title": "Revenue per Day",
        "type": "graph",
        "targets": [
          {
            "expr": "increase(erpgo_revenue_total[1d])",
            "legendFormat": "Daily Revenue"
          }
        ]
      },
      {
        "title": "Order Status Distribution",
        "type": "piechart",
        "targets": [
          {
            "expr": "erpgo_orders_by_status",
            "legendFormat": "{{status}}"
          }
        ]
      },
      {
        "title": "Top Products",
        "type": "table",
        "targets": [
          {
            "expr": "topk(10, erpgo_product_orders)",
            "legendFormat": "{{product_name}}"
          }
        ]
      }
    ]
  }
}
```

## Performance Monitoring

### Application Performance Monitoring (APM)

#### Custom Tracing with Jaeger

```go
// pkg/tracing/tracing.go
package tracing

import (
    "context"
    "github.com/opentracing/opentracing"
    "github.com/uber/jaeger-client-go"
    jaegercfg "github.com/uber/jaeger-client-go/config"
)

func InitTracing(serviceName string) (opentracing.Tracer, io.Closer, error) {
    cfg := jaegercfg.Configuration{
        ServiceName: serviceName,
        Sampler: &jaegercfg.SamplerConfig{
            Type:  jaeger.SamplerTypeConst,
            Param: 1,
        },
        Reporter: &jaegercfg.ReporterConfig{
            LogSpans: true,
            LocalAgentHostPort: "jaeger:6831",
        },
    }

    tracer, closer, err := cfg.NewTracer()
    if err != nil {
        return nil, nil, err
    }

    return tracer, closer, nil
}

func StartSpan(operationName string, ctx context.Context) (opentracing.Span, context.Context) {
    span := opentracing.SpanFromContext(ctx)
    if span == nil {
        span = opentracing.StartSpan(operationName)
    } else {
        span = opentracing.StartSpan(operationName, opentracing.ChildOf(span.Context()))
    }

    return span, opentracing.ContextWithSpan(ctx, span)
}
```

#### Middleware for HTTP Tracing

```go
// internal/interfaces/http/middleware/tracing.go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/opentracing/opentracing-go"
    "github.com/opentracing/opentracing-go/ext"
)

func TracingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        span := opentracing.SpanFromContext(c.Request.Context())
        if span == nil {
            span = opentracing.StartSpan(c.Request.URL.Path)
        } else {
            span = opentracing.StartSpan(c.Request.URL.Path, opentracing.ChildOf(span.Context()))
        }
        defer span.Finish()

        ext.SpanKindRPCClient.Set(span)
        ext.HTTPUrl.Set(span, c.Request.URL.String())
        ext.HTTPMethod.Set(span, c.Request.Method)

        // Inject span context into headers
        span.Tracer().Inject(
            span.Context(),
            opentracing.TextMap,
            opentracing.HTTPHeadersCarrier(c.Request.Header),
        )

        c.Next()
    }
}
```

### Database Performance Monitoring

```go
// pkg/metrics/database_metrics.go
package metrics

import (
    "context"
    "database/sql"
    "time"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    dbQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "erpgo_database_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"query_type", "table"},
    )

    dbQueryErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "erpgo_database_query_errors_total",
            Help: "Total number of database query errors",
        },
        []string{"query_type", "table", "error_type"},
    )
)

func InstrumentDB(db *sql.DB) *sql.DB {
    return &instrumentedDB{db: db}
}

type instrumentedDB struct {
    db *sql.DB
}

func (idb *instrumentedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    start := time.Now()
    table, queryType := parseQuery(query)

    rows, err := idb.db.QueryContext(ctx, query, args...)

    duration := time.Since(start).Seconds()
    dbQueryDuration.WithLabelValues(queryType, table).Observe(duration)

    if err != nil {
        dbQueryErrors.WithLabelValues(queryType, table, getErrorType(err)).Inc()
    }

    return rows, err
}
```

## Security Monitoring

### Security Metrics

```go
// pkg/metrics/security_metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    // Authentication metrics
    authAttempts = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "erpgo_auth_attempts_total",
            Help: "Total number of authentication attempts",
        },
        []string{"result"}, // success, failure
    )

    authFailuresByIP = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "erpgo_auth_failures_by_ip_total",
            Help: "Authentication failures by IP address",
        },
        []string{"ip"},
    )

    // Authorization metrics
    authzDenials = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "erpgo_authz_denials_total",
            Help: "Total number of authorization denials",
        },
        []string{"resource", "action"},
    )

    // Security events
    suspiciousActivities = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "erpgo_suspicious_activities_total",
            Help: "Total number of suspicious activities detected",
        },
        []string{"activity_type"},
    )
)
```

### Security Alerting Rules

```yaml
# security-alert-rules.yml
groups:
  - name: security
    rules:
      # Brute force attack detection
      - alert: BruteForceAttack
        expr: |
          rate(erpgo_auth_attempts_total{result="failure"}[5m]) > 10
        for: 2m
        labels:
          severity: critical
          category: security
        annotations:
          summary: "Potential brute force attack detected"
          description: |
            High rate of authentication failures detected.
            Failure rate: {{ $value }} attempts/minute

      # Suspicious IP activity
      - alert: SuspiciousIPActivity
        expr: |
          rate(erpgo_auth_failures_by_ip_total[5m]) > 5
        for: 1m
        labels:
          severity: warning
          category: security
        annotations:
          summary: "Suspicious activity from IP address"
          description: |
            High failure rate from IP {{ $labels.ip }}.
            Failure rate: {{ $value }} attempts/minute

      # Authorization failures
      - alert: HighAuthorizationFailures
        expr: |
          rate(erpgo_authz_denials_total[5m]) > 20
        for: 3m
        labels:
          severity: warning
          category: security
        annotations:
          summary: "High authorization failure rate"
          description: |
            Authorization failure rate is {{ $value }} attempts/minute.
            This might indicate a security issue or misconfiguration.
```

## Incident Management

### Incident Response Workflow

1. **Detection**
   - Automated alerts from monitoring systems
   - User reports
   - Manual monitoring

2. **Triage**
   - Assess severity and impact
   - Determine affected systems
   - Notify appropriate teams

3. **Investigation**
   - Review logs and metrics
   - Identify root cause
   - Document findings

4. **Resolution**
   - Implement fix
   - Verify resolution
   - Restore services

5. **Post-Incident**
   - Write incident report
   - Update monitoring/alerting
   - Implement preventive measures

### Incident Severity Levels

| Severity | Description | Response Time | Example |
|----------|-------------|---------------|---------|
| Critical | System down, major impact | 15 minutes | Complete outage |
| High | Significant degradation | 1 hour | Performance issues |
| Medium | Limited impact | 4 hours | Feature unavailable |
| Low | Minor issues | 24 hours | Non-critical bugs |

### On-Call Procedures

```yaml
# on-call-handbook.yml
procedures:
  critical_alert:
    steps:
      - "Acknowledge alert within 5 minutes"
      - "Join incident bridge"
      - "Assess impact and scope"
      - "Implement immediate mitigation"
      - "Communicate with stakeholders"
      - "Document actions taken"

  performance_degradation:
    steps:
      - "Check system resources (CPU, memory, disk)"
      - "Review application metrics"
      - "Analyze database performance"
      - "Check external dependencies"
      - "Implement scaling if needed"

  security_incident:
    steps:
      - "Assess security impact"
      - "Contain the threat"
      - "Preserve evidence"
      - "Notify security team"
      - "Follow security playbooks"
```

## Monitoring Setup

### Docker Compose Monitoring Stack

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: erpgo-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./configs/prometheus.yml:/etc/prometheus/prometheus.yml
      - ./configs/alert_rules.yml:/etc/prometheus/alert_rules.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=30d'
      - '--web.enable-lifecycle'
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: erpgo-grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./configs/grafana/provisioning:/etc/grafana/provisioning
      - ./configs/grafana/dashboards:/var/lib/grafana/dashboards
    networks:
      - monitoring

  alertmanager:
    image: prom/alertmanager:latest
    container_name: erpgo-alertmanager
    ports:
      - "9093:9093"
    volumes:
      - ./configs/alertmanager.yml:/etc/alertmanager/alertmanager.yml
      - alertmanager_data:/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
    networks:
      - monitoring

  loki:
    image: grafana/loki:latest
    container_name: erpgo-loki
    ports:
      - "3100:3100"
    volumes:
      - ./configs/loki.yml:/etc/loki/local-config.yaml
      - loki_data:/loki
    command: -config.file=/etc/loki/local-config.yaml
    networks:
      - monitoring

  promtail:
    image: grafana/promtail:latest
    container_name: erpgo-promtail
    volumes:
      - ./configs/promtail.yml:/etc/promtail/config.yml
      - ./logs:/var/log/erpgo:ro
      - /var/log:/var/log:ro
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
    command: -config.file=/etc/promtail/config.yml
    networks:
      - monitoring

  node-exporter:
    image: prom/node-exporter:latest
    container_name: erpgo-node-exporter
    ports:
      - "9100:9100"
    command:
      - '--path.rootfs=/host'
    volumes:
      - /:/host:ro,rslave
    networks:
      - monitoring

  cadvisor:
    image: gcr.io/cadvisor/cadvisor:latest
    container_name: erpgo-cadvisor
    ports:
      - "8081:8080"
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:ro
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro
      - /dev/disk/:/dev/disk:ro
    privileged: true
    devices:
      - /dev/kmsg
    networks:
      - monitoring

volumes:
  prometheus_data:
  grafana_data:
  alertmanager_data:
  loki_data:

networks:
  monitoring:
    driver: bridge
```

### Setup Commands

```bash
#!/bin/bash
# setup-monitoring.sh

echo "Setting up ERPGo monitoring stack..."

# Create monitoring directory
mkdir -p monitoring/{configs,data,logs}

# Copy configuration files
cp configs/prometheus.yml monitoring/configs/
cp configs/alert_rules.yml monitoring/configs/
cp configs/alertmanager.yml monitoring/configs/
cp configs/loki.yml monitoring/configs/
cp configs/promtail.yml monitoring/configs/

# Create Grafana provisioning directories
mkdir -p monitoring/configs/grafana/{provisioning/{datasources,dashboards},dashboards}

# Create Grafana datasource configuration
cat > monitoring/configs/grafana/provisioning/datasources/prometheus.yml <<EOF
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true

  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
EOF

# Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

echo "Monitoring stack started!"
echo "Grafana: http://localhost:3001 (admin/admin)"
echo "Prometheus: http://localhost:9090"
echo "AlertManager: http://localhost:9093"
```

## Best Practices

### Monitoring Best Practices

1. **SLI/SLO Definition**
   - Define Service Level Indicators (SLIs)
   - Set Service Level Objectives (SLOs)
   - Monitor error budgets

2. **Alert Design**
   - Alerts should be actionable
   - Include runbook links
   - Use severity levels appropriately
   - Avoid alert fatigue

3. **Dashboard Design**
   - Use clear, descriptive titles
   - Include relevant context
   - Use consistent color schemes
   - Add annotations for important events

4. **Metrics Design**
   - Use consistent naming conventions
   - Include relevant labels
   - Avoid high cardinality labels
   - Document metric definitions

### Performance Monitoring Tips

1. **Application Performance**
   - Monitor request latency percentiles
   - Track error rates by endpoint
   - Monitor resource utilization
   - Set up synthetic monitoring

2. **Database Performance**
   - Monitor connection pool usage
   - Track slow queries
   - Monitor index usage
   - Watch for lock contention

3. **Infrastructure Performance**
   - Monitor CPU, memory, disk, network
   - Track container resource usage
   - Monitor auto-scaling events
   - Set capacity alerts

### Security Monitoring Best Practices

1. **Authentication Monitoring**
   - Track login success/failure rates
   - Monitor for brute force attacks
   - Track suspicious IP addresses
   - Monitor password reset requests

2. **Authorization Monitoring**
   - Track access denied events
   - Monitor privilege escalation attempts
   - Track unusual access patterns
   - Monitor API key usage

3. **Data Protection**
   - Monitor data access patterns
   - Track data export events
   - Monitor sensitive data access
   - Set up data loss prevention alerts

---

**Note**: This monitoring setup provides comprehensive visibility into the ERPGo system. Regular review and updates of monitoring configurations ensure continued effectiveness and relevance.