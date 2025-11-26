package security

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// SecurityTestSuite groups all security tests
type SecurityTestSuite struct {
	suite.Suite
}

// TestSecuritySuite runs the security test suite
func TestSecuritySuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}

// Test wrapper functions to run individual tests
func (s *SecurityTestSuite) TestSQLInjectionPrevention() {
	TestSQLInjectionPrevention(s.T())
}

func (s *SecurityTestSuite) TestSQLColumnWhitelistIntegration() {
	TestSQLColumnWhitelistIntegration(s.T())
}

func (s *SecurityTestSuite) TestSQLOrderByClauseValidation() {
	TestSQLOrderByClauseValidation(s.T())
}

func (s *SecurityTestSuite) TestXSSPrevention() {
	TestXSSPrevention(s.T())
}

func (s *SecurityTestSuite) TestCSRFTokenGeneration() {
	TestCSRFTokenGeneration(s.T())
}

func (s *SecurityTestSuite) TestCSRFDoubleSubmitCookie() {
	TestCSRFDoubleSubmitCookie(s.T())
}

func (s *SecurityTestSuite) TestRateLimitBypassAttempts() {
	TestRateLimitBypassAttempts(s.T())
}

func (s *SecurityTestSuite) TestRateLimitWithRealLimiter() {
	TestRateLimitWithRealLimiter(s.T())
}

func (s *SecurityTestSuite) TestRateLimitHeaderSpoofing() {
	TestRateLimitHeaderSpoofing(s.T())
}

func (s *SecurityTestSuite) TestRateLimitBypassWithMultipleIPs() {
	TestRateLimitBypassWithMultipleIPs(s.T())
}

func (s *SecurityTestSuite) TestRateLimitBypassWithUserAgentRotation() {
	TestRateLimitBypassWithUserAgentRotation(s.T())
}

func (s *SecurityTestSuite) TestInputSanitization() {
	TestInputSanitization(s.T())
}

func (s *SecurityTestSuite) TestSecurityHeadersPresent() {
	TestSecurityHeadersPresent(s.T())
}

func (s *SecurityTestSuite) TestPaginationValidation() {
	TestPaginationValidation(s.T())
}
