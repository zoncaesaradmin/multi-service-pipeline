package ruleenginelib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// LoggerConfig contains config needed to setup a logger instance
type LoggerConfig struct {
	// relates to container name, used in mapping to log directory
	Component string
	// relates to logical group of functionality
	// and seen in actual log entries with "service"
	// service/deployment with multiple containers
	ServiceName string
	// full path to the log file
	FilePath string
	Level    zerolog.Level
}

// Logger with zerolog instance
type Logger struct {
	logger *zerolog.Logger
}

func (log *Logger) Info(msg string) {
	log.logger.Info().Msg(msg)
}

func (log *Logger) Debug(msg string) {
	log.logger.Debug().Msg(msg)
}

func (log *Logger) Warn(msg string) {
	log.logger.Warn().Msg(msg)
}

func (log *Logger) Trace(msg string) {
	log.logger.Trace().Msg(msg)
}

func (log *Logger) Error(msg string, err error) {
	log.logger.Error().Stack().Err(errors.WithStack(err)).Msg(msg)
}

func (log *Logger) Fatal(msg string, err error) {
	log.logger.Fatal().Stack().Err(errors.WithStack(err)).Msg(msg)
}

func (log *Logger) Panic(msg string, err error) {
	log.logger.Panic().Stack().Err(errors.WithStack(err)).Msg(msg)
}

func (log *Logger) Infof(format string, params ...interface{}) {
	log.logger.Info().Msgf(format, params...)
}

func (log *Logger) Debugf(format string, params ...interface{}) {
	log.logger.Debug().Msgf(format, params...)
}

func (log *Logger) Warnf(format string, params ...interface{}) {
	log.logger.Warn().Msgf(format, params...)
}

func (log *Logger) Tracef(format string, params ...interface{}) {
	log.logger.Trace().Msgf(format, params...)
}

func (log *Logger) Errorf(msg string, err error, params ...interface{}) {
	log.logger.Error().Stack().Err(errors.WithStack(err)).Msgf(msg, params...)
}

func (log *Logger) Fatalf(msg string, err error, params ...interface{}) {
	log.logger.Fatal().Stack().Err(errors.WithStack(err)).Msgf(msg, params...)
}

func (log *Logger) Panicf(msg string, err error, params ...interface{}) {
	log.logger.Panic().Stack().Err(errors.WithStack(err)).Msgf(msg, params...)
}

// WithField returns a new Logger instance with the specified field added to the context
func (log *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := log.logger.With().Interface(key, value).Logger()
	return &Logger{logger: &newLogger}
}

// WithFields returns a new Logger instance with multiple fields added to the context
func (log *Logger) WithFields(fields map[string]interface{}) *Logger {
	context := log.logger.With()
	for key, value := range fields {
		context = context.Interface(key, value)
	}
	newLogger := context.Logger()
	return &Logger{logger: &newLogger}
}

func CreateLoggerInstance(svcName string, logFilePath string, level zerolog.Level) *Logger {
	logConfig := LoggerConfig{
		ServiceName: svcName,
		FilePath:    logFilePath,
		Level:       level,
	}

	fmt.Printf("[Logger Debug] Attempting to use log file path: %s\n", logConfig.FilePath)
	if logFilePath != "" {
		if _, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			panic(fmt.Sprintf("Failed to open log file %v: %v", logFilePath, err))
		}
	}

	var logfile io.Writer
	var err error

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	//zerolog.SetGlobalLevel(zerolog.DebugLevel)

	zerolog.TimeFieldFormat = time.RFC3339Nano
	//zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string { return filepath.Base(file) + ":" + strconv.Itoa(line) }

	if _, err := os.Stat(logConfig.FilePath); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(logConfig.FilePath), 0755)
	}

	logfile, err = os.OpenFile(logConfig.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file %s: %v\n", logConfig.FilePath, err)
		os.Exit(-1)
	}

	logger := zerolog.New(logfile).With().Timestamp().Str("service", logConfig.ServiceName).Logger()
	logger.Level(logConfig.Level)

	return &Logger{logger: &logger}
}

func zerologLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel // Default to Info if unknown level
	}
}
