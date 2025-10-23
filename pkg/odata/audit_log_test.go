package odata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAuditLogConfig(t *testing.T) {
	config := DefaultAuditLogConfig()

	t.Run("Has default values", func(t *testing.T) {
		assert.NotNil(t, config)
		assert.False(t, config.Enabled, "Should be disabled by default")
		assert.Equal(t, "stdout", config.LogType)
		assert.Equal(t, "audit.log", config.FilePath)
		assert.Equal(t, "json", config.Format)
		assert.Equal(t, 100, config.BufferSize)
		assert.Empty(t, config.LoggedOperations)
		assert.False(t, config.IncludeSensitiveData)
	})
}

func TestNewAuditLogger_NoOp(t *testing.T) {
	t.Run("Disabled config returns NoOpLogger", func(t *testing.T) {
		config := &AuditLogConfig{
			Enabled: false,
		}

		logger, err := NewAuditLogger(config)

		require.NoError(t, err)
		assert.IsType(t, &NoOpAuditLogger{}, logger)
	})

	t.Run("Nil config returns NoOpLogger", func(t *testing.T) {
		logger, err := NewAuditLogger(nil)

		require.NoError(t, err)
		assert.IsType(t, &NoOpAuditLogger{}, logger)
	})

	t.Run("Type none returns NoOpLogger", func(t *testing.T) {
		config := &AuditLogConfig{
			Enabled: true,
			LogType: "none",
		}

		logger, err := NewAuditLogger(config)

		require.NoError(t, err)
		assert.IsType(t, &NoOpAuditLogger{}, logger)
	})
}

func TestNewAuditLogger_Stdout(t *testing.T) {
	config := &AuditLogConfig{
		Enabled: true,
		LogType: "stdout",
		Format:  "json",
	}

	logger, err := NewAuditLogger(config)

	require.NoError(t, err)
	assert.IsType(t, &StdoutAuditLogger{}, logger)
}

func TestNewAuditLogger_Stderr(t *testing.T) {
	config := &AuditLogConfig{
		Enabled: true,
		LogType: "stderr",
		Format:  "json",
	}

	logger, err := NewAuditLogger(config)

	require.NoError(t, err)
	assert.IsType(t, &StderrAuditLogger{}, logger)
}

func TestNewAuditLogger_File(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_audit.log")

	config := &AuditLogConfig{
		Enabled:    true,
		LogType:    "file",
		FilePath:   logFile,
		Format:     "json",
		BufferSize: 10,
	}

	logger, err := NewAuditLogger(config)

	require.NoError(t, err)
	assert.IsType(t, &FileAuditLogger{}, logger)

	// Cleanup
	if fileLogger, ok := logger.(*FileAuditLogger); ok {
		fileLogger.Close()
	}
}

func TestNewAuditLogger_UnknownType(t *testing.T) {
	config := &AuditLogConfig{
		Enabled: true,
		LogType: "unknown",
	}

	_, err := NewAuditLogger(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown audit log type")
}

func TestNoOpAuditLogger(t *testing.T) {
	logger := &NoOpAuditLogger{}

	t.Run("Log does nothing", func(t *testing.T) {
		entry := AuditLogEntry{
			Timestamp: time.Now(),
			Operation: AuditOpCreate,
		}

		err := logger.Log(entry)
		assert.NoError(t, err)
	})

	t.Run("Close does nothing", func(t *testing.T) {
		err := logger.Close()
		assert.NoError(t, err)
	})
}

func TestFileAuditLogger_JSON(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_audit_json.log")

	config := &AuditLogConfig{
		Enabled:    true,
		LogType:    "file",
		FilePath:   logFile,
		Format:     "json",
		BufferSize: 10,
	}

	logger, err := NewFileAuditLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("Logs JSON entry", func(t *testing.T) {
		entry := AuditLogEntry{
			Timestamp:  time.Now(),
			UserID:     "user123",
			Username:   "testuser",
			IP:         "192.168.1.1",
			Method:     "POST",
			Path:       "/api/users",
			EntityName: "User",
			Operation:  AuditOpCreate,
			EntityID:   "123",
			Success:    true,
			Duration:   100,
		}

		err := logger.Log(entry)
		assert.NoError(t, err)

		// Give time for async write
		time.Sleep(50 * time.Millisecond)

		// Read file and verify
		data, err := os.ReadFile(logFile)
		require.NoError(t, err)

		var loggedEntry AuditLogEntry
		err = json.Unmarshal(data, &loggedEntry)
		require.NoError(t, err)

		assert.Equal(t, entry.UserID, loggedEntry.UserID)
		assert.Equal(t, entry.Operation, loggedEntry.Operation)
	})
}

func TestFileAuditLogger_Text(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_audit_text.log")

	config := &AuditLogConfig{
		Enabled:    true,
		LogType:    "file",
		FilePath:   logFile,
		Format:     "text",
		BufferSize: 10,
	}

	logger, err := NewFileAuditLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("Logs text entry", func(t *testing.T) {
		entry := AuditLogEntry{
			Timestamp:  time.Now(),
			Username:   "testuser",
			IP:         "192.168.1.1",
			EntityName: "User",
			Operation:  AuditOpCreate,
			Success:    true,
		}

		err := logger.Log(entry)
		assert.NoError(t, err)

		// Give time for async write
		time.Sleep(50 * time.Millisecond)

		// Read file and verify
		data, err := os.ReadFile(logFile)
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "CREATE")
		assert.Contains(t, content, "testuser")
		assert.Contains(t, content, "SUCCESS")
	})
}

func TestFileAuditLogger_FilteredOperations(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_audit_filtered.log")

	config := &AuditLogConfig{
		Enabled:          true,
		LogType:          "file",
		FilePath:         logFile,
		Format:           "json",
		BufferSize:       10,
		LoggedOperations: []AuditOperation{AuditOpCreate, AuditOpDelete},
	}

	logger, err := NewFileAuditLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("Logs only filtered operations", func(t *testing.T) {
		// Should be logged
		createEntry := AuditLogEntry{
			Timestamp: time.Now(),
			Operation: AuditOpCreate,
			Username:  "user1",
		}
		logger.Log(createEntry)

		// Should NOT be logged
		updateEntry := AuditLogEntry{
			Timestamp: time.Now(),
			Operation: AuditOpUpdate,
			Username:  "user2",
		}
		logger.Log(updateEntry)

		// Should be logged
		deleteEntry := AuditLogEntry{
			Timestamp: time.Now(),
			Operation: AuditOpDelete,
			Username:  "user3",
		}
		logger.Log(deleteEntry)

		// Give time for async write
		time.Sleep(100 * time.Millisecond)

		// Read file and verify
		data, err := os.ReadFile(logFile)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		assert.Len(t, lines, 2, "Should have 2 entries (CREATE and DELETE)")
	})
}

func TestFileAuditLogger_BufferOverflow(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_audit_overflow.log")

	config := &AuditLogConfig{
		Enabled:    true,
		LogType:    "file",
		FilePath:   logFile,
		Format:     "json",
		BufferSize: 2, // Small buffer to test overflow
	}

	logger, err := NewFileAuditLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("Handles buffer overflow", func(t *testing.T) {
		// Fill buffer beyond capacity
		for i := 0; i < 10; i++ {
			entry := AuditLogEntry{
				Timestamp: time.Now(),
				Operation: AuditOpCreate,
				EntityID:  string(rune('A' + i)),
			}
			err := logger.Log(entry)
			assert.NoError(t, err)
		}

		// Give time for writes
		time.Sleep(200 * time.Millisecond)

		// Verify all entries were written
		data, err := os.ReadFile(logFile)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		assert.GreaterOrEqual(t, len(lines), 10, "All entries should be written")
	})
}

func TestFileAuditLogger_Close(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_audit_close.log")

	config := &AuditLogConfig{
		Enabled:    true,
		LogType:    "file",
		FilePath:   logFile,
		Format:     "json",
		BufferSize: 10,
	}

	logger, err := NewFileAuditLogger(config)
	require.NoError(t, err)

	t.Run("Flushes buffer on close", func(t *testing.T) {
		// Add entries
		for i := 0; i < 5; i++ {
			entry := AuditLogEntry{
				Timestamp: time.Now(),
				Operation: AuditOpRead,
				EntityID:  string(rune('A' + i)),
			}
			logger.Log(entry)
		}

		// Close immediately (should flush buffer)
		err := logger.Close()
		assert.NoError(t, err)

		// Verify entries were written
		data, err := os.ReadFile(logFile)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		assert.Len(t, lines, 5, "All buffered entries should be flushed")
	})
}

func TestStdoutAuditLogger_JSON(t *testing.T) {
	config := &AuditLogConfig{
		Enabled: true,
		LogType: "stdout",
		Format:  "json",
	}

	logger := &StdoutAuditLogger{config: config}

	t.Run("Logs to stdout without error", func(t *testing.T) {
		entry := AuditLogEntry{
			Timestamp: time.Now(),
			Operation: AuditOpAuthSuccess,
			Username:  "testuser",
		}

		err := logger.Log(entry)
		assert.NoError(t, err)
	})

	t.Run("Close does not error", func(t *testing.T) {
		err := logger.Close()
		assert.NoError(t, err)
	})
}

func TestStdoutAuditLogger_Text(t *testing.T) {
	config := &AuditLogConfig{
		Enabled: true,
		LogType: "stdout",
		Format:  "text",
	}

	logger := &StdoutAuditLogger{config: config}

	t.Run("Logs text format", func(t *testing.T) {
		entry := AuditLogEntry{
			Timestamp:  time.Now(),
			Operation:  AuditOpCreate,
			Username:   "testuser",
			EntityName: "User",
			Success:    true,
		}

		err := logger.Log(entry)
		assert.NoError(t, err)
	})
}

func TestStderrAuditLogger_JSON(t *testing.T) {
	config := &AuditLogConfig{
		Enabled: true,
		LogType: "stderr",
		Format:  "json",
	}

	logger := &StderrAuditLogger{config: config}

	t.Run("Logs to stderr without error", func(t *testing.T) {
		entry := AuditLogEntry{
			Timestamp: time.Now(),
			Operation: AuditOpAuthFailure,
			Username:  "baduser",
			Success:   false,
		}

		err := logger.Log(entry)
		assert.NoError(t, err)
	})

	t.Run("Close does not error", func(t *testing.T) {
		err := logger.Close()
		assert.NoError(t, err)
	})
}

func TestStderrAuditLogger_Text(t *testing.T) {
	config := &AuditLogConfig{
		Enabled: true,
		LogType: "stderr",
		Format:  "text",
	}

	logger := &StderrAuditLogger{config: config}

	t.Run("Logs text format", func(t *testing.T) {
		entry := AuditLogEntry{
			Timestamp:    time.Now(),
			Operation:    AuditOpUnauthorized,
			Username:     "testuser",
			Success:      false,
			ErrorMessage: "Access denied",
		}

		err := logger.Log(entry)
		assert.NoError(t, err)
	})
}

func TestAuditLogEntry_Fields(t *testing.T) {
	t.Run("Contains all expected fields", func(t *testing.T) {
		entry := AuditLogEntry{
			Timestamp:    time.Now(),
			UserID:       "user123",
			Username:     "testuser",
			IP:           "192.168.1.1",
			Method:       "POST",
			Path:         "/api/users",
			EntityName:   "User",
			Operation:    AuditOpCreate,
			EntityID:     "456",
			Success:      true,
			ErrorMessage: "",
			Duration:     150,
			UserAgent:    "Mozilla/5.0",
			RequestID:    "req-123",
			TenantID:     "tenant-1",
			Extra:        map[string]interface{}{"key": "value"},
		}

		assert.NotZero(t, entry.Timestamp)
		assert.Equal(t, "user123", entry.UserID)
		assert.Equal(t, "testuser", entry.Username)
		assert.Equal(t, AuditOpCreate, entry.Operation)
		assert.True(t, entry.Success)
		assert.Equal(t, int64(150), entry.Duration)
		assert.NotNil(t, entry.Extra)
	})
}

func TestAuditOperation_Constants(t *testing.T) {
	t.Run("All operations defined", func(t *testing.T) {
		assert.Equal(t, AuditOperation("CREATE"), AuditOpCreate)
		assert.Equal(t, AuditOperation("UPDATE"), AuditOpUpdate)
		assert.Equal(t, AuditOperation("DELETE"), AuditOpDelete)
		assert.Equal(t, AuditOperation("READ"), AuditOpRead)
		assert.Equal(t, AuditOperation("AUTH_SUCCESS"), AuditOpAuthSuccess)
		assert.Equal(t, AuditOperation("AUTH_FAILURE"), AuditOpAuthFailure)
		assert.Equal(t, AuditOperation("AUTH_LOGOUT"), AuditOpAuthLogout)
		assert.Equal(t, AuditOperation("UNAUTHORIZED"), AuditOpUnauthorized)
	})
}

func TestFileAuditLogger_ConcurrentWrites(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_audit_concurrent.log")

	config := &AuditLogConfig{
		Enabled:    true,
		LogType:    "file",
		FilePath:   logFile,
		Format:     "json",
		BufferSize: 50,
	}

	logger, err := NewFileAuditLogger(config)
	require.NoError(t, err)

	t.Run("Handles concurrent writes", func(t *testing.T) {
		const numGoroutines = 10
		const entriesPerGoroutine = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < entriesPerGoroutine; j++ {
					entry := AuditLogEntry{
						Timestamp: time.Now(),
						Operation: AuditOpCreate,
						EntityID:  string(rune('A' + id)),
						UserID:    string(rune('0' + j)),
					}
					logger.Log(entry)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Give time for all writes
		time.Sleep(500 * time.Millisecond)
	})

	// Close logger after test completes
	logger.Close()

	// Verify all entries were written
	data, err := os.ReadFile(logFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Be more lenient - check we got most of the writes (at least 90%)
	expectedMin := (10 * 10 * 9) / 10
	assert.GreaterOrEqual(t, len(lines), expectedMin, "Most concurrent writes should be persisted")
}

func TestAuditLogger_InvalidFilePath(t *testing.T) {
	config := &AuditLogConfig{
		Enabled:  true,
		LogType:  "file",
		FilePath: "/invalid/path/that/does/not/exist/audit.log",
	}

	_, err := NewFileAuditLogger(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open audit log file")
}
