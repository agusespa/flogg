package logger

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
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
}

// NewLogger creates a new FileLogger instance.
//
// Parameters:
//   - devMode: a boolean indicating whether the logger should output more detailed messages suitable for debugging.
//   - appDir: a string representing the subdirectory where log files should be stored. This should be a relative path, and will result in `user_home_dir/[appDir]/logs`.
func NewLogger(devMode bool, appDir string) *FileLogger {
	if devMode {
		log.Println("INFO logger running in development mode")
	}

	currentUser, err := user.Current()
	if err != nil {
		message := fmt.Sprintf("FATAL failed getting the current os user: %s", err.Error())
		log.Fatal(message)
	}

	homeDir := currentUser.HomeDir
	logDir := filepath.Join(homeDir, appDir, "logs")
	if err = os.MkdirAll(logDir, 0755); err != nil {
		message := fmt.Sprintf("FATAL failed creating log directory: %s", err.Error())
		log.Fatal(message)
	}

	var fileLogger *log.Logger
	logFile, err := getUserLogFile(logDir)
	if err != nil {
		message := fmt.Sprintf("FATAL failed getting log file: %s", err.Error())
		log.Fatal(message)
	} else {
		fileLogger = log.New(logFile, "", log.LstdFlags)
	}

	return &FileLogger{DevMode: devMode, LogDir: logDir, CurrentLogFile: logFile, FileLog: fileLogger}
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
	err := l.refreshLogFile()
	if err != nil {
		message := fmt.Sprintf("FATAL failed refreshing log file: %s", err.Error())
		log.Fatal(message)
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
	l.CurrentLogFile = logFile
	l.FileLog = log.New(logFile, "", log.LstdFlags)
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
		for i := 1; i < len(filteredFiles); i++ {
			latestNum := strings.Split(logFileName, "_")[1]
			currentNum := strings.Split(filteredFiles[i], "_")[1]
			if currentNum > latestNum {
				logFileName = filteredFiles[i]
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
