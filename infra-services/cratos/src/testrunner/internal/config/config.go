package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the testrunner configuration
type Config struct {
	Service  ServiceConfig  `yaml:"service"`
	MessageBus MessageBusConfig `yaml:"messagebus"`
	Testdata   TestdataConfig   `yaml:"testdata"`
	Validation ValidationConfig `yaml:"validation"`
}

// ServiceConfig contains service-specific settings
type ServiceConfig struct {
	BinaryPath string        `yaml:"binaryPath"`
	Port       int           `yaml:"port"`
	Timeout    time.Duration `yaml:"timeout"`
}

// MessageBusConfig contains message bus settings
type MessageBusConfig struct {
	Type          string            `yaml:"type"`
	KafkaConfig   *KafkaConfig      `yaml:"kafka,omitempty"`
	LocalConfig   *LocalConfig      `yaml:"local,omitempty"`
	ExtraSettings map[string]string `yaml:"extra_settings,omitempty"`
}

// KafkaConfig contains Kafka-specific settings
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
}

// LocalConfig contains local message bus settings
type LocalConfig struct {
	BufferSize int `yaml:"bufferSize"`
}

// TestdataConfig contains test data settings
type TestdataConfig struct {
	ScenariosPath string `yaml:"scenariosPath"`
	FixturesPath  string `yaml:"fixturesPath"`
}

// ValidationConfig contains validation settings
type ValidationConfig struct {
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int           `yaml:"maxRetries"`
	RetryDelay time.Duration `yaml:"retryDelay"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	if config.Service.Port == 0 {
		config.Service.Port = 8080
	}
	if config.Service.Timeout == 0 {
		config.Service.Timeout = 30 * time.Second
	}
	if config.Validation.Timeout == 0 {
		config.Validation.Timeout = 60 * time.Second
	}
	if config.Validation.MaxRetries == 0 {
		config.Validation.MaxRetries = 3
	}
	if config.Validation.RetryDelay == 0 {
		config.Validation.RetryDelay = 1 * time.Second
	}

	return &config, nil
}
