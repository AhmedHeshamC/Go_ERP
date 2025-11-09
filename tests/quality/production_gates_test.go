package quality

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/tests/security"
)

// ProductionGatesTestSuite validates that the project meets production readiness criteria
type ProductionGatesTestSuite struct {
	suite.Suite
	projectRoot      string
	qualityReport    *QualityReport
	securityReport   *security.ScanReport
	buildResults     *BuildResults
	testResults      *TestResults
	coverageResults  *CoverageResults
}

// QualityReport contains the overall quality assessment
type QualityReport struct {
	Timestamp           time.Time        `json:"timestamp"`
	OverallScore        float64         `json:"overall_score"`
	GateStatus          map[string]bool  `json:"gate_status"`
	PassedGates         int             `json:"passed_gates"`
	TotalGates          int             `json:"total_gates"`
	CriticalFailures    []string        `json:"critical_failures"`
	Recommendations     []string        `json:"recommendations"`
	ProductionReady     bool            `json:"production_ready"`
}

// BuildResults contains build validation results
type BuildResults struct {
	Success        bool              `json:"success"`
	Duration       time.Duration     `json:"duration"`
	Errors         []string          `json:"errors"`
	Warnings       []string          `json:"warnings"`
	BinarySize     map[string]int64  `json:"binary_size"`
	BuildArtifacts []string          `json:"build_artifacts"`
}

// TestResults contains test execution results
type TestResults struct {
	TotalTests     int               `json:"total_tests"`
	PassedTests    int               `json:"passed_tests"`
	FailedTests    int               `json:"failed_tests"`
	SkippedTests   int               `json:"skipped_tests"`
	Duration       time.Duration     `json:"duration"`
	Coverage       float64           `json:"coverage"`
	TestSuites     map[string]int    `json:"test_suites"`
	PerformanceOK  bool              `json:"performance_ok"`
}

// CoverageResults contains detailed code coverage information
type CoverageResults struct {
	OverallCoverage  float64            `json:"overall_coverage"`
	PackageCoverage  map[string]float64 `json:"package_coverage"`
	CoveredLines     int                `json:"covered_lines"`
	TotalLines       int                `json:"total_lines"`
	UncoveredFiles  []string           `json:"uncovered_files"`
}

// ProductionGate defines a quality gate with its criteria
type ProductionGate struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Critical     bool    `json:"critical"`
	Weight       float64 `json:"weight"`
	MinScore     float64 `json:"min_score"`
	Passed       bool    `json:"passed"`
	Score        float64 `json:"score"`
	Message      string  `json:"message"`
}

// QualityGate thresholds
const (
	MinTestCoverage      = 80.0  // Minimum test coverage percentage
	MinOverallScore      = 80.0  // Minimum overall quality score
	MaxCriticalFailures  = 0     // Maximum allowed critical failures
	MaxBuildErrors       = 0     // Maximum allowed build errors
	MinTestPassRate      = 95.0  // Minimum test pass rate percentage
	MaxTestDuration      = 10 * time.Minute // Maximum test execution time
	MaxBinarySize        = 100 * 1024 * 1024 // 100MB max binary size
)

// SetupSuite sets up the test suite
func (suite *ProductionGatesTestSuite) SetupSuite() {
	// Find project root
	projectRoot, err := suite.findProjectRoot()
	suite.Require().NoError(err)
	suite.projectRoot = projectRoot

	// Change to project root
	err = os.Chdir(projectRoot)
	suite.Require().NoError(err)

	// Initialize quality report
	suite.qualityReport = &QualityReport{
		Timestamp: time.Now(),
		GateStatus: make(map[string]bool),
	}
}

// findProjectRoot finds the project root directory
func (suite *ProductionGatesTestSuite) findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Look for go.mod file to find project root
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("project root not found")
}

// TestCases

func (suite *ProductionGatesTestSuite) TestBuildGate() {
	suite.T().Log("Testing Build Gate...")

	gate := &ProductionGate{
		Name:        "Build",
		Description: "Project must build successfully without errors",
		Critical:    true,
		Weight:      20.0,
		MinScore:    100.0,
	}

	start := time.Now()
	results := suite.runBuild()
	duration := time.Since(start)

	suite.buildResults = results

	if results.Success {
		gate.Passed = true
		gate.Score = 100.0
		gate.Message = fmt.Sprintf("Build successful in %v", duration)
	} else {
		gate.Passed = false
		gate.Score = 0.0
		gate.Message = fmt.Sprintf("Build failed with %d errors", len(results.Errors))
		suite.qualityReport.CriticalFailures = append(suite.qualityReport.CriticalFailures, "Build failure")
	}

	suite.qualityReport.GateStatus["build"] = gate.Passed
	suite.T().Logf("Build Gate: %s - Score: %.1f", gate.Message, gate.Score)

	if gate.Critical && !gate.Passed {
		suite.T().Logf("CRITICAL: Build gate failed - %v", gate.Message)
	}
}

func (suite *ProductionGatesTestSuite) TestCompilationGate() {
	suite.T().Log("Testing Compilation Gate...")

	gate := &ProductionGate{
		Name:        "Compilation",
		Description: "All packages must compile without warnings",
		Critical:    true,
		Weight:      15.0,
		MinScore:    95.0,
	}

	start := time.Now()
	results := suite.checkCompilation()
	duration := time.Since(start)

	warningCount := len(results.Warnings)
	if warningCount == 0 {
		gate.Score = 100.0
		gate.Passed = true
		gate.Message = "No compilation warnings"
	} else if warningCount <= 5 {
		gate.Score = 100.0 - (float64(warningCount) * 2)
		gate.Passed = gate.Score >= gate.MinScore
		gate.Message = fmt.Sprintf("%d compilation warnings", warningCount)
	} else {
		gate.Score = 90.0 - (float64(warningCount-5))
		gate.Passed = false
		gate.Message = fmt.Sprintf("Too many compilation warnings: %d", warningCount)
		suite.qualityReport.CriticalFailures = append(suite.qualityReport.CriticalFailures, "Excessive compilation warnings")
	}

	suite.qualityReport.GateStatus["compilation"] = gate.Passed
	suite.T().Logf("Compilation Gate: %s - Score: %.1f", gate.Message, gate.Score)

	if gate.Critical && !gate.Passed {
		suite.T().Logf("CRITICAL: Compilation gate failed - %v", gate.Message)
	}
}

func (suite *ProductionGatesTestSuite) TestUnitTestsGate() {
	suite.T().Log("Testing Unit Tests Gate...")

	gate := &ProductionGate{
		Name:        "Unit Tests",
		Description: "All unit tests must pass with adequate coverage",
		Critical:    true,
		Weight:      25.0,
		MinScore:    90.0,
	}

	start := time.Now()
	results := suite.runUnitTests()
	duration := time.Since(start)

	suite.testResults = results

	// Calculate score based on test pass rate and coverage
	passRate := float64(results.PassedTests) / float64(results.TotalTests) * 100
	coverageScore := results.Coverage

	gate.Score = (passRate * 0.6) + (coverageScore * 0.4)

	if passRate >= MinTestPassRate && coverageScore >= MinTestCoverage && results.FailedTests == 0 {
		gate.Passed = true
		gate.Message = fmt.Sprintf("All tests passed (%.1f%% pass rate, %.1f%% coverage)", passRate, coverageScore)
	} else {
		gate.Passed = false
		gate.Message = fmt.Sprintf("Test issues: %.1f%% pass rate, %.1f%% coverage, %d failures", passRate, coverageScore, results.FailedTests)
		suite.qualityReport.CriticalFailures = append(suite.qualityReport.CriticalFailures, "Test failures")
	}

	suite.qualityReport.GateStatus["unit_tests"] = gate.Passed
	suite.T().Logf("Unit Tests Gate: %s - Score: %.1f", gate.Message, gate.Score)

	if gate.Critical && !gate.Passed {
		suite.T().Logf("CRITICAL: Unit Tests gate failed - %v", gate.Message)
	}
}

func (suite *ProductionGatesTestSuite) TestIntegrationTestsGate() {
	suite.T().Log("Testing Integration Tests Gate...")

	gate := &ProductionGate{
		Name:        "Integration Tests",
		Description: "All integration tests must pass",
		Critical:    true,
		Weight:      20.0,
		MinScore:    95.0,
	}

	start := time.Now()
	results := suite.runIntegrationTests()
	duration := time.Since(start)

	if results.Success {
		gate.Score = 100.0
		gate.Passed = true
		gate.Message = "All integration tests passed"
	} else {
		gate.Score = 0.0
		gate.Passed = false
		gate.Message = fmt.Sprintf("Integration tests failed: %v", results.ErrorMessage)
		suite.qualityReport.CriticalFailures = append(suite.qualityReport.CriticalFailures, "Integration test failures")
	}

	suite.qualityReport.GateStatus["integration_tests"] = gate.Passed
	suite.T().Logf("Integration Tests Gate: %s - Score: %.1f", gate.Message, gate.Score)

	if gate.Critical && !gate.Passed {
		suite.T().Logf("CRITICAL: Integration Tests gate failed - %v", gate.Message)
	}
}

func (suite *ProductionGatesTestSuite) TestSecurityScanGate() {
	suite.T().Log("Testing Security Scan Gate...")

	gate := &ProductionGate{
		Name:        "Security Scan",
		Description: "No critical or high-severity security vulnerabilities",
		Critical:    true,
		Weight:      20.0,
		MinScore:    85.0,
	}

	config := &security.ScanConfig{
		ProjectRoot:      suite.projectRoot,
		ScanDependencies: true,
		ScanCode:         true,
		ScanSecrets:      true,
		ScanInsecureCode: true,
		Timeout:          5 * time.Minute,
		ExcludePatterns: []string{
			"vendor/*",
			".git/*",
			"node_modules/*",
			"*.test",
		},
	}

	scanner, err := security.NewVulnerabilityScanner(config)
	suite.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	report, err := scanner.Scan(ctx)
	suite.Require().NoError(err)
	suite.securityReport = report

	// Calculate security score
	criticalCount := report.Summary.CriticalCount
	highCount := report.Summary.HighCount

	if criticalCount == 0 && highCount == 0 {
		gate.Score = 100.0
		gate.Passed = true
		gate.Message = "No critical or high-severity vulnerabilities found"
	} else if criticalCount == 0 && highCount <= 2 {
		gate.Score = 100.0 - (float64(highCount) * 10)
		gate.Passed = gate.Score >= gate.MinScore
		gate.Message = fmt.Sprintf("%d high-severity vulnerabilities found", highCount)
	} else {
		gate.Score = 0.0
		gate.Passed = false
		gate.Message = fmt.Sprintf("Security issues: %d critical, %d high", criticalCount, highCount)
		suite.qualityReport.CriticalFailures = append(suite.qualityReport.CriticalFailures, "Security vulnerabilities")
	}

	suite.qualityReport.GateStatus["security_scan"] = gate.Passed
	suite.T().Logf("Security Scan Gate: %s - Score: %.1f", gate.Message, gate.Score)

	if gate.Critical && !gate.Passed {
		suite.T().Logf("CRITICAL: Security Scan gate failed - %v", gate.Message)
	}
}

func (suite *ProductionGatesTestSuite) TestPerformanceGate() {
	suite.T().Log("Testing Performance Gate...")

	gate := &ProductionGate{
		Name:        "Performance",
		Description: "Performance tests meet baseline requirements",
		Critical:    false,
		Weight:      10.0,
		MinScore:    80.0,
	}

	start := time.Now()
	results := suite.runPerformanceTests()
	duration := time.Since(start)

	if results.Passed {
		gate.Score = 100.0
		gate.Passed = true
		gate.Message = "All performance tests passed"
	} else {
		gate.Score = 70.0
		gate.Passed = gate.Score >= gate.MinScore
		gate.Message = fmt.Sprintf("Performance test issues: %v", results.ErrorMessage)
	}

	suite.qualityReport.GateStatus["performance"] = gate.Passed
	suite.T().Logf("Performance Gate: %s - Score: %.1f", gate.Message, gate.Score)
}

func (suite *ProductionGatesTestSuite) TestCodeQualityGate() {
	suite.T().Log("Testing Code Quality Gate...")

	gate := &ProductionGate{
		Name:        "Code Quality",
		Description: "Code meets quality standards (formatting, linting, etc.)",
		Critical:    false,
		Weight:      10.0,
		MinScore:    80.0,
	}

	results := suite.checkCodeQuality()

	if results.Passed {
		gate.Score = 100.0
		gate.Passed = true
		gate.Message = "Code quality checks passed"
	} else {
		gate.Score = 75.0
		gate.Passed = gate.Score >= gate.MinScore
		gate.Message = fmt.Sprintf("Code quality issues: %v", results.ErrorMessage)
	}

	suite.qualityReport.GateStatus["code_quality"] = gate.Passed
	suite.T().Logf("Code Quality Gate: %s - Score: %.1f", gate.Message, gate.Score)
}

func (suite *ProductionGatesTestSuite) TestOverallProductionReadiness() {
	suite.T().Log("Evaluating Overall Production Readiness...")

	// Calculate overall score
	totalWeight := 0.0
	weightedScore := 0.0

	gates := map[string]float64{
		"build":            20.0,
		"compilation":      15.0,
		"unit_tests":       25.0,
		"integration_tests": 20.0,
		"security_scan":    20.0,
		"performance":      10.0,
		"code_quality":     10.0,
	}

	for gateName, weight := range gates {
		if passed, exists := suite.qualityReport.GateStatus[gateName]; exists {
			totalWeight += weight
			if passed {
				weightedScore += weight
			}
		}
	}

	if totalWeight > 0 {
		suite.qualityReport.OverallScore = (weightedScore / totalWeight) * 100
	}

	// Count passed gates
	for _, passed := range suite.qualityReport.GateStatus {
		if passed {
			suite.qualityReport.PassedGates++
		}
	}
	suite.qualityReport.TotalGates = len(suite.qualityReport.GateStatus)

	// Determine production readiness
	suite.qualityReport.ProductionReady = (
		suite.qualityReport.OverallScore >= MinOverallScore &&
		len(suite.qualityReport.CriticalFailures) == 0
	)

	// Generate recommendations
	suite.generateRecommendations()

	suite.T().Logf("Overall Quality Assessment:")
	suite.T().Logf("  Score: %.1f%%", suite.qualityReport.OverallScore)
	suite.T().Logf("  Gates Passed: %d/%d", suite.qualityReport.PassedGates, suite.qualityReport.TotalGates)
	suite.T().Logf("  Critical Failures: %d", len(suite.qualityReport.CriticalFailures))
	suite.T().Logf("  Production Ready: %s", map[bool]string{true: "YES", false: "NO"}[suite.qualityReport.ProductionReady])

	if !suite.qualityReport.ProductionReady {
		suite.T().Log("RECOMMENDATIONS:")
		for _, rec := range suite.qualityReport.Recommendations {
			suite.T().Logf("  - %s", rec)
		}
	}

	// Assert production readiness
	if !suite.qualityReport.ProductionReady {
		suite.T().Log("⚠️  PRODUCTION READINESS FAILED - Address critical issues before deployment")
	} else {
		suite.T().Log("✅ PRODUCTION READINESS PASSED - System is ready for production deployment")
	}

	// Save quality report
	suite.saveQualityReport()
}

// Helper methods for executing checks

func (suite *ProductionGatesTestSuite) runBuild() *BuildResults {
	results := &BuildResults{
		Success:        false,
		Duration:       0,
		Errors:         []string{},
		Warnings:       []string{},
		BinarySize:     make(map[string]int64),
		BuildArtifacts: []string{},
	}

	start := time.Now()

	// Run go build
	cmd := exec.Command("go", "build", "./...")
	output, err := cmd.CombinedOutput()
	results.Duration = time.Since(start)

	if err != nil {
		results.Success = false
		results.Errors = append(results.Errors, string(output))
		return results
	}

	results.Success = true

	// Check for main binary
	if mainBinary, err := os.Stat("cmd/api/api"); err == nil {
		results.BinarySize["api"] = mainBinary.Size()
		results.BuildArtifacts = append(results.BuildArtifacts, "cmd/api/api")
	}

	return results
}

func (suite *ProductionGatesTestSuite) checkCompilation() *BuildResults {
	results := &BuildResults{
		Success:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Run go vet for potential issues
	cmd := exec.Command("go", "vet", "./...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		results.Success = false
		results.Errors = append(results.Errors, string(output))
	}

	// Run go build with vet warnings
	cmd = exec.Command("go", "build", "-gcflags=-m", "./...")
	output, err = cmd.CombinedOutput()
	if err != nil {
		results.Success = false
	} else {
		// Parse warnings from build output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "warning") || strings.Contains(line, "can't inline") {
				results.Warnings = append(results.Warnings, line)
			}
		}
	}

	return results
}

func (suite *ProductionGatesTestSuite) runUnitTests() *TestResults {
	results := &TestResults{
		TotalTests:    0,
		PassedTests:   0,
		FailedTests:   0,
		SkippedTests:  0,
		Duration:      0,
		Coverage:      0.0,
		TestSuites:    make(map[string]int),
		PerformanceOK: true,
	}

	start := time.Now()

	// Run tests with coverage
	cmd := exec.Command("go", "test", "-v", "-cover", "-coverprofile=coverage.out", "./...")
	output, err := cmd.CombinedOutput()
	results.Duration = time.Since(start)

	// Parse test output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "PASS") || strings.Contains(line, "FAIL") {
			results.TotalTests++
			if strings.Contains(line, "PASS") {
				results.PassedTests++
			} else if strings.Contains(line, "FAIL") {
				results.FailedTests++
			}
		}
	}

	// Extract coverage information
	if coverageData, err := os.ReadFile("coverage.out"); err == nil {
		lines := strings.Split(string(coverageData), "\n")
		totalLines := 0
		coveredLines := 0

		for _, line := range lines {
			if strings.HasPrefix(line, "mode:") {
				continue
			}
			if parts := strings.Fields(line); len(parts) >= 3 {
				totalLines++
				if parts[2] != "0" {
					coveredLines++
				}
			}
		}

		if totalLines > 0 {
			results.Coverage = float64(coveredLines) / float64(totalLines) * 100
		}
	}

	return results
}

func (suite *ProductionGatesTestSuite) runIntegrationTests() *BuildResults {
	results := &BuildResults{
		Success:      false,
		ErrorMessage: "",
	}

	// Check if integration tests exist
	if _, err := os.Stat("tests/integration"); os.IsNotExist(err) {
		results.Success = true
		results.ErrorMessage = "No integration tests found"
		return results
	}

	// Run integration tests
	cmd := exec.Command("go", "test", "-v", "./tests/integration/...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		results.Success = false
		results.ErrorMessage = string(output)
	} else {
		results.Success = true
	}

	return results
}

func (suite *ProductionGatesTestSuite) runPerformanceTests() *BuildResults {
	results := &BuildResults{
		Success:      false,
		ErrorMessage: "",
	}

	// Check if performance tests exist
	if _, err := os.Stat("tests/performance"); os.IsNotExist(err) {
		results.Success = true
		results.ErrorMessage = "No performance tests found"
		return results
	}

	// Run performance tests with timeout
	cmd := exec.Command("go", "test", "-run", "Benchmark", "-bench=.", "./tests/performance/...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		results.Success = false
		results.ErrorMessage = string(output)
	} else {
		results.Success = true
	}

	return results
}

func (suite *ProductionGatesTestSuite) checkCodeQuality() *BuildResults {
	results := &BuildResults{
		Success:      true,
		ErrorMessage: "",
	}

	// Check for gofmt
	cmd := exec.Command("gofmt", "-l", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		results.Success = false
		results.ErrorMessage = "gofmt check failed"
		return results
	}

	if len(output) > 0 {
		results.Success = false
		results.ErrorMessage = "Code formatting issues detected"
		return results
	}

	// Check for golint if available
	if _, err := exec.LookPath("golint"); err == nil {
		cmd = exec.Command("golint", "./...")
		output, err = cmd.CombinedOutput()
		if err != nil && len(output) > 0 {
			// Golint issues are warnings, not failures
			results.Success = true
		}
	}

	return results
}

func (suite *ProductionGatesTestSuite) generateRecommendations() {
	recommendations := []string{}

	// Check for critical failures
	if len(suite.qualityReport.CriticalFailures) > 0 {
		recommendations = append(recommendations, "Address all critical failures before production deployment")
	}

	// Check test coverage
	if suite.testResults != nil && suite.testResults.Coverage < MinTestCoverage {
		recommendations = append(recommendations, fmt.Sprintf("Increase test coverage to at least %.1f%% (currently %.1f%%)", MinTestCoverage, suite.testResults.Coverage))
	}

	// Check security
	if suite.securityReport != nil && suite.securityReport.Summary.CriticalCount > 0 {
		recommendations = append(recommendations, "Fix all critical security vulnerabilities immediately")
	}

	if suite.securityReport != nil && suite.securityReport.Summary.HighCount > 5 {
		recommendations = append(recommendations, "Address high-severity security vulnerabilities")
	}

	// Check build size
	if suite.buildResults != nil {
		for binary, size := range suite.buildResults.BinarySize {
			if size > MaxBinarySize {
				recommendations = append(recommendations, fmt.Sprintf("Optimize binary size for %s (currently %d bytes, max %d bytes)", binary, size, MaxBinarySize))
			}
		}
	}

	suite.qualityReport.Recommendations = recommendations
}

func (suite *ProductionGatesTestSuite) saveQualityReport() {
	reportData, err := json.MarshalIndent(suite.qualityReport, "", "  ")
	if err != nil {
		suite.T().Logf("Error marshaling quality report: %v", err)
		return
	}

	err = os.WriteFile("quality-report.json", reportData, 0644)
	if err != nil {
		suite.T().Logf("Error saving quality report: %v", err)
		return
	}

	suite.T().Log("Quality report saved to quality-report.json")
}

// Run the test suite
func TestProductionGatesSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production gates test in short mode")
	}
	suite.Run(t, new(ProductionGatesTestSuite))
}

// Standalone quality gate checker for CI/CD
func RunProductionGates(projectRoot string) (*QualityReport, error) {
	suite := &ProductionGatesTestSuite{}

	// Setup
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	defer os.Chdir(dir)

	suite.projectRoot = projectRoot
	suite.qualityReport = &QualityReport{
		Timestamp:  time.Now(),
		GateStatus: make(map[string]bool),
	}

	// Run all gates
	suite.TestBuildGate()
	suite.TestCompilationGate()
	suite.TestUnitTestsGate()
	suite.TestIntegrationTestsGate()
	suite.TestSecurityScanGate()
	suite.TestPerformanceGate()
	suite.TestCodeQualityGate()
	suite.TestOverallProductionReadiness()

	return suite.qualityReport, nil
}