package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

// DashboardConfig holds configuration for the performance dashboard
type DashboardConfig struct {
	Enabled    bool   `json:"enabled"`
	Port       int    `json:"port"`
	Path       string `json:"path"`
	RefreshInterval time.Duration `json:"refresh_interval"`
	HistorySize int    `json:"history_size"`
}

// PerformanceData represents current performance metrics
type PerformanceData struct {
	Timestamp          time.Time              `json:"timestamp"`
	QueryMetrics       QueryMetrics           `json:"query_metrics"`
	ConnectionStats    ConnectionStats        `json:"connection_stats"`
	CacheStats         CacheStats             `json:"cache_stats"`
	SlowQueries        []SlowQuery            `json:"slow_queries"`
	TableSizes         []TableSize            `json:"table_sizes"`
	IndexUsage         []IndexUsage           `json:"index_usage"`
	QueryPatterns      []QueryPattern         `json:"query_patterns"`
	SystemMetrics      SystemMetrics          `json:"system_metrics"`
}

// QueryMetrics holds query performance metrics
type QueryMetrics struct {
	TotalQueries       int64         `json:"total_queries"`
	QueriesPerSecond   float64       `json:"queries_per_second"`
	AvgResponseTime    time.Duration `json:"avg_response_time"`
	MaxResponseTime    time.Duration `json:"max_response_time"`
	MinResponseTime    time.Duration `json:"min_response_time"`
	P95ResponseTime    time.Duration `json:"p95_response_time"`
	P99ResponseTime    time.Duration `json:"p99_response_time"`
	ErrorRate          float64       `json:"error_rate"`
	SlowQueriesCount   int64         `json:"slow_queries_count"`
}

// ConnectionStats holds connection pool statistics
type ConnectionStats struct {
	ActiveConnections  int `json:"active_connections"`
	IdleConnections    int `json:"idle_connections"`
	TotalConnections   int `json:"total_connections"`
	MaxConnections     int `json:"max_connections"`
	ConnectionUtilization float64 `json:"connection_utilization"`
}

// CacheStats holds cache performance statistics
type CacheStats struct {
	Hits               int64   `json:"hits"`
	Misses             int64   `json:"misses"`
	Sets               int64   `json:"sets"`
	HitRate            float64 `json:"hit_rate"`
	MissRate           float64 `json:"miss_rate"`
	Evictions          int64   `json:"evictions"`
	MemoryUsage        int64   `json:"memory_usage"`
}

// SlowQuery represents a slow query entry
type SlowQuery struct {
	Query              string        `json:"query"`
	CallCount          int64         `json:"call_count"`
	TotalTime          time.Duration `json:"total_time"`
	MeanTime           time.Duration `json:"mean_time"`
	Rows               int64         `json:"rows"`
	HitPercent         float64       `json:"hit_percent"`
	LastExecuted       time.Time     `json:"last_executed"`
}

// TableSize represents table size information
type TableSize struct {
	TableName          string `json:"table_name"`
	TotalSize          string `json:"total_size"`
	IndexSize          string `json:"index_size"`
	TableSize          string `json:"table_size"`
	RowCount           int64  `json:"row_count"`
}

// IndexUsage represents index usage statistics
type IndexUsage struct {
	SchemaName         string `json:"schema_name"`
	TableName          string `json:"table_name"`
	IndexName          string `json:"index_name"`
	IdxScan            int64  `json:"idx_scan"`
	IdxTupRead         int64  `json:"idx_tup_read"`
	IdxTupFetch        int64  `json:"idx_tup_fetch"`
	UsagePercent       float64 `json:"usage_percent"`
}

// SystemMetrics holds system-level performance metrics
type SystemMetrics struct {
	CPUUsage           float64 `json:"cpu_usage"`
	MemoryUsage        int64   `json:"memory_usage"`
	DiskUsage          float64 `json:"disk_usage"`
	NetworkIO          int64   `json:"network_io"`
	DatabaseCPUTime    float64 `json:"database_cpu_time"`
}

// PerformanceDashboard provides a web-based performance monitoring interface
type PerformanceDashboard struct {
	config         *DashboardConfig
	db             *Database
	perfDB         *PerformanceDB
	queryCache     QueryCache
	logger         *zerolog.Logger
	history        []PerformanceData
	metrics        *PerformanceMetrics
	server         *http.Server
}

// NewPerformanceDashboard creates a new performance dashboard
func NewPerformanceDashboard(config *DashboardConfig, db *Database, perfDB *PerformanceDB, queryCache QueryCache, logger *zerolog.Logger) *PerformanceDashboard {
	if logger == nil {
		l := zerolog.Nop()
		logger = &l
	}

	if config.Path == "" {
		config.Path = "/db-dashboard"
	}
	if config.RefreshInterval <= 0 {
		config.RefreshInterval = 5 * time.Second
	}
	if config.HistorySize <= 0 {
		config.HistorySize = 100
	}

	dashboard := &PerformanceDashboard{
		config:     config,
		db:         db,
		perfDB:     perfDB,
		queryCache: queryCache,
		logger:     logger,
		history:    make([]PerformanceData, 0, config.HistorySize),
	}

	if perfDB != nil {
		dashboard.metrics = perfDB.metrics
	}

	return dashboard
}

// Start starts the dashboard server
func (pd *PerformanceDashboard) Start() error {
	if !pd.config.Enabled {
		pd.logger.Info().Msg("Performance dashboard disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc(pd.config.Path, pd.handleDashboard)
	mux.HandleFunc(pd.config.Path+"/api", pd.handleAPI)
	mux.Handle(pd.config.Path+"/metrics", promhttp.Handler())

	// Add static file serving for assets
	mux.Handle(pd.config.Path+"/static/", http.StripPrefix(pd.config.Path, http.FileServer(http.Dir("./web/static"))))

	pd.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", pd.config.Port),
		Handler: mux,
	}

	// Start data collection
	go pd.collectDataLoop()

	pd.logger.Info().
		Int("port", pd.config.Port).
		Str("path", pd.config.Path).
		Msg("Performance dashboard started")

	return pd.server.ListenAndServe()
}

// Stop stops the dashboard server
func (pd *PerformanceDashboard) Stop() error {
	if pd.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return pd.server.Shutdown(ctx)
	}
	return nil
}

// collectDataLoop continuously collects performance data
func (pd *PerformanceDashboard) collectDataLoop() {
	ticker := time.NewTicker(pd.config.RefreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		data := pd.collectCurrentData()
		pd.addToHistory(data)
	}
}

// collectCurrentData gathers current performance metrics
func (pd *PerformanceDashboard) collectCurrentData() PerformanceData {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data := PerformanceData{
		Timestamp: time.Now(),
	}

	// Collect query metrics
	if pd.perfDB != nil {
		patterns := pd.perfDB.GetQueryPatterns()
		data.QueryMetrics = pd.calculateQueryMetrics(patterns)
		data.QueryPatterns = pd.formatQueryPatterns(patterns)
	}

	// Collect connection stats
	data.ConnectionStats = pd.collectConnectionStats()

	// Collect cache stats
	data.CacheStats = pd.collectCacheStats()

	// Collect slow queries
	slowQueries, _ := pd.collectSlowQueries(ctx)
	data.SlowQueries = slowQueries

	// Collect table sizes
	tableSizes, _ := pd.collectTableSizes(ctx)
	data.TableSizes = tableSizes

	// Collect index usage
	indexUsage, _ := pd.collectIndexUsage(ctx)
	data.IndexUsage = indexUsage

	// Collect system metrics
	data.SystemMetrics = pd.collectSystemMetrics()

	return data
}

// calculateQueryMetrics calculates query performance metrics
func (pd *PerformanceDashboard) calculateQueryMetrics(patterns map[string]*PerformanceQueryPattern) QueryMetrics {
	metrics := QueryMetrics{}

	var totalTime time.Duration
	var count int64
	var maxTime, minTime time.Duration

	minTime = time.Hour // Initialize with high value

	for _, pattern := range patterns {
		count += pattern.Count
		totalTime += pattern.TotalTime
		if pattern.MaxTime > maxTime {
			maxTime = pattern.MaxTime
		}
		if pattern.MinTime < minTime {
			minTime = pattern.MinTime
		}
	}

	if count > 0 {
		metrics.TotalQueries = count
		metrics.AvgResponseTime = totalTime / time.Duration(count)
		metrics.MaxResponseTime = maxTime
		metrics.MinResponseTime = minTime

		// Calculate queries per second based on recent activity
		// This is a simplified calculation
		metrics.QueriesPerSecond = float64(count) / 60.0 // Assuming 1 minute window
	}

	// Collect slow queries count
	if pd.metrics != nil {
		// Prometheus counters don't have .Get() method
		// For now, return a placeholder value
		metrics.SlowQueriesCount = 0
	}

	return metrics
}

// collectConnectionStats collects connection pool statistics
func (pd *PerformanceDashboard) collectConnectionStats() ConnectionStats {
	stats := ConnectionStats{}

	if pd.db != nil {
		poolStats := pd.db.Stats()
		stats.ActiveConnections = int(poolStats.AcquiredConns())
		stats.IdleConnections = int(poolStats.IdleConns())
		stats.TotalConnections = int(poolStats.TotalConns())
		stats.MaxConnections = int(poolStats.MaxConns())

		if stats.MaxConnections > 0 {
			stats.ConnectionUtilization = float64(stats.ActiveConnections) / float64(stats.MaxConnections) * 100
		}
	}

	return stats
}

// collectCacheStats collects cache performance statistics
func (pd *PerformanceDashboard) collectCacheStats() CacheStats {
	stats := CacheStats{}

	if pd.metrics != nil {
		// Prometheus counters don't have .Get() method
		// For now, return placeholder values
		stats.Hits = 0
		stats.Misses = 0
		stats.Sets = 0

		total := stats.Hits + stats.Misses
		if total > 0 {
			stats.HitRate = float64(stats.Hits) / float64(total) * 100
			stats.MissRate = float64(stats.Misses) / float64(total) * 100
		}
	}

	// Get Redis stats if available
	if pd.queryCache != nil {
		if redisCache, ok := pd.queryCache.(*RedisCache); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			redisStats, err := redisCache.GetStats(ctx)
			if err == nil {
				if redisInfo, ok := redisStats["memory_usage"].(int64); ok {
					stats.MemoryUsage = redisInfo
				}
			}
		}
	}

	return stats
}

// collectSlowQueries collects slow query information
func (pd *PerformanceDashboard) collectSlowQueries(ctx context.Context) ([]SlowQuery, error) {
	// Query pg_stat_statements for slow queries
	query := `
		SELECT
			query,
			calls,
			total_time,
			mean_time,
			rows,
			100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
		FROM pg_stat_statements
		WHERE mean_time > 100
		ORDER BY mean_time DESC
		LIMIT 10
	`

	rows, err := pd.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slowQueries []SlowQuery
	for rows.Next() {
		var sq SlowQuery
		if err := rows.Scan(
			&sq.Query,
			&sq.CallCount,
			&sq.TotalTime,
			&sq.MeanTime,
			&sq.Rows,
			&sq.HitPercent,
		); err != nil {
			continue
		}
		sq.LastExecuted = time.Now()
		slowQueries = append(slowQueries, sq)
	}

	return slowQueries, nil
}

// collectTableSizes collects table size information
func (pd *PerformanceDashboard) collectTableSizes(ctx context.Context) ([]TableSize, error) {
	query := `
		SELECT
			table_name,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size,
			pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) as index_size,
			pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size,
			n_tup_ins + n_tup_upd + n_tup_del as row_count
		FROM pg_stat_user_tables
		ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
		LIMIT 20
	`

	rows, err := pd.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableSizes []TableSize
	for rows.Next() {
		var ts TableSize
		if err := rows.Scan(
			&ts.TableName,
			&ts.TotalSize,
			&ts.IndexSize,
			&ts.TableSize,
			&ts.RowCount,
		); err != nil {
			continue
		}
		tableSizes = append(tableSizes, ts)
	}

	return tableSizes, nil
}

// collectIndexUsage collects index usage statistics
func (pd *PerformanceDashboard) collectIndexUsage(ctx context.Context) ([]IndexUsage, error) {
	query := `
		SELECT
			schemaname,
			relname as table_name,
			indexrelname as index_name,
			idx_scan,
			idx_tup_read,
			idx_tup_fetch
		FROM pg_stat_user_indexes
		ORDER BY idx_scan DESC
		LIMIT 50
	`

	rows, err := pd.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexUsage []IndexUsage
	for rows.Next() {
		var iu IndexUsage
		if err := rows.Scan(
			&iu.SchemaName,
			&iu.TableName,
			&iu.IndexName,
			&iu.IdxScan,
			&iu.IdxTupRead,
			&iu.IdxTupFetch,
		); err != nil {
			continue
		}

		// Calculate usage percentage (simplified)
		if iu.IdxTupRead > 0 {
			iu.UsagePercent = float64(iu.IdxTupFetch) / float64(iu.IdxTupRead) * 100
		}

		indexUsage = append(indexUsage, iu)
	}

	return indexUsage, nil
}

// collectSystemMetrics collects system-level metrics
func (pd *PerformanceDashboard) collectSystemMetrics() SystemMetrics {
	// This is a simplified implementation
	// In a real system, you would collect these from system monitoring tools
	return SystemMetrics{
		CPUUsage:        0.0,
		MemoryUsage:     0,
		DiskUsage:       0.0,
		NetworkIO:       0,
		DatabaseCPUTime: 0.0,
	}
}

// formatQueryPatterns formats query patterns for display
func (pd *PerformanceDashboard) formatQueryPatterns(patterns map[string]*PerformanceQueryPattern) []QueryPattern {
	var result []QueryPattern
	for _, pattern := range patterns {
		// Convert PerformanceQueryPattern to QueryPattern for display
		result = append(result, QueryPattern{
			Name: pattern.Query, // Use query as name for display
			Query: pattern.Query,
			Args:  []interface{}{},
			Weight: 1,
		})
	}

	// Limit to top 20
	if len(result) > 20 {
		result = result[:20]
	}

	return result
}

// addToHistory adds performance data to history with size limit
func (pd *PerformanceDashboard) addToHistory(data PerformanceData) {
	pd.history = append(pd.history, data)

	// Keep only the most recent entries
	if len(pd.history) > pd.config.HistorySize {
		pd.history = pd.history[1:]
	}
}

// handleDashboard serves the main dashboard HTML
func (pd *PerformanceDashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := pd.generateDashboardHTML()
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleAPI serves the performance data as JSON
func (pd *PerformanceDashboard) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data := map[string]interface{}{
		"current": pd.getCurrentData(),
		"history": pd.history,
		"config":  pd.config,
	}

	json.NewEncoder(w).Encode(data)
}

// getCurrentData returns the most recent performance data
func (pd *PerformanceDashboard) getCurrentData() PerformanceData {
	if len(pd.history) == 0 {
		return PerformanceData{Timestamp: time.Now()}
	}
	return pd.history[len(pd.history)-1]
}

// generateDashboardHTML generates the dashboard HTML
func (pd *PerformanceDashboard) generateDashboardHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>Database Performance Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .dashboard { max-width: 1200px; margin: 0 auto; }
        .header { background: #2c3e50; color: white; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .metrics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .metric-card { background: white; padding: 20px; border-radius: 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
        .metric-value { font-size: 2em; font-weight: bold; color: #3498db; }
        .metric-label { color: #7f8c8d; margin-top: 5px; }
        .chart-container { background: white; padding: 20px; border-radius: 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); margin-bottom: 20px; }
        .table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        .table th, .table td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        .table th { background-color: #f8f9fa; }
        .status-good { color: #27ae60; }
        .status-warning { color: #f39c12; }
        .status-error { color: #e74c3c; }
        .refresh-button { background: #3498db; color: white; border: none; padding: 10px 20px; border-radius: 5px; cursor: pointer; }
        .refresh-button:hover { background: #2980b9; }
    </style>
</head>
<body>
    <div class="dashboard">
        <div class="header">
            <h1>Database Performance Dashboard</h1>
            <p>Real-time monitoring and analytics for database performance</p>
            <button class="refresh-button" onclick="refreshData()">Refresh Data</button>
        </div>

        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-value" id="queries-per-second">-</div>
                <div class="metric-label">Queries per Second</div>
            </div>
            <div class="metric-card">
                <div class="metric-value" id="avg-response-time">-</div>
                <div class="metric-label">Avg Response Time</div>
            </div>
            <div class="metric-card">
                <div class="metric-value" id="connection-utilization">-</div>
                <div class="metric-label">Connection Utilization</div>
            </div>
            <div class="metric-card">
                <div class="metric-value" id="cache-hit-rate">-</div>
                <div class="metric-label">Cache Hit Rate</div>
            </div>
        </div>

        <div class="chart-container">
            <h3>Query Response Time Trend</h3>
            <canvas id="response-time-chart" width="400" height="200"></canvas>
        </div>

        <div class="chart-container">
            <h3>Connection Pool Usage</h3>
            <canvas id="connection-chart" width="400" height="200"></canvas>
        </div>

        <div class="chart-container">
            <h3>Slow Queries</h3>
            <div id="slow-queries-table">
                <table class="table">
                    <thead>
                        <tr>
                            <th>Query</th>
                            <th>Calls</th>
                            <th>Mean Time</th>
                            <th>Hit %</th>
                        </tr>
                    </thead>
                    <tbody id="slow-queries-body">
                        <tr><td colspan="4">Loading...</td></tr>
                    </tbody>
                </table>
            </div>
        </div>

        <div class="chart-container">
            <h3>Table Sizes</h3>
            <div id="table-sizes-table">
                <table class="table">
                    <thead>
                        <tr>
                            <th>Table</th>
                            <th>Total Size</th>
                            <th>Rows</th>
                        </tr>
                    </thead>
                    <tbody id="table-sizes-body">
                        <tr><td colspan="3">Loading...</td></tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <script>
        let responseTimeChart, connectionChart;

        function initCharts() {
            const ctx1 = document.getElementById('response-time-chart').getContext('2d');
            responseTimeChart = new Chart(ctx1, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Avg Response Time (ms)',
                        data: [],
                        borderColor: '#3498db',
                        fill: false
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });

            const ctx2 = document.getElementById('connection-chart').getContext('2d');
            connectionChart = new Chart(ctx2, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Active Connections',
                        data: [],
                        borderColor: '#e74c3c',
                        fill: false
                    }, {
                        label: 'Idle Connections',
                        data: [],
                        borderColor: '#27ae60',
                        fill: false
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });
        }

        function updateMetrics(data) {
            document.getElementById('queries-per-second').textContent = data.query_metrics.queries_per_second.toFixed(1);
            document.getElementById('avg-response-time').textContent = (data.query_metrics.avg_response_time / 1000000).toFixed(1) + 'ms';
            document.getElementById('connection-utilization').textContent = data.connection_stats.connection_utilization.toFixed(1) + '%';
            document.getElementById('cache-hit-rate').textContent = data.cache_stats.hit_rate.toFixed(1) + '%';

            // Update colors based on thresholds
            updateMetricColor('avg-response-time', data.query_metrics.avg_response_time, 100000000); // 100ms threshold
            updateMetricColor('connection-utilization', data.connection_stats.connection_utilization, 80); // 80% threshold
            updateMetricColor('cache-hit-rate', data.cache_stats.hit_rate, 90, true); // 90% threshold (higher is better)
        }

        function updateMetricColor(elementId, value, threshold, higherIsBetter = false) {
            const element = document.getElementById(elementId);
            element.classList.remove('status-good', 'status-warning', 'status-error');

            if (higherIsBetter) {
                if (value >= threshold) element.classList.add('status-good');
                else if (value >= threshold * 0.8) element.classList.add('status-warning');
                else element.classList.add('status-error');
            } else {
                if (value <= threshold) element.classList.add('status-good');
                else if (value <= threshold * 1.5) element.classList.add('status-warning');
                else element.classList.add('status-error');
            }
        }

        function updateCharts(history) {
            const labels = history.map(d => new Date(d.timestamp).toLocaleTimeString());
            const avgResponseTimes = history.map(d => d.query_metrics.avg_response_time / 1000000);
            const activeConns = history.map(d => d.connection_stats.active_connections);
            const idleConns = history.map(d => d.connection_stats.idle_connections);

            responseTimeChart.data.labels = labels;
            responseTimeChart.data.datasets[0].data = avgResponseTimes;
            responseTimeChart.update();

            connectionChart.data.labels = labels;
            connectionChart.data.datasets[0].data = activeConns;
            connectionChart.data.datasets[1].data = idleConns;
            connectionChart.update();
        }

        function updateTables(data) {
            // Update slow queries table
            const slowQueriesBody = document.getElementById('slow-queries-body');
            slowQueriesBody.innerHTML = '';

            if (data.slow_queries && data.slow_queries.length > 0) {
                data.slow_queries.forEach(query => {
                    const row = slowQueriesBody.insertRow();
                    row.insertCell(0).textContent = query.query.substring(0, 100) + (query.query.length > 100 ? '...' : '');
                    row.insertCell(1).textContent = query.call_count;
                    row.insertCell(2).textContent = (query.mean_time / 1000000).toFixed(2) + 'ms';
                    row.insertCell(3).textContent = query.hit_percent.toFixed(1) + '%';
                });
            } else {
                slowQueriesBody.innerHTML = '<tr><td colspan="4">No slow queries detected</td></tr>';
            }

            // Update table sizes
            const tableSizesBody = document.getElementById('table-sizes-body');
            tableSizesBody.innerHTML = '';

            if (data.table_sizes && data.table_sizes.length > 0) {
                data.table_sizes.forEach(table => {
                    const row = tableSizesBody.insertRow();
                    row.insertCell(0).textContent = table.table_name;
                    row.insertCell(1).textContent = table.total_size;
                    row.insertCell(2).textContent = table.row_count.toLocaleString();
                });
            } else {
                tableSizesBody.innerHTML = '<tr><td colspan="3">No table data available</td></tr>';
            }
        }

        async function refreshData() {
            try {
                const response = await fetch('/db-dashboard/api');
                const data = await response.json();

                updateMetrics(data.current);
                updateCharts(data.history);
                updateTables(data.current);
            } catch (error) {
                console.error('Error refreshing data:', error);
            }
        }

        // Initialize dashboard
        document.addEventListener('DOMContentLoaded', function() {
            initCharts();
            refreshData();
            setInterval(refreshData, 5000); // Refresh every 5 seconds
        });
    </script>
</body>
</html>`
}