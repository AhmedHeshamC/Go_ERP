package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	ServerPort  int    `env:"SERVER_PORT" envDefault:"8080"`
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	DebugMode   bool   `env:"DEBUG_MODE" envDefault:"false"`

	// Database configuration
	DatabaseURL     string `env:"DATABASE_URL" envDefault:"postgres://localhost/erp?sslmode=disable"`
	MaxConnections  int    `env:"MAX_CONNECTIONS" envDefault:"20"`
	MinConnections  int    `env:"MIN_CONNECTIONS" envDefault:"5"`
	ConnMaxLifetime int    `env:"CONN_MAX_LIFETIME" envDefault:"3600"` // seconds
	ConnMaxIdleTime int    `env:"CONN_MAX_IDLE_TIME" envDefault:"1800"` // seconds

	// SSL Configuration
	DatabaseSSLMode string `env:"DATABASE_SSL_MODE" envDefault:"require"`
	DatabaseSSLCert string `env:"DATABASE_SSL_CERT"`
	DatabaseSSLKey  string `env:"DATABASE_SSL_KEY"`
	DatabaseSSLCA   string `env:"DATABASE_SSL_CA"`
	DatabaseSSLHost string `env:"DATABASE_SSL_HOST"`

	// Redis configuration
	RedisURL      string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`
	RedisPoolSize int    `env:"REDIS_POOL_SIZE" envDefault:"10"`

	// JWT configuration
	JWTSecret     string        `env:"JWT_SECRET" envDefault:"your-super-secret-jwt-key"`
	JWTExpiry     time.Duration `env:"JWT_EXPIRY" envDefault:"24h"`
	RefreshExpiry time.Duration `env:"REFRESH_EXPIRY" envDefault:"168h"` // 7 days

	// CORS configuration
	CORSOrigins []string `env:"CORS_ORIGINS" envDefault:"http://localhost:3000,http://localhost:8080"`
	CORSMethods []string `env:"CORS_METHODS" envDefault:"GET,POST,PUT,DELETE,OPTIONS"`
	CORSHeaders []string `env:"CORS_HEADERS" envDefault:"Origin,Content-Type,Accept,Authorization"`

	// Production CORS configuration
	ProductionCORSOrigins []string `env:"PRODUCTION_CORS_ORIGINS"`
	CORSEnvironmentWhitelist  bool     `env:"CORS_ENVIRONMENT_WHITELIST" envDefault:"true"`
	CORSMaxAge               int      `env:"CORS_MAX_AGE" envDefault:"86400"`
	CORSCredentialsEnabled   bool     `env:"CORS_CREDENTIALS_ENABLED" envDefault:"true"`

	// Rate limiting
	RateLimitEnabled bool          `env:"RATE_LIMIT_ENABLED" envDefault:"true"`
	RateLimitRPS     int           `env:"RATE_LIMIT_RPS" envDefault:"100"`
	RateLimitBurst   int           `env:"RATE_LIMIT_BURST" envDefault:"200"`
	RateLimitTTL     time.Duration `env:"RATE_LIMIT_TTL" envDefault:"1h"`

	// File storage
	StorageType string `env:"STORAGE_TYPE" envDefault:"local"` // local, s3
	UploadPath  string `env:"UPLOAD_PATH" envDefault:"./uploads"`

	// S3 configuration (if using S3)
	S3Bucket    string `env:"S3_BUCKET"`
	S3Region    string `env:"S3_REGION"`
	S3AccessKey string `env:"S3_ACCESS_KEY"`
	S3SecretKey string `env:"S3_SECRET_KEY"`
	S3Endpoint  string `env:"S3_ENDPOINT"`

	// Email configuration
	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUsername string `env:"SMTP_USERNAME"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPEmail    string `env:"SMTP_EMAIL"`
	EmailFrom    string `env:"EMAIL_FROM"`

	// Monitoring and metrics
	MetricsEnabled bool   `env:"METRICS_ENABLED" envDefault:"true"`
	MetricsPath    string `env:"METRICS_PATH" envDefault:"/metrics"`
	TracingEnabled bool   `env:"TRACING_ENABLED" envDefault:"false"`
	TracingURL     string `env:"TRACING_URL"`

	// Security
	BcryptCost       int  `env:"BCRYPT_COST" envDefault:"12"`
	MaxLoginAttempts int  `env:"MAX_LOGIN_ATTEMPTS" envDefault:"5"`
	LockoutDuration  time.Duration `env:"LOCKOUT_DURATION" envDefault:"15m"`

	// API configuration
	APIVersion     string `env:"API_VERSION" envDefault:"v1"`
	APIPrefix      string `env:"API_PREFIX" envDefault:"/api"`
	APIDocsEnabled bool   `env:"API_DOCS_ENABLED" envDefault:"true"`
	APIDocsPath    string `env:"API_DOCS_PATH" envDefault:"/docs"`

	// Cache configuration
	CacheEnabled     bool          `env:"CACHE_ENABLED" envDefault:"true"`
	CacheDefaultTTL  time.Duration `env:"CACHE_DEFAULT_TTL" envDefault:"5m"`
	CacheUserTTL     time.Duration `env:"CACHE_USER_TTL" envDefault:"10m"`
	CacheProductTTL  time.Duration `env:"CACHE_PRODUCT_TTL" envDefault:"15m"`
	CacheInventoryTTL time.Duration `env:"CACHE_INVENTORY_TTL" envDefault:"1m"`

	// Background jobs
	WorkerEnabled    bool `env:"WORKER_ENABLED" envDefault:"true"`
	WorkerCount      int  `env:"WORKER_COUNT" envDefault:"5"`
	JobRetryAttempts int  `env:"JOB_RETRY_ATTEMPTS" envDefault:"3"`

	// Structured configuration objects (computed from env vars)
	// These are not loaded from env directly but computed in Load()
	RateLimit *RateLimitConfig `json:"-"`
	Redis     *RedisConfig     `json:"-"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file if it exists
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("error parsing environment variables: %w", err)
	}

	// Populate structured configuration objects
	cfg.populateStructuredConfigs()

	// Validate required configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Parse comma-separated values from environment variables
	// These are already parsed by env/v10 from comma-separated env vars

	return cfg, nil
}

// validate validates the configuration
func (c *Config) validate() error {
	if c.JWTSecret == "" || c.JWTSecret == "your-super-secret-jwt-key" {
		return fmt.Errorf("JWT_SECRET must be set to a secure value")
	}

	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535")
	}

	if c.StorageType == "s3" {
		if c.S3Bucket == "" || c.S3Region == "" || c.S3AccessKey == "" || c.S3SecretKey == "" {
			return fmt.Errorf("S3 configuration is incomplete when STORAGE_TYPE is 's3'")
		}
	}

	return nil
}

// IsDevelopment returns true if the environment is development
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// populateStructuredConfigs populates structured configuration objects from environment variables
func (c *Config) populateStructuredConfigs() {
	// Populate RateLimit config
	c.RateLimit = &RateLimitConfig{
		Enabled:           c.RateLimitEnabled,
		RequestsPerSecond: float64(c.RateLimitRPS),
		BurstSize:         c.RateLimitBurst,
		StorageType:       "redis", // Default to Redis for production
		LogRequests:       true,
		LogRejections:     true,
		TTL:               c.RateLimitTTL,
	}

	// Populate Redis config
	c.Redis = &RedisConfig{
		URL:       c.RedisURL,
		Password:  c.RedisPassword,
		DB:        c.RedisDB,
		PoolSize:  c.RedisPoolSize,
	}
}

// GetRedisAddress returns the Redis address for rate limiting
func (c *Config) GetRedisAddress() string {
	if c.Redis != nil && c.Redis.URL != "" {
		return c.Redis.URL
	}
	return "localhost:6379" // Default fallback
}

// GetDatabaseConfig returns database connection configuration
func (c *Config) GetDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		URL:             c.DatabaseURL,
		MaxConnections:  c.MaxConnections,
		MinConnections:  c.MinConnections,
		ConnMaxLifetime: time.Duration(c.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(c.ConnMaxIdleTime) * time.Second,
		SSLMode:         c.DatabaseSSLMode,
		SSLCert:         c.DatabaseSSLCert,
		SSLKey:          c.DatabaseSSLKey,
		SSLCA:           c.DatabaseSSLCA,
		SSLHost:         c.DatabaseSSLHost,
	}
}

// GetRedisConfig returns Redis connection configuration
func (c *Config) GetRedisConfig() RedisConfig {
	return RedisConfig{
		URL:      c.RedisURL,
		Password: c.RedisPassword,
		DB:       c.RedisDB,
		PoolSize: c.RedisPoolSize,
	}
}

// GetJWTConfig returns JWT configuration
func (c *Config) GetJWTConfig() JWTConfig {
	return JWTConfig{
		Secret:     c.JWTSecret,
		Expiry:     c.JWTExpiry,
		RefreshExpiry: c.RefreshExpiry,
	}
}

// GetCORSConfig returns CORS configuration based on environment
func (c *Config) GetCORSConfig() CORSConfig {
	if c.IsProduction() {
		origins := c.ProductionCORSOrigins
		if len(origins) == 0 {
			origins = c.CORSOrigins
		}
		return CORSConfig{
			Origins:         origins,
			Methods:         c.CORSMethods,
			Headers:         c.CORSHeaders,
			MaxAge:          c.CORSMaxAge,
			Credentials:     c.CORSCredentialsEnabled,
			EnvironmentWhitelist: c.CORSEnvironmentWhitelist,
			IsProduction:    true,
		}
	}

	return CORSConfig{
		Origins:         c.CORSOrigins,
		Methods:         c.CORSMethods,
		Headers:         c.CORSHeaders,
		MaxAge:          c.CORSMaxAge,
		Credentials:     c.CORSCredentialsEnabled,
		EnvironmentWhitelist: c.CORSEnvironmentWhitelist,
		IsProduction:    false,
	}
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string
	MaxConnections  int
	MinConnections  int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	SSLMode         string
	SSLCert         string
	SSLKey          string
	SSLCA           string
	SSLHost         string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
	PoolSize int
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret        string
	Expiry        time.Duration
	RefreshExpiry time.Duration
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Origins              []string
	Methods              []string
	Headers              []string
	MaxAge               int
	Credentials          bool
	EnvironmentWhitelist bool
	IsProduction         bool
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool    `json:"enabled"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	BurstSize         int     `json:"burst_size"`
	StorageType       string  `json:"storage_type"`
	LogRequests       bool    `json:"log_requests"`
	LogRejections     bool    `json:"log_rejections"`
	TTL               time.Duration `json:"ttl"`
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetEnvInt gets an integer environment variable with a default value
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool gets a boolean environment variable with a default value
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetEnvDuration gets a duration environment variable with a default value
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}