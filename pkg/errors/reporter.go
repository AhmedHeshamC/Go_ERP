package errors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Severity represents the error severity level
type Severity string

const (
	SeverityDebug   Severity = "debug"
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
	SeverityFatal   Severity = "fatal"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeBusiness       ErrorType = "business"
	ErrorTypeSystem         ErrorType = "system"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeDatabase       ErrorType = "database"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeAuthorization  ErrorType = "authorization"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeExternal       ErrorType = "external"
	ErrorTypeSecurity       ErrorType = "security"
	ErrorTypePerformance    ErrorType = "performance"
)

// Context represents error context information
type Context struct {
	UserID        string                 `json:"user_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Tags          map[string]string      `json:"tags,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

// StackFrame represents a single stack frame
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
}

// ErrorReport represents a comprehensive error report
type ErrorReport struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error"`
	Type        ErrorType              `json:"type"`
	Severity    Severity               `json:"severity"`
	Context     *Context               `json:"context,omitempty"`
	StackFrames []StackFrame           `json:"stack_frames,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
	Platform    *PlatformInfo          `json:"platform,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	Release     string                 `json:"release,omitempty"`
	Fingerprint string                 `json:"fingerprint,omitempty"`
	Handled     bool                   `json:"handled"`
	Occurrences int                    `json:"occurrences"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
}

// PlatformInfo represents platform information
type PlatformInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Hostname     string `json:"hostname"`
	Runtime      string `json:"runtime"`
	Version      string `json:"version"`
}

// Config holds the error reporter configuration
type Config struct {
	// Basic settings
	Enabled     bool   `json:"enabled"`
	Environment string `json:"environment"`
	Release     string `json:"release"`
	Debug       bool   `json:"debug"`

	// Reporting settings
	SampleRate     float64       `json:"sample_rate"`
	MaxErrors      int           `json:"max_errors"`
	FlushInterval  time.Duration `json:"flush_interval"`
	AsyncReporting bool          `json:"async_reporting"`

	// Filter settings
	IgnoreErrors      []string `json:"ignore_errors"`
	IgnoreStatusCodes []int    `json:"ignore_status_codes"`
	MinSeverity       Severity `json:"min_severity"`

	// Context settings
	IncludeRequestData bool `json:"include_request_data"`
	IncludeUserData    bool `json:"include_user_data"`
	IncludeSystemData  bool `json:"include_system_data"`

	// Sensitive data settings
	SanitizeFields     []string `json:"sanitize_fields"`
	EnableSanitization bool     `json:"enable_sanitization"`

	// Performance settings
	MaxStackFrames    int           `json:"max_stack_frames"`
	MaxCustomDataSize int           `json:"max_custom_data_size"`
	Timeout           time.Duration `json:"timeout"`

	// Integration settings
	SentryEnabled    bool   `json:"sentry_enabled"`
	SentryDSN        string `json:"sentry_dsn"`
	DataDogEnabled   bool   `json:"datadog_enabled"`
	DataDogAPIKey    string `json:"datadog_api_key"`
	DataDogSite      string `json:"datadog_site"`
	CustomWebhookURL string `json:"custom_webhook_url"`
}

// DefaultConfig returns a default error reporter configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:            true,
		Environment:        "development",
		Release:            "1.0.0",
		Debug:              true,
		SampleRate:         1.0,
		MaxErrors:          1000,
		FlushInterval:      5 * time.Second,
		AsyncReporting:     true,
		IgnoreErrors:       []string{},
		IgnoreStatusCodes:  []int{404, 401},
		MinSeverity:        SeverityError,
		IncludeRequestData: true,
		IncludeUserData:    true,
		IncludeSystemData:  true,
		SanitizeFields: []string{
			"password", "token", "secret", "key", "auth",
			"credit_card", "ssn", "social_security", "api_key",
		},
		EnableSanitization: true,
		MaxStackFrames:     20,
		MaxCustomDataSize:  1024 * 64, // 64KB
		Timeout:            10 * time.Second,
		SentryEnabled:      false,
		DataDogEnabled:     false,
		DataDogSite:        "datadoghq.com",
	}
}

// ProductionConfig returns a production-safe configuration
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Debug = false
	config.SampleRate = 0.1 // Sample 10% of errors
	config.MinSeverity = SeverityError
	config.IncludeUserData = false
	config.EnableSanitization = true
	config.MaxStackFrames = 10
	return config
}

// Reporter represents the error reporter
type Reporter struct {
	config      *Config
	logger      *zerolog.Logger
	queue       chan *ErrorReport
	buffer      []*ErrorReport
	bufferMu    sync.RWMutex
	errorCounts map[string]int
	countsMu    sync.RWMutex
	client      *http.Client
	sanitizer   *sanitizer
}

// sanitizer handles sensitive data sanitization
type sanitizer struct {
	fields map[string]bool
}

// NewReporter creates a new error reporter
func NewReporter(config *Config, logger *zerolog.Logger) (*Reporter, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	reporter := &Reporter{
		config:      config,
		logger:      logger,
		queue:       make(chan *ErrorReport, config.MaxErrors),
		buffer:      make([]*ErrorReport, 0, config.MaxErrors),
		errorCounts: make(map[string]int),
		client: &http.Client{
			Timeout: config.Timeout,
		},
		sanitizer: newSanitizer(config.SanitizeFields),
	}

	// Start background worker if async reporting is enabled
	if config.AsyncReporting {
		go reporter.worker()
	}

	return reporter, nil
}

// newSanitizer creates a new sanitizer
func newSanitizer(fields []string) *sanitizer {
	fieldMap := make(map[string]bool)
	for _, field := range fields {
		fieldMap[strings.ToLower(field)] = true
	}
	return &sanitizer{fields: fieldMap}
}

// Report reports an error
func (r *Reporter) Report(ctx context.Context, err error, severity Severity, errorType ErrorType, message string, context *Context, customData map[string]interface{}) {
	if !r.config.Enabled {
		return
	}

	// Check sampling
	if !r.shouldSample(err) {
		return
	}

	// Create error report
	report := r.createErrorReport(ctx, err, severity, errorType, message, context, customData)

	// Sanitize sensitive data
	if r.config.EnableSanitization {
		r.sanitizeReport(report)
	}

	// Handle reporting
	if r.config.AsyncReporting {
		select {
		case r.queue <- report:
		default:
			r.logger.Warn().Msg("Error reporting queue is full, dropping error")
		}
	} else {
		r.processReport(report)
	}
}

// ReportHTTPError reports an HTTP error
func (r *Reporter) ReportHTTPError(ctx context.Context, err error, statusCode int, message string, requestContext *Context) {
	// Check if status code should be ignored
	for _, ignoredCode := range r.config.IgnoreStatusCodes {
		if statusCode == ignoredCode {
			return
		}
	}

	severity := r.mapStatusCodeToSeverity(statusCode)
	errorType := r.mapStatusCodeToErrorType(statusCode)

	r.Report(ctx, err, severity, errorType, message, requestContext, nil)
}

// ReportPanic reports a panic recovery
func (r *Reporter) ReportPanic(ctx context.Context, recovered interface{}, stack []byte, context *Context) {
	message := fmt.Sprintf("Panic recovered: %v", recovered)

	// Parse stack trace
	var stackFrames []StackFrame
	if stack != nil {
		stackFrames = r.parseStackFrames(stack)
	}

	report := &ErrorReport{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		Message:     message,
		Error:       fmt.Sprintf("%v", recovered),
		Type:        ErrorTypeSystem,
		Severity:    SeverityFatal,
		Context:     context,
		StackFrames: stackFrames,
		Platform:    r.getPlatformInfo(),
		Environment: r.config.Environment,
		Release:     r.config.Release,
		Handled:     false,
		Occurrences: 1,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
	}

	if r.config.AsyncReporting {
		select {
		case r.queue <- report:
		default:
			r.logger.Warn().Msg("Error reporting queue is full, dropping panic report")
		}
	} else {
		r.processReport(report)
	}
}

// createErrorReport creates an error report
func (r *Reporter) createErrorReport(ctx context.Context, err error, severity Severity, errorType ErrorType, message string, contextInfo *Context, customData map[string]interface{}) *ErrorReport {
	report := &ErrorReport{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		Message:     message,
		Error:       err.Error(),
		Type:        errorType,
		Severity:    severity,
		Context:     contextInfo,
		CustomData:  customData,
		Platform:    r.getPlatformInfo(),
		Environment: r.config.Environment,
		Release:     r.config.Release,
		StackFrames: r.captureStackFrames(),
		Handled:     true,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
	}

	// Generate fingerprint
	report.Fingerprint = r.generateFingerprint(report)

	// Update occurrence count
	report.Occurrences = r.updateErrorCount(report.Fingerprint)

	return report
}

// captureStackFrames captures the current stack frames
func (r *Reporter) captureStackFrames() []StackFrame {
	var frames []StackFrame
	pcs := make([]uintptr, r.config.MaxStackFrames)
	n := runtime.Callers(3, pcs) // Skip this function and the caller
	pcs = pcs[:n]

	frames = make([]StackFrame, 0, n)
	for _, pc := range pcs {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		file, line := fn.FileLine(pc)
		frame := StackFrame{
			Function: fn.Name(),
			File:     file,
			Line:     line,
		}

		frames = append(frames, frame)
	}

	return frames
}

// parseStackFrames parses a stack trace from bytes
func (r *Reporter) parseStackFrames(stack []byte) []StackFrame {
	var frames []StackFrame
	lines := strings.Split(string(stack), "\n")

	for i := 0; i < len(lines); i += 2 {
		if i+1 >= len(lines) {
			break
		}

		line := strings.TrimSpace(lines[i])
		fileLine := strings.TrimSpace(lines[i+1])

		if strings.HasPrefix(line, "goroutine ") || !strings.Contains(fileLine, ":") {
			continue
		}

		// Parse function name
		function := line
		if idx := strings.LastIndex(function, "."); idx > 0 {
			function = function[idx+1:]
		}

		// Parse file and line number
		parts := strings.Split(fileLine, ":")
		if len(parts) >= 2 {
			frame := StackFrame{
				Function: function,
				File:     parts[0],
			}

			if lineNum, err := fmt.Sscanf(parts[1], "%d", &frame.Line); err == nil && lineNum == 1 {
				frames = append(frames, frame)
			}
		}
	}

	return frames
}

// getPlatformInfo returns platform information
func (r *Reporter) getPlatformInfo() *PlatformInfo {
	hostname, _ := os.Hostname()

	return &PlatformInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Hostname:     hostname,
		Runtime:      "go",
		Version:      runtime.Version(),
	}
}

// generateFingerprint generates a unique fingerprint for an error
func (r *Reporter) generateFingerprint(report *ErrorReport) string {
	// Create a fingerprint based on error type, message, and stack frames
	var buffer bytes.Buffer
	buffer.WriteString(string(report.Type))
	buffer.WriteString("|")
	buffer.WriteString(report.Error)
	buffer.WriteString("|")

	// Include top 3 stack frames for fingerprinting
	for i := 0; i < len(report.StackFrames) && i < 3; i++ {
		frame := report.StackFrames[i]
		buffer.WriteString(fmt.Sprintf("%s:%d:%s", frame.File, frame.Line, frame.Function))
		buffer.WriteString("|")
	}

	return buffer.String()
}

// updateErrorCount updates the error occurrence count
func (r *Reporter) updateErrorCount(fingerprint string) int {
	r.countsMu.Lock()
	defer r.countsMu.Unlock()

	r.errorCounts[fingerprint]++
	return r.errorCounts[fingerprint]
}

// shouldSample determines if an error should be sampled
func (r *Reporter) shouldSample(err error) bool {
	// Check if error should be ignored
	for _, ignoredError := range r.config.IgnoreErrors {
		if strings.Contains(err.Error(), ignoredError) {
			return false
		}
	}

	// Apply sampling rate
	if r.config.SampleRate < 1.0 {
		// Simple random sampling
		return rand.Float64() <= r.config.SampleRate // #nosec G404 - Used for sampling, not security
	}

	return true
}

// sanitizeReport sanitizes sensitive data in the report
func (r *Reporter) sanitizeReport(report *ErrorReport) {
	if report.Context != nil {
		r.sanitizeContext(report.Context)
	}

	if report.CustomData != nil {
		if sanitizedData, ok := r.sanitizeData(report.CustomData).(map[string]interface{}); ok {
			report.CustomData = sanitizedData
		}
	}
}

// sanitizeContext sanitizes sensitive data in context
func (r *Reporter) sanitizeContext(ctx *Context) {
	// Sanitize extra data
	if ctx.Extra != nil {
		if sanitizedData, ok := r.sanitizeData(ctx.Extra).(map[string]interface{}); ok {
			ctx.Extra = sanitizedData
		}
	}

	// Redact sensitive tags
	if ctx.Tags != nil {
		for key := range ctx.Tags {
			if r.sanitizer.shouldSanitize(key) {
				ctx.Tags[key] = "[REDACTED]"
			}
		}
	}
}

// sanitizeData recursively sanitizes sensitive data
func (r *Reporter) sanitizeData(data interface{}) interface{} {
	switch v := data.(type) {
	case string:
		return v
	case map[string]interface{}:
		for key, value := range v {
			if r.sanitizer.shouldSanitize(key) {
				v[key] = "[REDACTED]"
			} else {
				v[key] = r.sanitizeData(value)
			}
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = r.sanitizeData(item)
		}
		return v
	default:
		return v
	}
}

// shouldSanitize checks if a field should be sanitized
func (s *sanitizer) shouldSanitize(field string) bool {
	lowerField := strings.ToLower(field)
	for sensitiveField := range s.fields {
		if strings.Contains(lowerField, sensitiveField) {
			return true
		}
	}
	return false
}

// mapStatusCodeToSeverity maps HTTP status codes to severity levels
func (r *Reporter) mapStatusCodeToSeverity(statusCode int) Severity {
	switch {
	case statusCode >= 500:
		return SeverityError
	case statusCode >= 400:
		return SeverityWarning
	case statusCode >= 300:
		return SeverityInfo
	default:
		return SeverityDebug
	}
}

// mapStatusCodeToErrorType maps HTTP status codes to error types
func (r *Reporter) mapStatusCodeToErrorType(statusCode int) ErrorType {
	switch {
	case statusCode == 400:
		return ErrorTypeValidation
	case statusCode == 401:
		return ErrorTypeAuthentication
	case statusCode == 403:
		return ErrorTypeAuthorization
	case statusCode == 429:
		return ErrorTypeRateLimit
	case statusCode >= 500:
		return ErrorTypeSystem
	default:
		return ErrorTypeBusiness
	}
}

// processReport processes an error report
func (r *Reporter) processReport(report *ErrorReport) {
	// Add to buffer
	r.bufferMu.Lock()
	r.buffer = append(r.buffer, report)
	if len(r.buffer) > r.config.MaxErrors {
		r.buffer = r.buffer[1:] // Remove oldest
	}
	r.bufferMu.Unlock()

	// Send to integrations
	if r.config.SentryEnabled {
		r.sendToSentry(report)
	}

	if r.config.DataDogEnabled {
		r.sendToDataDog(report)
	}

	if r.config.CustomWebhookURL != "" {
		r.sendToCustomWebhook(report)
	}

	// Log locally
	r.logReport(report)
}

// worker processes error reports in the background
func (r *Reporter) worker() {
	ticker := time.NewTicker(r.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case report := <-r.queue:
			r.processReport(report)
		case <-ticker.C:
			// Periodic flush can be handled here if needed
		}
	}
}

// sendToSentry sends error report to Sentry
func (r *Reporter) sendToSentry(report *ErrorReport) {
	// Implementation would send to Sentry API
	r.logger.Debug().Str("error_id", report.ID).Msg("Sending error to Sentry")
}

// sendToDataDog sends error report to DataDog
func (r *Reporter) sendToDataDog(report *ErrorReport) {
	// Implementation would send to DataDog API
	r.logger.Debug().Str("error_id", report.ID).Msg("Sending error to DataDog")
}

// sendToCustomWebhook sends error report to custom webhook
func (r *Reporter) sendToCustomWebhook(report *ErrorReport) {
	payload, err := json.Marshal(report)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to marshal error report for webhook")
		return
	}

	req, err := http.NewRequest("POST", r.config.CustomWebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to create webhook request")
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to send webhook request")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		r.logger.Warn().Int("status", resp.StatusCode).Msg("Webhook request failed")
	}
}

// logReport logs the error report locally
func (r *Reporter) logReport(report *ErrorReport) {
	event := r.logger.WithLevel(r.mapSeverityToZerolog(report.Severity))
	event = event.Str("error_id", report.ID)
	event = event.Str("error_type", string(report.Type))
	event = event.Str("message", report.Message)
	event = event.Str("error", report.Error)
	event = event.Bool("handled", report.Handled)
	event = event.Int("occurrences", report.Occurrences)

	if report.Context != nil {
		if report.Context.UserID != "" {
			event = event.Str("user_id", report.Context.UserID)
		}
		if report.Context.RequestID != "" {
			event = event.Str("request_id", report.Context.RequestID)
		}
		if report.Context.CorrelationID != "" {
			event = event.Str("correlation_id", report.Context.CorrelationID)
		}
	}

	if report.Fingerprint != "" {
		event = event.Str("fingerprint", report.Fingerprint)
	}

	event.Msg("Error reported")
}

// mapSeverityToZerolog maps severity to zerolog level
func (r *Reporter) mapSeverityToZerolog(severity Severity) zerolog.Level {
	switch severity {
	case SeverityDebug:
		return zerolog.DebugLevel
	case SeverityInfo:
		return zerolog.InfoLevel
	case SeverityWarning:
		return zerolog.WarnLevel
	case SeverityError:
		return zerolog.ErrorLevel
	case SeverityFatal:
		return zerolog.FatalLevel
	default:
		return zerolog.ErrorLevel
	}
}

// GetStats returns reporter statistics
func (r *Reporter) GetStats() map[string]interface{} {
	r.countsMu.RLock()
	defer r.countsMu.RUnlock()

	r.bufferMu.RLock()
	defer r.bufferMu.RUnlock()

	return map[string]interface{}{
		"enabled":           r.config.Enabled,
		"environment":       r.config.Environment,
		"queue_size":        len(r.queue),
		"buffer_size":       len(r.buffer),
		"unique_errors":     len(r.errorCounts),
		"total_occurrences": r.calculateTotalOccurrences(),
		"sample_rate":       r.config.SampleRate,
		"sentry_enabled":    r.config.SentryEnabled,
		"datadog_enabled":   r.config.DataDogEnabled,
	}
}

// calculateTotalOccurrences calculates total error occurrences
func (r *Reporter) calculateTotalOccurrences() int {
	total := 0
	for _, count := range r.errorCounts {
		total += count
	}
	return total
}

// Global reporter instance
var globalReporter *Reporter

// InitGlobalReporter initializes the global error reporter
func InitGlobalReporter(config *Config, logger *zerolog.Logger) error {
	reporter, err := NewReporter(config, logger)
	if err != nil {
		return err
	}
	globalReporter = reporter
	return nil
}

// GetGlobalReporter returns the global error reporter
func GetGlobalReporter() *Reporter {
	if globalReporter == nil {
		// Create with default config if not initialized
		nopLogger := zerolog.Nop()
		globalReporter, _ = NewReporter(DefaultConfig(), &nopLogger)
	}
	return globalReporter
}

// Global convenience functions

// Report reports an error using the global reporter
func Report(ctx context.Context, err error, severity Severity, errorType ErrorType, message string, context *Context, customData map[string]interface{}) {
	if globalReporter != nil {
		globalReporter.Report(ctx, err, severity, errorType, message, context, customData)
	}
}

// ReportHTTPError reports an HTTP error using the global reporter
func ReportHTTPError(ctx context.Context, err error, statusCode int, message string, requestContext *Context) {
	if globalReporter != nil {
		globalReporter.ReportHTTPError(ctx, err, statusCode, message, requestContext)
	}
}

// ReportPanic reports a panic using the global reporter
func ReportPanic(ctx context.Context, recovered interface{}, stack []byte, context *Context) {
	if globalReporter != nil {
		globalReporter.ReportPanic(ctx, recovered, stack, context)
	}
}

// RecoveryMiddleware creates a Gin middleware that recovers from panics and reports them
func RecoveryMiddleware(logger *zerolog.Logger) gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultWriter, func(c *gin.Context, recovered interface{}) {
		// Capture stack trace
		stack := make([]byte, 4*1024) // 4KB stack trace
		length := runtime.Stack(stack, false)
		stack = stack[:length]

		// Create context
		errorContext := &Context{
			RequestID:     c.GetHeader("X-Request-ID"),
			CorrelationID: c.GetHeader("X-Correlation-ID"),
			IPAddress:     c.ClientIP(),
			UserAgent:     c.GetHeader("User-Agent"),
		}

		// Report panic
		ReportPanic(c.Request.Context(), recovered, stack, errorContext)

		// Log the panic
		logger.Error().
			Interface("panic", recovered).
			Str("stack", string(stack)).
			Str("request_id", errorContext.RequestID).
			Msg("Panic recovered")
	})
}
