package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/rs/zerolog"
)

// **Feature: production-readiness, Property 9: Login Rate Limiting**
// For any IP address, if more than 5 login attempts occur within 15 minutes,
// subsequent attempts must be rejected with rate limit error
// **Validates: Requirements 5.1**
func TestProperty_LoginRateLimiting(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: More than 5 attempts within 15 minutes must be rejected
	properties.Property("login attempts exceeding 5 per 15 minutes are rejected", prop.ForAll(
		func(ipAddress string, attemptCount int) bool {
			// Skip invalid test cases
			if ipAddress == "" || attemptCount < 0 {
				return true
			}

			// Create a test rate limiter with memory store
			logger := zerolog.Nop()
			config := &AuthLimiterConfig{
				MaxLoginAttempts:    5,
				LoginWindow:         15 * time.Minute,
				LockoutDuration:     15 * time.Minute,
				StorageType:         StorageMemory,
				EnableNotifications: false,
			}

			limiter, err := NewEnhancedRateLimiter(config, &logger)
			if err != nil {
				t.Logf("Failed to create limiter: %v", err)
				return false
			}

			ctx := context.Background()

			// Make login attempts
			var lastAllowed bool
			var lastErr error
			for i := 0; i < attemptCount; i++ {
				lastAllowed, lastErr = limiter.AllowLogin(ctx, ipAddress)
			}

			// If we made more than 5 attempts, the last one should be rejected
			if attemptCount > 5 {
				return !lastAllowed && lastErr != nil
			}

			// If we made 5 or fewer attempts, they should all be allowed
			if attemptCount <= 5 && attemptCount > 0 {
				return lastAllowed && lastErr == nil
			}

			// Zero attempts is a valid edge case
			return true
		},
		genIPAddress(),
		gen.IntRange(0, 20), // Test with 0-20 attempts
	))

	// Property: Attempts from different IPs should be independent
	properties.Property("rate limits are independent per IP", prop.ForAll(
		func(ip1 string, ip2 string) bool {
			// Skip if IPs are the same or empty
			if ip1 == "" || ip2 == "" || ip1 == ip2 {
				return true
			}

			logger := zerolog.Nop()
			config := &AuthLimiterConfig{
				MaxLoginAttempts:    5,
				LoginWindow:         15 * time.Minute,
				LockoutDuration:     15 * time.Minute,
				StorageType:         StorageMemory,
				EnableNotifications: false,
			}

			limiter, err := NewEnhancedRateLimiter(config, &logger)
			if err != nil {
				return false
			}

			ctx := context.Background()

			// Exhaust rate limit for ip1
			for i := 0; i < 6; i++ {
				limiter.AllowLogin(ctx, ip1)
			}

			// ip1 should be rate limited
			allowed1, _ := limiter.AllowLogin(ctx, ip1)

			// ip2 should still be allowed
			allowed2, _ := limiter.AllowLogin(ctx, ip2)

			return !allowed1 && allowed2
		},
		genIPAddress(),
		genIPAddress(),
	))

	// Property: Rate limit should reset after the window expires
	properties.Property("rate limit resets after window expires", prop.ForAll(
		func(ipAddress string) bool {
			if ipAddress == "" {
				return true
			}

			logger := zerolog.Nop()
			// Use a very short window for testing
			config := &AuthLimiterConfig{
				MaxLoginAttempts:    5,
				LoginWindow:         100 * time.Millisecond,
				LockoutDuration:     100 * time.Millisecond,
				StorageType:         StorageMemory,
				EnableNotifications: false,
			}

			limiter, err := NewEnhancedRateLimiter(config, &logger)
			if err != nil {
				return false
			}

			ctx := context.Background()

			// Exhaust rate limit
			for i := 0; i < 6; i++ {
				limiter.AllowLogin(ctx, ipAddress)
			}

			// Should be rate limited
			allowed1, _ := limiter.AllowLogin(ctx, ipAddress)

			// Wait for window to expire
			time.Sleep(150 * time.Millisecond)

			// Should be allowed again
			allowed2, _ := limiter.AllowLogin(ctx, ipAddress)

			return !allowed1 && allowed2
		},
		genIPAddress(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: production-readiness, Property 10: Account Lockout After Failed Logins**
// For any user account, if 5 consecutive login failures occur,
// the account must be locked for 15 minutes regardless of correct password
// **Validates: Requirements 5.2**
func TestProperty_AccountLockoutAfterFailedLogins(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: 5 consecutive failures must lock the account
	properties.Property("account locks after 5 failed login attempts", prop.ForAll(
		func(username string, failureCount int) bool {
			// Skip invalid test cases
			if username == "" || failureCount < 0 {
				return true
			}

			logger := zerolog.Nop()
			config := &AuthLimiterConfig{
				MaxLoginAttempts:    5,
				LoginWindow:         15 * time.Minute,
				LockoutDuration:     15 * time.Minute,
				StorageType:         StorageMemory,
				EnableNotifications: false,
			}

			limiter, err := NewEnhancedRateLimiter(config, &logger)
			if err != nil {
				return false
			}

			ctx := context.Background()

			// Record failed login attempts
			for i := 0; i < failureCount; i++ {
				limiter.RecordFailedLogin(ctx, username)
			}

			// Check if account is locked
			isLocked, _, err := limiter.IsAccountLocked(ctx, username)
			if err != nil {
				return false
			}

			// After 5 or more failures, account should be locked
			if failureCount >= 5 {
				return isLocked
			}

			// With fewer than 5 failures, account should not be locked
			return !isLocked
		},
		genUsername(),
		gen.IntRange(0, 10), // Test with 0-10 failures
	))

	// Property: Locked account should reject login attempts
	properties.Property("locked accounts reject login attempts", prop.ForAll(
		func(username string) bool {
			if username == "" {
				return true
			}

			logger := zerolog.Nop()
			config := &AuthLimiterConfig{
				MaxLoginAttempts:    5,
				LoginWindow:         15 * time.Minute,
				LockoutDuration:     15 * time.Minute,
				StorageType:         StorageMemory,
				EnableNotifications: false,
			}

			limiter, err := NewEnhancedRateLimiter(config, &logger)
			if err != nil {
				return false
			}

			ctx := context.Background()

			// Lock the account by recording 5 failures
			for i := 0; i < 5; i++ {
				limiter.RecordFailedLogin(ctx, username)
			}

			// Verify account is locked
			isLocked, _, _ := limiter.IsAccountLocked(ctx, username)
			if !isLocked {
				return false
			}

			// Try to login - should be rejected
			allowed, err := limiter.AllowLogin(ctx, username)

			return !allowed && err != nil
		},
		genUsername(),
	))

	// Property: Account lockout should expire after lockout duration
	properties.Property("account lockout expires after duration", prop.ForAll(
		func(username string) bool {
			if username == "" {
				return true
			}

			logger := zerolog.Nop()
			// Use a very short lockout for testing
			config := &AuthLimiterConfig{
				MaxLoginAttempts:    5,
				LoginWindow:         15 * time.Minute,
				LockoutDuration:     100 * time.Millisecond,
				StorageType:         StorageMemory,
				EnableNotifications: false,
			}

			limiter, err := NewEnhancedRateLimiter(config, &logger)
			if err != nil {
				return false
			}

			ctx := context.Background()

			// Lock the account
			for i := 0; i < 5; i++ {
				limiter.RecordFailedLogin(ctx, username)
			}

			// Verify locked
			isLocked1, _, _ := limiter.IsAccountLocked(ctx, username)

			// Wait for lockout to expire
			time.Sleep(150 * time.Millisecond)

			// Should be unlocked now
			isLocked2, _, _ := limiter.IsAccountLocked(ctx, username)

			return isLocked1 && !isLocked2
		},
		genUsername(),
	))

	// Property: Different accounts should have independent lockouts
	properties.Property("account lockouts are independent", prop.ForAll(
		func(user1 string, user2 string) bool {
			// Skip if usernames are the same or empty
			if user1 == "" || user2 == "" || user1 == user2 {
				return true
			}

			logger := zerolog.Nop()
			config := &AuthLimiterConfig{
				MaxLoginAttempts:    5,
				LoginWindow:         15 * time.Minute,
				LockoutDuration:     15 * time.Minute,
				StorageType:         StorageMemory,
				EnableNotifications: false,
			}

			limiter, err := NewEnhancedRateLimiter(config, &logger)
			if err != nil {
				return false
			}

			ctx := context.Background()

			// Lock user1's account
			for i := 0; i < 5; i++ {
				limiter.RecordFailedLogin(ctx, user1)
			}

			// Check both accounts
			isLocked1, _, _ := limiter.IsAccountLocked(ctx, user1)
			isLocked2, _, _ := limiter.IsAccountLocked(ctx, user2)

			// user1 should be locked, user2 should not be
			return isLocked1 && !isLocked2
		},
		genUsername(),
		genUsername(),
	))

	// Property: Unlock should clear the lockout
	properties.Property("unlock clears account lockout", prop.ForAll(
		func(username string) bool {
			if username == "" {
				return true
			}

			logger := zerolog.Nop()
			config := &AuthLimiterConfig{
				MaxLoginAttempts:    5,
				LoginWindow:         15 * time.Minute,
				LockoutDuration:     15 * time.Minute,
				StorageType:         StorageMemory,
				EnableNotifications: false,
			}

			limiter, err := NewEnhancedRateLimiter(config, &logger)
			if err != nil {
				return false
			}

			ctx := context.Background()

			// Lock the account
			for i := 0; i < 5; i++ {
				limiter.RecordFailedLogin(ctx, username)
			}

			// Verify locked
			isLocked1, _, _ := limiter.IsAccountLocked(ctx, username)

			// Unlock the account
			err = limiter.UnlockAccount(ctx, username)
			if err != nil {
				return false
			}

			// Should be unlocked now
			isLocked2, _, _ := limiter.IsAccountLocked(ctx, username)

			return isLocked1 && !isLocked2
		},
		genUsername(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Generators

// genIPAddress generates random IP addresses for testing
func genIPAddress() gopter.Gen {
	return gen.OneGenOf(
		// Valid IPv4 addresses
		gen.Const("192.168.1.1"),
		gen.Const("10.0.0.1"),
		gen.Const("172.16.0.1"),
		gen.Const("8.8.8.8"),
		gen.Const("1.1.1.1"),
		// Generate random IPv4
		gen.SliceOfN(4, gen.IntRange(0, 255)).Map(func(octets []int) string {
			return fmt.Sprintf("%d.%d.%d.%d", octets[0], octets[1], octets[2], octets[3])
		}),
		// IPv6 addresses
		gen.Const("::1"),
		gen.Const("2001:db8::1"),
		gen.Const("fe80::1"),
	)
}

// genUsername generates random usernames for testing
func genUsername() gopter.Gen {
	return gen.OneGenOf(
		// Common usernames
		gen.Const("user@example.com"),
		gen.Const("admin@example.com"),
		gen.Const("test@example.com"),
		gen.Const("john.doe@example.com"),
		gen.Const("jane.smith@example.com"),
		// Random alphanumeric usernames
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		// Email-like usernames
		gen.Identifier().Map(func(id string) string {
			return id + "@example.com"
		}),
	)
}
