package logger

import (
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

type Logger interface {
	LogFatal(err error)
	LogError(err error)
	LogWarn(message string)
	LogInfo(message string)
	LogDebug(message string)
}

type FileLogger struct {
	DevMode        bool
	LogDir         string
	CurrentLogFile *os.File
	FileLog        *log.Logger
	MaxLogAgeDays  int
	stopCleanup    chan struct{}
	mu             sync.Mutex
}

// NewLogger creates a new FileLogger instance.
//
// Parameters:
//   - devMode: a boolean indicating whether the logger should output more detailed messages suitable for debugging.
//   - appDir: a string representing the subdirectory where log files should be stored. This should be a relative path, and will result in `user_home_dir/[appDir]/logs`.
//   - maxLogAgeDays: maximum age of log files in days before cleanup (0 = no cleanup).
func NewLogger(devMode bool, appDir string, maxLogAgeDays int) (*FileLogger, error) {
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
	message := fmt.Sprintf("FATAL %s", err.Error())
	l.logToFile(message)
	log.Fatal(message)
}

func (l *FileLogger) LogError(err error) {
	message := fmt.Sprintf("ERROR %s", err.Error())
	log.Println(message)
	l.logToFile(message)
}

func (l *FileLogger) LogWarn(message string) {
	message = fmt.Sprintf("WARNING %s", message)
	log.Println(message)
	l.logToFile(message)
}

func (l *FileLogger) LogInfo(message string) {
	message = fmt.Sprintf("INFO %s", message)
	log.Println(message)
	l.logToFile(message)
}

func (l *FileLogger) LogDebug(message string) {
	message = fmt.Sprintf("DEBUG %s", message)
	l.logToFile(message)

	if l.DevMode {
		log.Println(message)
	}
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
