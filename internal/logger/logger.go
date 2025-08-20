package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/http"
)

var (
	instance *Logger
	once     sync.Once
)

type Logger struct {
	logger     zerolog.Logger
	mu         sync.RWMutex
	level      zerolog.Level
	outputs    []io.Writer
	fileWriter *lumberjack.Logger
}

type Config struct {
	Level      string `json:"level"`
	Console    bool   `json:"console"`
	File       bool   `json:"file"`
	FilePath   string `json:"file_path"`
	MaxSize    int    `json:"max_size"`    // megabytes
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`     // days
	Compress   bool   `json:"compress"`
	JSONFormat bool   `json:"json_format"`
	Caller     bool   `json:"caller"`
}

func Get() *Logger {
	once.Do(func() {
		instance = &Logger{}
		instance.initialize(DefaultConfig())
	})
	return instance
}

func Initialize(cfg Config) {
	Get().initialize(cfg)
}

func DefaultConfig() Config {
	dataDir := getDataDir()
	return Config{
		Level:      "info",
		Console:    true,
		File:       true,
		FilePath:   filepath.Join(dataDir, "logs", "winramp.log"),
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
		JSONFormat: false,
		Caller:     true,
	}
}

func (l *Logger) initialize(cfg Config) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Parse log level
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}
	l.level = level

	// Reset outputs
	l.outputs = []io.Writer{}

	// Console output
	if cfg.Console {
		var consoleWriter io.Writer
		if cfg.JSONFormat {
			consoleWriter = os.Stdout
		} else {
			consoleWriter = zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: "15:04:05",
				FormatLevel: func(i interface{}) string {
					return strings.ToUpper(fmt.Sprintf("%-5s", i))
				},
				FormatMessage: func(i interface{}) string {
					return fmt.Sprintf("%s", i)
				},
				FormatFieldName: func(i interface{}) string {
					return fmt.Sprintf("%s:", i)
				},
				FormatFieldValue: func(i interface{}) string {
					return fmt.Sprintf("%s", i)
				},
			}
		}
		l.outputs = append(l.outputs, consoleWriter)
	}

	// File output
	if cfg.File {
		// Ensure log directory exists
		logDir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Failed to create log directory: %v\n", err)
		}

		l.fileWriter = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		l.outputs = append(l.outputs, l.fileWriter)
	}

	// Create multi-writer
	multi := zerolog.MultiLevelWriter(l.outputs...)

	// Create logger
	l.logger = zerolog.New(multi).
		Level(level).
		With().
		Timestamp().
		Logger()

	// Add caller info if enabled
	if cfg.Caller {
		l.logger = l.logger.With().Caller().Logger()
	}

	// Set global logger
	log.Logger = l.logger
}

func (l *Logger) Debug(msg string, fields ...Field) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	event := l.logger.Debug()
	for _, field := range fields {
		event = field.Apply(event)
	}
	event.Msg(msg)
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	event := l.logger.Info()
	for _, field := range fields {
		event = field.Apply(event)
	}
	event.Msg(msg)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	event := l.logger.Warn()
	for _, field := range fields {
		event = field.Apply(event)
	}
	event.Msg(msg)
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	event := l.logger.Error()
	for _, field := range fields {
		event = field.Apply(event)
	}
	event.Msg(msg)
}

func (l *Logger) Fatal(msg string, fields ...Field) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	event := l.logger.Fatal()
	for _, field := range fields {
		event = field.Apply(event)
	}
	event.Msg(msg)
}

func (l *Logger) Panic(msg string, fields ...Field) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	event := l.logger.Panic()
	for _, field := range fields {
		event = field.Apply(event)
	}
	event.Msg(msg)
}

func (l *Logger) WithField(key string, value interface{}) *LoggerContext {
	return &LoggerContext{
		logger: l.logger.With().Interface(key, value).Logger(),
	}
}

func (l *Logger) WithFields(fields map[string]interface{}) *LoggerContext {
	ctx := l.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &LoggerContext{
		logger: ctx.Logger(),
	}
}

func (l *Logger) SetLevel(level string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		return err
	}
	
	l.level = lvl
	l.logger = l.logger.Level(lvl)
	return nil
}

func (l *Logger) GetLevel() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level.String()
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.fileWriter != nil {
		return l.fileWriter.Close()
	}
	return nil
}

type LoggerContext struct {
	logger zerolog.Logger
}

func (lc *LoggerContext) Debug(msg string) {
	lc.logger.Debug().Msg(msg)
}

func (lc *LoggerContext) Info(msg string) {
	lc.logger.Info().Msg(msg)
}

func (lc *LoggerContext) Warn(msg string) {
	lc.logger.Warn().Msg(msg)
}

func (lc *LoggerContext) Error(msg string) {
	lc.logger.Error().Msg(msg)
}

type Field struct {
	Key   string
	Value interface{}
}

func (f Field) Apply(event *zerolog.Event) *zerolog.Event {
	return event.Interface(f.Key, f.Value)
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Package-level convenience functions
func Debug(msg string, fields ...Field) {
	Get().Debug(msg, fields...)
}

func Info(msg string, fields ...Field) {
	Get().Info(msg, fields...)
}

func Warn(msg string, fields ...Field) {
	Get().Warn(msg, fields...)
}

func ErrorLog(msg string, fields ...Field) {
	Get().Error(msg, fields...)
}

func Fatal(msg string, fields ...Field) {
	Get().Fatal(msg, fields...)
}

func Panic(msg string, fields ...Field) {
	Get().Panic(msg, fields...)
}

func WithField(key string, value interface{}) *LoggerContext {
	return Get().WithField(key, value)
}

func WithFields(fields map[string]interface{}) *LoggerContext {
	return Get().WithFields(fields)
}

func getDataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "WinRamp")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "winramp")
}

// Middleware for HTTP logging
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:        200,
		}
		
		next.ServeHTTP(wrapped, r)
		
		Get().Info("HTTP Request",
			String("method", r.Method),
			String("path", r.URL.Path),
			Int("status", wrapped.status),
			Duration("duration", time.Since(start)),
			String("remote_addr", r.RemoteAddr),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}