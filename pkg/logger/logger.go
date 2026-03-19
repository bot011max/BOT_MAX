package logger

import (
    "log"
    "os"
    "runtime"
    "fmt"
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

type Logger struct {
    level LogLevel
    file  *os.File
}

func NewLogger(level LogLevel, logFile string) *Logger {
    file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal(err)
    }
    
    return &Logger{
        level: level,
        file:  file,
    }
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
    if level < l.level {
        return
    }
    
    _, file, line, _ := runtime.Caller(2)
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    
    msg := fmt.Sprintf(format, args...)
    logLine := fmt.Sprintf("[%s] [%s] %s:%d - %s\n", 
        timestamp, levelToString(level), file, line, msg)
    
    l.file.WriteString(logLine)
    
    if level == FATAL {
        os.Exit(1)
    }
}

func levelToString(level LogLevel) string {
    switch level {
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
