package testdata

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

// TestScenario represents a test scenario
type TestScenario struct {
	Name           string                 `yaml:"name" json:"name"`
	Description    string                 `yaml:"description" json:"description"`
	Input          map[string]interface{} `yaml:"input" json:"input"`
	ExpectedOutput map[string]interface{} `yaml:"expected_output" json:"expected_output"`
	Timeout        time.Duration          `yaml:"timeout" json:"timeout"`
}

// Loader handles loading test scenarios from files
type Loader struct {
	scenariosPath string
}

// NewLoader creates a new test data loader
func NewLoader(scenariosPath string) *Loader {
	return &Loader{
		scenariosPath: scenariosPath,
	}
}

// LoadAllScenarios loads all test scenarios from the scenarios directory
func (l *Loader) LoadAllScenarios() ([]TestScenario, error) {
	scenarios := make([]TestScenario, 0)

	// Walk through scenarios directory
	err := filepath.Walk(l.scenariosPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if info.IsDir() || (filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml") {
			return nil
		}

		scenario, err := l.LoadScenario(path)
		if err != nil {
			return fmt.Errorf("failed to load scenario from %s: %w", path, err)
		}

		scenarios = append(scenarios, scenario)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk scenarios directory: %w", err)
	}

	return scenarios, nil
}

// LoadScenario loads a single test scenario from a file
func (l *Loader) LoadScenario(filepath string) (TestScenario, error) {
	var scenario TestScenario

	data, err := os.ReadFile(filepath)
	if err != nil {
		return scenario, fmt.Errorf("failed to read scenario file: %w", err)
	}

	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return scenario, fmt.Errorf("failed to parse scenario file: %w", err)
	}

	// Apply defaults
	if scenario.Timeout == 0 {
		scenario.Timeout = 30 * time.Second
	}

	return scenario, nil
}

// LoadFixture loads fixture data from a JSON or YAML file
func (l *Loader) LoadFixture(filename string) (map[string]interface{}, error) {
	fixturePath := filepath.Join(l.scenariosPath, "..", "fixtures", filename)
	
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file: %w", err)
	}

	var fixture map[string]interface{}
	if err := yaml.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("failed to parse fixture file: %w", err)
	}

	return fixture, nil
}
