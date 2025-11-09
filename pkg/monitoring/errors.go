package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityDebug    ErrorSeverity = "debug"
	ErrorSeverityInfo     ErrorSeverity = "info"
	ErrorSeverityWarning  ErrorSeverity = "warning"
	ErrorSeverityError    ErrorSeverity = "error"
	ErrorSeverityCritical ErrorSeverity = "critical"
	ErrorSeverityFatal    ErrorSeverity = "fatal"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	ErrorCategorySystem     ErrorCategory = "system"
	ErrorCategoryNetwork    ErrorCategory = "network"
	ErrorCategoryDatabase   ErrorCategory = "database"
	ErrorCategoryAuth       ErrorCategory = "auth"
	ErrorCategoryBusiness   ErrorCategory = "business"
	ErrorCategoryValidation ErrorCategory = "validation"
	ErrorCategoryExternal   ErrorCategory = "external"
	ErrorCategorySecurity   ErrorCategory = "security"
)

// ErrorEvent represents a tracked error event
type ErrorEvent struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Message      string                 `json:"message"`
	ErrorType    string                 `json:"error_type"`
	Severity     ErrorSeverity          `json:"severity"`
	Category     ErrorCategory          `json:"category"`
	Component    string                 `json:"component"`
	UserID       string                 `json:"user_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	StackTrace   string                 `json:"stack_trace,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Recoverable  bool                   `json:"recoverable"`
	Count        int                    `json:"count"`
	FirstSeen    time.Time              `json:"first_seen"`
	LastSeen     time.Time              `json:"last_seen"`
	AffectedUsers []string               `json:"affected_users,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

// ErrorAggregation represents aggregated error statistics
type ErrorAggregation struct {
	ErrorType     string    `json:"error_type"`
	Category      string    `json:"category"`
	Component     string    `json:"component"`
	Severity      string    `json:"severity"`
	Count         int       `json:"count"`
	UniqueUsers   int       `json:"unique_users"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	AffectedUsers []string  `json:"affected_users"`
	Trend         float64   `json:"trend"` // Rate change percentage
}

// ErrorTracker tracks and aggregates errors
type ErrorTracker struct {
	errors      map[string]*ErrorEvent
	errorsMutex sync.RWMutex
	aggregations map[string]*ErrorAggregation
	aggregationsMutex sync.RWMutex
	maxErrors   int
	maxAge      time.Duration
	exporters   []ErrorExporter
}

// ErrorExporter exports error events to external systems
type ErrorExporter interface {
	Export(ctx context.Context, event *ErrorEvent) error
	ExportAggregated(ctx context.Context, aggregation []ErrorAggregation) error
	Shutdown(ctx context.Context) error
}

// ConsoleErrorExporter exports errors to console
type ConsoleErrorExporter struct{}

// NewConsoleErrorExporter creates a new console error exporter
func NewConsoleErrorExporter() *ConsoleErrorExporter {
	return &ConsoleErrorExporter{}
}

// Export exports an error event to console
func (cee *ConsoleErrorExporter) Export(ctx context.Context, event *ErrorEvent) error {
	fmt.Printf("[ERROR] %s [%s] %s - %s (%s)\n",
		event.Timestamp.Format(time.RFC3339),
		event.Severity,
		event.Component,
		event.Message,
		event.ErrorType)
	return nil
}

// ExportAggregated exports aggregated errors to console
func (cee *ConsoleErrorExporter) ExportAggregated(ctx context.Context, aggregations []ErrorAggregation) error {
	for _, agg := range aggregations {
		fmt.Printf("[ERROR_AGG] %s: %d occurrences, %d users, trend: %.1f%%\n",
			agg.ErrorType,
			agg.Count,
			agg.UniqueUsers,
			agg.Trend)
	}
	return nil
}

// Shutdown closes the console exporter
func (cee *ConsoleErrorExporter) Shutdown(ctx context.Context) error {
	return nil
}

// NewErrorTracker creates a new error tracker
func NewErrorTracker(maxErrors int, maxAge time.Duration) *ErrorTracker {
	return &ErrorTracker{
		errors:       make(map[string]*ErrorEvent),
		aggregations: make(map[string]*ErrorAggregation),
		maxErrors:    maxErrors,
		maxAge:       maxAge,
		exporters:    []ErrorExporter{NewConsoleErrorExporter()},
	}
}

// TrackError tracks an error event
func (et *ErrorTracker) TrackError(ctx context.Context, err error, component string, severity ErrorSeverity, category ErrorCategory, context map[string]interface{}) {
	if err == nil {
		return
	}

	event := &ErrorEvent{
		ID:          GenerateErrorID(err),
		Timestamp:   time.Now().UTC(),
		Message:     err.Error(),
		ErrorType:   fmt.Sprintf("%T", err),
		Severity:    severity,
		Category:    category,
		Component:   component,
		StackTrace:  getStackTrace(),
		Context:     context,
		Recoverable: isRecoverable(err),
		Count:       1,
		FirstSeen:   time.Now().UTC(),
		LastSeen:    time.Now().UTC(),
		Tags:        extractTags(err),
	}

	// Extract user and request information from context
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		event.UserID = userID
	}
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		event.RequestID = requestID
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		event.TraceID = traceID
	}

	et.errorsMutex.Lock()
	defer et.errorsMutex.Unlock()

	// Check if error already exists and update aggregation
	if existing, exists := et.errors[event.ID]; exists {
		existing.Count++
		existing.LastSeen = event.Timestamp
		if event.UserID != "" && !contains(existing.AffectedUsers, event.UserID) {
			existing.AffectedUsers = append(existing.AffectedUsers, event.UserID)
		}
		et.updateAggregation(existing)
	} else {
		et.errors[event.ID] = event
		if event.UserID != "" {
			event.AffectedUsers = []string{event.UserID}
		}
		et.updateAggregation(event)
	}

	// Clean old errors
	et.cleanOldErrors()

	// Export error asynchronously
	go func() {
		for _, exporter := range et.exporters {
			if err := exporter.Export(ctx, event); err != nil {
				fmt.Printf("Failed to export error: %v\n", err)
			}
		}
	}()
}

// TrackPanic tracks a panic event
func (et *ErrorTracker) TrackPanic(ctx context.Context, recovered interface{}, component string) {
	var err error
	switch v := recovered.(type) {
	case error:
		err = v
	case string:
		err = fmt.Errorf("panic: %s", v)
	default:
		err = fmt.Errorf("panic: %v", v)
	}

	context := map[string]interface{}{
		"panic":        true,
		"stack_trace":  getStackTrace(),
		"recovered":    fmt.Sprintf("%v", recovered),
	}

	et.TrackError(ctx, err, component, ErrorSeverityCritical, ErrorCategorySystem, context)
}

// GetErrors returns recent errors
func (et *ErrorTracker) GetErrors(limit int, severity ErrorSeverity, category ErrorCategory) []*ErrorEvent {
	et.errorsMutex.RLock()
	defer et.errorsMutex.RUnlock()

	var filtered []*ErrorEvent
	for _, event := range et.errors {
		if (severity == "" || event.Severity == severity) &&
			(category == "" || event.Category == category) {
			filtered = append(filtered, event)
		}
	}

	// Sort by timestamp (most recent first)
	// For simplicity, just return the most recent
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	return filtered
}

// GetAggregations returns error aggregations
func (et *ErrorTracker) GetAggregations() []ErrorAggregation {
	et.aggregationsMutex.RLock()
	defer et.aggregationsMutex.RUnlock()

	var result []ErrorAggregation
	for _, agg := range et.aggregations {
		result = append(result, *agg)
	}

	return result
}

// GetErrorStats returns error statistics
func (et *ErrorTracker) GetErrorStats() map[string]interface{} {
	et.errorsMutex.RLock()
	et.aggregationsMutex.RLock()
	defer et.errorsMutex.RUnlock()
	defer et.aggregationsMutex.RUnlock()

	stats := map[string]interface{}{
		"total_errors":     len(et.errors),
		"total_aggregated": len(et.aggregations),
		"severity_breakdown": make(map[string]int),
		"category_breakdown": make(map[string]int),
		"component_breakdown": make(map[string]int),
	}

	severityBreakdown := stats["severity_breakdown"].(map[string]int)
	categoryBreakdown := stats["category_breakdown"].(map[string]int)
	componentBreakdown := stats["component_breakdown"].(map[string]int)

	for _, event := range et.errors {
		severityBreakdown[string(event.Severity)]++
		categoryBreakdown[string(event.Category)]++
		componentBreakdown[event.Component]++
	}

	return stats
}

// AddExporter adds an error exporter
func (et *ErrorTracker) AddExporter(exporter ErrorExporter) {
	et.exporters = append(et.exporters, exporter)
}

// updateAggregation updates error aggregation
func (et *ErrorTracker) updateAggregation(event *ErrorEvent) {
	key := fmt.Sprintf("%s:%s:%s", event.ErrorType, event.Category, event.Component)

	et.aggregationsMutex.Lock()
	defer et.aggregationsMutex.Unlock()

	agg, exists := et.aggregations[key]
	if !exists {
		agg = &ErrorAggregation{
			ErrorType:     event.ErrorType,
			Category:      string(event.Category),
			Component:     event.Component,
			Severity:      string(event.Severity),
			Count:         0,
			FirstSeen:     event.Timestamp,
			LastSeen:      event.Timestamp,
			AffectedUsers: []string{},
		}
		et.aggregations[key] = agg
	}

	agg.Count++
	agg.LastSeen = event.Timestamp

	// Update affected users
	if event.UserID != "" && !contains(agg.AffectedUsers, event.UserID) {
		agg.AffectedUsers = append(agg.AffectedUsers, event.UserID)
	}
	agg.UniqueUsers = len(agg.AffectedUsers)

	// Calculate trend (simplified)
	agg.Trend = calculateTrend(agg.Count, time.Since(agg.FirstSeen))
}

// cleanOldErrors removes old errors
func (et *ErrorTracker) cleanOldErrors() {
	cutoff := time.Now().Add(-et.maxAge)
	toDelete := []string{}

	for id, event := range et.errors {
		if event.Timestamp.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}

	// Also limit total number of errors
	if len(et.errors)-len(toDelete) > et.maxErrors {
		// Sort by timestamp and remove oldest
		type errorWithTime struct {
			id       string
			event    *ErrorEvent
			timestamp time.Time
		}

		var sorted []errorWithTime
		for id, event := range et.errors {
			sorted = append(sorted, errorWithTime{id, event, event.Timestamp})
		}

		// Simple sort (would need proper implementation)
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i].timestamp.After(sorted[j].timestamp) {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		excess := len(sorted) - et.maxErrors
		for i := 0; i < excess; i++ {
			toDelete = append(toDelete, sorted[i].id)
		}
	}

	for _, id := range toDelete {
		delete(et.errors, id)
	}
}

// Shutdown shuts down the error tracker
func (et *ErrorTracker) Shutdown(ctx context.Context) error {
	var errors []error

	for _, exporter := range et.exporters {
		if err := exporter.Shutdown(ctx); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// Helper functions

func GenerateErrorID(err error) string {
	return fmt.Sprintf("%s:%s", fmt.Sprintf("%T", err), err.Error())
}

func getStackTrace() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

func isRecoverable(err error) bool {
	// Simple heuristic - can be made more sophisticated
	errStr := err.Error()
	recoverableErrors := []string{
		"connection refused",
		"timeout",
		"temporary",
		"network",
	}

	for _, rec := range recoverableErrors {
		if contains(errStr, rec) {
			return true
		}
	}

	return false
}

func extractTags(err error) []string {
	// Extract tags from error type or message
	var tags []string
	errStr := err.Error()

	// Security-related tags
	securityKeywords := []string{"auth", "login", "permission", "unauthorized"}
	for _, keyword := range securityKeywords {
		if contains(errStr, keyword) {
			tags = append(tags, "security")
			break
		}
	}

	// Performance-related tags
	perfKeywords := []string{"timeout", "slow", "performance"}
	for _, keyword := range perfKeywords {
		if contains(errStr, keyword) {
			tags = append(tags, "performance")
			break
		}
	}

	return tags
}

func calculateTrend(count int, duration time.Duration) float64 {
	// Simple trend calculation - rate per hour
	if duration == 0 {
		return 0
	}
	rate := float64(count) / duration.Hours()
	return rate
}

func contains(slice interface{}, item interface{}) bool {
	switch s := slice.(type) {
	case []string:
		for _, a := range s {
			if a == item {
				return true
			}
		}
	case string:
		str := s
		if substr, ok := item.(string); ok {
			return len(str) >= len(substr) && str[:len(substr)] == substr
		}
	}
	return false
}

// Global error tracker instance
var GlobalErrorTracker *ErrorTracker

// InitializeErrorTracker initializes the global error tracker
func InitializeErrorTracker(maxErrors int, maxAge time.Duration) {
	GlobalErrorTracker = NewErrorTracker(maxErrors, maxAge)
}

// GetErrorTracker returns the global error tracker
func GetErrorTracker() *ErrorTracker {
	if GlobalErrorTracker == nil {
		GlobalErrorTracker = NewErrorTracker(1000, 24*time.Hour)
	}
	return GlobalErrorTracker
}

// Convenience functions for global error tracker

// TrackError tracks an error using the global tracker
func TrackError(ctx context.Context, err error, component string, severity ErrorSeverity, category ErrorCategory, context map[string]interface{}) {
	GetErrorTracker().TrackError(ctx, err, component, severity, category, context)
}

// TrackPanic tracks a panic using the global tracker
func TrackPanic(ctx context.Context, recovered interface{}, component string) {
	GetErrorTracker().TrackPanic(ctx, recovered, component)
}

// GetErrors returns recent errors using the global tracker
func GetErrors(limit int, severity ErrorSeverity, category ErrorCategory) []*ErrorEvent {
	return GetErrorTracker().GetErrors(limit, severity, category)
}

// GetErrorStats returns error statistics using the global tracker
func GetErrorStats() map[string]interface{} {
	return GetErrorTracker().GetErrorStats()
}