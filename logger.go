package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

type Logger interface {
	LogFatal(err error)
	LogError(err error)
	LogWarn(message string)
	LogInfo(message string)
	LogDebug(message string)
	LogFatalWith(err error, fields map[string]interface{})
	LogErrorWith(err error, fields map[string]interface{})
	LogWarnWith(message string, fields map[string]interface{})
	LogInfoWith(message string, fields map[string]interface{})
	LogDebugWith(message string, fields map[string]interface{})
}

type LogFormat int

const (
	LogFormatText LogFormat = iota
	LogFormatJSON
)

type FileLogger struct {
	DevMode        bool
	LogDir         string
	CurrentLogFile *os.File
	FileLog        *log.Logger
	MaxLogAgeDays  int
	MinLevel       LogLevel
	Format         LogFormat
	stopCleanup    chan struct{}
	mu             sync.Mutex
}

// NewLogger creates a new FileLogger instance.
//
// Parameters:
//   - devMode: a boolean indicating whether the logger should output more detailed messages suitable for debugging.
//   - appDir: a string representing the subdirectory where log files should be stored. This should be a relative path, and will result in `user_home_dir/[appDir]/logs`.
//   - maxLogAgeDays: maximum age of log files in days before cleanup (0 = no cleanup).
//   - minLevel: minimum log level to write (logs below this level are ignored).
//   - format: log format (LogFormatText or LogFormatJSON).
func NewLogger(devMode bool, appDir string, maxLogAgeDays int, minLevel LogLevel, format LogFormat) (*FileLogger, error) {
	if devMode {
		log.Println("INFO logger running in development mode")
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed getting the current os user: %w", err)
	}

	homeDir := currentUser.HomeDir
	logDir := filepath.Join(homeDir, appDir, "logs")
	if err = os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed creating log directory: %w", err)
	}

	var fileLogger *log.Logger
	logFile, err := getUserLogFile(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed getting log file: %w", err)
	} else {
		fileLogger = log.New(logFile, "", log.LstdFlags)
	}

	logger := &FileLogger{
		DevMode:        devMode,
		LogDir:         logDir,
		CurrentLogFile: logFile,
		FileLog:        fileLogger,
		MaxLogAgeDays:  maxLogAgeDays,
		MinLevel:       minLevel,
		Format:         format,
		stopCleanup:    make(chan struct{}),
	}

	if err := logger.cleanupOldLogs(); err != nil {
		log.Printf("WARNING failed to cleanup old logs: %s", err.Error())
	}

	if maxLogAgeDays > 0 {
		go logger.periodicCleanup()
	}

	return logger, nil
}

func (l *FileLogger) LogFatal(err error) {
	l.LogFatalWith(err, nil)
}

func (l *FileLogger) LogFatalWith(err error, fields map[string]interface{}) {
	if l.MinLevel > LogLevelFatal {
		return
	}
	message := l.formatMessage("FATAL", err.Error(), fields)
	l.logToFile(message)
	log.Fatal(message)
}

func (l *FileLogger) LogError(err error) {
	l.LogErrorWith(err, nil)
}

func (l *FileLogger) LogErrorWith(err error, fields map[string]interface{}) {
	if l.MinLevel > LogLevelError {
		return
	}
	message := l.formatMessage("ERROR", err.Error(), fields)
	log.Println(message)
	l.logToFile(message)
}

func (l *FileLogger) LogWarn(message string) {
	l.LogWarnWith(message, nil)
}

func (l *FileLogger) LogWarnWith(message string, fields map[string]interface{}) {
	if l.MinLevel > LogLevelWarn {
		return
	}
	formatted := l.formatMessage("WARNING", message, fields)
	log.Println(formatted)
	l.logToFile(formatted)
}

func (l *FileLogger) LogInfo(message string) {
	l.LogInfoWith(message, nil)
}

func (l *FileLogger) LogInfoWith(message string, fields map[string]interface{}) {
	if l.MinLevel > LogLevelInfo {
		return
	}
	formatted := l.formatMessage("INFO", message, fields)
	log.Println(formatted)
	l.logToFile(formatted)
}

func (l *FileLogger) LogDebug(message string) {
	l.LogDebugWith(message, nil)
}

func (l *FileLogger) LogDebugWith(message string, fields map[string]interface{}) {
	if l.MinLevel > LogLevelDebug {
		return
	}
	formatted := l.formatMessage("DEBUG", message, fields)
	l.logToFile(formatted)

	if l.DevMode {
		log.Println(formatted)
	}
}

func (l *FileLogger) formatMessage(level, message string, fields map[string]interface{}) string {
	if l.Format == LogFormatJSON {
		entry := map[string]interface{}{
			"level":   level,
			"message": message,
			"time":    time.Now().Format(time.RFC3339),
		}
		for k, v := range fields {
			entry[k] = v
		}
		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			return fmt.Sprintf("%s %s fields_error=%v", level, message, err)
		}
		return string(jsonBytes)
	}

	// Text format
	if fields == nil || len(fields) == 0 {
		return fmt.Sprintf("%s %s", level, message)
	}

	var fieldStrs []string
	for k, v := range fields {
		fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
	}
	return fmt.Sprintf("%s %s %s", level, message, strings.Join(fieldStrs, " "))
}

func (l *FileLogger) logToFile(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := l.refreshLogFile()
	if err != nil {
		log.Printf("FATAL failed refreshing log file: %s", err.Error())
		return
	}

	l.FileLog.Println(message)
}

func (l *FileLogger) refreshLogFile() error {
	filename := filepath.Base(l.CurrentLogFile.Name())

	now := time.Now()
	y, m, d := now.Date()
	date := fmt.Sprintf(`%d-%d-%d`, y, m, d)

	var newFileName string
	if !strings.HasPrefix(filename, date) {
		newFileName = fmt.Sprintf(`%s_1.log`, date)
	} else {
		info, err := l.CurrentLogFile.Stat()
		if err != nil {
			return err
		}

		if info.Size() < 10000000 {
			return nil
		}

		oldName := filename[:len(filename)-4]
		currNum := strings.Split(oldName, "_")[1]
		num, err := strconv.Atoi(currNum)
		if err != nil {
			return err
		}
		newFileName = fmt.Sprintf(`%s_%d.log`, date, num+1)
	}

	logFile, err := os.OpenFile(filepath.Join(l.LogDir, newFileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Close the old file before switching to the new one
	oldFile := l.CurrentLogFile
	l.CurrentLogFile = logFile
	l.FileLog = log.New(logFile, "", log.LstdFlags)

	if err := oldFile.Close(); err != nil {
		log.Printf("WARNING failed to close old log file: %s", err.Error())
	}

	return nil
}

// Close stops the periodic cleanup goroutine and closes the current log file.
// Should be called when the logger is no longer needed.
func (l *FileLogger) Close() error {
	if l.stopCleanup != nil {
		close(l.stopCleanup)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.CurrentLogFile != nil {
		return l.CurrentLogFile.Close()
	}
	return nil
}

func (l *FileLogger) periodicCleanup() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := l.cleanupOldLogs(); err != nil {
				log.Printf("WARNING periodic cleanup failed: %s", err.Error())
			}
		case <-l.stopCleanup:
			return
		}
	}
}

func (l *FileLogger) cleanupOldLogs() error {
	if l.MaxLogAgeDays <= 0 {
		return nil
	}

	files, err := os.ReadDir(l.LogDir)
	if err != nil {
		return err
	}

	now := time.Now()
	cutoffTime := now.AddDate(0, 0, -l.MaxLogAgeDays)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".log") {
			continue
		}

		info, err := f.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(filepath.Join(l.LogDir, f.Name())); err != nil {
				log.Printf("WARNING failed to remove old log file %s: %s", f.Name(), err.Error())
			}
		}
	}

	return nil
}

func getUserLogFile(logDir string) (*os.File, error) {
	files, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	y, m, d := now.Date()
	date := fmt.Sprintf(`%d-%d-%d`, y, m, d)

	var filteredFiles []string

	for _, f := range files {
		filename := f.Name()
		if strings.HasPrefix(filename, date) {
			filteredFiles = append(filteredFiles, filename[:len(filename)-4])
		}
	}

	var logFileName string

	if len(filteredFiles) > 0 {
		logFileName = filteredFiles[0]
		maxNum := 0

		for _, filename := range filteredFiles {
			parts := strings.Split(filename, "_")
			if len(parts) != 2 {
				continue
			}
			num, err := strconv.Atoi(parts[1])
			if err != nil {
				continue
			}
			if num > maxNum {
				maxNum = num
				logFileName = filename
			}
		}
	} else {
		logFileName = fmt.Sprintf(`%s_1`, date)
	}

	logFileName = fmt.Sprintf(`%s.log`, logFileName)
	logFile, err := os.OpenFile(filepath.Join(logDir, logFileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return logFile, nil
}
