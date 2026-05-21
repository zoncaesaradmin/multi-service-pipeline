package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfFilePath(t *testing.T) {
	absPath := "/tmp/absolute.yaml"
	if got := ResolveConfFilePath(absPath); got != absPath {
		t.Fatalf("ResolveConfFilePath() = %q, want %q", got, absPath)
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_config.yaml")
	if err := os.WriteFile(testFile, []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(originalWD)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if got := ResolveConfFilePath("test_config.yaml"); got != "test_config.yaml" {
		t.Fatalf("ResolveConfFilePath() = %q, want local test path", got)
	}

	t.Setenv("SERVICE_HOME", "/srv/app")
	if got := ResolveConfFilePath("kafka-consumer.yaml"); got != filepath.Join("/srv/app", "conf", "kafka-consumer.yaml") {
		t.Fatalf("ResolveConfFilePath() = %q, want SERVICE_HOME-based path", got)
	}
}

func TestLoadConfigMap(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := []byte("host: localhost\nport: 9092\nenabled: true\n")
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config := LoadConfigMap(configPath)
	if config["host"] != "localhost" {
		t.Fatalf("host = %v, want localhost", config["host"])
	}
	if config["port"] != 9092 {
		t.Fatalf("port = %v, want 9092", config["port"])
	}
	if config["enabled"] != true {
		t.Fatalf("enabled = %v, want true", config["enabled"])
	}
}

func TestLoadConfigMapWithError(t *testing.T) {
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validPath, []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("write valid config: %v", err)
	}

	if _, err := LoadConfigMapWithError(validPath); err != nil {
		t.Fatalf("LoadConfigMapWithError(valid) error = %v", err)
	}

	if _, err := LoadConfigMapWithError(filepath.Join(tmpDir, "missing.yaml")); err == nil {
		t.Fatal("expected error for missing config file")
	}

	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte(":\n  broken"), 0644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	if _, err := LoadConfigMapWithError(invalidPath); err == nil {
		t.Fatal("expected parse error for invalid YAML")
	}
}
