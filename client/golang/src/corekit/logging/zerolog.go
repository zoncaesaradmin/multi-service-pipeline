package logging

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog"
)

// ZerologLogger implements Logger interface using zerolog
type ZerologLogger struct {
	mu       sync.RWMutex
	logger   zerolog.Logger
	level    Level
	fields   Fields
	context  context.Context
	errorKey string
	config   *LoggerConfig
	file     *os.File
}

// NewLoggerWithConfig creates a new ZerologLogger with comprehensive configuration
func NewLoggerWithConfig(config *LoggerConfig) (*ZerologLogger, error) {
	// Open the log file for writing (create if not exists, append if exists)
	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", config.FilePath, err)
	}

	//set global logger to lowest level so that
	// explicit logger instance level can always take effect
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// Configure zerolog to write to the file with JSON format
	logger := zerolog.New(file).With().
		Timestamp().
		Str("service", config.ServiceName).
		Logger().
		Level(levelToZerolog(config.Level))

	return &ZerologLogger{
		logger:   logger,
		level:    config.Level,
		fields:   make(Fields),
		errorKey: "error",
		config:   config,
		file:     file,
	}, nil
}

// Close closes the log file
func (z *ZerologLogger) Close() error {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.file != nil {
		err := z.file.Close()
		z.file = nil
		return err
	}
	return nil
}

// SetLevel sets the logging level
func (z *ZerologLogger) SetLevel(level Level) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.level = level
	z.logger = z.logger.Level(levelToZerolog(level))
}

// GetLevel returns the current logging level
func (z *ZerologLogger) GetLevel() Level {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.level
}

// IsLevelEnabled checks if the given level is enabled
func (z *ZerologLogger) IsLevelEnabled(level Level) bool {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return level >= z.level
}

// levelToZerolog converts our Level to zerolog.Level
func levelToZerolog(level Level) zerolog.Level {
	switch level {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	case PanicLevel:
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// getEvent creates a zerolog event with current fields
func (z *ZerologLogger) getEvent(level Level) *zerolog.Event {
	var event *zerolog.Event

	switch level {
	case DebugLevel:
		event = z.logger.Debug()
	case InfoLevel:
		event = z.logger.Info()
	case WarnLevel:
		event = z.logger.Warn()
	case ErrorLevel:
		event = z.logger.Error()
	case FatalLevel:
		event = z.logger.Fatal()
	case PanicLevel:
		event = z.logger.Panic()
	default:
		event = z.logger.Info()
	}

	// Add fields
	z.mu.RLock()
	for key, value := range z.fields {
		event = event.Interface(key, value)
	}
	z.mu.RUnlock()

	return event
}

// Basic logging methods
func (z *ZerologLogger) Debug(msg string) {
	if !z.IsLevelEnabled(DebugLevel) {
		return
	}
	z.getEvent(DebugLevel).Msg(msg)
}

func (z *ZerologLogger) Info(msg string) {
	if !z.IsLevelEnabled(InfoLevel) {
		return
	}
	z.getEvent(InfoLevel).Msg(msg)
}

func (z *ZerologLogger) Warn(msg string) {
	if !z.IsLevelEnabled(WarnLevel) {
		return
	}
	z.getEvent(WarnLevel).Msg(msg)
}

func (z *ZerologLogger) Error(msg string) {
	if !z.IsLevelEnabled(ErrorLevel) {
		return
	}
	z.getEvent(ErrorLevel).Msg(msg)
}

func (z *ZerologLogger) Fatal(msg string) {
	z.getEvent(FatalLevel).Msg(msg)
}

func (z *ZerologLogger) Panic(msg string) {
	z.getEvent(PanicLevel).Msg(msg)
}

// Formatted logging methods
func (z *ZerologLogger) Debugf(format string, args ...interface{}) {
	if !z.IsLevelEnabled(DebugLevel) {
		return
	}
	z.getEvent(DebugLevel).Msgf(format, args...)
}

func (z *ZerologLogger) Infof(format string, args ...interface{}) {
	if !z.IsLevelEnabled(InfoLevel) {
		return
	}
	z.getEvent(InfoLevel).Msgf(format, args...)
}

func (z *ZerologLogger) Warnf(format string, args ...interface{}) {
	if !z.IsLevelEnabled(WarnLevel) {
		return
	}
	z.getEvent(WarnLevel).Msgf(format, args...)
}

func (z *ZerologLogger) Errorf(format string, args ...interface{}) {
	if !z.IsLevelEnabled(ErrorLevel) {
		return
	}
	z.getEvent(ErrorLevel).Msgf(format, args...)
}

func (z *ZerologLogger) Fatalf(format string, args ...interface{}) {
	z.getEvent(FatalLevel).Msgf(format, args...)
}

func (z *ZerologLogger) Panicf(format string, args ...interface{}) {
	z.getEvent(PanicLevel).Msgf(format, args...)
}

// Variadic logging methods
func (z *ZerologLogger) Debugw(msg string, keysAndValues ...interface{}) {
	z.WithFields(keysAndValuesToFields(keysAndValues...)).Debug(msg)
}

func (z *ZerologLogger) Infow(msg string, keysAndValues ...interface{}) {
	z.WithFields(keysAndValuesToFields(keysAndValues...)).Info(msg)
}

func (z *ZerologLogger) Warnw(msg string, keysAndValues ...interface{}) {
	z.WithFields(keysAndValuesToFields(keysAndValues...)).Warn(msg)
}

func (z *ZerologLogger) Errorw(msg string, keysAndValues ...interface{}) {
	z.WithFields(keysAndValuesToFields(keysAndValues...)).Error(msg)
}

func (z *ZerologLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	z.WithFields(keysAndValuesToFields(keysAndValues...)).Fatal(msg)
}

func (z *ZerologLogger) Panicw(msg string, keysAndValues ...interface{}) {
	z.WithFields(keysAndValuesToFields(keysAndValues...)).Panic(msg)
}

// Structured logging with fields
func (z *ZerologLogger) WithFields(fields Fields) Logger {
	newLogger := z.Clone().(*ZerologLogger)
	newLogger.mu.Lock()
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	newLogger.mu.Unlock()
	return newLogger
}

func (z *ZerologLogger) WithField(key string, value interface{}) Logger {
	return z.WithFields(Fields{key: value})
}

func (z *ZerologLogger) WithError(err error) Logger {
	if err == nil {
		return z
	}
	return z.WithField(z.errorKey, err.Error())
}

// Context-aware logging
func (z *ZerologLogger) WithContext(ctx context.Context) Logger {
	newLogger := z.Clone().(*ZerologLogger)
	newLogger.context = ctx
	// Update the underlying zerolog logger with context
	newLogger.logger = newLogger.logger.With().Ctx(ctx).Logger()
	return newLogger
}

// Log at specific level
func (z *ZerologLogger) Log(level Level, msg string) {
	if !z.IsLevelEnabled(level) {
		return
	}
	z.getEvent(level).Msg(msg)
}

func (z *ZerologLogger) Logf(level Level, format string, args ...interface{}) {
	if !z.IsLevelEnabled(level) {
		return
	}
	z.getEvent(level).Msgf(format, args...)
}

func (z *ZerologLogger) Logw(level Level, msg string, keysAndValues ...interface{}) {
	z.WithFields(keysAndValuesToFields(keysAndValues...)).Log(level, msg)
}

// Clone creates a copy of the logger (shares the same file and config)
func (z *ZerologLogger) Clone() Logger {
	z.mu.RLock()
	defer z.mu.RUnlock()

	newFields := make(Fields)
	for k, v := range z.fields {
		newFields[k] = v
	}

	return &ZerologLogger{
		logger:   z.logger,
		level:    z.level,
		fields:   newFields,
		context:  z.context,
		errorKey: z.errorKey,
		config:   z.config,
		file:     z.file, // Share the same file
	}
}
