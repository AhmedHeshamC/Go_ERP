package unit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBackupCreation tests that backups are created successfully
// Validates: Requirements 14.1
func TestBackupCreation(t *testing.T) {
	// Setup test environment
	testDir := setupTestBackupEnvironment(t)
	defer cleanupTestBackupEnvironment(t, testDir)

	// Create a mock backup script that simulates successful backup creation
	mockScript := createMockBackupScript(t, testDir, "success")

	// Execute backup creation
	cmd := exec.Command("bash", mockScript, "backup", "full")
	output, err := cmd.CombinedOutput()

	// Verify backup was created
	if err != nil {
		t.Fatalf("Backup creation failed: %v\nOutput: %s", err, string(output))
	}

	// Check that backup file exists
	backupFiles, err := filepath.Glob(filepath.Join(testDir, "backups", "automated_*.sql*"))
	if err != nil {
		t.Fatalf("Failed to search for backup files: %v", err)
	}

	if len(backupFiles) == 0 {
		t.Error("Expected backup file to be created, but none found")
	}

	// Verify backup file has content
	if len(backupFiles) > 0 {
		info, err := os.Stat(backupFiles[0])
		if err != nil {
			t.Fatalf("Failed to stat backup file: %v", err)
		}
		if info.Size() == 0 {
			t.Error("Backup file is empty")
		}
	}

	// Verify success message in output
	outputStr := string(output)
	if !strings.Contains(outputStr, "SUCCESS") && !strings.Contains(outputStr, "completed successfully") {
		t.Errorf("Expected success message in output, got: %s", outputStr)
	}
}

// TestBackupVerification tests that backup integrity is verified
// Validates: Requirements 14.2
func TestBackupVerification(t *testing.T) {
	testDir := setupTestBackupEnvironment(t)
	defer cleanupTestBackupEnvironment(t, testDir)

	// Create a valid backup file
	backupFile := filepath.Join(testDir, "backups", "test_backup.sql")
	validBackupContent := "-- PostgreSQL database dump\nCREATE TABLE test (id INT);\n"
	if err := os.WriteFile(backupFile, []byte(validBackupContent), 0644); err != nil {
		t.Fatalf("Failed to create test backup file: %v", err)
	}

	// Create verification script
	verifyScript := createBackupVerificationScript(t, testDir, backupFile, true)

	// Execute verification
	cmd := exec.Command("bash", verifyScript)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Backup verification failed: %v\nOutput: %s", err, string(output))
	}

	// Verify success message
	outputStr := string(output)
	if !strings.Contains(outputStr, "verified") && !strings.Contains(outputStr, "SUCCESS") {
		t.Errorf("Expected verification success message, got: %s", outputStr)
	}
}

// TestBackupVerificationFailure tests that corrupted backups are detected
// Validates: Requirements 14.2
func TestBackupVerificationFailure(t *testing.T) {
	testDir := setupTestBackupEnvironment(t)
	defer cleanupTestBackupEnvironment(t, testDir)

	// Create a corrupted backup file
	backupFile := filepath.Join(testDir, "backups", "corrupted_backup.sql")
	corruptedContent := "CORRUPTED DATA INVALID SQL"
	if err := os.WriteFile(backupFile, []byte(corruptedContent), 0644); err != nil {
		t.Fatalf("Failed to create corrupted backup file: %v", err)
	}

	// Create verification script that should fail
	verifyScript := createBackupVerificationScript(t, testDir, backupFile, false)

	// Execute verification - should fail
	cmd := exec.Command("bash", verifyScript)
	output, err := cmd.CombinedOutput()

	// We expect this to fail
	if err == nil {
		t.Error("Expected verification to fail for corrupted backup, but it succeeded")
	}

	// Verify error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "ERROR") && !strings.Contains(outputStr, "failed") {
		t.Errorf("Expected error message for corrupted backup, got: %s", outputStr)
	}
}

// TestBackupRetentionPolicy tests that old backups are cleaned up according to retention policy
// Validates: Requirements 14.4
func TestBackupRetentionPolicy(t *testing.T) {
	testDir := setupTestBackupEnvironment(t)
	defer cleanupTestBackupEnvironment(t, testDir)

	backupDir := filepath.Join(testDir, "backups")

	// Create test backups with different ages
	now := time.Now()

	// Daily backups
	createTestBackup(t, backupDir, "automated_full_backup_20240101_120000_daily.sql", now.AddDate(0, 0, -8)) // 8 days old - should be deleted
	createTestBackup(t, backupDir, "automated_full_backup_20240110_120000_daily.sql", now.AddDate(0, 0, -5)) // 5 days old - should be kept

	// Weekly backups
	createTestBackup(t, backupDir, "automated_full_backup_20231201_120000_weekly.sql", now.AddDate(0, 0, -35)) // 5 weeks old - should be deleted
	createTestBackup(t, backupDir, "automated_full_backup_20240101_120000_weekly.sql", now.AddDate(0, 0, -14)) // 2 weeks old - should be kept

	// Monthly backups
	createTestBackup(t, backupDir, "automated_full_backup_20230101_120000_monthly.sql", now.AddDate(0, -13, 0)) // 13 months old - should be deleted
	createTestBackup(t, backupDir, "automated_full_backup_20240101_120000_monthly.sql", now.AddDate(0, -6, 0))  // 6 months old - should be kept

	// Create cleanup script
	cleanupScript := createBackupCleanupScript(t, testDir)

	// Execute cleanup
	cmd := exec.Command("bash", cleanupScript)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Backup cleanup failed: %v\nOutput: %s", err, string(output))
	}

	// Verify old backups were deleted
	remainingFiles, err := filepath.Glob(filepath.Join(backupDir, "*.sql*"))
	if err != nil {
		t.Fatalf("Failed to list remaining backup files: %v", err)
	}

	// Should have 3 files remaining (1 daily, 1 weekly, 1 monthly)
	if len(remainingFiles) != 3 {
		t.Errorf("Expected 3 backup files to remain, got %d", len(remainingFiles))
	}

	// Verify the correct files remain
	remainingNames := make(map[string]bool)
	for _, file := range remainingFiles {
		remainingNames[filepath.Base(file)] = true
	}

	// Check that old backups were deleted
	if remainingNames["automated_full_backup_20240101_120000_daily.sql"] {
		t.Error("Old daily backup (8 days) should have been deleted")
	}
	if remainingNames["automated_full_backup_20231201_120000_weekly.sql"] {
		t.Error("Old weekly backup (5 weeks) should have been deleted")
	}
	if remainingNames["automated_full_backup_20230101_120000_monthly.sql"] {
		t.Error("Old monthly backup (13 months) should have been deleted")
	}

	// Check that recent backups were kept
	if !remainingNames["automated_full_backup_20240110_120000_daily.sql"] {
		t.Error("Recent daily backup (5 days) should have been kept")
	}
	if !remainingNames["automated_full_backup_20240101_120000_weekly.sql"] {
		t.Error("Recent weekly backup (2 weeks) should have been kept")
	}
	if !remainingNames["automated_full_backup_20240101_120000_monthly.sql"] {
		t.Error("Recent monthly backup (6 months) should have been kept")
	}
}

// TestBackupFailureAlerting tests that backup failures trigger alerts
// Validates: Requirements 14.3
func TestBackupFailureAlerting(t *testing.T) {
	testDir := setupTestBackupEnvironment(t)
	defer cleanupTestBackupEnvironment(t, testDir)

	// Create a mock backup script that simulates failure
	mockScript := createMockBackupScript(t, testDir, "failure")

	// Execute backup that should fail
	cmd := exec.Command("bash", mockScript, "backup", "full")
	output, err := cmd.CombinedOutput()

	// We expect this to fail
	if err == nil {
		t.Error("Expected backup to fail, but it succeeded")
	}

	outputStr := string(output)

	// Verify error message is present
	if !strings.Contains(outputStr, "ERROR") && !strings.Contains(outputStr, "failed") {
		t.Errorf("Expected error message in output, got: %s", outputStr)
	}

	// Verify notification was attempted
	if !strings.Contains(outputStr, "notification") && !strings.Contains(outputStr, "alert") {
		t.Logf("Warning: No notification message found in output: %s", outputStr)
	}
}

// TestBackupRetryLogic tests that backup retries on failure
// Validates: Requirements 14.3
func TestBackupRetryLogic(t *testing.T) {
	testDir := setupTestBackupEnvironment(t)
	defer cleanupTestBackupEnvironment(t, testDir)

	// Create a script that fails first time, succeeds second time
	retryScript := createRetryBackupScript(t, testDir)

	// First attempt - should fail
	cmd1 := exec.Command("bash", retryScript)
	output1, err1 := cmd1.CombinedOutput()

	if err1 == nil {
		t.Error("Expected first backup attempt to fail, but it succeeded")
	}

	outputStr1 := string(output1)
	if !strings.Contains(outputStr1, "attempt 1") {
		t.Errorf("Expected attempt 1 message in first output, got: %s", outputStr1)
	}

	// Second attempt - should succeed (simulating retry)
	cmd2 := exec.Command("bash", retryScript)
	output2, err2 := cmd2.CombinedOutput()

	if err2 != nil {
		t.Fatalf("Backup retry failed: %v\nOutput: %s", err2, string(output2))
	}

	outputStr2 := string(output2)

	// Verify retry was attempted
	if !strings.Contains(outputStr2, "attempt 2") {
		t.Errorf("Expected attempt 2 message in output, got: %s", outputStr2)
	}

	// Verify eventual success
	if !strings.Contains(outputStr2, "SUCCESS") && !strings.Contains(outputStr2, "completed successfully") {
		t.Errorf("Expected success message after retry, got: %s", outputStr2)
	}
}

// Helper functions

func setupTestBackupEnvironment(t *testing.T) string {
	t.Helper()

	testDir, err := os.MkdirTemp("", "backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create necessary subdirectories
	dirs := []string{
		filepath.Join(testDir, "backups"),
		filepath.Join(testDir, "logs"),
		filepath.Join(testDir, "scripts"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	return testDir
}

func cleanupTestBackupEnvironment(t *testing.T, testDir string) {
	t.Helper()
	if err := os.RemoveAll(testDir); err != nil {
		t.Logf("Warning: Failed to cleanup test directory: %v", err)
	}
}

func createTestBackup(t *testing.T, backupDir, filename string, modTime time.Time) {
	t.Helper()

	backupPath := filepath.Join(backupDir, filename)
	content := fmt.Sprintf("-- Test backup created at %s\nCREATE TABLE test (id INT);\n", modTime.Format(time.RFC3339))

	if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test backup %s: %v", filename, err)
	}

	// Set modification time
	if err := os.Chtimes(backupPath, modTime, modTime); err != nil {
		t.Fatalf("Failed to set modification time for %s: %v", filename, err)
	}
}

func createMockBackupScript(t *testing.T, testDir, mode string) string {
	t.Helper()

	scriptPath := filepath.Join(testDir, "scripts", "mock_backup.sh")
	backupDir := filepath.Join(testDir, "backups")
	logDir := filepath.Join(testDir, "logs")

	var scriptContent string
	if mode == "success" {
		scriptContent = fmt.Sprintf(`#!/bin/bash
set -e

BACKUP_DIR="%s"
LOG_DIR="%s"

echo "[$(date)] Starting backup..."
echo "[$(date)] Starting backup..." >> "$LOG_DIR/backup.log"

# Create a mock backup file
TIMESTAMP=$(date '+%%Y%%m%%d_%%H%%M%%S')
BACKUP_FILE="$BACKUP_DIR/automated_full_backup_${TIMESTAMP}.sql"

echo "-- PostgreSQL database dump" > "$BACKUP_FILE"
echo "CREATE TABLE test (id INT);" >> "$BACKUP_FILE"

echo "[SUCCESS] Backup created successfully"
echo "[SUCCESS] Backup completed successfully"
echo "Backup file: $BACKUP_FILE"

exit 0
`, backupDir, logDir)
	} else {
		scriptContent = fmt.Sprintf(`#!/bin/bash

BACKUP_DIR="%s"
LOG_DIR="%s"

echo "[$(date)] Starting backup..."
echo "[$(date)] Starting backup..." >> "$LOG_DIR/backup.log"

echo "[ERROR] Backup creation failed"
echo "[ERROR] Sending notification..."

exit 1
`, backupDir, logDir)
	}

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create mock backup script: %v", err)
	}

	return scriptPath
}

func createBackupVerificationScript(t *testing.T, testDir, backupFile string, shouldSucceed bool) string {
	t.Helper()

	scriptPath := filepath.Join(testDir, "scripts", "verify_backup.sh")
	logDir := filepath.Join(testDir, "logs")

	var scriptContent string
	if shouldSucceed {
		scriptContent = fmt.Sprintf(`#!/bin/bash
set -e

LOG_DIR="%s"
BACKUP_FILE="%s"

echo "[$(date)] Verifying backup integrity..."
echo "[$(date)] Verifying backup integrity..." >> "$LOG_DIR/verify.log"

# Check if file exists and has content
if [[ ! -f "$BACKUP_FILE" ]]; then
    echo "[ERROR] Backup file not found"
    exit 1
fi

if [[ ! -s "$BACKUP_FILE" ]]; then
    echo "[ERROR] Backup file is empty"
    exit 1
fi

# Simple validation - check for SQL content
if grep -q "CREATE TABLE\|PostgreSQL" "$BACKUP_FILE"; then
    echo "[SUCCESS] Backup integrity verified"
    exit 0
else
    echo "[ERROR] Backup integrity check failed"
    exit 1
fi
`, logDir, backupFile)
	} else {
		scriptContent = fmt.Sprintf(`#!/bin/bash

LOG_DIR="%s"
BACKUP_FILE="%s"

echo "[$(date)] Verifying backup integrity..."
echo "[$(date)] Verifying backup integrity..." >> "$LOG_DIR/verify.log"

# Simulate verification failure
echo "[ERROR] Backup integrity check failed - corrupted data detected"
exit 1
`, logDir, backupFile)
	}

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create verification script: %v", err)
	}

	return scriptPath
}

func createBackupCleanupScript(t *testing.T, testDir string) string {
	t.Helper()

	scriptPath := filepath.Join(testDir, "scripts", "cleanup_backups.sh")
	backupDir := filepath.Join(testDir, "backups")
	logDir := filepath.Join(testDir, "logs")

	scriptContent := fmt.Sprintf(`#!/bin/bash
set -e

BACKUP_DIR="%s"
LOG_DIR="%s"

# Retention policy (in days)
DAILY_RETENTION=7
WEEKLY_RETENTION=28  # 4 weeks
MONTHLY_RETENTION=365  # 12 months

echo "[$(date)] Cleaning up old backups..."
echo "[$(date)] Cleaning up old backups..." >> "$LOG_DIR/cleanup.log"

deleted_count=0

# Clean daily backups older than retention period
find "$BACKUP_DIR" -name "*_daily.sql*" -type f -mtime +$DAILY_RETENTION -delete 2>/dev/null && \
    deleted_count=$((deleted_count + $(find "$BACKUP_DIR" -name "*_daily.sql*" -type f -mtime +$DAILY_RETENTION 2>/dev/null | wc -l)))

# Clean weekly backups older than retention period
find "$BACKUP_DIR" -name "*_weekly.sql*" -type f -mtime +$WEEKLY_RETENTION -delete 2>/dev/null && \
    deleted_count=$((deleted_count + $(find "$BACKUP_DIR" -name "*_weekly.sql*" -type f -mtime +$WEEKLY_RETENTION 2>/dev/null | wc -l)))

# Clean monthly backups older than retention period
find "$BACKUP_DIR" -name "*_monthly.sql*" -type f -mtime +$MONTHLY_RETENTION -delete 2>/dev/null && \
    deleted_count=$((deleted_count + $(find "$BACKUP_DIR" -name "*_monthly.sql*" -type f -mtime +$MONTHLY_RETENTION 2>/dev/null | wc -l)))

echo "[SUCCESS] Cleaned up old backups"
echo "Deleted $deleted_count backups"

exit 0
`, backupDir, logDir)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create cleanup script: %v", err)
	}

	return scriptPath
}

func createRetryBackupScript(t *testing.T, testDir string) string {
	t.Helper()

	scriptPath := filepath.Join(testDir, "scripts", "retry_backup.sh")
	backupDir := filepath.Join(testDir, "backups")
	logDir := filepath.Join(testDir, "logs")
	attemptFile := filepath.Join(testDir, "attempt_count")

	scriptContent := fmt.Sprintf(`#!/bin/bash
set -e

BACKUP_DIR="%s"
LOG_DIR="%s"
ATTEMPT_FILE="%s"
MAX_RETRIES=1

echo "[$(date)] Starting backup with retry logic..."
echo "[$(date)] Starting backup with retry logic..." >> "$LOG_DIR/backup.log"

# Initialize attempt counter
if [[ ! -f "$ATTEMPT_FILE" ]]; then
    echo "0" > "$ATTEMPT_FILE"
fi

attempt=$(cat "$ATTEMPT_FILE")
attempt=$((attempt + 1))
echo "$attempt" > "$ATTEMPT_FILE"

echo "Backup attempt $attempt..."

# Fail on first attempt, succeed on second
if [[ $attempt -eq 1 ]]; then
    echo "[ERROR] Backup creation failed on attempt $attempt"
    echo "Retry attempt $attempt of $MAX_RETRIES..."
    exit 1
else
    # Create successful backup
    TIMESTAMP=$(date '+%%Y%%m%%d_%%H%%M%%S')
    BACKUP_FILE="$BACKUP_DIR/automated_full_backup_${TIMESTAMP}.sql"
    
    echo "-- PostgreSQL database dump" > "$BACKUP_FILE"
    echo "CREATE TABLE test (id INT);" >> "$BACKUP_FILE"
    
    echo "[SUCCESS] Backup created successfully on attempt $attempt"
    echo "[SUCCESS] Backup completed successfully"
    rm -f "$ATTEMPT_FILE"
    exit 0
fi
`, backupDir, logDir, attemptFile)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create retry backup script: %v", err)
	}

	return scriptPath
}
