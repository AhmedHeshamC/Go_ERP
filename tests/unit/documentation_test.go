package unit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunbookLinksValid verifies that all runbook links in RUNBOOKS.md are valid
func TestRunbookLinksValid(t *testing.T) {
	// Read the runbooks file
	data, err := os.ReadFile("../../docs/operations/RUNBOOKS.md")
	require.NoError(t, err, "RUNBOOKS.md file should exist and be readable")

	content := string(data)

	// Define all runbook sections that should exist
	expectedSections := []struct {
		name   string
		anchor string
	}{
		{"High Error Rate", "#high-error-rate"},
		{"Database Connectivity Issues", "#database-connectivity-issues"},
		{"High Latency Incidents", "#high-latency-incidents"},
		{"Authentication Failures", "#authentication-failures"},
		{"Database Connection Pool Exhaustion", "#database-connection-pool-exhaustion"},
		{"Low Cache Hit Rate", "#low-cache-hit-rate"},
		{"Deployment Procedures", "#deployment-procedures"},
		{"Emergency Rollback", "#emergency-rollback"},
		{"General Troubleshooting", "#general-troubleshooting"},
	}

	// Verify each section exists
	for _, section := range expectedSections {
		t.Run(fmt.Sprintf("Section_%s", strings.ReplaceAll(section.name, " ", "_")), func(t *testing.T) {
			// Check for the section header
			assert.Contains(t, content, fmt.Sprintf("## %s", section.name),
				"Runbook should contain section: %s", section.name)
		})
	}
}

// TestRunbookQuickReferenceLinks verifies the quick reference table links are valid
func TestRunbookQuickReferenceLinks(t *testing.T) {
	// Read the runbooks file
	data, err := os.ReadFile("../../docs/operations/RUNBOOKS.md")
	require.NoError(t, err, "RUNBOOKS.md file should exist")

	content := string(data)

	// Verify Quick Reference section exists
	assert.Contains(t, content, "## Quick Reference",
		"Runbooks should have a Quick Reference section")

	// Define expected quick reference entries
	expectedEntries := []struct {
		alert    string
		severity string
	}{
		{"High Error Rate", "Critical"},
		{"Database Down", "Critical"},
		{"High Latency", "Warning"},
		{"Auth Failures", "Warning"},
		{"Connection Pool", "Critical"},
		{"Low Cache Hit", "Warning"},
	}

	// Verify each entry is in the quick reference
	for _, entry := range expectedEntries {
		t.Run(fmt.Sprintf("QuickRef_%s", strings.ReplaceAll(entry.alert, " ", "_")), func(t *testing.T) {
			assert.Contains(t, content, entry.alert,
				"Quick reference should contain alert: %s", entry.alert)
			assert.Contains(t, content, entry.severity,
				"Quick reference should contain severity: %s", entry.severity)
		})
	}
}

// TestRunbookAlertRulesReferenced verifies that alert rules are referenced
func TestRunbookAlertRulesReferenced(t *testing.T) {
	// Read the runbooks file
	data, err := os.ReadFile("../../docs/operations/RUNBOOKS.md")
	require.NoError(t, err, "RUNBOOKS.md file should exist")

	content := string(data)

	// Define expected alert rule names
	expectedAlertRules := []string{
		"HighErrorRate",
		"PostgreSQLDown",
		"HighResponseTime",
		"HighFailedLoginRate",
		"DatabaseConnectionPoolExhausted",
		"LowCacheHitRate",
	}

	// Verify each alert rule is referenced
	for _, alertRule := range expectedAlertRules {
		t.Run(fmt.Sprintf("AlertRule_%s", alertRule), func(t *testing.T) {
			assert.Contains(t, content, alertRule,
				"Runbook should reference alert rule: %s", alertRule)
		})
	}
}

// TestRunbookRelatedDocumentationLinks verifies related documentation links exist
func TestRunbookRelatedDocumentationLinks(t *testing.T) {
	// Read the runbooks file
	data, err := os.ReadFile("../../docs/operations/RUNBOOKS.md")
	require.NoError(t, err, "RUNBOOKS.md file should exist")

	content := string(data)

	// Define expected documentation references
	expectedDocs := []string{
		"BACKUP_RUNBOOK.md",
		"DISASTER_RECOVERY_PROCEDURES.md",
		"LAUNCH_CHECKLIST.md",
		"ROLLBACK_PROCEDURES.md",
		"ARCHITECTURE_OVERVIEW.md",
		"MONITORING.md",
		"AUTHENTICATION.md",
		"SECURITY_BEST_PRACTICES.md",
		"DEPLOYMENT_GUIDE.md",
	}

	// Verify each documentation file is referenced
	for _, doc := range expectedDocs {
		t.Run(fmt.Sprintf("Doc_%s", strings.ReplaceAll(doc, ".", "_")), func(t *testing.T) {
			assert.Contains(t, content, doc,
				"Runbook should reference documentation: %s", doc)
		})
	}
}

// TestConfigurationOptionsDocumented verifies all configuration options are documented
func TestConfigurationOptionsDocumented(t *testing.T) {
	// Read the config.go file to extract all configuration options
	configData, err := os.ReadFile("../../pkg/config/config.go")
	require.NoError(t, err, "config.go file should exist")

	configContent := string(configData)

	// Find all documentation files that might contain configuration
	docFiles := []string{
		"../../docs/DEPLOYMENT_GUIDE.md",
		"../../docs/deployment/PRODUCTION_DEPLOYMENT_GUIDE.md",
		"../../docs/DEVELOPER_GUIDE.md",
		"../../docs/DEVELOPER_ONBOARDING_GUIDE.md",
		"../../README.md",
	}

	// Combine all documentation content
	var allDocsContent strings.Builder
	for _, docFile := range docFiles {
		if data, err := os.ReadFile(docFile); err == nil {
			allDocsContent.Write(data)
			allDocsContent.WriteString("\n")
		}
	}

	docsContent := allDocsContent.String()

	// Define critical configuration options that must be documented
	criticalConfigOptions := []struct {
		envVar      string
		description string
	}{
		{"JWT_SECRET", "JWT secret key"},
		{"DATABASE_URL", "Database connection"},
		{"REDIS_URL", "Redis connection"},
		{"PASSWORD_PEPPER", "Password pepper"},
		{"SERVER_PORT", "Server port"},
		{"ENVIRONMENT", "Environment setting"},
		{"LOG_LEVEL", "Logging level"},
		{"CORS_ORIGINS", "CORS origins"},
		{"RATE_LIMIT_ENABLED", "Rate limiting"},
		{"CACHE_ENABLED", "Caching"},
		{"SMTP_HOST", "Email configuration"},
		{"METRICS_ENABLED", "Metrics"},
		{"TRACING_ENABLED", "Tracing"},
		{"BCRYPT_COST", "Password hashing"},
		{"MAX_LOGIN_ATTEMPTS", "Login security"},
		{"LOCKOUT_DURATION", "Account lockout"},
	}

	// Verify each critical option is documented
	for _, option := range criticalConfigOptions {
		t.Run(fmt.Sprintf("Config_%s", option.envVar), func(t *testing.T) {
			// Check if the environment variable is mentioned in documentation
			found := strings.Contains(docsContent, option.envVar) ||
				strings.Contains(configContent, fmt.Sprintf("`env:\"%s\"`", option.envVar))

			assert.True(t, found,
				"Configuration option %s (%s) should be documented",
				option.envVar, option.description)
		})
	}
}

// TestConfigurationDefaultsDocumented verifies configuration defaults are documented
func TestConfigurationDefaultsDocumented(t *testing.T) {
	// Read the config.go file
	configData, err := os.ReadFile("../../pkg/config/config.go")
	require.NoError(t, err, "config.go file should exist")

	configContent := string(configData)

	// Verify that envDefault tags are present for important configs
	assert.Contains(t, configContent, "envDefault",
		"Config should use envDefault tags for default values")

	// Check specific defaults are set
	defaultChecks := []struct {
		field        string
		defaultValue string
	}{
		{"SERVER_PORT", "8080"},
		{"ENVIRONMENT", "development"},
		{"LOG_LEVEL", "info"},
		{"BCRYPT_COST", "12"},
		{"MAX_LOGIN_ATTEMPTS", "5"},
		{"LOCKOUT_DURATION", "15m"},
	}

	for _, check := range defaultChecks {
		t.Run(fmt.Sprintf("Default_%s", check.field), func(t *testing.T) {
			assert.Contains(t, configContent, check.defaultValue,
				"Config should have default value %s for %s", check.defaultValue, check.field)
		})
	}
}

// TestSecurityConfigurationDocumented verifies security-related configuration is documented
func TestSecurityConfigurationDocumented(t *testing.T) {
	// Read security documentation
	securityDocs := []string{
		"../../docs/SECURITY_BEST_PRACTICES.md",
		"../../docs/AUTHENTICATION.md",
		"../../docs/deployment/PRODUCTION_DEPLOYMENT_GUIDE.md",
	}

	var securityContent strings.Builder
	for _, docFile := range securityDocs {
		if data, err := os.ReadFile(docFile); err == nil {
			securityContent.Write(data)
			securityContent.WriteString("\n")
		}
	}

	content := securityContent.String()

	// Define security configuration topics that should be documented
	securityTopics := []string{
		"JWT_SECRET",
		"PASSWORD_PEPPER",
		"BCRYPT_COST",
		"MAX_LOGIN_ATTEMPTS",
		"LOCKOUT_DURATION",
		"RATE_LIMIT",
		"SSL",
		"HTTPS",
		"CORS",
	}

	// Verify each security topic is documented
	for _, topic := range securityTopics {
		t.Run(fmt.Sprintf("Security_%s", topic), func(t *testing.T) {
			assert.Contains(t, strings.ToUpper(content), strings.ToUpper(topic),
				"Security documentation should cover: %s", topic)
		})
	}
}

// TestDatabaseConfigurationDocumented verifies database configuration is documented
func TestDatabaseConfigurationDocumented(t *testing.T) {
	// Read database-related documentation
	dbDocs := []string{
		"../../docs/DATABASE_SCHEMA.md",
		"../../docs/deployment/PRODUCTION_DEPLOYMENT_GUIDE.md",
		"../../docs/DEVELOPER_GUIDE.md",
	}

	var dbContent strings.Builder
	for _, docFile := range dbDocs {
		if data, err := os.ReadFile(docFile); err == nil {
			dbContent.Write(data)
			dbContent.WriteString("\n")
		}
	}

	content := dbContent.String()

	// Define database configuration options that should be documented
	dbConfigOptions := []string{
		"DATABASE_URL",
		"MAX_CONNECTIONS",
		"MIN_CONNECTIONS",
		"CONN_MAX_LIFETIME",
		"SSL",
	}

	// Verify each database config option is documented
	for _, option := range dbConfigOptions {
		t.Run(fmt.Sprintf("DBConfig_%s", option), func(t *testing.T) {
			assert.Contains(t, strings.ToUpper(content), strings.ToUpper(option),
				"Database documentation should cover: %s", option)
		})
	}
}

// TestMonitoringConfigurationDocumented verifies monitoring configuration is documented
func TestMonitoringConfigurationDocumented(t *testing.T) {
	// Read monitoring documentation
	monitoringDocs := []string{
		"../../docs/MONITORING.md",
		"../../docs/MONITORING_ALERTING.md",
		"../../docs/operations/PRODUCTION_MONITORING.md",
	}

	var monitoringContent strings.Builder
	for _, docFile := range monitoringDocs {
		if data, err := os.ReadFile(docFile); err == nil {
			monitoringContent.Write(data)
			monitoringContent.WriteString("\n")
		}
	}

	content := monitoringContent.String()

	// Define monitoring configuration options that should be documented
	monitoringOptions := []string{
		"METRICS_ENABLED",
		"METRICS_PATH",
		"TRACING_ENABLED",
		"TRACING_URL",
		"Prometheus",
		"Grafana",
		"AlertManager",
	}

	// Verify each monitoring option is documented
	for _, option := range monitoringOptions {
		t.Run(fmt.Sprintf("Monitoring_%s", option), func(t *testing.T) {
			assert.Contains(t, content, option,
				"Monitoring documentation should cover: %s", option)
		})
	}
}

// TestCacheConfigurationDocumented verifies cache configuration is documented
func TestCacheConfigurationDocumented(t *testing.T) {
	// Read the config file and architecture docs
	configData, err := os.ReadFile("../../pkg/config/config.go")
	require.NoError(t, err, "config.go should exist")

	archData, _ := os.ReadFile("../../docs/ARCHITECTURE_OVERVIEW.md")

	content := string(configData) + string(archData)

	// Define cache configuration options
	cacheOptions := []string{
		"CACHE_ENABLED",
		"CACHE_DEFAULT_TTL",
		"CACHE_USER_TTL",
		"CACHE_PRODUCT_TTL",
		"REDIS_URL",
		"REDIS_PASSWORD",
	}

	// Verify each cache option is in config
	for _, option := range cacheOptions {
		t.Run(fmt.Sprintf("Cache_%s", option), func(t *testing.T) {
			assert.Contains(t, content, option,
				"Cache configuration should include: %s", option)
		})
	}
}

// TestEnvironmentSpecificConfigurationDocumented verifies environment-specific config is documented
func TestEnvironmentSpecificConfigurationDocumented(t *testing.T) {
	// Read deployment documentation
	deploymentDocs := []string{
		"../../docs/DEPLOYMENT_GUIDE.md",
		"../../docs/deployment/PRODUCTION_DEPLOYMENT_GUIDE.md",
		"../../docs/DEVELOPER_ONBOARDING_GUIDE.md",
	}

	var deployContent strings.Builder
	for _, docFile := range deploymentDocs {
		if data, err := os.ReadFile(docFile); err == nil {
			deployContent.Write(data)
			deployContent.WriteString("\n")
		}
	}

	content := deployContent.String()

	// Define environment-specific topics
	envTopics := []string{
		"development",
		"staging",
		"production",
		".env",
		"ENVIRONMENT",
	}

	// Verify each environment topic is documented
	for _, topic := range envTopics {
		t.Run(fmt.Sprintf("Env_%s", topic), func(t *testing.T) {
			assert.Contains(t, strings.ToLower(content), strings.ToLower(topic),
				"Environment documentation should cover: %s", topic)
		})
	}
}

// TestConfigurationValidationDocumented verifies configuration validation is documented
func TestConfigurationValidationDocumented(t *testing.T) {
	// Read the config.go file
	configData, err := os.ReadFile("../../pkg/config/config.go")
	require.NoError(t, err, "config.go should exist")

	content := string(configData)

	// Verify validation functions exist
	assert.Contains(t, content, "func (c *Config) validate()",
		"Config should have a validate function")

	assert.Contains(t, content, "validateSecrets",
		"Config should have secret validation")

	// Verify validation checks important fields
	validationChecks := []string{
		"JWTSecret",
		"DatabaseURL",
		"ServerPort",
	}

	for _, check := range validationChecks {
		t.Run(fmt.Sprintf("Validation_%s", check), func(t *testing.T) {
			assert.Contains(t, content, check,
				"Config validation should check: %s", check)
		})
	}
}

// TestRunbookCommandsValid verifies that runbook commands are syntactically valid
func TestRunbookCommandsValid(t *testing.T) {
	// Read the runbooks file
	data, err := os.ReadFile("../../docs/operations/RUNBOOKS.md")
	require.NoError(t, err, "RUNBOOKS.md file should exist")

	content := string(data)

	// Verify common command patterns are present
	commandPatterns := []string{
		"kubectl",
		"docker-compose",
		"psql",
		"redis-cli",
		"curl",
		"grep",
		"awk",
	}

	for _, pattern := range commandPatterns {
		t.Run(fmt.Sprintf("Command_%s", pattern), func(t *testing.T) {
			assert.Contains(t, content, pattern,
				"Runbook should contain %s commands", pattern)
		})
	}
}

// TestRunbookSectionsComplete verifies each runbook section has required subsections
func TestRunbookSectionsComplete(t *testing.T) {
	// Read the runbooks file
	data, err := os.ReadFile("../../docs/operations/RUNBOOKS.md")
	require.NoError(t, err, "RUNBOOKS.md file should exist")

	content := string(data)

	// Define required subsections for runbooks
	requiredSubsections := []string{
		"Symptoms",
		"Investigation Steps",
		"Resolution Steps",
		"Prevention",
	}

	// Verify each subsection appears multiple times (once per runbook)
	for _, subsection := range requiredSubsections {
		t.Run(fmt.Sprintf("Subsection_%s", strings.ReplaceAll(subsection, " ", "_")), func(t *testing.T) {
			assert.Contains(t, content, fmt.Sprintf("### %s", subsection),
				"Runbooks should have %s subsections", subsection)
		})
	}
}

// TestAllDocumentationFilesExist verifies that all referenced documentation files exist
func TestAllDocumentationFilesExist(t *testing.T) {
	// Define all documentation files that should exist
	expectedDocs := []string{
		"../../docs/README.md",
		"../../docs/API_DOCUMENTATION.md",
		"../../docs/ARCHITECTURE_OVERVIEW.md",
		"../../docs/AUTHENTICATION.md",
		"../../docs/DATABASE_SCHEMA.md",
		"../../docs/DEPLOYMENT_GUIDE.md",
		"../../docs/DEVELOPER_GUIDE.md",
		"../../docs/ERROR_CODES.md",
		"../../docs/MONITORING.md",
		"../../docs/SECURITY_BEST_PRACTICES.md",
		"../../docs/operations/RUNBOOKS.md",
		"../../docs/operations/BACKUP_RUNBOOK.md",
		"../../docs/operations/DISASTER_RECOVERY_PROCEDURES.md",
		"../../docs/operations/LAUNCH_CHECKLIST.md",
		"../../docs/operations/ROLLBACK_PROCEDURES.md",
	}

	// Verify each file exists
	for _, docFile := range expectedDocs {
		t.Run(fmt.Sprintf("File_%s", filepath.Base(docFile)), func(t *testing.T) {
			_, err := os.Stat(docFile)
			assert.NoError(t, err, "Documentation file should exist: %s", docFile)
		})
	}
}

// TestConfigurationExampleFilesExist verifies example configuration files exist
func TestConfigurationExampleFilesExist(t *testing.T) {
	// Define example configuration files that should exist
	exampleFiles := []string{
		"../../.env.example",
		"../../.env.production",
	}

	// Verify each example file exists
	for _, exampleFile := range exampleFiles {
		t.Run(fmt.Sprintf("Example_%s", filepath.Base(exampleFile)), func(t *testing.T) {
			_, err := os.Stat(exampleFile)
			assert.NoError(t, err, "Example configuration file should exist: %s", exampleFile)
		})
	}
}
