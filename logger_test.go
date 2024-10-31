package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createTestFiles creates test log files in the given directory.
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

// removeTestFiles removes all files in the given directory.
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

	logger := &FileLogger{
		DevMode:        false,
		LogDir:         testLogDir,
		CurrentLogFile: initFile2,
		FileLog:        log.New(initFile2, "", log.LstdFlags),
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
