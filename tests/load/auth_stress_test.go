package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// AuthStressTestSuite contains authentication and authorization stress tests
type AuthStressTestSuite struct {
	baseURL     string
	config      AuthStressTestConfig
	results     *AuthStressTestResult
	framework   *LoadTestFramework
	userTokens  []string
	adminTokens []string
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// AuthStressTestConfig defines configuration for auth stress tests
type AuthStressTestConfig struct {
	Name               string
	BaseURL            string
	ConcurrentUsers    int
	RequestsPerUser    int
	TestDuration       time.Duration
	TimeoutPerRequest  time.Duration
	TargetRPS          int
	MaxErrorRate       float64
	MaxResponseTime    time.Duration
	BruteForceAttempts int
	CredentialReuse    bool
	TokenReuse         bool
	RoleBasedTests     bool
	PermissionTests    bool
}

// AuthStressTestResult contains results from auth stress tests
type AuthStressTestResult struct {
	TestName             string
	StartTime            time.Time
	EndTime              time.Time
	Duration             time.Duration
	TotalRequests        int64
	SuccessfulRequests   int64
	FailedRequests       int64
	RequestsPerSecond    float64
	AverageResponseTime  time.Duration
	P95ResponseTime      time.Duration
	ErrorRate            float64
	AuthMetrics          *AuthMetrics
	SecurityEvents       []SecurityEvent
	PerformanceBreakdown map[string]*TestBreakdown
}

// AuthMetrics tracks authentication-specific metrics
type AuthMetrics struct {
	LoginAttempts           int64
	SuccessfulLogins        int64
	FailedLogins            int64
	TokenRefreshes          int64
	RegistrationAttempts    int64
	SuccessfulRegistrations int64
	PasswordResets          int64
	RoleChanges             int64
	PermissionChecks        int64
	AuthorizationFailures   int64
	BruteForceAttempts      int64
	SessionCreations        int64
	SessionInvalidations    int64
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	EventType   string
	Timestamp   time.Time
	UserID      string
	IPAddress   string
	UserAgent   string
	Description string
	Severity    string // "low", "medium", "high", "critical"
}

// TestBreakdown breaks down test results by operation type
type TestBreakdown struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	AverageLatency     time.Duration
	P95Latency         time.Duration
	ErrorRate          float64
}

// UserCredentials represents user login credentials
type UserCredentials struct {
	Email    string
	Password string
	UserID   string
	Token    string
	Role     string
}

// NewAuthStressTestSuite creates a new auth stress test suite
func NewAuthStressTestSuite(config AuthStressTestConfig) *AuthStressTestSuite {
	ctx, cancel := context.WithCancel(context.Background())

	return &AuthStressTestSuite{
		baseURL: config.BaseURL,
		config:  config,
		results: &AuthStressTestResult{
			TestName:             config.Name,
			AuthMetrics:          &AuthMetrics{},
			SecurityEvents:       make([]SecurityEvent, 0),
			PerformanceBreakdown: make(map[string]*TestBreakdown),
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// RunAuthStressTest executes the authentication stress test
func (a *AuthStressTestSuite) RunAuthStressTest() (*AuthStressTestResult, error) {
	log.Printf("Starting authentication stress test: %s", a.config.Name)
	log.Printf("Configuration: Users=%d, Requests/User=%d, Duration=%v",
		a.config.ConcurrentUsers, a.config.RequestsPerUser, a.config.TestDuration)

	a.results.StartTime = time.Now()

	// Setup test data
	if err := a.setupTestData(); err != nil {
		return nil, fmt.Errorf("failed to setup test data: %w", err)
	}

	// Start security event monitoring
	a.startSecurityMonitoring()

	// Run different stress test scenarios
	testScenarios := []struct {
		name     string
		testFunc func() error
		weight   int // Relative weight in mixed workload
	}{
		{"LoginStress", a.runLoginStressTest, 30},
		{"TokenRefresh", a.runTokenRefreshTest, 20},
		{"RegistrationStress", a.runRegistrationStressTest, 15},
		{"BruteForceAttack", a.runBruteForceAttackTest, 10},
		{"RoleBasedAccess", a.runRoleBasedAccessTest, 15},
		{"PermissionChecks", a.runPermissionCheckTest, 10},
	}

	// Run individual scenarios
	for _, scenario := range testScenarios {
		if err := scenario.testFunc(); err != nil {
			log.Printf("Error in scenario %s: %v", scenario.name, err)
		}
	}

	// Run mixed workload test
	if err := a.runMixedWorkloadTest(testScenarios); err != nil {
		log.Printf("Error in mixed workload test: %v", err)
	}

	a.results.EndTime = time.Now()
	a.results.Duration = a.results.EndTime.Sub(a.results.StartTime)

	// Calculate final results
	a.calculateResults()

	log.Printf("Authentication stress test completed: %s", a.config.Name)
	log.Printf("Results: RPS=%.2f, Success Rate=%.2f%%, Login Success Rate=%.2f%%",
		a.results.RequestsPerSecond,
		(1-a.results.ErrorRate)*100,
		float64(a.results.AuthMetrics.SuccessfulLogins)/float64(a.results.AuthMetrics.LoginAttempts)*100)

	return a.results, nil
}

// setupTestData creates test users and tokens
func (a *AuthStressTestSuite) setupTestData() error {
	log.Printf("Setting up test data...")

	// Create regular users
	for i := 0; i < a.config.ConcurrentUsers; i++ {
		email := fmt.Sprintf("stressuser%d@example.com", i)
		password := fmt.Sprintf("password%d", i)

		token, userID, err := a.createTestUser(email, password, "customer")
		if err != nil {
			log.Printf("Warning: Failed to create user %s: %v", email, err)
			continue
		}

		a.userTokens = append(a.userTokens, token)
	}

	// Create admin users
	for i := 0; i < 5; i++ {
		email := fmt.Sprintf("stressadmin%d@example.com", i)
		password := fmt.Sprintf("adminpass%d", i)

		token, userID, err := a.createTestUser(email, password, "admin")
		if err != nil {
			log.Printf("Warning: Failed to create admin %s: %v", email, err)
			continue
		}

		a.adminTokens = append(a.adminTokens, token)
	}

	log.Printf("Created %d regular users and %d admin users", len(a.userTokens), len(a.adminTokens))
	return nil
}

// createTestUser creates a test user and returns auth token
func (a *AuthStressTestSuite) createTestUser(email, password, role string) (string, string, error) {
	// Register user
	registerReq := map[string]interface{}{
		"first_name": "Stress",
		"last_name":  "User",
		"email":      email,
		"password":   password,
		"role":       role,
	}

	body, _ := json.Marshal(registerReq)
	resp, err := http.Post(a.baseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	// Login and get token
	loginReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ = json.Marshal(loginReq)
	resp, err = http.Post(a.baseURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", "", err
	}

	token, ok := loginResp["token"].(string)
	if !ok {
		return "", "", fmt.Errorf("no token in login response")
	}

	userID, ok := loginResp["user_id"].(string)
	if !ok {
		userID = uuid.New().String()
	}

	return token, userID, nil
}

// runLoginStressTest tests login endpoint under stress
func (a *AuthStressTestSuite) runLoginStressTest() error {
	log.Printf("Running login stress test...")

	config := LoadTestConfig{
		Name:               "Login Stress Test",
		BaseURL:            a.baseURL,
		ConcurrentUsers:    a.config.ConcurrentUsers,
		RequestsPerUser:    a.config.RequestsPerUser,
		TestDuration:       a.config.TestDuration,
		TimeoutPerRequest:  a.config.TimeoutPerRequest,
		ThinkTime:          100 * time.Millisecond,
		TargetRPS:          a.config.TargetRPS,
		MaxErrorRate:       a.config.MaxErrorRate,
		MaxResponseTime:    a.config.MaxResponseTime,
		ExpectedStatusCode: 200,
	}

	requestFunc := func(user int, iteration int) (*http.Request, error) {
		email := fmt.Sprintf("stressuser%d@example.com", user%len(a.userTokens))
		password := fmt.Sprintf("password%d", user%len(a.userTokens))

		loginReq := map[string]interface{}{
			"email":    email,
			"password": password,
		}

		body, _ := json.Marshal(loginReq)
		req, err := http.NewRequest("POST", a.baseURL+"/api/v1/auth/login", bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		atomic.AddInt64(&a.results.AuthMetrics.LoginAttempts, 1)

		return req, nil
	}

	framework := NewLoadTestFramework(&config)
	result, err := framework.RunLoadTest(requestFunc)
	if err != nil {
		return err
	}

	// Update metrics
	atomic.AddInt64(&a.results.AuthMetrics.SuccessfulLogins, result.SuccessfulRequests)
	atomic.AddInt64(&a.results.AuthMetrics.FailedLogins, result.FailedRequests)

	// Store breakdown
	a.results.PerformanceBreakdown["login"] = &TestBreakdown{
		TotalRequests:      result.TotalRequests,
		SuccessfulRequests: result.SuccessfulRequests,
		FailedRequests:     result.FailedRequests,
		AverageLatency:     result.AverageResponseTime,
		P95Latency:         result.P95ResponseTime,
		ErrorRate:          result.ErrorRate,
	}

	return nil
}

// runTokenRefreshTest tests token refresh endpoint under stress
func (a *AuthStressTestSuite) runTokenRefreshTest() error {
	log.Printf("Running token refresh stress test...")

	config := LoadTestConfig{
		Name:               "Token Refresh Stress Test",
		BaseURL:            a.baseURL,
		ConcurrentUsers:    a.config.ConcurrentUsers,
		RequestsPerUser:    a.config.RequestsPerUser * 2, // More refreshes
		TestDuration:       a.config.TestDuration,
		TimeoutPerRequest:  a.config.TimeoutPerRequest,
		ThinkTime:          50 * time.Millisecond,
		TargetRPS:          a.config.TargetRPS * 2,
		MaxErrorRate:       a.config.MaxErrorRate,
		MaxResponseTime:    a.config.MaxResponseTime,
		ExpectedStatusCode: 200,
	}

	requestFunc := func(user int, iteration int) (*http.Request, error) {
		token := a.userTokens[user%len(a.userTokens)]

		refreshReq := map[string]interface{}{
			"token": token,
		}

		body, _ := json.Marshal(refreshReq)
		req, err := http.NewRequest("POST", a.baseURL+"/api/v1/auth/refresh", bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		atomic.AddInt64(&a.results.AuthMetrics.TokenRefreshes, 1)

		return req, nil
	}

	framework := NewLoadTestFramework(&config)
	result, err := framework.RunLoadTest(requestFunc)
	if err != nil {
		return err
	}

	// Store breakdown
	a.results.PerformanceBreakdown["token_refresh"] = &TestBreakdown{
		TotalRequests:      result.TotalRequests,
		SuccessfulRequests: result.SuccessfulRequests,
		FailedRequests:     result.FailedRequests,
		AverageLatency:     result.AverageResponseTime,
		P95Latency:         result.P95ResponseTime,
		ErrorRate:          result.ErrorRate,
	}

	return nil
}

// runRegistrationStressTest tests user registration under stress
func (a *AuthStressTestSuite) runRegistrationStressTest() error {
	log.Printf("Running registration stress test...")

	config := LoadTestConfig{
		Name:               "Registration Stress Test",
		BaseURL:            a.baseURL,
		ConcurrentUsers:    a.config.ConcurrentUsers / 2, // Fewer concurrent registrations
		RequestsPerUser:    a.config.RequestsPerUser / 2,
		TestDuration:       a.config.TestDuration,
		TimeoutPerRequest:  a.config.TimeoutPerRequest,
		ThinkTime:          200 * time.Millisecond,
		TargetRPS:          a.config.TargetRPS / 2,
		MaxErrorRate:       a.config.MaxErrorRate,
		MaxResponseTime:    a.config.MaxResponseTime,
		ExpectedStatusCode: 201,
	}

	requestFunc := func(user int, iteration int) (*http.Request, error) {
		registerReq := map[string]interface{}{
			"first_name": "Stress",
			"last_name":  "Reg",
			"email":      fmt.Sprintf("stressreg%d_%d@example.com", user, iteration),
			"password":   "password123",
			"role":       "customer",
		}

		body, _ := json.Marshal(registerReq)
		req, err := http.NewRequest("POST", a.baseURL+"/api/v1/auth/register", bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		atomic.AddInt64(&a.results.AuthMetrics.RegistrationAttempts, 1)

		return req, nil
	}

	framework := NewLoadTestFramework(&config)
	result, err := framework.RunLoadTest(requestFunc)
	if err != nil {
		return err
	}

	// Update metrics
	atomic.AddInt64(&a.results.AuthMetrics.SuccessfulRegistrations, result.SuccessfulRequests)

	// Store breakdown
	a.results.PerformanceBreakdown["registration"] = &TestBreakdown{
		TotalRequests:      result.TotalRequests,
		SuccessfulRequests: result.SuccessfulRequests,
		FailedRequests:     result.FailedRequests,
		AverageLatency:     result.AverageResponseTime,
		P95Latency:         result.P95ResponseTime,
		ErrorRate:          result.ErrorRate,
	}

	return nil
}

// runBruteForceAttackTest simulates brute force attacks
func (a *AuthStressTestSuite) runBruteForceAttackTest() error {
	log.Printf("Running brute force attack simulation...")

	config := LoadTestConfig{
		Name:               "Brute Force Attack Test",
		BaseURL:            a.baseURL,
		ConcurrentUsers:    a.config.BruteForceAttempts,
		RequestsPerUser:    10,
		TestDuration:       30 * time.Second,
		TimeoutPerRequest:  5 * time.Second,
		ThinkTime:          10 * time.Millisecond, // Very fast to simulate attack
		TargetRPS:          500,
		MaxErrorRate:       0.95, // Expect high failure rate
		MaxResponseTime:    5 * time.Second,
		ExpectedStatusCode: 401, // Expect unauthorized
	}

	requestFunc := func(user int, iteration int) (*http.Request, error) {
		// Simulate brute force with random passwords
		email := fmt.Sprintf("stressuser%d@example.com", user%len(a.userTokens))
		password := fmt.Sprintf("wrongpass%d", rand.Intn(10000))

		loginReq := map[string]interface{}{
			"email":    email,
			"password": password,
		}

		body, _ := json.Marshal(loginReq)
		req, err := http.NewRequest("POST", a.baseURL+"/api/v1/auth/login", bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		atomic.AddInt64(&a.results.AuthMetrics.BruteForceAttempts, 1)

		// Record security event
		a.recordSecurityEvent("brute_force_attempt", "", "attacker_ip", "attack_tool",
			fmt.Sprintf("Brute force attempt on %s", email), "high")

		return req, nil
	}

	framework := NewLoadTestFramework(&config)
	result, err := framework.RunLoadTest(requestFunc)
	if err != nil {
		return err
	}

	// Store breakdown
	a.results.PerformanceBreakdown["brute_force"] = &TestBreakdown{
		TotalRequests:      result.TotalRequests,
		SuccessfulRequests: result.SuccessfulRequests,
		FailedRequests:     result.FailedRequests,
		AverageLatency:     result.AverageResponseTime,
		P95Latency:         result.P95ResponseTime,
		ErrorRate:          result.ErrorRate,
	}

	return nil
}

// runRoleBasedAccessTest tests role-based access control
func (a *AuthStressTestSuite) runRoleBasedAccessTest() error {
	if !a.config.RoleBasedTests {
		return nil
	}

	log.Printf("Running role-based access test...")

	config := LoadTestConfig{
		Name:               "Role-Based Access Test",
		BaseURL:            a.baseURL,
		ConcurrentUsers:    a.config.ConcurrentUsers,
		RequestsPerUser:    a.config.RequestsPerUser,
		TestDuration:       a.config.TestDuration,
		TimeoutPerRequest:  a.config.TimeoutPerRequest,
		ThinkTime:          100 * time.Millisecond,
		TargetRPS:          a.config.TargetRPS,
		MaxErrorRate:       0.10, // Higher error rate expected due to permissions
		MaxResponseTime:    a.config.MaxResponseTime,
		ExpectedStatusCode: 200, // Some should succeed
	}

	requestFunc := func(user int, iteration int) (*http.Request, error) {
		var token string
		var endpoint string

		// Mix of admin and user tokens
		if rand.Float32() < 0.3 && len(a.adminTokens) > 0 {
			// Use admin token (30% of requests)
			token = a.adminTokens[user%len(a.adminTokens)]
			endpoint = "/api/v1/admin/users"
		} else {
			// Use user token (70% of requests)
			token = a.userTokens[user%len(a.userTokens)]
			endpoint = "/api/v1/users/profile"
		}

		req, err := http.NewRequest("GET", a.baseURL+endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		atomic.AddInt64(&a.results.AuthMetrics.PermissionChecks, 1)

		return req, nil
	}

	framework := NewLoadTestFramework(&config)
	result, err := framework.RunLoadTest(requestFunc)
	if err != nil {
		return err
	}

	// Store breakdown
	a.results.PerformanceBreakdown["role_based_access"] = &TestBreakdown{
		TotalRequests:      result.TotalRequests,
		SuccessfulRequests: result.SuccessfulRequests,
		FailedRequests:     result.FailedRequests,
		AverageLatency:     result.AverageResponseTime,
		P95Latency:         result.P95ResponseTime,
		ErrorRate:          result.ErrorRate,
	}

	return nil
}

// runPermissionCheckTest tests permission validation
func (a *AuthStressTestSuite) runPermissionCheckTest() error {
	if !a.config.PermissionTests {
		return nil
	}

	log.Printf("Running permission check test...")

	config := LoadTestConfig{
		Name:               "Permission Check Test",
		BaseURL:            a.baseURL,
		ConcurrentUsers:    a.config.ConcurrentUsers / 2,
		RequestsPerUser:    a.config.RequestsPerUser,
		TestDuration:       a.config.TestDuration,
		TimeoutPerRequest:  a.config.TimeoutPerRequest,
		ThinkTime:          50 * time.Millisecond,
		TargetRPS:          a.config.TargetRPS,
		MaxErrorRate:       0.20, // Higher error rate expected
		MaxResponseTime:    a.config.MaxResponseTime,
		ExpectedStatusCode: 200,
	}

	requestFunc := func(user int, iteration int) (*http.Request, error) {
		// Try to access admin endpoints with user tokens
		token := a.userTokens[user%len(a.userTokens)]
		endpoints := []string{
			"/api/v1/admin/users",
			"/api/v1/admin/products",
			"/api/v1/admin/orders",
			"/api/v1/admin/analytics",
		}

		endpoint := endpoints[iteration%len(endpoints)]

		req, err := http.NewRequest("GET", a.baseURL+endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		atomic.AddInt64(&a.results.AuthMetrics.PermissionChecks, 1)

		return req, nil
	}

	framework := NewLoadTestFramework(&config)
	result, err := framework.RunLoadTest(requestFunc)
	if err != nil {
		return err
	}

	// Store breakdown
	a.results.PerformanceBreakdown["permission_check"] = &TestBreakdown{
		TotalRequests:      result.TotalRequests,
		SuccessfulRequests: result.SuccessfulRequests,
		FailedRequests:     result.FailedRequests,
		AverageLatency:     result.AverageResponseTime,
		P95Latency:         result.P95ResponseTime,
		ErrorRate:          result.ErrorRate,
	}

	return nil
}

// runMixedWorkloadTest runs a mixed workload of all auth operations
func (a *AuthStressTestSuite) runMixedWorkloadTest(scenarios []struct {
	name     string
	testFunc func() error
	weight   int
}) error {
	log.Printf("Running mixed workload test...")

	config := LoadTestConfig{
		Name:               "Mixed Auth Workload Test",
		BaseURL:            a.baseURL,
		ConcurrentUsers:    a.config.ConcurrentUsers,
		RequestsPerUser:    a.config.RequestsPerUser * 2,
		TestDuration:       a.config.TestDuration,
		TimeoutPerRequest:  a.config.TimeoutPerRequest,
		ThinkTime:          75 * time.Millisecond,
		TargetRPS:          a.config.TargetRPS,
		MaxErrorRate:       a.config.MaxErrorRate,
		MaxResponseTime:    a.config.MaxResponseTime,
		ExpectedStatusCode: 200,
	}

	// Calculate total weight for weighted selection
	totalWeight := 0
	for _, scenario := range scenarios {
		totalWeight += scenario.weight
	}

	requestFunc := func(user int, iteration int) (*http.Request, error) {
		// Select scenario based on weighted distribution
		rand := (user*1000 + iteration) % totalWeight
		cumulative := 0

		for _, scenario := range scenarios {
			cumulative += scenario.weight
			if rand < cumulative {
				switch scenario.name {
				case "LoginStress":
					return a.createLoginRequest(user, iteration)
				case "TokenRefresh":
					return a.createTokenRefreshRequest(user, iteration)
				case "RegistrationStress":
					return a.createRegistrationRequest(user, iteration)
				case "BruteForceAttack":
					return a.createBruteForceRequest(user, iteration)
				case "RoleBasedAccess":
					return a.createRoleBasedAccessRequest(user, iteration)
				case "PermissionChecks":
					return a.createPermissionCheckRequest(user, iteration)
				}
			}
		}

		// Fallback to login
		return a.createLoginRequest(user, iteration)
	}

	framework := NewLoadTestFramework(&config)
	result, err := framework.RunLoadTest(requestFunc)
	if err != nil {
		return err
	}

	// Store breakdown
	a.results.PerformanceBreakdown["mixed_workload"] = &TestBreakdown{
		TotalRequests:      result.TotalRequests,
		SuccessfulRequests: result.SuccessfulRequests,
		FailedRequests:     result.FailedRequests,
		AverageLatency:     result.AverageResponseTime,
		P95Latency:         result.P95ResponseTime,
		ErrorRate:          result.ErrorRate,
	}

	return nil
}

// Request creation helpers
func (a *AuthStressTestSuite) createLoginRequest(user, iteration int) (*http.Request, error) {
	email := fmt.Sprintf("stressuser%d@example.com", user%len(a.userTokens))
	password := fmt.Sprintf("password%d", user%len(a.userTokens))

	loginReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(loginReq)
	req, err := http.NewRequest("POST", a.baseURL+"/api/v1/auth/login", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *AuthStressTestSuite) createTokenRefreshRequest(user, iteration int) (*http.Request, error) {
	token := a.userTokens[user%len(a.userTokens)]

	refreshReq := map[string]interface{}{
		"token": token,
	}

	body, _ := json.Marshal(refreshReq)
	req, err := http.NewRequest("POST", a.baseURL+"/api/v1/auth/refresh", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *AuthStressTestSuite) createRegistrationRequest(user, iteration int) (*http.Request, error) {
	registerReq := map[string]interface{}{
		"first_name": "Mixed",
		"last_name":  "Workload",
		"email":      fmt.Sprintf("mixed%d_%d@example.com", user, iteration),
		"password":   "password123",
		"role":       "customer",
	}

	body, _ := json.Marshal(registerReq)
	req, err := http.NewRequest("POST", a.baseURL+"/api/v1/auth/register", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *AuthStressTestSuite) createBruteForceRequest(user, iteration int) (*http.Request, error) {
	email := fmt.Sprintf("stressuser%d@example.com", user%len(a.userTokens))
	password := fmt.Sprintf("wrongpass%d", rand.Intn(10000))

	loginReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(loginReq)
	req, err := http.NewRequest("POST", a.baseURL+"/api/v1/auth/login", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *AuthStressTestSuite) createRoleBasedAccessRequest(user, iteration int) (*http.Request, error) {
	var token string
	var endpoint string

	if rand.Float32() < 0.3 && len(a.adminTokens) > 0 {
		token = a.adminTokens[user%len(a.adminTokens)]
		endpoint = "/api/v1/admin/users"
	} else {
		token = a.userTokens[user%len(a.userTokens)]
		endpoint = "/api/v1/users/profile"
	}

	req, err := http.NewRequest("GET", a.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

func (a *AuthStressTestSuite) createPermissionCheckRequest(user, iteration int) (*http.Request, error) {
	token := a.userTokens[user%len(a.userTokens)]
	endpoints := []string{
		"/api/v1/admin/users",
		"/api/v1/admin/products",
		"/api/v1/admin/orders",
	}

	endpoint := endpoints[iteration%len(endpoints)]

	req, err := http.NewRequest("GET", a.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

// startSecurityMonitoring starts monitoring security events
func (a *AuthStressTestSuite) startSecurityMonitoring() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-a.ctx.Done():
				return
			case <-ticker.C:
				// Check for suspicious patterns
				a.checkSuspiciousPatterns()
			}
		}
	}()
}

// checkSuspiciousPatterns checks for suspicious authentication patterns
func (a *AuthStressTestSuite) checkSuspiciousPatterns() {
	// Check for high failure rates
	if a.results.AuthMetrics.FailedLogins > a.results.AuthMetrics.SuccessfulLogins*2 {
		a.recordSecurityEvent("high_failure_rate", "", "multiple", "system",
			"High authentication failure rate detected", "medium")
	}

	// Check for brute force patterns
	if a.results.AuthMetrics.BruteForceAttempts > 100 {
		a.recordSecurityEvent("brute_force_detected", "", "multiple", "automated",
			"Brute force attack pattern detected", "high")
	}
}

// recordSecurityEvent records a security event
func (a *AuthStressTestSuite) recordSecurityEvent(eventType, userID, ipAddress, userAgent, description, severity string) {
	event := SecurityEvent{
		EventType:   eventType,
		Timestamp:   time.Now(),
		UserID:      userID,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Description: description,
		Severity:    severity,
	}

	a.results.SecurityEvents = append(a.results.SecurityEvents, event)

	// Limit security events storage
	if len(a.results.SecurityEvents) > 1000 {
		a.results.SecurityEvents = a.results.SecurityEvents[500:]
	}
}

// calculateResults calculates final test statistics
func (a *AuthStressTestSuite) calculateResults() {
	if a.results.TotalRequests == 0 {
		return
	}

	// Calculate total requests across all breakdowns
	totalRequests := int64(0)
	successfulRequests := int64(0)

	for _, breakdown := range a.results.PerformanceBreakdown {
		totalRequests += breakdown.TotalRequests
		successfulRequests += breakdown.SuccessfulRequests
	}

	a.results.TotalRequests = totalRequests
	a.results.SuccessfulRequests = successfulRequests
	a.results.FailedRequests = totalRequests - successfulRequests

	a.results.RequestsPerSecond = float64(totalRequests) / a.results.Duration.Seconds()
	a.results.ErrorRate = float64(a.results.FailedRequests) / float64(totalRequests)

	// Calculate average latency (weighted)
	if totalRequests > 0 {
		totalLatency := time.Duration(0)
		for _, breakdown := range a.results.PerformanceBreakdown {
			totalLatency += breakdown.AverageLatency * time.Duration(breakdown.TotalRequests)
		}
		a.results.AverageResponseTime = totalLatency / time.Duration(totalRequests)
	}

	// Calculate P95 (approximation)
	a.results.P95ResponseTime = a.results.AverageResponseTime * 2
}

// ValidateResults validates the auth stress test results
func (a *AuthStressTestSuite) ValidateResults() error {
	if a.results == nil {
		return fmt.Errorf("no test results to validate")
	}

	// Validate overall error rate
	if a.results.ErrorRate > a.config.MaxErrorRate {
		return fmt.Errorf("overall error rate %.2f%% exceeds maximum %.2f%%",
			a.results.ErrorRate*100, a.config.MaxErrorRate*100)
	}

	// Validate RPS
	if a.config.TargetRPS > 0 && a.results.RequestsPerSecond < float64(a.config.TargetRPS)*0.8 {
		return fmt.Errorf("RPS %.2f is below target %d (80%% threshold)",
			a.results.RequestsPerSecond, a.config.TargetRPS)
	}

	// Validate response time
	if a.config.MaxResponseTime > 0 && a.results.P95ResponseTime > a.config.MaxResponseTime {
		return fmt.Errorf("95th percentile response time %v exceeds maximum %v",
			a.results.P95ResponseTime, a.config.MaxResponseTime)
	}

	// Validate login success rate (should be high)
	if a.results.AuthMetrics.LoginAttempts > 0 {
		loginSuccessRate := float64(a.results.AuthMetrics.SuccessfulLogins) / float64(a.results.AuthMetrics.LoginAttempts)
		if loginSuccessRate < 0.90 { // 90% minimum login success rate
			return fmt.Errorf("login success rate %.2f%% is below 90%% threshold", loginSuccessRate*100)
		}
	}

	return nil
}

// Close closes the auth stress test suite
func (a *AuthStressTestSuite) Close() {
	a.cancel()
}

// TestAuthStressTestSuite runs the auth stress test suite
func TestAuthStressTestSuite(t *testing.T) {
	baseURL := "http://localhost:8080"

	config := AuthStressTestConfig{
		Name:               "Comprehensive Auth Stress Test",
		BaseURL:            baseURL,
		ConcurrentUsers:    50,
		RequestsPerUser:    20,
		TestDuration:       60 * time.Second,
		TimeoutPerRequest:  10 * time.Second,
		TargetRPS:          1000,
		MaxErrorRate:       0.05, // 5%
		MaxResponseTime:    500 * time.Millisecond,
		BruteForceAttempts: 20,
		CredentialReuse:    true,
		TokenReuse:         true,
		RoleBasedTests:     true,
		PermissionTests:    true,
	}

	suite := NewAuthStressTestSuite(config)
	defer suite.Close()

	result, err := suite.RunAuthStressTest()
	require.NoError(t, err)

	err = suite.ValidateResults()
	require.NoError(t, err, "Auth stress test validation failed: %v", err)

	// Print detailed results
	t.Logf("=== Auth Stress Test Results ===")
	t.Logf("Duration: %v", result.Duration)
	t.Logf("Total Requests: %d", result.TotalRequests)
	t.Logf("Successful Requests: %d", result.SuccessfulRequests)
	t.Logf("Failed Requests: %d", result.FailedRequests)
	t.Logf("Requests Per Second: %.2f", result.RequestsPerSecond)
	t.Logf("Error Rate: %.2f%%", result.ErrorRate*100)
	t.Logf("Average Response Time: %v", result.AverageResponseTime)
	t.Logf("95th Percentile Response Time: %v", result.P95ResponseTime)

	t.Logf("\n=== Authentication Metrics ===")
	t.Logf("Login Attempts: %d", result.AuthMetrics.LoginAttempts)
	t.Logf("Successful Logins: %d", result.AuthMetrics.SuccessfulLogins)
	t.Logf("Failed Logins: %d", result.AuthMetrics.FailedLogins)
	t.Logf("Token Refreshes: %d", result.AuthMetrics.TokenRefreshes)
	t.Logf("Registration Attempts: %d", result.AuthMetrics.RegistrationAttempts)
	t.Logf("Successful Registrations: %d", result.AuthMetrics.SuccessfulRegistrations)
	t.Logf("Brute Force Attempts: %d", result.AuthMetrics.BruteForceAttempts)
	t.Logf("Permission Checks: %d", result.AuthMetrics.PermissionChecks)

	t.Logf("\n=== Performance Breakdown ===")
	for operation, breakdown := range result.PerformanceBreakdown {
		t.Logf("%s:", operation)
		t.Logf("  Requests: %d (Success: %d, Failed: %d)",
			breakdown.TotalRequests, breakdown.SuccessfulRequests, breakdown.FailedRequests)
		t.Logf("  Error Rate: %.2f%%", breakdown.ErrorRate*100)
		t.Logf("  Average Latency: %v", breakdown.AverageLatency)
		t.Logf("  P95 Latency: %v", breakdown.P95Latency)
	}

	t.Logf("\n=== Security Events ===")
	t.Logf("Total Security Events: %d", len(result.SecurityEvents))
	severityCount := make(map[string]int)
	for _, event := range result.SecurityEvents {
		severityCount[event.Severity]++
	}
	for severity, count := range severityCount {
		t.Logf("%s: %d events", severity, count)
	}
}
