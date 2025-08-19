package app

import (
	"context"
	"sync"

	"servicegomodule/internal/config"
	"servicegomodule/internal/processing"
	"sharedgomodule/logging"
)

// Application represents the main application instance that holds configuration and dependencies
type Application struct {
	rawconfig          *config.RawConfig
	logger             logging.Logger
	processingPipeline *processing.Pipeline
	mutex              sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewApplication creates a new application instance
func NewApplication(cfg *config.RawConfig, logger logging.Logger) *Application {
	ctx, cancel := context.WithCancel(context.Background())

	// Create processing pipeline with configuration from config file
	processingConfig := processing.DefaultConfig(cfg)
	processingPipeline := processing.NewPipeline(processingConfig, logger.WithField("module", "processing"))

	return &Application{
		rawconfig:          cfg,
		logger:             logger,
		processingPipeline: processingPipeline,
		ctx:                ctx,
		cancel:             cancel,
	}
}

// Config returns the application configuration
func (app *Application) Config() *config.RawConfig {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.rawconfig
}

// Logger returns the application logger
func (app *Application) Logger() logging.Logger {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.logger
}

// Context returns the application context
func (app *Application) Context() context.Context {
	return app.ctx
}

// ProcessingPipeline returns the processing pipeline instance
func (app *Application) ProcessingPipeline() *processing.Pipeline {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.processingPipeline
}

// Start starts the application and its processing pipeline
func (app *Application) Start() error {
	app.logger.Info("Starting application...")

	// Start the processing pipeline
	if err := app.processingPipeline.Start(); err != nil {
		app.logger.Errorw("Failed to start processing pipeline", "error", err)
		return err
	}

	app.logger.Info("Application started successfully")
	return nil
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown() error {
	app.logger.Info("Shutting down application...")

	// Stop the processing pipeline
	if app.processingPipeline != nil {
		if err := app.processingPipeline.Stop(); err != nil {
			app.logger.Errorw("Error stopping processing pipeline", "error", err)
		}
	}

	// Cancel the application context
	app.cancel()

	app.logger.Info("Application shutdown completed")
	return nil
}

// IsShuttingDown returns true if the application is shutting down
func (app *Application) IsShuttingDown() bool {
	select {
	case <-app.ctx.Done():
		return true
	default:
		return false
	}
}
