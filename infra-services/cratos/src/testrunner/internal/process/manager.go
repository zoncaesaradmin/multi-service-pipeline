package process

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"testgomodule/internal/config"
)

// Manager handles service process lifecycle
type Manager struct {
	config    config.ServiceConfig
	process   *exec.Cmd
	isRunning bool
}

// NewManager creates a new process manager
func NewManager(cfg config.ServiceConfig) *Manager {
	return &Manager{
		config: cfg,
	}
}

// StartService starts the service process
func (m *Manager) StartService() error {
	if m.isRunning {
		return fmt.Errorf("service is already running")
	}

	// Check if binary exists
	if !filepath.IsAbs(m.config.BinaryPath) {
		abs, err := filepath.Abs(m.config.BinaryPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		m.config.BinaryPath = abs
	}

	if _, err := os.Stat(m.config.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("service binary not found: %s", m.config.BinaryPath)
	}

	// Start the process
	m.process = exec.Command(m.config.BinaryPath)
	m.process.Stdout = os.Stdout
	m.process.Stderr = os.Stderr

	// Set environment variables
	m.process.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", m.config.Port),
		"ENVIRONMENT=test",
	)

	if err := m.process.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	m.isRunning = true
	return nil
}

// StopService stops the service process
func (m *Manager) StopService() error {
	if !m.isRunning || m.process == nil {
		return nil
	}

	// Try graceful shutdown first
	if err := m.process.Process.Signal(syscall.SIGTERM); err != nil {
		// If graceful shutdown fails, force kill
		if err := m.process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill service process: %w", err)
		}
	}

	// Wait for process to exit
	go func() {
		m.process.Wait()
	}()

	// Give it some time to shut down gracefully
	time.Sleep(2 * time.Second)

	m.isRunning = false
	m.process = nil
	return nil
}

// WaitForReady waits for the service to be ready to accept requests
func (m *Manager) WaitForReady() error {
	if !m.isRunning {
		return fmt.Errorf("service is not running")
	}

	timeout := time.After(m.config.Timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	healthURL := fmt.Sprintf("http://localhost:%d/health", m.config.Port)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for service to be ready")
		case <-ticker.C:
			resp, err := http.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// IsRunning returns whether the service is currently running
func (m *Manager) IsRunning() bool {
	return m.isRunning && m.process != nil
}

// GetPID returns the process ID of the service
func (m *Manager) GetPID() int {
	if m.process != nil && m.process.Process != nil {
		return m.process.Process.Pid
	}
	return 0
}
