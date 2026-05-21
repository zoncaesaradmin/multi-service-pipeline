package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	targetDir := filepath.Join(t.TempDir(), "nested", "dir")
	if err := EnsureDir(targetDir, 0755); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}

	info, err := os.Stat(targetDir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected created path to be a directory")
	}

	if err := EnsureDir("", 0755); err != nil {
		t.Fatalf("EnsureDir(empty) error = %v", err)
	}
}
