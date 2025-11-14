package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func createTestFiles(logDir string, filenames []string) error {
	for _, filename := range filenames {
		path := filepath.Join(logDir, filename)
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

func removeTestFiles(logDir string) error {
	files, err := os.ReadDir(logDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		err := os.Remove(filepath.Join(logDir, file.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}

func TestGetUserLogFile(t *testing.T) {
	tempDir := os.TempDir()
	testLogDir := filepath.Join(tempDir, "test_logs")
	err := os.MkdirAll(testLogDir, 0755)
	if err != nil {
		t.Errorf("failed to create log directory: %s", err)
	}
	defer os.RemoveAll(testLogDir)

	now := time.Now()
	y, m, d := now.Date()
	date := fmt.Sprintf(`%d-%d-%d`, y, m, d)

	yesterday := now.AddDate(0, 0, -1)
	y, m, d = yesterday.Date()
	prevDate := fmt.Sprintf(`%d-%d-%d`, y, m, d)

	tests := []struct {
		name             string
		existingFiles    []string
		expectedFilename string
	}{
		{
			name:             "no existing files",
			existingFiles:    []string{},
			expectedFilename: fmt.Sprintf("%s_1.log", date),
		},
		{
			name:             "one existing file with same date",
			existingFiles:    []string{fmt.Sprintf("%s_1.log", date)},
			expectedFilename: fmt.Sprintf("%s_1.log", date),
		},
		{
			name:             "one existing file with older date",
			existingFiles:    []string{fmt.Sprintf("%s_1.log", prevDate)},
			expectedFilename: fmt.Sprintf("%s_1.log", date),
		},
		{
			name:             "multiple existing files",
			existingFiles:    []string{fmt.Sprintf("%s_1.log", prevDate), fmt.Sprintf("%s_2.log", prevDate), fmt.Sprintf("%s_1.log", date), fmt.Sprintf("%s_2.log", date), fmt.Sprintf("%s_3.log", date)},
			expectedFilename: fmt.Sprintf("%s_3.log", date),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = removeTestFiles(testLogDir)
			err = createTestFiles(testLogDir, tt.existingFiles)
			if err != nil {
				t.Errorf("failed to create test files: %s", err)
			}

			logFile, err := getUserLogFile(testLogDir)
			if err != nil {
				t.Errorf("failed to get user log file: %s", err)
			}
			defer logFile.Close()

			actualLogFileName := filepath.Base(logFile.Name())

			if actualLogFileName != tt.expectedFilename {
				t.Errorf("expected log file name %s; got %s", tt.expectedFilename, actualLogFileName)
			}
		})
	}
}

func TestRefreshLogFile(t *testing.T) {
	tempDir := os.TempDir()
	testLogDir := filepath.Join(tempDir, "test_logs")
	err := os.MkdirAll(testLogDir, 0755)
	if err != nil {
		t.Errorf("failed to create log directory: %s", err)
	}
	defer os.RemoveAll(testLogDir)

	now := time.Now()
	y, m, d := now.Date()
	date := fmt.Sprintf(`%d-%d-%d`, y, m, d)

	yesterday := now.AddDate(0, 0, -1)
	y, m, d = yesterday.Date()
	prevDate := fmt.Sprintf(`%d-%d-%d`, y, m, d)

	type LoggerTest struct {
		name           string
		initialLogger  FileLogger
		expectedLogger FileLogger
	}
	var tests [3]LoggerTest

	// Test case 1
	initialFilePath := filepath.Join(testLogDir, fmt.Sprintf("%s_1.log", prevDate))
	initFile1, err := os.OpenFile(initialFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		t.Errorf("failed to create file: %s", err)
	}

	expectedFilePath := filepath.Join(testLogDir, fmt.Sprintf("%s_1.log", date))
	expetedFile1, err := os.OpenFile(expectedFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		t.Errorf("failed to create file: %s", err)
	}

	test1 := &LoggerTest{
		name: "new log file on a new day",
		initialLogger: FileLogger{
			DevMode:        false,
			LogDir:         testLogDir,
			CurrentLogFile: initFile1,
			FileLog:        log.New(initFile1, "", log.LstdFlags),
		},
		expectedLogger: FileLogger{
			DevMode:        false,
			LogDir:         testLogDir,
			CurrentLogFile: expetedFile1,
			FileLog:        log.New(expetedFile1, "", log.LstdFlags),
		},
	}
	tests[0] = *test1

	// Test case 2
	initialFilePath = filepath.Join(testLogDir, fmt.Sprintf("%s_1.log", date))
	initFile2, err := os.OpenFile(initialFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		t.Errorf("failed to create file: %s", err)
	}
	err = initFile2.Truncate(500000)
	if err != nil {
		t.Errorf("failed to resize file: %s", err)
	}

	logger, err := NewLogger(false, testLogDir, 0, LogLevelDebug, LogFormatText)
	if err != nil {
		t.Errorf("failed to create logger: %s", err)
	}

	test2 := &LoggerTest{
		name:           "no new file if size is less than 10MB",
		initialLogger:  *logger,
		expectedLogger: *logger,
	}
	tests[1] = *test2

	// Test case 3
	initialFilePath = filepath.Join(testLogDir, fmt.Sprintf("%s_2.log", date))
	initFile3, err := os.OpenFile(initialFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		t.Errorf("failed to create file: %s", err)
	}
	err = initFile3.Truncate(10000001)
	if err != nil {
		t.Errorf("failed to resize file: %s", err)
	}

	expectedFilePath = filepath.Join(testLogDir, fmt.Sprintf("%s_3.log", date))
	expetedFile3, err := os.OpenFile(expectedFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		t.Errorf("Failed to create file: %s", err)
	}

	test3 := &LoggerTest{
		name: "new file if size exceeds 10MB",
		initialLogger: FileLogger{
			DevMode:        false,
			LogDir:         testLogDir,
			CurrentLogFile: initFile3,
			FileLog:        log.New(initFile3, "", log.LstdFlags),
		},
		expectedLogger: FileLogger{
			DevMode:        false,
			LogDir:         testLogDir,
			CurrentLogFile: expetedFile3,
			FileLog:        log.New(expetedFile3, "", log.LstdFlags),
		},
	}
	tests[2] = *test3

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = tt.initialLogger.refreshLogFile()
			if err != nil {
				t.Errorf("failed to refresh log file: %s", err)
			}

			actualLogFileName := filepath.Base(tt.initialLogger.CurrentLogFile.Name())
			expectedLogFileName := filepath.Base(tt.expectedLogger.CurrentLogFile.Name())

			if actualLogFileName != expectedLogFileName {
				t.Errorf("expected log file name %s; got %s", expectedLogFileName, actualLogFileName)
			}

			// TODO compare loggers
		})
	}
}

func TestConcurrency(t *testing.T) {
	tempDir := os.TempDir()
	testLogDir := filepath.Join(tempDir, "test_logs_concurrency")
	err := os.MkdirAll(testLogDir, 0755)
	if err != nil {
		t.Errorf("failed to create log directory: %s", err)
	}
	defer os.RemoveAll(testLogDir)

	logger, err := NewLogger(false, testLogDir, 0, LogLevelDebug, LogFormatText)
	if err != nil {
		t.Errorf("failed to create logger: %s", err)
	}

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			logger.LogInfo(fmt.Sprintf("test message %d", i))
		}(i)
	}
	wg.Wait()
}

func TestCleanupOldLogs(t *testing.T) {
	tempDir := os.TempDir()
	testLogDir := filepath.Join(tempDir, "test_logs_cleanup")
	err := os.MkdirAll(testLogDir, 0755)
	if err != nil {
		t.Errorf("failed to create log directory: %s", err)
	}
	defer os.RemoveAll(testLogDir)

	now := time.Now()
	oldFile := filepath.Join(testLogDir, "2025-10-01_1.log")
	recentFile := filepath.Join(testLogDir, "2025-11-10_1.log")
	nonLogFile := filepath.Join(testLogDir, "data.txt")

	for _, path := range []string{oldFile, recentFile, nonLogFile} {
		f, err := os.Create(path)
		if err != nil {
			t.Errorf("failed to create test file: %s", err)
		}
		f.Close()
	}

	oldTime := now.AddDate(0, 0, -10)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Errorf("failed to set file time: %s", err)
	}

	logger := &FileLogger{
		DevMode:       false,
		LogDir:        testLogDir,
		MaxLogAgeDays: 7,
	}

	if err := logger.cleanupOldLogs(); err != nil {
		t.Errorf("cleanup failed: %s", err)
	}

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("expected old log file to be deleted")
	}

	if _, err := os.Stat(recentFile); os.IsNotExist(err) {
		t.Errorf("expected recent log file to still exist")
	}

	if _, err := os.Stat(nonLogFile); os.IsNotExist(err) {
		t.Errorf("expected non-log file to still exist")
	}

	logger2 := &FileLogger{
		DevMode:       false,
		LogDir:        testLogDir,
		MaxLogAgeDays: 0,
	}

	if err := logger2.cleanupOldLogs(); err != nil {
		t.Errorf("cleanup failed: %s", err)
	}

	if _, err := os.Stat(recentFile); os.IsNotExist(err) {
		t.Errorf("expected recent log file to still exist after no-op cleanup")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tempDir := os.TempDir()
	testLogDir := filepath.Join(tempDir, "test_logs_level")
	err := os.MkdirAll(testLogDir, 0755)
	if err != nil {
		t.Errorf("failed to create log directory: %s", err)
	}
	defer os.RemoveAll(testLogDir)

	tests := []struct {
		name      string
		minLevel  LogLevel
		logFunc   func(*FileLogger)
		shouldLog bool
	}{
		{
			name:     "debug logged when min level is debug",
			minLevel: LogLevelDebug,
			logFunc: func(l *FileLogger) {
				l.LogDebug("debug message")
			},
			shouldLog: true,
		},
		{
			name:     "debug not logged when min level is info",
			minLevel: LogLevelInfo,
			logFunc: func(l *FileLogger) {
				l.LogDebug("debug message")
			},
			shouldLog: false,
		},
		{
			name:     "info not logged when min level is warn",
			minLevel: LogLevelWarn,
			logFunc: func(l *FileLogger) {
				l.LogInfo("info message")
			},
			shouldLog: false,
		},
		{
			name:     "error logged when min level is warn",
			minLevel: LogLevelWarn,
			logFunc: func(l *FileLogger) {
				l.LogError(fmt.Errorf("error message"))
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = removeTestFiles(testLogDir)
			if err != nil {
				t.Errorf("failed to remove test files: %s", err)
			}

			logger, err := NewLogger(false, testLogDir, 0, tt.minLevel, LogFormatText)
			if err != nil {
				t.Errorf("failed to create logger: %s", err)
			}
			defer logger.Close()

			// Get file size before logging
			info, err := logger.CurrentLogFile.Stat()
			if err != nil {
				t.Errorf("failed to stat log file: %s", err)
			}
			sizeBefore := info.Size()

			tt.logFunc(logger)

			// Get file size after logging
			info, err = logger.CurrentLogFile.Stat()
			if err != nil {
				t.Errorf("failed to stat log file: %s", err)
			}
			sizeAfter := info.Size()

			logWasWritten := sizeAfter > sizeBefore
			if logWasWritten != tt.shouldLog {
				t.Errorf("expected shouldLog=%v, but log was written=%v", tt.shouldLog, logWasWritten)
			}
		})
	}
}

func TestStructuredLogging(t *testing.T) {
	tempDir := os.TempDir()
	testLogDir := filepath.Join(tempDir, "test_logs_structured")
	err := os.MkdirAll(testLogDir, 0755)
	if err != nil {
		t.Errorf("failed to create log directory: %s", err)
	}
	defer os.RemoveAll(testLogDir)

	t.Run("text format with fields", func(t *testing.T) {
		err = removeTestFiles(testLogDir)
		if err != nil {
			t.Errorf("failed to remove test files: %s", err)
		}

		logger, err := NewLogger(false, testLogDir, 0, LogLevelDebug, LogFormatText)
		if err != nil {
			t.Errorf("failed to create logger: %s", err)
		}
		defer logger.Close()

		fields := map[string]interface{}{
			"user_id": 123,
			"action":  "login",
			"ip":      "192.168.1.1",
		}
		logger.LogInfoWith("user logged in", fields)

		content, err := os.ReadFile(logger.CurrentLogFile.Name())
		if err != nil {
			t.Errorf("failed to read log file: %s", err)
		}

		logContent := string(content)
		if !strings.Contains(logContent, "user logged in") {
			t.Errorf("expected log to contain message")
		}
		if !strings.Contains(logContent, "user_id=123") {
			t.Errorf("expected log to contain user_id field")
		}
		if !strings.Contains(logContent, "action=login") {
			t.Errorf("expected log to contain action field")
		}
		if !strings.Contains(logContent, "ip=192.168.1.1") {
			t.Errorf("expected log to contain ip field")
		}
	})

	t.Run("json format with fields", func(t *testing.T) {
		err = removeTestFiles(testLogDir)
		if err != nil {
			t.Errorf("failed to remove test files: %s", err)
		}

		logger, err := NewLogger(false, testLogDir, 0, LogLevelDebug, LogFormatJSON)
		if err != nil {
			t.Errorf("failed to create logger: %s", err)
		}
		defer logger.Close()

		fields := map[string]interface{}{
			"user_id": 456,
			"action":  "logout",
		}
		logger.LogInfoWith("user logged out", fields)

		content, err := os.ReadFile(logger.CurrentLogFile.Name())
		if err != nil {
			t.Errorf("failed to read log file: %s", err)
		}

		// Parse the last line as JSON
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		lastLine := lines[len(lines)-1]
		
		// Extract JSON from log line (skip timestamp prefix)
		jsonStart := strings.Index(lastLine, "{")
		if jsonStart == -1 {
			t.Errorf("expected JSON in log line")
			return
		}
		jsonStr := lastLine[jsonStart:]

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &entry); err != nil {
			t.Errorf("failed to parse JSON log: %s", err)
		}

		if entry["level"] != "INFO" {
			t.Errorf("expected level=INFO, got %v", entry["level"])
		}
		if entry["message"] != "user logged out" {
			t.Errorf("expected message='user logged out', got %v", entry["message"])
		}
		if entry["user_id"] != float64(456) {
			t.Errorf("expected user_id=456, got %v", entry["user_id"])
		}
		if entry["action"] != "logout" {
			t.Errorf("expected action=logout, got %v", entry["action"])
		}
	})

	t.Run("text format without fields", func(t *testing.T) {
		err = removeTestFiles(testLogDir)
		if err != nil {
			t.Errorf("failed to remove test files: %s", err)
		}

		logger, err := NewLogger(false, testLogDir, 0, LogLevelDebug, LogFormatText)
		if err != nil {
			t.Errorf("failed to create logger: %s", err)
		}
		defer logger.Close()

		logger.LogInfo("simple message")

		content, err := os.ReadFile(logger.CurrentLogFile.Name())
		if err != nil {
			t.Errorf("failed to read log file: %s", err)
		}

		logContent := string(content)
		if !strings.Contains(logContent, "INFO simple message") {
			t.Errorf("expected log to contain 'INFO simple message'")
		}
	})
}
