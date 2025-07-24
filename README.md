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

To create a new logger, use the `NewLogger` function. It takes two arguments:

- `devMode`: A boolean that, when `true`, enables more verbose logging for debugging purposes.
- `appDir`: A string representing the directory where log files will be stored. This is relative to the user's home directory. For example, an `appDir` of `"MyApp"` will store logs in `~_user_/MyApp/logs`.

```go
logger, err := flog.NewLogger(true, "MyApp")
if err != nil {
    log.Fatalf("Failed to create logger: %v", err)
}
```

### Logging Messages

`flogg` supports the following log levels:

- `LogFatal`: Logs a fatal error and exits the application.
- `LogError`: Logs an error message.
- `LogWarn`: Logs a warning message.
- `LogInfo`: Logs an informational message.
- `LogDebug`: Logs a debug message (only logged to the console if `devMode` is `true`).

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
