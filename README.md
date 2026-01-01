# Flogg

A simple file-based logging library for Go.

`flogg` provides a straightforward way to log messages to files, with support for log rotation and different log levels.

## Installation

To use `flogg` in your Go project, you can use `go get`:

```bash
go get github.com/agusespa/flogg
```

## Usage

### Importing the library

```go
import "github.com/agusespa/flogg"
```

### Creating a new logger instance

To create a new logger, use the `NewLogger` function. It takes five arguments:

- `devMode`: A boolean that, when `true`, enables more verbose logging for debugging purposes.
- `appDir`: A string representing the directory where log files will be stored. This is relative to the user's home directory. For example, an `appDir` of `"MyApp"` will store logs in `~_user_/MyApp/logs`.
- `maxLogAgeDays`: Maximum age of log files in days before automatic cleanup (0 = no cleanup).
- `minLevel`: Minimum log level to write. Logs below this level are ignored.
- `format`: Log format - `LogFormatText` for human-readable logs or `LogFormatJSON` for structured JSON logs.

```go
// Production: JSON format, only warnings and above
logger, err := flog.NewLogger(false, "MyApp", 30, flog.LogLevelWarn, flog.LogFormatJSON)
if err != nil {
    log.Fatalf("Failed to create logger: %v", err)
}
defer logger.Close()

// Development: text format, log everything
logger, err := flog.NewLogger(true, "MyApp", 30, flog.LogLevelDebug, flog.LogFormatText)
if err != nil {
    log.Fatalf("Failed to create logger: %v", err)
}
defer logger.Close()
```

### Logging Messages

`flogg` supports the following log levels (from lowest to highest priority):

- `LogDebug`: Logs a debug message (only logged to the console if `devMode` is `true`).
- `LogInfo`: Logs an informational message.
- `LogWarn`: Logs a warning message.
- `LogError`: Logs an error message.
- `LogFatal`: Logs a fatal error and exits the application.

### Log Level Filtering

You can control which logs are written by setting a minimum log level. Only logs at or above the specified level will be written to the file:

- `LogLevelDebug`: Log everything (debug, info, warn, error, fatal)
- `LogLevelInfo`: Log info and above (info, warn, error, fatal)
- `LogLevelWarn`: Log warnings and above (warn, error, fatal)
- `LogLevelError`: Log errors only (error, fatal)
- `LogLevelFatal`: Log only fatal errors

### Log Formats

`flogg` supports two output formats:

**Text Format** (`LogFormatText`) - Human-readable format:
```
2025/11/14 08:04:38 INFO user logged in user_id=123 ip=192.168.1.1 action=login
```

**JSON Format** (`LogFormatJSON`) - Structured JSON for parsing and analysis:
```json
{"action":"login","ip":"192.168.1.1","level":"INFO","message":"user logged in","time":"2025-11-14T08:04:38+01:00","user_id":123}
```

```go
// Simple logging
logger.LogInfo("This is an info message.")
logger.LogError(errors.New("this is an error"))
logger.LogWarn("This is a warning message.")
logger.LogDebug("This is a debug message.")
logger.LogFatal(errors.New("this is a fatal error")) // exits the application

// Structured logging with key-value pairs
logger.LogInfoWith("user logged in", map[string]interface{}{
    "user_id": 123,
    "ip": "192.168.1.1",
    "action": "login",
})

logger.LogErrorWith(errors.New("database connection failed"), map[string]interface{}{
    "host": "localhost",
    "port": 5432,
    "retry_count": 3,
})
```
