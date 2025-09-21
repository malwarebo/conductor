package utils

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	Service       string                 `json:"service"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}

type Logger struct {
	service string
	level   LogLevel
}

var defaultLogger = &Logger{
	service: "gopay",
	level:   LevelInfo,
}

func init() {
	if os.Getenv("LOG_LEVEL") == "debug" {
		defaultLogger.level = LevelDebug
	}
}

func NewLogger(service string) *Logger {
	return &Logger{
		service: service,
		level:   defaultLogger.level,
	}
}

func (l *Logger) Debug(ctx context.Context, message string, fields ...map[string]interface{}) {
	l.log(ctx, LevelDebug, message, fields...)
}

func (l *Logger) Info(ctx context.Context, message string, fields ...map[string]interface{}) {
	l.log(ctx, LevelInfo, message, fields...)
}

func (l *Logger) Warn(ctx context.Context, message string, fields ...map[string]interface{}) {
	l.log(ctx, LevelWarn, message, fields...)
}

func (l *Logger) Error(ctx context.Context, message string, fields ...map[string]interface{}) {
	l.log(ctx, LevelError, message, fields...)
}

func (l *Logger) log(ctx context.Context, level LogLevel, message string, fields ...map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp:     time.Now(),
		Level:         l.levelString(level),
		Message:       message,
		Service:       l.service,
		CorrelationID: GetCorrelationID(ctx),
		UserID:        GetUserID(ctx),
	}

	if len(fields) > 0 {
		entry.Fields = fields[0]
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	log.Println(string(jsonData))
}

func (l *Logger) levelString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value("correlation_id").(string); ok {
		return id
	}
	return ""
}

func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value("user_id").(string); ok {
		return id
	}
	return ""
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, "correlation_id", id)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, "user_id", userID)
}

func Debug(ctx context.Context, message string, fields ...map[string]interface{}) {
	defaultLogger.Debug(ctx, message, fields...)
}

func Info(ctx context.Context, message string, fields ...map[string]interface{}) {
	defaultLogger.Info(ctx, message, fields...)
}

func Warn(ctx context.Context, message string, fields ...map[string]interface{}) {
	defaultLogger.Warn(ctx, message, fields...)
}

func Error(ctx context.Context, message string, fields ...map[string]interface{}) {
	defaultLogger.Error(ctx, message, fields...)
}
