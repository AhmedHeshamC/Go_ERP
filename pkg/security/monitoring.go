package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"erpgo/pkg/cache"
)

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID           uuid.UUID              `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Level        SecurityLevel          `json:"level"`
	Type         SecurityEventType      `json:"type"`
	Category     SecurityEventCategory  `json:"category"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Source       string                 `json:"source"`
	ClientIP     string                 `json:"client_ip"`
	UserAgent    string                 `json:"user_agent"`
	UserID       string                 `json:"user_id,omitempty"`
	Username     string                 `json:"username,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	Path         string                 `json:"path"`
	Method       string                 `json:"method"`
	StatusCode   int                    `json:"status_code"`
	Severity     SecuritySeverity       `json:"severity"`
	RiskScore    int                    `json:"risk_score"`
	Confirmed    bool                   `json:"confirmed"`
	Investigated bool                   `json:"investigated"`
	Resolved     bool                   `json:"resolved"`
	Metadata     map[string]interface{} `json:"metadata"`
	Tags         []string               `json:"tags"`
}

// SecurityLevel represents the severity level of a security event
type SecurityLevel string

const (
	SecurityLevelInfo     SecurityLevel = "info"
	SecurityLevelWarning  SecurityLevel = "warning"
	SecurityLevelError    SecurityLevel = "error"
	SecurityLevelCritical SecurityLevel = "critical"
)

// SecurityEventType represents the type of security event
type SecurityEventType string

const (
	EventTypeAuthentication      SecurityEventType = "authentication"
	EventTypeAuthorization       SecurityEventType = "authorization"
	EventTypeInjection          SecurityEventType = "injection"
	EventTypeXSS                SecurityEventType = "xss"
	EventTypeCSRF               SecurityEventType = "csrf"
	EventTypeRateLimit          SecurityEventType = "rate_limit"
	EventTypeSuspiciousActivity SecurityEventType = "suspicious_activity"
	EventTypeDataBreach         SecurityEventType = "data_breach"
	EventTypeMalware            SecurityEventType = "malware"
	EventTypeDoS                SecurityEventType = "dos"
	EventTypeConfigChange       SecurityEventType = "config_change"
	EventTypePrivilegeEscalation SecurityEventType = "privilege_escalation"
)

// SecurityEventCategory represents the category of security event
type SecurityEventCategory string

const (
	CategoryThreat     SecurityEventCategory = "threat"
	CategoryVulnerability SecurityEventCategory = "vulnerability"
	CategoryIncident   SecurityEventCategory = "incident"
	CategoryAnomaly    SecurityEventCategory = "anomaly"
	CategoryCompliance SecurityEventCategory = "compliance"
)

// SecuritySeverity represents the severity of a security event
type SecuritySeverity string

const (
	SeverityLow      SecuritySeverity = "low"
	SeverityMedium   SecuritySeverity = "medium"
	SeverityHigh     SecuritySeverity = "high"
	SeverityCritical SecuritySeverity = "critical"
)

// SecurityConfig holds configuration for security monitoring
type SecurityConfig struct {
	// Event collection
	Enabled                 bool          `json:"enabled"`
	RetentionPeriod         time.Duration `json:"retention_period"`
	MaxEventsPerSecond      int           `json:"max_events_per_second"`
	BufferSize              int           `json:"buffer_size"`
	FlushInterval           time.Duration `json:"flush_interval"`

	// Alerting
	AlertEnabled            bool          `json:"alert_enabled"`
	AlertThresholds         map[SecurityLevel]int `json:"alert_thresholds"`
	AlertCooldown           time.Duration `json:"alert_cooldown"`
	AlertChannels           []string      `json:"alert_channels"`

	// Alert channel configurations
	EmailConfig             *EmailAlertConfig    `json:"email_config,omitempty"`
	SlackConfig             *SlackAlertConfig    `json:"slack_config,omitempty"`
	WebhookConfig           *WebhookAlertConfig  `json:"webhook_config,omitempty"`

	// Risk scoring
	RiskScoringEnabled      bool          `json:"risk_scoring_enabled"`
	BaseRiskScore           int           `json:"base_risk_score"`
	MaliciousIPScore        int           `json:"malicious_ip_score"`
	SuspiciousPatternScore  int           `json:"suspicious_pattern_score"`
	FailedAuthScore         int           `json:"failed_auth_score"`
	PrivilegeEscalationScore int          `json:"privilege_escalation_score"`

	// Detection rules
	FailedAuthThreshold     int           `json:"failed_auth_threshold"`
	FailedAuthWindow        time.Duration `json:"failed_auth_window"`
	RateLimitThreshold      int           `json:"rate_limit_threshold"`
	RateLimitWindow         time.Duration `json:"rate_limit_window"`
	SuspiciousPatterns      []string      `json:"suspicious_patterns"`
	MaliciousIPs            []string      `json:"malicious_ips"`

	// Threat intelligence
	ThreatIntelEnabled      bool          `json:"threat_intel_enabled"`
	ThreatIntelSources      []string      `json:"threat_intel_sources"`
	ThreatIntelCacheTTL     time.Duration `json:"threat_intel_cache_ttl"`

	// Storage
	StoreInCache            bool          `json:"store_in_cache"`
	StoreInFile             bool          `json:"store_in_file"`
	StoreInDatabase         bool          `json:"store_in_database"`

	// Logging
	LogLevel                SecurityLevel `json:"log_level"`
	LogSecurityEvents       bool          `json:"log_security_events"`
	LogThreatIntelligence   bool          `json:"log_threat_intelligence"`
}

// EmailAlertConfig holds configuration for email alerts
type EmailAlertConfig struct {
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	SMTPUsername string   `json:"smtp_username"`
	SMTPPassword string   `json:"smtp_password"`
	FromEmail    string   `json:"from_email"`
	ToEmails     []string `json:"to_emails"`
	Subject      string   `json:"subject"`
	UseTLS       bool     `json:"use_tls"`
}

// SlackAlertConfig holds configuration for Slack alerts
type SlackAlertConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	IconEmoji  string `json:"icon_emoji"`
}

// WebhookAlertConfig holds configuration for webhook alerts
type WebhookAlertConfig struct {
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Timeout  time.Duration     `json:"timeout"`
	Retries  int               `json:"retries"`
}

// ThreatIntelResponse represents a threat intelligence response
type ThreatIntelResponse struct {
	IsMalicious   bool      `json:"is_malicious"`
	Confidence    float64   `json:"confidence"`
	LastSeen      time.Time `json:"last_seen"`
	ThreatTypes   []string  `json:"threat_types"`
	Reputation    int       `json:"reputation"` // -100 to 100
	Source        string    `json:"source"`
	FirstSeen     time.Time `json:"first_seen"`
	ASN           string    `json:"asn"`
	Country       string    `json:"country"`
	Organization  string    `json:"organization"`
}

// PatternMatcher represents a compiled pattern for security detection
type PatternMatcher struct {
	Pattern     *regexp.Regexp `json:"-"`
	RawPattern  string         `json:"raw_pattern"`
	Description string         `json:"description"`
	Severity    SecuritySeverity `json:"severity"`
	ThreatType  string         `json:"threat_type"`
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		Enabled:                true,
		RetentionPeriod:        30 * 24 * time.Hour, // 30 days
		MaxEventsPerSecond:     1000,
		BufferSize:             10000,
		FlushInterval:          5 * time.Second,

		AlertEnabled:           true,
		AlertThresholds: map[SecurityLevel]int{
			SecurityLevelInfo:     0,   // No alert for info
			SecurityLevelWarning:  10,  // Alert after 10 warnings
			SecurityLevelError:    5,   // Alert after 5 errors
			SecurityLevelCritical: 1,   // Alert immediately for critical
		},
		AlertCooldown:          15 * time.Minute,
		AlertChannels:          []string{"email", "slack", "webhook"},

		RiskScoringEnabled:     true,
		BaseRiskScore:          10,
		MaliciousIPScore:       50,
		SuspiciousPatternScore: 30,
		FailedAuthScore:        20,
		PrivilegeEscalationScore: 40,

		FailedAuthThreshold:    5,
		FailedAuthWindow:       5 * time.Minute,
		RateLimitThreshold:     100,
		RateLimitWindow:        time.Minute,
		SuspiciousPatterns: []string{
			`(?i)(union\s+select|select\s+.*\s+from\s+|insert\s+into)`,
			`(?i)(<script|javascript:|vbscript:)`,
			`(?i)(\.\./|\.\.\\|%2e%2e%2f)`,
			`(?i)(wget\s+|curl\s+|nc\s+|netcat\s+)`,
		},
		MaliciousIPs: []string{},

		StoreInCache:           true,
		StoreInFile:            true,
		StoreInDatabase:        false, // Implement if needed

		LogLevel:               SecurityLevelWarning,
		LogSecurityEvents:      true,
		LogThreatIntelligence:  true,
	}
}

// SecurityMonitor represents a security monitoring system
type SecurityMonitor struct {
	config     SecurityConfig
	cache      cache.Cache
	logger     zerolog.Logger
	eventChan  chan *SecurityEvent
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex

	// State tracking
	failedAuths        map[string][]time.Time       // IP -> failed auth attempts
	rateLimitHits      map[string][]time.Time       // IP -> rate limit hits
	alertCounters      map[SecurityLevel]int        // Level -> count since last alert
	lastAlertTimes     map[SecurityLevel]time.Time  // Level -> last alert time
	threatCache        map[string]bool              // IP -> is malicious (simple cache)
	threatIntelCache   map[string]*ThreatIntelResponse // IP -> detailed threat intel

	// Pattern matching
	compiledPatterns   []*PatternMatcher            // Pre-compiled regex patterns
	httpClient         *http.Client                 // HTTP client for external services
}

// NewSecurityMonitor creates a new security monitor
func NewSecurityMonitor(config SecurityConfig, cache cache.Cache, logger zerolog.Logger) *SecurityMonitor {
	monitor := &SecurityMonitor{
		config:            config,
		cache:             cache,
		logger:            logger,
		eventChan:         make(chan *SecurityEvent, config.BufferSize),
		stopChan:          make(chan struct{}),
		failedAuths:       make(map[string][]time.Time),
		rateLimitHits:     make(map[string][]time.Time),
		alertCounters:     make(map[SecurityLevel]int),
		lastAlertTimes:    make(map[SecurityLevel]time.Time),
		threatCache:       make(map[string]bool),
		threatIntelCache:  make(map[string]*ThreatIntelResponse),
		compiledPatterns:  compilePatterns(config.SuspiciousPatterns),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	return monitor
}

// compilePatterns compiles regex patterns for efficient pattern matching
func compilePatterns(patterns []string) []*PatternMatcher {
	var matchers []*PatternMatcher

	for i, pattern := range patterns {
		// Compile regex
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Skip invalid patterns
			continue
		}

		// Determine threat type based on pattern content
		threatType := "unknown"
		severity := SeverityMedium

		switch {
		case strings.Contains(strings.ToLower(pattern), "union.*select"):
			threatType = "sql_injection"
			severity = SeverityHigh
		case strings.Contains(strings.ToLower(pattern), "script"):
			threatType = "xss"
			severity = SeverityHigh
		case strings.Contains(strings.ToLower(pattern), "../"):
			threatType = "path_traversal"
			severity = SeverityMedium
		case strings.Contains(strings.ToLower(pattern), "wget|curl|nc"):
			threatType = "command_injection"
			severity = SeverityCritical
		}

		matchers = append(matchers, &PatternMatcher{
			Pattern:     re,
			RawPattern:  pattern,
			Description: fmt.Sprintf("Pattern %d: %s", i+1, threatType),
			Severity:    severity,
			ThreatType:  threatType,
		})
	}

	return matchers
}

// queryThreatIntelligence queries external threat intelligence sources
func (m *SecurityMonitor) queryThreatIntelligence(ip string) (*ThreatIntelResponse, error) {
	if !m.config.ThreatIntelEnabled {
		return nil, fmt.Errorf("threat intelligence not enabled")
	}

	// Check cache first
	if cached, exists := m.threatIntelCache[ip]; exists {
		return cached, nil
	}

	// This is a mock implementation - in production, you would integrate with:
	// - VirusTotal API
	// - AbuseIPDB
	// - Spamhaus
	// - Custom threat feeds

	// For now, create a mock response based on IP patterns
	response := &ThreatIntelResponse{
		IsMalicious: false,
		Confidence:  0.0,
		LastSeen:    time.Now(),
		ThreatTypes: []string{},
		Reputation:  0,
		Source:      "internal_mock",
		FirstSeen:   time.Now().Add(-24 * time.Hour),
	}

	// Simple heuristics for demo purposes
	if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "172.") {
		// Private IP - not malicious
		response.Reputation = 10
	} else if ip == "127.0.0.1" || ip == "::1" {
		// localhost - safe
		response.Reputation = 20
	} else {
		// Public IP - unknown reputation
		response.Reputation = 0
		response.Confidence = 0.5
	}

	// Cache the result
	m.threatIntelCache[ip] = response

	if m.config.LogThreatIntelligence {
		m.logger.Info().
			Str("ip", ip).
			Bool("malicious", response.IsMalicious).
			Int("reputation", response.Reputation).
			Str("source", response.Source).
			Msg("Threat intelligence lookup completed")
	}

	return response, nil
}

// Start starts the security monitor
func (m *SecurityMonitor) Start() {
	if !m.config.Enabled {
		return
	}

	// Start event processor
	m.wg.Add(1)
	go m.eventProcessor()

	// Start cleanup routine
	m.wg.Add(1)
	go m.cleanupRoutine()

	m.logger.Info().Msg("Security monitor started")
}

// Stop stops the security monitor
func (m *SecurityMonitor) Stop() {
	if !m.config.Enabled {
		return
	}

	close(m.stopChan)
	close(m.eventChan)
	m.wg.Wait()

	m.logger.Info().Msg("Security monitor stopped")
}

// LogSecurityEvent logs a security event
func (m *SecurityMonitor) LogSecurityEvent(event *SecurityEvent) {
	if !m.config.Enabled {
		return
	}

	// Ensure event has required fields
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Calculate risk score if enabled
	if m.config.RiskScoringEnabled {
		event.RiskScore = m.calculateRiskScore(event)
	}

	// Check for alert conditions
	m.checkAlertConditions(event)

	// Send event to processor
	select {
	case m.eventChan <- event:
	default:
		m.logger.Warn().Msg("Security event channel full, dropping event")
	}
}

// AuthenticationFailed logs a failed authentication attempt
func (m *SecurityMonitor) AuthenticationFailed(clientIP, userAgent, userID, reason string) {
	m.LogSecurityEvent(&SecurityEvent{
		Type:        EventTypeAuthentication,
		Category:    CategoryThreat,
		Title:       "Authentication Failed",
		Description: fmt.Sprintf("Failed authentication attempt from %s: %s", clientIP, reason),
		Source:      "auth_service",
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		UserID:      userID,
		Level:       SecurityLevelWarning,
		Severity:    SeverityMedium,
		Metadata: map[string]interface{}{
			"reason": reason,
		},
		Tags: []string{"auth", "failure"},
	})
}

// UnauthorizedAccess logs an unauthorized access attempt
func (m *SecurityMonitor) UnauthorizedAccess(clientIP, userAgent, userID, path, method string) {
	m.LogSecurityEvent(&SecurityEvent{
		Type:        EventTypeAuthorization,
		Category:    CategoryThreat,
		Title:       "Unauthorized Access Attempt",
		Description: fmt.Sprintf("Unauthorized access attempt to %s %s from %s", method, path, clientIP),
		Source:      "auth_middleware",
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		UserID:      userID,
		Path:        path,
		Method:      method,
		Level:       SecurityLevelWarning,
		Severity:    SeverityMedium,
		Tags:        []string{"auth", "unauthorized"},
	})
}

// SuspiciousActivity logs suspicious activity
func (m *SecurityMonitor) SuspiciousActivity(clientIP, userAgent, userID, activity string, metadata map[string]interface{}) {
	m.LogSecurityEvent(&SecurityEvent{
		Type:        EventTypeSuspiciousActivity,
		Category:    CategoryAnomaly,
		Title:       "Suspicious Activity Detected",
		Description: fmt.Sprintf("Suspicious activity detected from %s: %s", clientIP, activity),
		Source:      "security_monitor",
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		UserID:      userID,
		Level:       SecurityLevelWarning,
		Severity:    SeverityMedium,
		Metadata:    metadata,
		Tags:        []string{"suspicious", "activity"},
	})
}

// InjectionAttempt logs an injection attempt
func (m *SecurityMonitor) InjectionAttempt(clientIP, userAgent, userID, injectionType, payload string) {
	m.LogSecurityEvent(&SecurityEvent{
		Type:        EventTypeInjection,
		Category:    CategoryThreat,
		Title:       "Injection Attempt Detected",
		Description: fmt.Sprintf("%s injection attempt detected from %s", injectionType, clientIP),
		Source:      "input_validator",
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		UserID:      userID,
		Level:       SecurityLevelError,
		Severity:    SeverityHigh,
		Metadata: map[string]interface{}{
			"injection_type": injectionType,
			"payload":        payload,
		},
		Tags: []string{"injection", "threat"},
	})
}

// RateLimitExceeded logs a rate limit violation
func (m *SecurityMonitor) RateLimitExceeded(clientIP, userAgent, userID, endpoint string) {
	m.LogSecurityEvent(&SecurityEvent{
		Type:        EventTypeRateLimit,
		Category:    CategoryAnomaly,
		Title:       "Rate Limit Exceeded",
		Description: fmt.Sprintf("Rate limit exceeded for %s from %s", endpoint, clientIP),
		Source:      "rate_limiter",
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		UserID:      userID,
		Level:       SecurityLevelWarning,
		Severity:    SeverityMedium,
		Metadata: map[string]interface{}{
			"endpoint": endpoint,
		},
		Tags: []string{"rate_limit", "anomaly"},
	})
}

// PrivilegeEscalation logs a privilege escalation attempt
func (m *SecurityMonitor) PrivilegeEscalation(clientIP, userAgent, userID, attemptedRole string) {
	m.LogSecurityEvent(&SecurityEvent{
		Type:        EventTypePrivilegeEscalation,
		Category:    CategoryThreat,
		Title:       "Privilege Escalation Attempt",
		Description: fmt.Sprintf("Privilege escalation attempt to %s from %s", attemptedRole, clientIP),
		Source:      "auth_middleware",
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		UserID:      userID,
		Level:       SecurityLevelError,
		Severity:    SeverityHigh,
		Metadata: map[string]interface{}{
			"attempted_role": attemptedRole,
		},
		Tags: []string{"privilege", "escalation"},
	})
}

// calculateRiskScore calculates the risk score for an event
func (m *SecurityMonitor) calculateRiskScore(event *SecurityEvent) int {
	score := m.config.BaseRiskScore

	// Add score based on event type
	switch event.Type {
	case EventTypeInjection:
		score += 40
	case EventTypeXSS:
		score += 35
	case EventTypeAuthentication:
		score += 20
	case EventTypeAuthorization:
		score += 25
	case EventTypePrivilegeEscalation:
		score += m.config.PrivilegeEscalationScore
	}

	// Add score based on client IP
	if m.isMaliciousIP(event.ClientIP) {
		score += m.config.MaliciousIPScore
	}

	// Add score based on suspicious patterns
	if m.hasSuspiciousPattern(event.Description) {
		score += m.config.SuspiciousPatternScore
	}

	// Cap the score at 100
	if score > 100 {
		score = 100
	}

	return score
}

// isMaliciousIP checks if an IP is known to be malicious
func (m *SecurityMonitor) isMaliciousIP(ip string) bool {
	// Check cache first
	m.mu.RLock()
	if malicious, exists := m.threatCache[ip]; exists {
		m.mu.RUnlock()
		return malicious
	}
	m.mu.RUnlock()

	// Check configured malicious IPs
	for _, maliciousIP := range m.config.MaliciousIPs {
		if ip == maliciousIP {
			m.mu.Lock()
			m.threatCache[ip] = true
			m.mu.Unlock()
			return true
		}
	}

	// Query threat intelligence
	if m.config.ThreatIntelEnabled {
		threatIntel, err := m.queryThreatIntelligence(ip)
		if err == nil && threatIntel != nil {
			isMalicious := threatIntel.IsMalicious || threatIntel.Reputation < -50

			m.mu.Lock()
			m.threatCache[ip] = isMalicious
			m.mu.Unlock()

			return isMalicious
		}
	}

	// Default to non-malicious if no threat intelligence available
	m.mu.Lock()
	m.threatCache[ip] = false
	m.mu.Unlock()
	return false
}

// hasSuspiciousPattern checks if text contains suspicious patterns using compiled regex
func (m *SecurityMonitor) hasSuspiciousPattern(text string) bool {
	// Use compiled patterns for efficient matching
	for _, matcher := range m.compiledPatterns {
		if matcher.Pattern.MatchString(text) {
			if m.config.LogThreatIntelligence {
				m.logger.Warn().
					Str("pattern", matcher.RawPattern).
					Str("threat_type", matcher.ThreatType).
					Str("severity", string(matcher.Severity)).
					Msg("Suspicious pattern detected")
			}
			return true
		}
	}

	// Fallback to simple string matching for any remaining patterns
	for _, pattern := range m.config.SuspiciousPatterns {
		if strings.Contains(strings.ToLower(text), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// checkAlertConditions checks if an alert should be triggered
func (m *SecurityMonitor) checkAlertConditions(event *SecurityEvent) {
	if !m.config.AlertEnabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment counter
	m.alertCounters[event.Level]++

	// Check if threshold is met
	threshold, exists := m.config.AlertThresholds[event.Level]
	if !exists || threshold <= 0 {
		return
	}

	if m.alertCounters[event.Level] >= threshold {
		// Check cooldown
		if lastAlert, exists := m.lastAlertTimes[event.Level]; exists {
			if time.Since(lastAlert) < m.config.AlertCooldown {
				return
			}
		}

		// Trigger alert
		m.triggerAlert(event)

		// Reset counter and update last alert time
		m.alertCounters[event.Level] = 0
		m.lastAlertTimes[event.Level] = time.Now()
	}
}

// triggerAlert triggers a security alert through configured channels
func (m *SecurityMonitor) triggerAlert(event *SecurityEvent) {
	m.logger.Error().
		Str("event_type", string(event.Type)).
		Str("level", string(event.Level)).
		Str("client_ip", event.ClientIP).
		Int("risk_score", event.RiskScore).
		Msg("Security alert triggered")

	// Send alerts through configured channels
	for _, channel := range m.config.AlertChannels {
		switch channel {
		case "email":
			m.sendEmailAlert(event)
		case "slack":
			m.sendSlackAlert(event)
		case "webhook":
			m.sendWebhookAlert(event)
		default:
			m.logger.Warn().Str("channel", channel).Msg("Unknown alert channel")
		}
	}
}

// sendEmailAlert sends an email alert
func (m *SecurityMonitor) sendEmailAlert(event *SecurityEvent) {
	if m.config.EmailConfig == nil {
		m.logger.Warn().Msg("Email alert requested but no email configuration provided")
		return
	}

	// This is a mock implementation - in production, integrate with an SMTP library
	m.logger.Info().
		Str("smtp_host", m.config.EmailConfig.SMTPHost).
		Str("from", m.config.EmailConfig.FromEmail).
		Strs("to", m.config.EmailConfig.ToEmails).
		Str("subject", fmt.Sprintf("Security Alert: %s", event.Type)).
		Msg("Email alert sent (mock implementation)")
}

// sendSlackAlert sends a Slack alert
func (m *SecurityMonitor) sendSlackAlert(event *SecurityEvent) {
	if m.config.SlackConfig == nil || m.config.SlackConfig.WebhookURL == "" {
		m.logger.Warn().Msg("Slack alert requested but no Slack configuration provided")
		return
	}

	// Create Slack message payload
	payload := map[string]interface{}{
		"channel":   m.config.SlackConfig.Channel,
		"username":  m.config.SlackConfig.Username,
		"icon_emoji": m.config.SlackConfig.IconEmoji,
		"text":      fmt.Sprintf("🚨 Security Alert: %s", event.Type),
		"attachments": []map[string]interface{}{
			{
				"color": "danger",
				"fields": []map[string]interface{}{
					{
						"title": "Event Type",
						"value": string(event.Type),
						"short": true,
					},
					{
						"title": "Severity",
						"value": string(event.Level),
						"short": true,
					},
					{
						"title": "Client IP",
						"value": event.ClientIP,
						"short": true,
					},
					{
						"title": "Risk Score",
						"value": fmt.Sprintf("%d", event.RiskScore),
						"short": true,
					},
					{
						"title": "Description",
						"value": event.Description,
						"short": false,
					},
				},
				"ts": event.Timestamp.Unix(),
			},
		},
	}

	// Convert to JSON to validate payload structure
	_, err := json.Marshal(payload)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to marshal Slack payload")
		return
	}

	// Send to Slack webhook (mock implementation for now)
	m.logger.Info().
		Str("webhook_url", m.config.SlackConfig.WebhookURL).
		Str("channel", m.config.SlackConfig.Channel).
		Msg("Slack alert sent (mock implementation)")
}

// sendWebhookAlert sends a webhook alert
func (m *SecurityMonitor) sendWebhookAlert(event *SecurityEvent) {
	if m.config.WebhookConfig == nil || m.config.WebhookConfig.URL == "" {
		m.logger.Warn().Msg("Webhook alert requested but no webhook configuration provided")
		return
	}

	// Create webhook payload
	payload := map[string]interface{}{
		"alert_id":   event.ID.String(),
		"timestamp":  event.Timestamp,
		"event":      event,
		"severity":   string(event.Level),
		"risk_score": event.RiskScore,
	}

	// Convert to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to marshal webhook payload")
		return
	}

	// Send webhook (mock implementation for now)
	m.logger.Info().
		Str("url", m.config.WebhookConfig.URL).
		Str("method", m.config.WebhookConfig.Method).
		Int("payload_size", len(payloadBytes)).
		Msg("Webhook alert sent (mock implementation)")
}

// eventProcessor processes security events
func (m *SecurityMonitor) eventProcessor() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.FlushInterval)
	defer ticker.Stop()

	eventBatch := make([]*SecurityEvent, 0, 100)

	for {
		select {
		case event := <-m.eventChan:
			eventBatch = append(eventBatch, event)

			// Process batch if it reaches the buffer size
			if len(eventBatch) >= 100 {
				m.processEventBatch(eventBatch)
				eventBatch = eventBatch[:0]
			}

		case <-ticker.C:
			// Process any pending events
			if len(eventBatch) > 0 {
				m.processEventBatch(eventBatch)
				eventBatch = eventBatch[:0]
			}

		case <-m.stopChan:
			// Process remaining events before stopping
			if len(eventBatch) > 0 {
				m.processEventBatch(eventBatch)
			}
			return
		}
	}
}

// processEventBatch processes a batch of security events
func (m *SecurityMonitor) processEventBatch(events []*SecurityEvent) {
	for _, event := range events {
		// Log event if enabled
		if m.config.LogSecurityEvents {
			m.logEvent(event)
		}

		// Store event if enabled
		if m.config.StoreInCache {
			m.storeEvent(event)
		}

		// Update state tracking
		m.updateStateTracking(event)
	}
}

// logEvent logs a security event
func (m *SecurityMonitor) logEvent(event *SecurityEvent) {
	logEvent := m.logger.WithLevel(m.getLogLevel(event.Level))

	logEvent.
		Str("event_id", event.ID.String()).
		Str("event_type", string(event.Type)).
		Str("category", string(event.Category)).
		Str("level", string(event.Level)).
		Str("severity", string(event.Severity)).
		Str("title", event.Title).
		Str("client_ip", event.ClientIP).
		Str("user_id", event.UserID).
		Int("risk_score", event.RiskScore).
		Msg("Security event")

	// Log additional metadata if it's a high-severity event
	if event.Severity == SeverityHigh || event.Severity == SeverityCritical {
		logEvent.Interface("metadata", event.Metadata).Msg("Security event details")
	}
}

// storeEvent stores a security event in cache
func (m *SecurityMonitor) storeEvent(event *SecurityEvent) {
	key := fmt.Sprintf("security:event:%s", event.ID.String())
	m.cache.SetJSON(context.Background(), key, event, m.config.RetentionPeriod)
}

// updateStateTracking updates internal state tracking
func (m *SecurityMonitor) updateStateTracking(event *SecurityEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Track failed authentication attempts
	if event.Type == EventTypeAuthentication && event.Level == SecurityLevelWarning {
		m.failedAuths[event.ClientIP] = append(m.failedAuths[event.ClientIP], now)
		m.cleanupOldEntries(event.ClientIP, m.failedAuths, m.config.FailedAuthWindow)
	}

	// Track rate limit hits
	if event.Type == EventTypeRateLimit {
		m.rateLimitHits[event.ClientIP] = append(m.rateLimitHits[event.ClientIP], now)
		m.cleanupOldEntries(event.ClientIP, m.rateLimitHits, m.config.RateLimitWindow)
	}
}

// cleanupOldEntries removes old entries from tracking maps
func (m *SecurityMonitor) cleanupOldEntries(key string, entries map[string][]time.Time, window time.Duration) {
	cutoff := time.Now().Add(-window)
	if times, exists := entries[key]; exists {
		var validTimes []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}
		if len(validTimes) > 0 {
			entries[key] = validTimes
		} else {
			delete(entries, key)
		}
	}
}

// cleanupRoutine performs periodic cleanup
func (m *SecurityMonitor) cleanupRoutine() {
	defer m.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performCleanup()

		case <-m.stopChan:
			return
		}
	}
}

// performCleanup performs cleanup operations
func (m *SecurityMonitor) performCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean up old failed auth entries
	for ip := range m.failedAuths {
		m.cleanupOldEntries(ip, m.failedAuths, m.config.FailedAuthWindow)
	}

	// Clean up old rate limit entries
	for ip := range m.rateLimitHits {
		m.cleanupOldEntries(ip, m.rateLimitHits, m.config.RateLimitWindow)
	}

	// Clean up threat cache (re-evaluate periodically)
	for ip := range m.threatCache {
		delete(m.threatCache, ip)
	}
}

// getLogLevel converts security level to zerolog level
func (m *SecurityMonitor) getLogLevel(level SecurityLevel) zerolog.Level {
	switch level {
	case SecurityLevelInfo:
		return zerolog.InfoLevel
	case SecurityLevelWarning:
		return zerolog.WarnLevel
	case SecurityLevelError:
		return zerolog.ErrorLevel
	case SecurityLevelCritical:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// GetSecurityEvents retrieves security events from cache/memory
func (m *SecurityMonitor) GetSecurityEvents(ctx context.Context, limit int, offset int) ([]*SecurityEvent, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	events := make([]*SecurityEvent, 0)

	// Try to retrieve events from cache first
	if m.config.StoreInCache && m.cache != nil {
		// Get event keys from cache
		eventKeys, err := m.getEventKeysFromCache(ctx, limit, offset)
		if err == nil && len(eventKeys) > 0 {
			for _, key := range eventKeys {
				if eventDataStr, err := m.cache.Get(ctx, key); err == nil && eventDataStr != "" {
					// Try to unmarshal JSON string to SecurityEvent
					var event SecurityEvent
					if unmarshalErr := json.Unmarshal([]byte(eventDataStr), &event); unmarshalErr == nil {
						events = append(events, &event)
					}
				}
			}
			return events, nil
		}
	}

	// Fallback to in-memory storage (if events are stored in memory)
	m.mu.RLock()
	defer m.mu.RUnlock()

	// This would be populated by the event processor in a real implementation
	// For now, return mock events if cache is not available
	if len(events) == 0 {
		// Generate some mock events for demonstration
		mockEvent := &SecurityEvent{
			ID:          uuid.New(),
			Timestamp:   time.Now().Add(-time.Hour),
			Level:       SecurityLevelWarning,
			Type:        EventTypeSuspiciousActivity,
			Category:    CategoryThreat,
			Title:       "Sample Security Event",
			Description: "This is a sample security event for demonstration purposes",
			Source:      "security-monitor",
			ClientIP:    "192.168.1.100",
			Path:        "/api/v1/users",
			Method:      "POST",
			StatusCode:  200,
			Severity:    SeverityMedium,
			RiskScore:   25,
			Confirmed:   false,
		}
		events = append(events, mockEvent)
	}

	return events, nil
}

// getEventKeysFromCache retrieves security event keys from cache
func (m *SecurityMonitor) getEventKeysFromCache(ctx context.Context, limit int, offset int) ([]string, error) {
	// In a real implementation, this would query cache for event keys
	// For now, return empty slice as this depends on cache implementation
	return []string{}, nil
}

// GetSecurityStats returns security statistics
func (m *SecurityMonitor) GetSecurityStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"failed_auth_count":      len(m.failedAuths),
		"rate_limit_hits":        len(m.rateLimitHits),
		"malicious_ips_cached":   len(m.threatCache),
		"alert_counters":         m.alertCounters,
		"last_alert_times":       m.lastAlertTimes,
	}
}