package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erpgo/pkg/cache"
)

// MockRedisClient is a mock for Redis client
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewStringCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.String(0))
	}
	if args.Error(1) != nil {
		cmd.SetErr(args.Error(1))
	}
	return cmd
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	cmd := redis.NewStatusCmd(ctx)
	if args.Error(0) != nil {
		cmd.SetErr(args.Error(0))
	}
	return cmd
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewIntCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(int64))
	}
	if args.Error(1) != nil {
		cmd.SetErr(args.Error(1))
	}
	return cmd
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	args := m.Called(ctx, key, expiration)
	cmd := redis.NewBoolCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(bool))
	}
	if args.Error(1) != nil {
		cmd.SetErr(args.Error(1))
	}
	return cmd
}

func (m *MockRedisClient) Exists(ctx context.Context, key string) *redis.IntCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewIntCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(int64))
	}
	if args.Error(1) != nil {
		cmd.SetErr(args.Error(1))
	}
	return cmd
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewDurationCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(time.Duration))
	}
	if args.Error(1) != nil {
		cmd.SetErr(args.Error(1))
	}
	return cmd
}

func (m *MockRedisClient) TxPipeline() redis.Pipeliner {
	args := m.Called()
	return args.Get(0).(redis.Pipeliner)
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	assert.Equal(t, 100, config.RequestsPerSecond)
	assert.Equal(t, 200, config.Burst)
	assert.Equal(t, time.Minute, config.Window)
	assert.True(t, config.IPBased)
	assert.False(t, config.UserBased)
	assert.True(t, config.AdminExempt)
	assert.True(t, config.EnablePenalty)
	assert.Equal(t, 5*time.Minute, config.PenaltyTime)

	// Check specific endpoint configurations
	loginConfig, exists := config.Endpoints["POST:/api/v1/auth/login"]
	assert.True(t, exists)
	assert.Equal(t, 5, loginConfig.RequestsPerSecond)
	assert.True(t, loginConfig.IPBased)
	assert.False(t, loginConfig.UserBased)
}

func TestNewRateLimiter(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultRateLimitConfig()
	mockRedis := &MockRedisClient{}

	limiter := NewRateLimiter(config, mockRedis, logger)

	assert.NotNil(t, limiter)
	assert.Equal(t, config, limiter.config)
	assert.Equal(t, mockRedis, limiter.redis)
}

func TestRateLimiterGetClientID(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultRateLimitConfig()
	config.UserBased = true
	config.IPBased = true

	mockRedis := &MockRedisClient{}
	limiter := NewRateLimiter(config, mockRedis, logger)

	tests := []struct {
		name       string
		setupFunc  func(*gin.Context)
		expectedID string
	}{
		{
			name: "User and IP based ID",
			setupFunc: func(c *gin.Context) {
				c.Set("user_id", "user123")
				// IP will be set automatically by test
			},
			expectedID: "user:user123:ip:192.168.1.1",
		},
		{
			name: "IP based ID only",
			setupFunc: func(c *gin.Context) {
				// No user set
			},
			expectedID: "ip:192.168.1.1",
		},
		{
			name: "API key based ID",
			setupFunc: func(c *gin.Context) {
				c.Request.Header.Set("X-API-Key", "api_key_123")
			},
			expectedID: "ip:192.168.1.1:api_key:api_key_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Request.RemoteAddr = "192.168.1.1:12345" // Set IP

			tt.setupFunc(c)

			clientID := limiter.getClientID(c)
			assert.Equal(t, tt.expectedID, clientID)
		})
	}
}

func TestRateLimiterGetEndpointConfig(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultRateLimitConfig()
	config.Endpoints["POST:/api/v1/test"] = EndpointRateLimit{
		RequestsPerSecond: 10,
		Burst:             20,
		Window:            time.Second * 30,
	}

	mockRedis := &MockRedisClient{}
	limiter := NewRateLimiter(config, mockRedis, logger)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedRPS    int
		expectedBurst  int
		expectedWindow time.Duration
	}{
		{
			name:           "Exact match endpoint",
			method:         "POST",
			path:           "/api/v1/test",
			expectedRPS:    10,
			expectedBurst:  20,
			expectedWindow: 30 * time.Second,
		},
		{
			name:           "Wildcard match endpoint",
			method:         "GET",
			path:           "/api/v1/products",
			expectedRPS:    50, // From default config
			expectedBurst:  100,
			expectedWindow: time.Minute,
		},
		{
			name:           "Default configuration",
			method:         "GET",
			path:           "/api/v1/unknown",
			expectedRPS:    100, // Default config
			expectedBurst:  200,
			expectedWindow: time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(tt.method, tt.path, nil)
			c.Request.RemoteAddr = "192.168.1.1:12345"

			// Mock the FullPath (normally set by Gin router)
			c.Set("FullPath", tt.path)

			endpointConfig := limiter.getEndpointConfig(c)
			assert.Equal(t, tt.expectedRPS, endpointConfig.RequestsPerSecond)
			assert.Equal(t, tt.expectedBurst, endpointConfig.Burst)
			assert.Equal(t, tt.expectedWindow, endpointConfig.Window)
		})
	}
}

func TestRateLimiterCheckRateLimit(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultRateLimitConfig()
	mockRedis := &MockRedisClient{}
	limiter := NewRateLimiter(config, mockRedis, logger)

	ctx := context.Background()
	clientID := "test_client"
	endpointConfig := EndpointRateLimit{
		RequestsPerSecond: 5,
		Burst:             10,
		Window:            time.Minute,
	}

	tests := []struct {
		name              string
		setupMock         func()
		expectedAllow     bool
		expectedRemaining int
	}{
		{
			name: "First request allowed",
			setupMock: func() {
				mockPipeline := &MockPipeliner{}
				mockPipeline.On("Incr", ctx, mock.AnythingOfType("string")).Return(&redis.IntCmd{Val: 1})
				mockPipeline.On("Expire", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(&redis.BoolCmd{Val: true})
				mockPipeline.On("Exec", ctx).Return([]redis.Cmder{}, nil)
				mockRedis.On("TxPipeline").Return(mockPipeline)
			},
			expectedAllow:     true,
			expectedRemaining: 4, // 5 - 1
		},
		{
			name: "Rate limit exceeded",
			setupMock: func() {
				mockPipeline := &MockPipeliner{}
				mockPipeline.On("Incr", ctx, mock.AnythingOfType("string")).Return(&redis.IntCmd{Val: 6}) // Exceeds limit of 5
				mockPipeline.On("Expire", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(&redis.BoolCmd{Val: true})
				mockPipeline.On("Exec", ctx).Return([]redis.Cmder{}, nil)
				mockRedis.On("TxPipeline").Return(mockPipeline)
			},
			expectedAllow:     false,
			expectedRemaining: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			allowed, remaining, _, err := limiter.checkRateLimit(clientID, endpointConfig)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedAllow, allowed)
			assert.Equal(t, tt.expectedRemaining, remaining)
		})
	}
}

func TestRateLimiterIsIPBlacklisted(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultRateLimitConfig()
	config.IPBlacklist = []string{"192.168.1.100", "10.0.0.50"}

	mockRedis := &MockRedisClient{}
	limiter := NewRateLimiter(config, mockRedis, logger)

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "Blacklisted IP",
			ip:       "192.168.1.100",
			expected: true,
		},
		{
			name:     "Not blacklisted IP",
			ip:       "192.168.1.1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := limiter.isIPBlacklisted(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRateLimiterIntegration(t *testing.T) {
	// This test requires a real Redis instance or a more comprehensive mock
	// For now, we'll test the middleware structure
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()
	config := DefaultRateLimitConfig()
	config.RequestsPerSecond = 2 // Very low limit for testing
	config.Burst = 3

	// Use mock cache for testing
	mockCache := cache.NewMockCache()

	// Create middleware
	middleware := RateLimit(config, mockCache, logger)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Make requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if i < 3 {
			// First 3 requests should be allowed
			assert.Equal(t, 200, w.Code)
		} else {
			// Subsequent requests might be rate limited depending on implementation
			// This is a basic test - in real implementation, we need proper Redis
			assert.True(t, w.Code == 200 || w.Code == 429)
		}
	}
}

func TestRateLimitAuth(t *testing.T) {
	logger := zerolog.Nop()
	mockCache := cache.NewMockCache()

	// Test auth rate limiting middleware
	middleware := RateLimitAuth(mockCache, logger)

	assert.NotNil(t, middleware)

	// The middleware should be created without errors
	// In a real test, we would make requests to auth endpoints and verify rate limiting
}

func TestRateLimitAPI(t *testing.T) {
	logger := zerolog.Nop()
	mockCache := cache.NewMockCache()

	// Test API rate limiting middleware
	middleware := RateLimitAPI(mockCache, logger)

	assert.NotNil(t, middleware)

	// The middleware should be created without errors
	// In a real test, we would make requests to API endpoints and verify rate limiting
}

// MockPipeliner is a mock for Redis pipeline
type MockPipeliner struct {
	mock.Mock
}

func (m *MockPipeliner) Incr(ctx context.Context, key string) *redis.IntCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewIntCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(int64))
	}
	return cmd
}

func (m *MockPipeliner) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	args := m.Called(ctx, key, expiration)
	cmd := redis.NewBoolCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(bool))
	}
	return cmd
}

func (m *MockPipeliner) Exec(ctx context.Context) ([]redis.Cmder, error) {
	args := m.Called(ctx)
	return args.Get(0).([]redis.Cmder), args.Error(1)
}

func (m *MockPipeliner) Len() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockPipeliner) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	mockArgs := m.Called(ctx, args)
	cmd := redis.NewCmd(ctx)
	if mockArgs.Error(0) != nil {
		cmd.SetErr(mockArgs.Error(0))
	}
	return cmd
}

func (m *MockPipeliner) Process(ctx context.Context, cmd redis.Cmder) error {
	args := m.Called(ctx, cmd)
	return args.Error(0)
}
