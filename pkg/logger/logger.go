package logger

import (
    "fmt"
    "log"
    "os"
    "runtime"
    "time"
)

type LogLevel int

const (
    DEBUG LogLevel = iota
    INFO
    WARN
    ERROR
    FATAL
)

func (l LogLevel) String() string {
    switch l {
    case DEBUG:
        return "DEBUG"
    case INFO:
        return "INFO"
    case WARN:
        return "WARN"
    case ERROR:
        return "ERROR"
    case FATAL:
        return "FATAL"
    default:
        return "UNKNOWN"
    }
}

type Logger struct {
    level  LogLevel
    file   *os.File
    logger *log.Logger
}

func NewLogger(level LogLevel, logFile string) *Logger {
    file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal("Failed to open log file:", err)
    }

    return &Logger{
        level:  level,
        file:   file,
        logger: log.New(file, "", 0),
    }
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
    if level < l.level {
        return
    }

    _, file, line, ok := runtime.Caller(2)
    if !ok {
        file = "unknown"
        line = 0
    }

    timestamp := time.Now().Format("2006-01-02 15:04:05.000")
    msg := fmt.Sprintf(format, args...)
    
    l.logger.Printf("[%s] [%s] %s:%d - %s", timestamp, level.String(), file, line, msg)

    if level == FATAL {
        l.file.Close()
        os.Exit(1)
    }
}

func (l *Logger) Debug(format string, args ...interface{}) {
    l.log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
    l.log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
    l.log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
    l.log(ERROR, format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
    l.log(FATAL, format, args...)
}

func (l *Logger) Close() {
    l.file.Close()
}
