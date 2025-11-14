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

To create a new logger, use the `NewLogger` function. It takes four arguments:

- `devMode`: A boolean that, when `true`, enables more verbose logging for debugging purposes.
- `appDir`: A string representing the directory where log files will be stored. This is relative to the user's home directory. For example, an `appDir` of `"MyApp"` will store logs in `~_user_/MyApp/logs`.
- `maxLogAgeDays`: Maximum age of log files in days before automatic cleanup (0 = no cleanup).
- `minLevel`: Minimum log level to write. Logs below this level are ignored.

```go
// Production: only log warnings and above
logger, err := flog.NewLogger(false, "MyApp", 30, flog.LogLevelWarn)
if err != nil {
    log.Fatalf("Failed to create logger: %v", err)
}
defer logger.Close()

// Development: log everything including debug messages
logger, err := flog.NewLogger(true, "MyApp", 30, flog.LogLevelDebug)
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

```go
// Log a fatal error
logger.LogFatal(errors.New("this is a fatal error"))

// Log an error
logger.LogError(errors.New("this is an error"))

// Log a warning
logger.LogWarn("This is a warning message.")

// Log an info message
logger.LogInfo("This is an info message.")

// Log a debug message
logger.LogDebug("This is a debug message.")
```

## License

[MIT](https://choosealicense.com/licenses/mit/)
