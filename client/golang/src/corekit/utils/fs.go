package utils

import "os"

// EnsureDir creates a directory path if it does not already exist.
// Empty paths are ignored so callers can safely pass optional env-derived paths.
func EnsureDir(path string, perm os.FileMode) error {
	if path == "" {
		return nil
	}
	return os.MkdirAll(path, perm)
}
